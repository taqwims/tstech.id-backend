package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
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
	Features       string    `json:"features"`
	FeatureModules string    `json:"feature_modules"`
	ActiveUnits    string    `json:"active_units"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	DaysRemaining  int       `json:"days_remaining"`
	OwnerEmail     string    `json:"owner_email,omitempty"`
	Message        string    `json:"message"`
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

// GetFeaturedProducts returns all featured SaaS products for landing page
func (s *SaaSService) GetFeaturedProducts() ([]model.SaaSProduct, error) {
	return s.saasRepo.GetFeaturedProducts()
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
	if err != nil || (plan.SaaSProductID != prod.ID && (plan.ProductID == nil || *plan.ProductID != prod.ID)) {
		return nil, nil, errors.New("paket langganan tidak valid untuk produk ini")
	}

	cleanSlug, fullDomain, err := s.ValidateSubdomain(prod.Slug, req.SubdomainSlug)
	if err != nil {
		return nil, nil, err
	}

	// Calculate price & duration based on billing cycle
	var price float64
	var periodDuration time.Duration
	var cycleText string
	billingCycle := strings.ToLower(strings.TrimSpace(req.BillingCycle))

	switch billingCycle {
	case "yearly", "12_months", "annual":
		billingCycle = "yearly"
		basePrice := plan.PriceYearly
		if basePrice <= 0 {
			basePrice = plan.PriceMonthly * 12
		}
		if plan.DiscountPct > 0 {
			price = math.Round(basePrice * (1.0 - float64(plan.DiscountPct)/100.0))
		} else {
			price = basePrice
		}
		periodDuration = 365 * 24 * time.Hour
		cycleText = "1 Tahun (12 Bulan)"

	case "6_months", "semi_annual", "6-months":
		billingCycle = "6_months"
		basePrice := plan.PriceMonthly * 6
		// Give half of yearly discount if discount available
		disc := float64(plan.DiscountPct) * 0.5
		if disc > 0 {
			price = math.Round(basePrice * (1.0 - disc/100.0))
		} else {
			price = basePrice
		}
		periodDuration = 180 * 24 * time.Hour
		cycleText = "6 Bulan"

	case "3_months", "quarterly", "3-months":
		billingCycle = "3_months"
		basePrice := plan.PriceMonthly * 3
		// Give 25% of yearly discount if available
		disc := float64(plan.DiscountPct) * 0.25
		if disc > 0 {
			price = math.Round(basePrice * (1.0 - disc/100.0))
		} else {
			price = basePrice
		}
		periodDuration = 90 * 24 * time.Hour
		cycleText = "3 Bulan"

	default: // monthly
		billingCycle = "monthly"
		price = plan.PriceMonthly
		periodDuration = 30 * 24 * time.Hour
		cycleText = "1 Bulan"
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
	invTitle := fmt.Sprintf("Langganan %s - %s (%s)", prod.Name, sub.TenantName, cycleText)

	paymentMethod := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if paymentMethod == "" {
		paymentMethod = "pakasir"
	}

	inv := &model.Invoice{
		InvoiceNumber:  invNum,
		UserID:         userID,
		SubscriptionID: &sub.ID,
		Title:          invTitle,
		Description:    fmt.Sprintf("Paket: %s (%s) | Subdomain: https://%s", plan.Name, cycleText, sub.FullSubdomain),
		Amount:         price,
		TotalAmount:    price,
		Status:         "unpaid",
		PaymentMethod:  paymentMethod,
		DueDate:        now.Add(24 * time.Hour),
	}

	// Dynamic Payment Gateway Processing
	if s.paymentSvc != nil {
		settings, _ := s.paymentSvc.GetAdminSettings()
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
			PaymentMethod: paymentMethod,
			BuyerName:     userName,
			BuyerEmail:    userEmail,
			BuyerPhone:    userPhone,
			ProductName:   invTitle,
		}

		if (paymentMethod == "pakasir" || paymentMethod == "gateway") && settings != nil && settings.PakasirEnabled {
			inv.PaymentMethod = "pakasir"
			checkoutURL, transID, err := s.paymentSvc.callPakasirAPI(0, payReq, settings)
			if err == nil {
				// Pakasir checkout URL formatting with invoice ref
				slug := settings.PakasirProjectSlug
				if slug == "" {
					slug = s.cfg.PakasirProjectSlug
				}
				if slug == "" {
					slug = "tstech"
				}
				redirectURL := fmt.Sprintf("%s/dashboard/invoices", s.cfg.AppURL)
				checkoutURL = fmt.Sprintf("https://app.pakasir.com/pay/%s/%d?order_id=%s&redirect=%s",
					slug, int64(price), invNum, redirectURL)
				if settings.PakasirQRISOnly || s.cfg.PakasirQRISOnly {
					checkoutURL += "&qris_only=1"
				}
				inv.PaymentURL = checkoutURL
				inv.GatewayTransID = transID
			}
		} else if paymentMethod == "mayar" && settings != nil && settings.MayarEnabled {
			inv.PaymentMethod = "mayar"
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
		"feature_modules":     sub.CustomModules,
		"active_units":        sub.ActiveUnits,
		"exp":                 time.Now().Add(24 * time.Hour).Unix(), // 24 hours valid window
		"iat":                 time.Now().Unix(),
		"jti":                 generateRandomJTI(),
	}

	if claims["feature_modules"] == "" && sub.SaaSPlan != nil {
		claims["feature_modules"] = sub.SaaSPlan.FeatureModules
	}
	if claims["active_units"] == "" && sub.SaaSPlan != nil {
		claims["active_units"] = sub.SaaSPlan.AllowedUnits
	}
	if claims["active_units"] == "" {
		claims["active_units"] = `["sdit","mts","ma"]`
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

// CancelSubscription cancels a subscription and any pending unpaid invoices
func (s *SaaSService) CancelSubscription(subID uint) error {
	sub, err := s.saasRepo.GetSubscriptionByID(subID)
	if err != nil {
		return errors.New("langganan tidak ditemukan")
	}

	sub.Status = "cancelled"
	if err := s.saasRepo.UpdateSubscription(sub); err != nil {
		return err
	}

	// Cancel any pending/unpaid invoices associated with this subscription
	s.db.Model(&model.Invoice{}).
		Where("subscription_id = ? AND status IN ?", sub.ID, []string{"unpaid", "waiting_confirmation", "pending"}).
		Update("status", "cancelled")

	return nil
}

// DeleteSubscription permanently removes a subscription record and cancels pending invoices
func (s *SaaSService) DeleteSubscription(subID uint) error {
	sub, err := s.saasRepo.GetSubscriptionByID(subID)
	if err != nil {
		return errors.New("langganan tidak ditemukan")
	}

	// Cancel any pending/unpaid invoices associated with this subscription
	s.db.Model(&model.Invoice{}).
		Where("subscription_id = ? AND status IN ?", sub.ID, []string{"unpaid", "waiting_confirmation", "pending"}).
		Update("status", "cancelled")

	return s.saasRepo.DeleteSubscription(subID)
}

func getCycleDuration(cycle string) time.Duration {
	switch strings.ToLower(strings.TrimSpace(cycle)) {
	case "yearly", "12_months", "annual":
		return 365 * 24 * time.Hour
	case "6_months", "semi_annual", "6-months":
		return 180 * 24 * time.Hour
	case "3_months", "quarterly", "3-months":
		return 90 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
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

			sub.EndDate = baseTime.Add(getCycleDuration(sub.BillingCycle))
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
				sub.EndDate = baseTime.Add(getCycleDuration(sub.BillingCycle))
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

	// Dynamic Feature Modules: subscription override OR plan default
	if sub.CustomModules != "" {
		dto.FeatureModules = sub.CustomModules
	} else if sub.SaaSPlan != nil && sub.SaaSPlan.FeatureModules != "" {
		dto.FeatureModules = sub.SaaSPlan.FeatureModules
	}

	// Dynamic Active Units: subscription override OR plan allowed units OR default
	if sub.ActiveUnits != "" {
		dto.ActiveUnits = sub.ActiveUnits
	} else if sub.SaaSPlan != nil && sub.SaaSPlan.AllowedUnits != "" {
		dto.ActiveUnits = sub.SaaSPlan.AllowedUnits
	} else {
		dto.ActiveUnits = `["sdit","mts","ma"]`
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
