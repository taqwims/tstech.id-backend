package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tstech/backend/internal/config"
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
	"gorm.io/gorm"
)

type SaaSService struct {
	db         *gorm.DB
	cfg        *config.Config
	saasRepo   repository.SaaSRepository
	userRepo   *repository.UserRepo
	paymentSvc *PaymentService
}

func NewSaaSService(
	db *gorm.DB,
	cfg *config.Config,
	saasRepo repository.SaaSRepository,
	userRepo *repository.UserRepo,
	paymentSvc *PaymentService,
) *SaaSService {
	return &SaaSService{
		db:         db,
		cfg:        cfg,
		saasRepo:   saasRepo,
		userRepo:   userRepo,
		paymentSvc: paymentSvc,
	}
}

// DTOs
type SubscribeRequest struct {
	SaaSProductID uint   `json:"saas_product_id"`
	SaaSPlanID    uint   `json:"saas_plan_id"`
	TenantName    string `json:"tenant_name"`    // e.g. "SMK Negeri 1 Surabaya"
	SubdomainSlug string `json:"subdomain_slug"` // e.g. "smkn1"
	BillingCycle  string `json:"billing_cycle"`  // "monthly" or "yearly"
	PaymentMethod string `json:"payment_method"` // "mayar" or "manual_transfer"
}

type SaaSSubscriptionDTO struct {
	model.SaaSSubscription
	DaysRemaining int    `json:"days_remaining"`
	IsExpired     bool   `json:"is_expired"`
	StatusBadge   string `json:"status_badge"` // "active", "expiring_soon", "expired", "pending_payment"
	DirectAppURL  string `json:"direct_app_url"`
}

type LicenseVerifyDTO struct {
	Valid         bool      `json:"valid"`
	IsActive      bool      `json:"is_active"`
	Status        string    `json:"status"` // "active", "expiring_soon", "expired", "trial", "pending_payment"
	TenantID      uint      `json:"tenant_id"`
	TenantName    string    `json:"tenant_name"`
	SubdomainSlug string    `json:"subdomain_slug"`
	FullSubdomain string    `json:"full_subdomain"`
	CustomDomain  string    `json:"custom_domain,omitempty"`
	ProductSlug   string    `json:"product_slug"`
	ProductName   string    `json:"product_name"`
	PlanCode      string    `json:"plan_code"`
	PlanName      string    `json:"plan_name"`
	MaxUsers      int       `json:"max_users"`
	MaxStorageGB  int       `json:"max_storage_gb"`
	Features      string    `json:"features"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	DaysRemaining int       `json:"days_remaining"`
	OwnerEmail    string    `json:"owner_email,omitempty"`
	Message       string    `json:"message"`
}

var reservedSubdomains = map[string]bool{
	"admin": true, "api": true, "app": true, "auth": true, "billing": true,
	"cdn": true, "dev": true, "mail": true, "ns1": true, "ns2": true,
	"root": true, "staging": true, "static": true, "test": true, "www": true,
	"demo": true, "docs": true, "dashboard": true, "portal": true, "hub": true,
}

var subdomainRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

// ValidateSubdomain checks format and availability
func (s *SaaSService) ValidateSubdomain(productSlug, rawSubdomain string) (string, string, error) {
	slug := strings.ToLower(strings.TrimSpace(rawSubdomain))
	slug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(slug, "-")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	if len(slug) < 3 {
		return "", "", errors.New("subdomain minimal 3 karakter (huruf, angka, atau tanda hubung '-')")
	}
	if len(slug) > 40 {
		return "", "", errors.New("subdomain maksimal 40 karakter")
	}
	if !subdomainRegex.MatchString(slug) {
		return "", "", errors.New("format subdomain tidak valid (hanya huruf kecil, angka, dan '-')")
	}
	if reservedSubdomains[slug] {
		return "", "", fmt.Errorf("subdomain '%s' adalah kata yang dilindungi dan tidak dapat digunakan", slug)
	}

	prod, err := s.saasRepo.GetProductBySlug(productSlug)
	if err != nil {
		return "", "", fmt.Errorf("produk SaaS tidak ditemukan: %w", err)
	}

	baseDomain := prod.BaseDomain
	if s.cfg != nil && s.cfg.SaaSBaseDomain != "" {
		baseDomain = fmt.Sprintf("%s.%s", prod.Slug, s.cfg.SaaSBaseDomain)
	} else if baseDomain == "" {
		baseDomain = fmt.Sprintf("%s.tstech.id", prod.Slug)
	}
	fullSubdomain := fmt.Sprintf("%s.%s", slug, baseDomain)

	available, err := s.saasRepo.CheckSubdomainAvailable(fullSubdomain)
	if err != nil {
		return "", "", err
	}
	if !available {
		return "", "", fmt.Errorf("subdomain '%s' sudah digunakan oleh instansi/pengguna lain", fullSubdomain)
	}

	return slug, fullSubdomain, nil
}

// GetActiveProducts returns all active SaaS products with plans
func (s *SaaSService) GetActiveProducts() ([]model.SaaSProduct, error) {
	return s.saasRepo.GetActiveProducts()
}

// GetProductBySlug returns product detail
func (s *SaaSService) GetProductBySlug(slug string) (*model.SaaSProduct, error) {
	return s.saasRepo.GetProductBySlug(slug)
}

// Subscribe handles client subscription purchase & invoice creation
func (s *SaaSService) Subscribe(userID uint, req SubscribeRequest) (*model.SaaSSubscription, *model.Invoice, error) {
	if req.TenantName == "" {
		return nil, nil, errors.New("nama instansi/usaha (tenant) wajib diisi")
	}

	prod, err := s.saasRepo.GetProductByID(req.SaaSProductID)
	if err != nil {
		return nil, nil, errors.New("produk SaaS tidak ditemukan")
	}

	plan, err := s.saasRepo.GetPlanByID(req.SaaSPlanID)
	if err != nil || plan.SaaSProductID != prod.ID {
		return nil, nil, errors.New("paket langganan tidak valid untuk produk ini")
	}

	cleanSlug, fullDomain, err := s.ValidateSubdomain(prod.Slug, req.SubdomainSlug)
	if err != nil {
		return nil, nil, err
	}

	// Calculate price
	var price float64
	var periodDuration time.Duration
	billingCycle := strings.ToLower(req.BillingCycle)
	if billingCycle == "yearly" {
		price = plan.PriceYearly
		if price <= 0 {
			price = plan.PriceMonthly * 12 * 0.85 // 15% discount fallback
		}
		periodDuration = 365 * 24 * time.Hour
	} else {
		billingCycle = "monthly"
		price = plan.PriceMonthly
		periodDuration = 30 * 24 * time.Hour
	}

	now := time.Now()
	subNum := fmt.Sprintf("SUB-%s-%04d", now.Format("200601"), time.Now().Unix()%10000)

	sub := &model.SaaSSubscription{
		SubscriptionNumber: subNum,
		UserID:             userID,
		SaaSProductID:      prod.ID,
		SaaSPlanID:         plan.ID,
		TenantName:         strings.TrimSpace(req.TenantName),
		SubdomainSlug:      cleanSlug,
		FullSubdomain:      fullDomain,
		Status:             "pending_payment",
		BillingCycle:       billingCycle,
		PriceAmount:        price,
		StartDate:          now,
		EndDate:            now.Add(periodDuration),
		AutoRenew:          true,
	}

	if err := s.saasRepo.CreateSubscription(sub); err != nil {
		return nil, nil, fmt.Errorf("gagal membuat data langganan: %w", err)
	}

	// Create Invoice
	invNum := fmt.Sprintf("INV-%s-%04d", now.Format("20060102"), time.Now().Unix()%10000)
	cycleText := "1 Bulan"
	if billingCycle == "yearly" {
		cycleText = "1 Tahun"
	}
	invTitle := fmt.Sprintf("Langganan %s - %s (%s)", prod.Name, sub.TenantName, cycleText)

	paymentMethod := strings.ToLower(req.PaymentMethod)
	if paymentMethod != "mayar" && paymentMethod != "manual_transfer" {
		paymentMethod = "mayar"
	}

	inv := &model.Invoice{
		InvoiceNumber:  invNum,
		UserID:         userID,
		SubscriptionID: &sub.ID,
		Title:          invTitle,
		Description:    fmt.Sprintf("Paket: %s | Subdomain: https://%s", plan.Name, sub.FullSubdomain),
		Amount:         price,
		TotalAmount:    price,
		Status:         "unpaid",
		PaymentMethod:  paymentMethod,
		DueDate:        now.Add(24 * time.Hour),
	}

	// If Mayar selected, try to create Mayar payment session
	if paymentMethod == "mayar" && s.paymentSvc != nil {
		user, _ := s.userRepo.FindByID(userID)
		userName := "Pelanggan TsTech"
		userEmail := "client@tstech.id"
		userPhone := "081234567890"
		if user != nil {
			if user.Name != "" {
				userName = user.Name
			}
			if user.Email != "" {
				userEmail = user.Email
			}
			if user.Phone != "" {
				userPhone = user.Phone
			}
		}

		payReq := &CreatePaymentRequest{
			Amount:        int64(price),
			PaymentType:   "subscription",
			PaymentMethod: "mayar",
			BuyerName:     userName,
			BuyerEmail:    userEmail,
			BuyerPhone:    userPhone,
			ProductName:   invTitle,
		}

		settings, err := s.paymentSvc.GetAdminSettings()
		if err == nil && settings != nil && settings.MayarEnabled {
			paymentURL, transID, qrURL, err := s.paymentSvc.callMayarAPI(0, payReq, settings)
			if err == nil {
				inv.PaymentURL = paymentURL
				inv.GatewayTransID = transID
				inv.QRCodeURL = qrURL
			}
		}
	}

	if err := s.saasRepo.CreateInvoice(inv); err != nil {
		return nil, nil, fmt.Errorf("gagal membuat invoice: %w", err)
	}

	return sub, inv, nil
}

// GetUserSubscriptions returns formatted list with remaining days and status
func (s *SaaSService) GetUserSubscriptions(userID uint) ([]SaaSSubscriptionDTO, error) {
	subs, err := s.saasRepo.GetUserSubscriptions(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var dtos []SaaSSubscriptionDTO

	for _, sub := range subs {
		daysRem := int(time.Until(sub.EndDate).Hours() / 24)
		isExp := now.After(sub.EndDate)
		statusBadge := sub.Status

		if sub.Status == "active" {
			if isExp {
				statusBadge = "expired"
			} else if daysRem <= 7 {
				statusBadge = "expiring_soon"
			}
		}

		protocol := "https"
		if s.cfg != nil && s.cfg.SaaSSSOProtocol != "" {
			protocol = s.cfg.SaaSSSOProtocol
		} else if s.cfg != nil && s.cfg.APIEnv == "development" {
			protocol = "http"
		}

		directURL := fmt.Sprintf("%s://%s", protocol, sub.FullSubdomain)
		if s.cfg != nil && s.cfg.SaaSDevURLOverride != "" {
			devTpl := strings.ReplaceAll(s.cfg.SaaSDevURLOverride, "{tenant}", sub.SubdomainSlug)
			if sub.SaaSProduct != nil {
				devTpl = strings.ReplaceAll(devTpl, "{product}", sub.SaaSProduct.Slug)
			}
			directURL = strings.TrimRight(devTpl, "/")
		}

		dtos = append(dtos, SaaSSubscriptionDTO{
			SaaSSubscription: sub,
			DaysRemaining:    daysRem,
			IsExpired:        isExp,
			StatusBadge:      statusBadge,
			DirectAppURL:     directURL,
		})
	}

	return dtos, nil
}

// GetSubscriptionByID returns subscription details
func (s *SaaSService) GetSubscriptionByID(userID uint, subID uint, isAdmin bool) (*model.SaaSSubscription, error) {
	sub, err := s.saasRepo.GetSubscriptionByID(subID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && sub.UserID != userID {
		return nil, errors.New("akses ditolak")
	}
	return sub, nil
}

// GenerateSSOToken creates a signed 1-minute JWT token to auto-login to the satellite SaaS app
func (s *SaaSService) GenerateSSOToken(userID uint, subID uint) (string, error) {
	sub, err := s.saasRepo.GetSubscriptionByID(subID)
	if err != nil {
		return "", errors.New("langganan SaaS tidak ditemukan")
	}
	if sub.UserID != userID {
		return "", errors.New("akses ditolak")
	}

	now := time.Now()
	if now.After(sub.EndDate) && sub.Status != "trial" {
		return "", errors.New("masa aktif langganan telah berakhir. Silakan perpanjang langganan untuk mengakses aplikasi.")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", errors.New("user tidak ditemukan")
	}

	// Secret key used for signing: product secret or main jwt secret
	secret := s.cfg.JWTSecret
	if sub.Product != nil && sub.Product.APISecretKey != "" {
		secret = sub.Product.APISecretKey
	} else if sub.SaaSProduct != nil && sub.SaaSProduct.APISecretKey != "" {
		secret = sub.SaaSProduct.APISecretKey
	}

	planCode := ""
	planName := ""
	maxUsers := 0
	maxStorageGB := 0
	if sub.SaaSPlan != nil {
		planCode = sub.SaaSPlan.Code
		planName = sub.SaaSPlan.Name
		maxUsers = sub.SaaSPlan.MaxUsers
		maxStorageGB = sub.SaaSPlan.MaxStorageGB
	}

	productSlug := ""
	if sub.Product != nil {
		productSlug = sub.Product.Slug
	} else if sub.SaaSProduct != nil {
		productSlug = sub.SaaSProduct.Slug
	}

	claims := jwt.MapClaims{
		"iss":                 "tstech.id",
		"sub":                 fmt.Sprintf("%d", user.ID),
		"email":               user.Email,
		"name":                user.Name,
		"role":                "admin",
		"product_slug":        productSlug,
		"tenant_slug":         sub.SubdomainSlug,
		"full_subdomain":      sub.FullSubdomain,
		"subscription_number": sub.SubscriptionNumber,
		"subscription_id":     sub.SubscriptionNumber,
		"plan_id":             sub.SaaSPlanID,
		"plan_code":           planCode,
		"plan_name":           planName,
		"max_users":           maxUsers,
		"max_storage_gb":      maxStorageGB,
		"exp":                 time.Now().Add(5 * time.Minute).Unix(), // 5 minutes valid window
		"iat":                 time.Now().Unix(),
		"jti":                 generateRandomJTI(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("gagal menandatangani SSO token: %w", err)
	}

	protocol := "https"
	if s.cfg != nil && s.cfg.SaaSSSOProtocol != "" {
		protocol = s.cfg.SaaSSSOProtocol
	} else if s.cfg != nil && s.cfg.APIEnv == "development" {
		protocol = "http"
	}

	redirectURL := fmt.Sprintf("%s://%s/auth/sso?token=%s", protocol, sub.FullSubdomain, signedToken)
	if s.cfg != nil && s.cfg.SaaSDevURLOverride != "" {
		devTpl := strings.ReplaceAll(s.cfg.SaaSDevURLOverride, "{tenant}", sub.SubdomainSlug)
		if sub.SaaSProduct != nil {
			devTpl = strings.ReplaceAll(devTpl, "{product}", sub.SaaSProduct.Slug)
		}
		baseDev := strings.TrimRight(devTpl, "/")
		redirectURL = fmt.Sprintf("%s/auth/sso?token=%s", baseDev, signedToken)
	}

	return redirectURL, nil
}

// GetUserInvoices returns all invoices for a user
func (s *SaaSService) GetUserInvoices(userID uint) ([]model.Invoice, error) {
	return s.saasRepo.GetUserInvoices(userID)
}

// GetInvoiceByID returns invoice details
func (s *SaaSService) GetInvoiceByID(userID uint, invID uint, isAdmin bool) (*model.Invoice, error) {
	inv, err := s.saasRepo.GetInvoiceByID(invID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && inv.UserID != userID {
		return nil, errors.New("akses ditolak")
	}
	return inv, nil
}

// SubmitManualPaymentProof stores client manual transfer proof
func (s *SaaSService) SubmitManualPaymentProof(userID uint, invID uint, proofURL, bankName, senderName string) (*model.Invoice, error) {
	inv, err := s.saasRepo.GetInvoiceByID(invID)
	if err != nil {
		return nil, errors.New("invoice tidak ditemukan")
	}
	if inv.UserID != userID {
		return nil, errors.New("akses ditolak")
	}

	inv.ProofURL = proofURL
	inv.BankName = bankName
	inv.AccountHolder = senderName
	inv.PaymentMethod = "manual_transfer"
	inv.Status = "waiting_confirmation"

	if err := s.saasRepo.UpdateInvoice(inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// ApproveManualInvoice marks invoice as paid and activates the subscription
func (s *SaaSService) ApproveManualInvoice(invID uint) (*model.Invoice, error) {
	inv, err := s.saasRepo.GetInvoiceByID(invID)
	if err != nil {
		return nil, errors.New("invoice tidak ditemukan")
	}

	now := time.Now()
	inv.Status = "paid"
	inv.PaidAt = &now

	if err := s.saasRepo.UpdateInvoice(inv); err != nil {
		return nil, err
	}

	// Activate or extend subscription
	if inv.SubscriptionID != nil {
		sub, err := s.saasRepo.GetSubscriptionByID(*inv.SubscriptionID)
		if err == nil {
			sub.Status = "active"
			// If already active and end_date is in the future, extend from end_date
			baseTime := now
			if sub.EndDate.After(now) {
				baseTime = sub.EndDate
			}

			if sub.BillingCycle == "yearly" {
				sub.EndDate = baseTime.Add(365 * 24 * time.Hour)
			} else {
				sub.EndDate = baseTime.Add(30 * 24 * time.Hour)
			}
			sub.StartDate = now
			s.saasRepo.UpdateSubscription(sub)
		}
	}

	return inv, nil
}

// HandleMayarWebhook processes webhook from Mayar.id
func (s *SaaSService) HandleMayarWebhook(event, status, transactionID, invoiceNumber string) error {
	if status == "SUCCESS" || status == "SETTLED" || status == "PAID" || event == "payment.received" {
		inv, err := s.saasRepo.GetInvoiceByNumber(invoiceNumber)
		if err != nil {
			return fmt.Errorf("invoice not found: %s", invoiceNumber)
		}
		if inv.Status == "paid" {
			return nil // already handled
		}

		now := time.Now()
		inv.Status = "paid"
		inv.PaidAt = &now
		inv.GatewayTransID = transactionID
		s.saasRepo.UpdateInvoice(inv)

		if inv.SubscriptionID != nil {
			sub, err := s.saasRepo.GetSubscriptionByID(*inv.SubscriptionID)
			if err == nil {
				sub.Status = "active"
				baseTime := now
				if sub.EndDate.After(now) {
					baseTime = sub.EndDate
				}
				if sub.BillingCycle == "yearly" {
					sub.EndDate = baseTime.Add(365 * 24 * time.Hour)
				} else {
					sub.EndDate = baseTime.Add(30 * 24 * time.Hour)
				}
				s.saasRepo.UpdateSubscription(sub)
			}
		}
	}
	return nil
}

// VerifyLicense checks subscription validity for satellite SaaS apps
func (s *SaaSService) VerifyLicense(domainOrSlug, secretKey string) (*LicenseVerifyDTO, error) {
	clean := strings.TrimSpace(domainOrSlug)
	clean = strings.TrimPrefix(clean, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.Split(clean, ":")[0] // strip port
	clean = strings.Trim(clean, "/")
	clean = strings.ToLower(clean)

	if clean == "" {
		return nil, errors.New("subdomain atau domain wajib diisi")
	}

	sub, err := s.saasRepo.GetSubscriptionBySubdomain(clean)
	if err != nil {
		// Fallback: If passed full subdomain like "smkn1.schola.tstech.id", try first segment "smkn1"
		parts := strings.Split(clean, ".")
		if len(parts) > 1 {
			sub, err = s.saasRepo.GetSubscriptionBySubdomain(parts[0])
		}
	}

	if err != nil || sub == nil {
		return &LicenseVerifyDTO{
			Valid:    false,
			IsActive: false,
			Status:   "not_found",
			Message:  fmt.Sprintf("Langganan untuk domain '%s' tidak ditemukan atau belum terdaftar", clean),
		}, nil
	}

	// If product has API secret and secretKey is provided, check matching
	prodSecret := ""
	if sub.Product != nil && sub.Product.APISecretKey != "" {
		prodSecret = sub.Product.APISecretKey
	} else if sub.SaaSProduct != nil && sub.SaaSProduct.APISecretKey != "" {
		prodSecret = sub.SaaSProduct.APISecretKey
	}
	if prodSecret != "" && secretKey != "" {
		if prodSecret != secretKey {
			return nil, errors.New("API Secret Key tidak valid")
		}
	}

	now := time.Now()
	daysRem := int(time.Until(sub.EndDate).Hours() / 24)
	isExpired := now.After(sub.EndDate)

	isActive := false
	status := sub.Status

	if sub.Status == "active" {
		if isExpired {
			status = "expired"
			isActive = false
		} else {
			isActive = true
			if daysRem <= 7 {
				status = "expiring_soon"
			}
		}
	} else if sub.Status == "trial" {
		if isExpired {
			status = "expired"
			isActive = false
		} else {
			isActive = true
		}
	} else {
		isActive = false
	}

	dto := &LicenseVerifyDTO{
		Valid:         true,
		IsActive:      isActive,
		Status:        status,
		TenantID:      sub.ID,
		TenantName:    sub.TenantName,
		SubdomainSlug: sub.SubdomainSlug,
		FullSubdomain: sub.FullSubdomain,
		CustomDomain:  sub.CustomDomain,
		StartDate:     sub.StartDate,
		EndDate:       sub.EndDate,
		DaysRemaining: daysRem,
	}

	if sub.Product != nil {
		dto.ProductSlug = sub.Product.Slug
		dto.ProductName = sub.Product.Title
	} else if sub.SaaSProduct != nil {
		dto.ProductSlug = sub.SaaSProduct.Slug
		dto.ProductName = sub.SaaSProduct.Name
	}
	if sub.SaaSPlan != nil {
		dto.PlanCode = sub.SaaSPlan.Code
		dto.PlanName = sub.SaaSPlan.Name
		dto.MaxUsers = sub.SaaSPlan.MaxUsers
		dto.MaxStorageGB = sub.SaaSPlan.MaxStorageGB
		dto.Features = sub.SaaSPlan.Features
	}
	if sub.User != nil {
		dto.OwnerEmail = sub.User.Email
	}

	if isActive {
		dto.Message = fmt.Sprintf("Langganan aktif (%s s/d %s)", sub.TenantName, sub.EndDate.Format("02 Jan 2006"))
	} else {
		dto.Message = fmt.Sprintf("Masa aktif langganan %s telah berakhir pada %s. Silakan lakukan perpanjangan di tstech.id/dashboard", sub.TenantName, sub.EndDate.Format("02 Jan 2006"))
	}

	return dto, nil
}

func generateRandomJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
