package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kotban/backend/internal/config"
	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/repository"
	"gorm.io/gorm"
)

type PaymentService struct {
	db        *gorm.DB
	cfg       *config.Config
	orderRepo *repository.OrderRepo
	userRepo  *repository.UserRepo
	emailSvc  *EmailService
}

func NewPaymentService(db *gorm.DB, cfg *config.Config, orderRepo *repository.OrderRepo, userRepo *repository.UserRepo, emailSvc *EmailService) *PaymentService {
	return &PaymentService{
		db:        db,
		cfg:       cfg,
		orderRepo: orderRepo,
		userRepo:  userRepo,
		emailSvc:  emailSvc,
	}
}

// CreatePaymentRequest holds the data needed to create a payment
type CreatePaymentRequest struct {
	OrderID       *uint  `json:"order_id"`
	ProjectID     *uint  `json:"project_id"`
	Amount        int64  `json:"amount"`
	PaymentType   string `json:"payment_type"`   // dp / pelunasan / full_payment
	PaymentMethod string `json:"payment_method"` // mayar / ipaymu / manual
	BuyerName     string `json:"buyer_name"`
	BuyerEmail    string `json:"buyer_email"`
	BuyerPhone    string `json:"buyer_phone"`
	ProductName   string `json:"product_name"`
	Notes         string `json:"notes"`
}

// CreatePaymentResponse is returned after creating a payment
type CreatePaymentResponse struct {
	InvoiceNumber  string              `json:"invoice_number"`
	PaymentURL     string              `json:"payment_url"`
	QRCodeURL      string              `json:"qr_code_url,omitempty"`
	QRString       string              `json:"qr_string,omitempty"`
	PaymentID      uint                `json:"payment_id"`
	PaymentMethod  string              `json:"payment_method"`
	GatewayTransID string              `json:"gateway_trans_id,omitempty"`
	Status         string              `json:"status"`
	BankAccounts   []model.BankAccount `json:"bank_accounts,omitempty"`
	Instructions   string              `json:"instructions,omitempty"`
	WhatsApp       string              `json:"whatsapp,omitempty"`
}

// GenerateInvoiceNumber creates a unique sequential invoice number
func (s *PaymentService) GenerateInvoiceNumber() string {
	now := time.Now()
	var count int64
	s.db.Model(&model.Payment{}).Count(&count)
	return fmt.Sprintf("INV/%d%02d/%04d", now.Year(), int(now.Month()), count+1)
}

// GetSettingValue helper to get a setting key from database with fallback
func (s *PaymentService) GetSettingValue(key, fallback string) string {
	var item model.PaymentSetting
	if err := s.db.Where("`key` = ? OR \"key\" = ?", key, key).First(&item).Error; err == nil && item.Value != "" {
		return item.Value
	}
	return fallback
}

// GetAdminSettings retrieves the complete payment configuration for admin
func (s *PaymentService) GetAdminSettings() (*model.PaymentSettingsConfig, error) {
	var settings []model.PaymentSetting
	s.db.Find(&settings)

	mapSettings := make(map[string]string)
	for _, setting := range settings {
		mapSettings[setting.Key] = setting.Value
	}

	pakasirEnabled := mapSettings["pakasir_enabled"] != "false"
	pakasirQRISOnly := mapSettings["pakasir_qris_only"] == "true"
	mayarEnabled := mapSettings["mayar_enabled"] != "false"
	mayarIsProd := mapSettings["mayar_is_production"] == "true"
	ipaymuEnabled := mapSettings["ipaymu_enabled"] != "false"
	ipaymuIsProd := mapSettings["ipaymu_is_production"] == "true"
	manualEnabled := mapSettings["manual_transfer_enabled"] != "false"

	var bankAccounts []model.BankAccount
	if rawBanks, ok := mapSettings["manual_bank_accounts"]; ok && rawBanks != "" {
		_ = json.Unmarshal([]byte(rawBanks), &bankAccounts)
	}

	if len(bankAccounts) == 0 {
		bankAccounts = []model.BankAccount{
			{BankName: "BCA", AccountNumber: "8735098231", AccountHolder: "PT KOTBAN SOLUSI TEKNOLOGI", Icon: "bca"},
			{BankName: "Bank Mandiri", AccountNumber: "1370019827364", AccountHolder: "PT KOTBAN SOLUSI TEKNOLOGI", Icon: "mandiri"},
			{BankName: "Bank Syariah Indonesia (BSI)", AccountNumber: "7219082341", AccountHolder: "PT KOTBAN SOLUSI TEKNOLOGI", Icon: "bsi"},
		}
	}

	cfgResp := &model.PaymentSettingsConfig{
		PakasirEnabled:        pakasirEnabled,
		PakasirProjectSlug:    s.GetSettingValue("pakasir_project_slug", s.cfg.PakasirProjectSlug),
		PakasirAPIKey:         s.GetSettingValue("pakasir_api_key", s.cfg.PakasirAPIKey),
		PakasirQRISOnly:       pakasirQRISOnly,
		MayarEnabled:          mayarEnabled,
		MayarAPIKey:           mapSettings["mayar_api_key"],
		MayarIsProduction:     mayarIsProd,
		MayarWebhookToken:     mapSettings["mayar_webhook_token"],
		IpaymuEnabled:         ipaymuEnabled,
		IpaymuVA:              s.GetSettingValue("ipaymu_va", s.cfg.IpaymuVA),
		IpaymuAPIKey:          s.GetSettingValue("ipaymu_api_key", s.cfg.IpaymuAPIKey),
		IpaymuIsProduction:    ipaymuIsProd,
		ManualTransferEnabled: manualEnabled,
		ManualBankAccounts:    bankAccounts,
		ManualInstructions:    s.GetSettingValue("manual_instructions", "Silakan lakukan transfer tepat sesuai total nominal yang tertera ke salah satu rekening resmi di atas. Setelah transfer berhasil, harap unggah bukti transfer melalui halaman ini atau kirimkan konfirmasi via WhatsApp kami agar pesanan Anda dapat langsung diproses."),
		ManualWhatsApp:        s.GetSettingValue("manual_whatsapp", s.cfg.WhatsAppNumber),
		DefaultGateway:        s.GetSettingValue("default_gateway", "customer_choice"),
	}

	return cfgResp, nil
}

// SaveAdminSettings updates payment configuration in database
func (s *PaymentService) SaveAdminSettings(cfg *model.PaymentSettingsConfig) error {
	saveKey := func(key, val string) {
		var item model.PaymentSetting
		if err := s.db.Where("`key` = ? OR \"key\" = ?", key, key).First(&item).Error; err != nil {
			s.db.Create(&model.PaymentSetting{Key: key, Value: val, UpdatedAt: time.Now()})
		} else {
			item.Value = val
			item.UpdatedAt = time.Now()
			s.db.Save(&item)
		}
	}

	saveKey("pakasir_enabled", strconv.FormatBool(cfg.PakasirEnabled))
	saveKey("pakasir_project_slug", cfg.PakasirProjectSlug)
	saveKey("pakasir_api_key", cfg.PakasirAPIKey)
	saveKey("pakasir_qris_only", strconv.FormatBool(cfg.PakasirQRISOnly))

	saveKey("mayar_enabled", strconv.FormatBool(cfg.MayarEnabled))
	saveKey("mayar_api_key", cfg.MayarAPIKey)
	saveKey("mayar_is_production", strconv.FormatBool(cfg.MayarIsProduction))
	saveKey("mayar_webhook_token", cfg.MayarWebhookToken)

	saveKey("ipaymu_enabled", strconv.FormatBool(cfg.IpaymuEnabled))
	saveKey("ipaymu_va", cfg.IpaymuVA)
	saveKey("ipaymu_api_key", cfg.IpaymuAPIKey)
	saveKey("ipaymu_is_production", strconv.FormatBool(cfg.IpaymuIsProduction))

	saveKey("manual_transfer_enabled", strconv.FormatBool(cfg.ManualTransferEnabled))
	banksJson, _ := json.Marshal(cfg.ManualBankAccounts)
	saveKey("manual_bank_accounts", string(banksJson))
	saveKey("manual_instructions", cfg.ManualInstructions)
	saveKey("manual_whatsapp", cfg.ManualWhatsApp)
	saveKey("default_gateway", cfg.DefaultGateway)

	return nil
}

// GetPublicPaymentMethods returns safe payment options for client checkout
func (s *PaymentService) GetPublicPaymentMethods() (map[string]interface{}, error) {
	cfg, err := s.GetAdminSettings()
	if err != nil {
		return nil, err
	}

	var methods []model.PublicPaymentMethod

	// 1. Pakasir (QRIS & Virtual Account)
	if cfg.PakasirEnabled {
		methods = append(methods, model.PublicPaymentMethod{
			ID:          "pakasir",
			Name:        "Pakasir Payment (QRIS & VA)",
			Description: "Pembayaran instan via QRIS (Semua Bank & E-Wallet: BCA, Mandiri, BRI, BNI, GoPay, OVO, ShopeePay, DANA) & Virtual Account",
			Badge:       "Instan & Otomatis",
			Type:        "gateway",
			IsEnabled:   true,
		})
	}

	// 2. Mayar.id
	if cfg.MayarEnabled {
		methods = append(methods, model.PublicPaymentMethod{
			ID:          "mayar",
			Name:        "Mayar.id Payment Gateway",
			Description: "QRIS Instan, Virtual Account (BCA, Mandiri, BRI, BNI), E-Wallet (GoPay, OVO, ShopeePay), & Kartu Kredit",
			Badge:       "Otomatis & Cepat",
			Type:        "gateway",
			IsEnabled:   true,
		})
	}

	// 3. iPaymu
	if cfg.IpaymuEnabled {
		methods = append(methods, model.PublicPaymentMethod{
			ID:          "ipaymu",
			Name:        "iPaymu Payment Gateway",
			Description: "Virtual Account Bank, QRIS, Indomaret & Alfamart",
			Badge:       "Otomatis",
			Type:        "gateway",
			IsEnabled:   true,
		})
	}

	// 4. Manual Bank Transfer
	if cfg.ManualTransferEnabled {
		methods = append(methods, model.PublicPaymentMethod{
			ID:           "manual",
			Name:         "Transfer Bank Manual",
			Description:  "Transfer langsung ke rekening resmi PT Kotban Solusi Teknologi (BCA / Mandiri / BSI)",
			Badge:        "Verifikasi Tim Kotban",
			Type:         "manual",
			IsEnabled:    true,
			BankAccounts: cfg.ManualBankAccounts,
			Instructions: cfg.ManualInstructions,
			WhatsApp:     cfg.ManualWhatsApp,
		})
	}

	return map[string]interface{}{
		"default_gateway": cfg.DefaultGateway,
		"methods":         methods,
	}, nil
}

// CreatePayment creates a payment record and executes the appropriate gateway logic
func (s *PaymentService) CreatePayment(req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	settings, err := s.GetAdminSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load payment settings: %w", err)
	}

	method := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if method == "" {
		if settings.DefaultGateway != "" && settings.DefaultGateway != "customer_choice" {
			method = settings.DefaultGateway
		} else if settings.PakasirEnabled {
			method = "pakasir"
		} else if settings.MayarEnabled {
			method = "mayar"
		} else if settings.IpaymuEnabled {
			method = "ipaymu"
		} else {
			method = "manual"
		}
	}

	invoiceNumber := s.GenerateInvoiceNumber()

	payment := &model.Payment{
		InvoiceNumber: invoiceNumber,
		OrderID:       req.OrderID,
		ProjectID:     req.ProjectID,
		Amount:        req.Amount,
		PaymentType:   req.PaymentType,
		PaymentMethod: method,
		Status:        "pending",
		Notes:         req.Notes,
	}

	if err := s.db.Create(payment).Error; err != nil {
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	switch method {
	case "pakasir":
		paymentURL, transID, err := s.callPakasirAPI(payment.ID, req, settings)
		if err != nil {
			return nil, fmt.Errorf("gagal membuat link pembayaran Pakasir: %w", err)
		}
		payment.PaymentURL = paymentURL
		payment.GatewayTransID = transID
		s.db.Save(payment)

		return &CreatePaymentResponse{
			InvoiceNumber:  invoiceNumber,
			PaymentURL:     paymentURL,
			PaymentID:      payment.ID,
			PaymentMethod:  "pakasir",
			GatewayTransID: transID,
			Status:         payment.Status,
		}, nil

	case "mayar":
		paymentURL, transID, qrURL, err := s.callMayarAPI(payment.ID, req, settings)
		if err != nil {
			return nil, fmt.Errorf("gagal membuat sesi pembayaran Mayar: %w", err)
		}
		payment.PaymentURL = paymentURL
		payment.GatewayTransID = transID
		s.db.Save(payment)

		return &CreatePaymentResponse{
			InvoiceNumber:  invoiceNumber,
			PaymentURL:     paymentURL,
			QRCodeURL:      qrURL,
			PaymentID:      payment.ID,
			PaymentMethod:  "mayar",
			GatewayTransID: transID,
			Status:         payment.Status,
			BankAccounts:   settings.ManualBankAccounts,
			Instructions:   settings.ManualInstructions,
			WhatsApp:       settings.ManualWhatsApp,
		}, nil

	case "ipaymu":
		paymentURL, transID, err := s.callIpaymuAPI(payment.ID, req, settings)
		if err != nil {
			return nil, fmt.Errorf("gagal membuat link pembayaran iPaymu: %w", err)
		}
		payment.PaymentURL = paymentURL
		payment.IpaymuTransID = transID
		payment.GatewayTransID = transID
		s.db.Save(payment)

		return &CreatePaymentResponse{
			InvoiceNumber:  invoiceNumber,
			PaymentURL:     paymentURL,
			PaymentID:      payment.ID,
			PaymentMethod:  "ipaymu",
			GatewayTransID: transID,
			Status:         payment.Status,
		}, nil

	case "manual", "bank_transfer":
		payment.PaymentMethod = "manual"
		s.db.Save(payment)

		return &CreatePaymentResponse{
			InvoiceNumber: invoiceNumber,
			PaymentID:     payment.ID,
			PaymentMethod: "manual",
			Status:        payment.Status,
			BankAccounts:  settings.ManualBankAccounts,
			Instructions:  settings.ManualInstructions,
			WhatsApp:      settings.ManualWhatsApp,
		}, nil

	default:
		// Default fallback to manual or mayar
		payment.PaymentMethod = "manual"
		s.db.Save(payment)
		return &CreatePaymentResponse{
			InvoiceNumber: invoiceNumber,
			PaymentID:     payment.ID,
			PaymentMethod: "manual",
			Status:        payment.Status,
			BankAccounts:  settings.ManualBankAccounts,
			Instructions:  settings.ManualInstructions,
			WhatsApp:      settings.ManualWhatsApp,
		}, nil
	}
}

// callPakasirAPI generates a Pakasir direct payment checkout URL
func (s *PaymentService) callPakasirAPI(paymentID uint, req *CreatePaymentRequest, settings *model.PaymentSettingsConfig) (string, string, error) {
	slug := settings.PakasirProjectSlug
	if slug == "" {
		slug = s.cfg.PakasirProjectSlug
	}
	if slug == "" {
		slug = "kotban"
	}

	refID := fmt.Sprintf("KTB-%d", paymentID)
	if req.OrderID != nil && *req.OrderID > 0 {
		refID = fmt.Sprintf("KTB-ORD-%d-%d", *req.OrderID, paymentID)
	} else if req.ProjectID != nil && *req.ProjectID > 0 {
		refID = fmt.Sprintf("KTB-PRJ-%d-%d", *req.ProjectID, paymentID)
	}

	redirectURL := fmt.Sprintf("%s/pemesanan/sukses?payment_id=%d", s.cfg.AppURL, paymentID)

	// Format URL: https://app.pakasir.com/pay/{slug}/{amount}?order_id={order_id}&redirect={redirectURL}
	checkoutURL := fmt.Sprintf("https://app.pakasir.com/pay/%s/%d?order_id=%s&redirect=%s",
		url.PathEscape(slug),
		req.Amount,
		url.QueryEscape(refID),
		url.QueryEscape(redirectURL),
	)

	if settings.PakasirQRISOnly || s.cfg.PakasirQRISOnly {
		checkoutURL += "&qris_only=1"
	}

	log.Printf("💳 Pakasir Payment Link Generated: %s (Order ID: %s, Project Slug: %s)", checkoutURL, refID, slug)
	return checkoutURL, refID, nil
}

// callMayarAPI calls Mayar.id REST API to create a payment checkout link and dynamic QRIS
func (s *PaymentService) callMayarAPI(paymentID uint, req *CreatePaymentRequest, settings *model.PaymentSettingsConfig) (string, string, string, error) {
	apiKey := settings.MayarAPIKey
	if apiKey == "" {
		apiKey = s.cfg.MayarAPIKey
	}
	if apiKey == "" {
		return "", "", "", errors.New("API Key Mayar belum dikonfigurasi di Admin Panel atau .env. Silakan atur API Key Mayar terlebih dahulu atau pilih metode pembayaran Pakasir / Transfer Bank.")
	}

	// Mayar Endpoints: Sandbox = api.mayar.io | Production = api.mayar.id
	baseURL := "https://api.mayar.io/hl/v1"
	if settings.MayarIsProduction || s.cfg.MayarIsProduction {
		baseURL = "https://api.mayar.id/hl/v1"
	}

	apiURL := fmt.Sprintf("%s/invoice/create", baseURL)
	log.Printf("💳 Sending Mayar API Request to %s (Environment: %s)", apiURL, baseURL)

	redirectURL := fmt.Sprintf("%s/pemesanan/sukses?payment_id=%d", s.cfg.AppURL, paymentID)

	// Sanitize buyer fields for Mayar requirements
	buyerName := strings.TrimSpace(req.BuyerName)
	if buyerName == "" {
		buyerName = "Pelanggan Kotban"
	}

	buyerEmail := strings.TrimSpace(req.BuyerEmail)
	if buyerEmail == "" || !strings.Contains(buyerEmail, "@") {
		buyerEmail = "client@kotban.com"
	}

	// Mayar strictly requires 'mobile' to have length >= 10
	cleanPhone := ""
	for _, c := range req.BuyerPhone {
		if c >= '0' && c <= '9' {
			cleanPhone += string(c)
		}
	}
	if len(cleanPhone) < 10 {
		cleanPhone = "081234567890"
	}

	productDesc := req.ProductName
	if strings.TrimSpace(productDesc) == "" {
		productDesc = fmt.Sprintf("Pembayaran Tagihan Kotban #%d", paymentID)
	}

	bodyMap := map[string]interface{}{
		"name":        buyerName,
		"email":       buyerEmail,
		"amount":      req.Amount,
		"mobile":      cleanPhone,
		"redirectUrl": redirectURL,
		"description": productDesc,
		"expiredAt":   time.Now().Add(48 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z"),
		"items": []map[string]interface{}{
			{
				"name":        productDesc,
				"description": productDesc,
				"quantity":    1,
				"rate":        req.Amount,
			},
		},
		"extraData": map[string]string{
			"paymentId":     fmt.Sprintf("%d", paymentID),
			"transactionId": fmt.Sprintf("KTB-%d", paymentID),
		},
	}

	jsonBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return "", "", "", err
	}

	log.Printf("💳 Sending Mayar API Request to %s | Payload: %s", apiURL, string(jsonBytes))

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", "", "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("❌ Mayar HTTP Error: %v", err)
		return "", "", "", fmt.Errorf("koneksi ke server Mayar gagal: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("📩 Mayar Response Status: %d | Body: %s", resp.StatusCode, string(respBody))

	var mayarResp struct {
		StatusCode int    `json:"statusCode"`
		Messages   string `json:"messages"`
		Data       struct {
			ID            string `json:"id"`
			TransactionID string `json:"transactionId"`
			Link          string `json:"link"`
			PaymentURL    string `json:"paymentUrl"`
			URL           string `json:"url"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &mayarResp); err != nil {
		log.Printf("❌ Failed to parse Mayar JSON: %v", err)
		return "", "", "", fmt.Errorf("respon dari Mayar tidak valid: %s", string(respBody))
	}

	targetURL := mayarResp.Data.Link
	if targetURL == "" {
		targetURL = mayarResp.Data.PaymentURL
	}
	if targetURL == "" {
		targetURL = mayarResp.Data.URL
	}

	transID := mayarResp.Data.TransactionID
	if transID == "" {
		transID = mayarResp.Data.ID
	}
	if transID == "" {
		transID = fmt.Sprintf("MAYAR-%d", paymentID)
	}

	// Request dynamic QRIS from Mayar for native in-app QRIS rendering
	fetchMayarQRIS := func() string {
		qrEndpoint := fmt.Sprintf("%s/qrcode/create", baseURL)
		qrBodyMap := map[string]interface{}{"amount": req.Amount}
		qrJson, _ := json.Marshal(qrBodyMap)
		qrReq, qrErr := http.NewRequest("POST", qrEndpoint, bytes.NewBuffer(qrJson))
		if qrErr != nil {
			return ""
		}
		qrReq.Header.Set("Content-Type", "application/json")
		qrReq.Header.Set("Authorization", "Bearer "+apiKey)
		qrResp, qrDoErr := client.Do(qrReq)
		if qrDoErr != nil {
			return ""
		}
		defer qrResp.Body.Close()
		if qrResp.StatusCode < 400 {
			qrBytes, _ := io.ReadAll(qrResp.Body)
			var qrRes struct {
				Data struct {
					URL string `json:"url"`
				} `json:"data"`
			}
			if json.Unmarshal(qrBytes, &qrRes) == nil && qrRes.Data.URL != "" {
				log.Printf("📱 Mayar Native Dynamic QRIS URL: %s", qrRes.Data.URL)
				return qrRes.Data.URL
			}
		}
		return ""
	}

	if resp.StatusCode < 400 && targetURL != "" {
		log.Printf("✅ Mayar Payment URL Generated: %s", targetURL)
		qrURL := fetchMayarQRIS()
		return targetURL, transID, qrURL, nil
	}

	// If invoice/create returned an error or 500, attempt single payment endpoint as fallback
	log.Printf("⚠️ Retrying Mayar via /payment/create fallback...")
	apiURLFallback := fmt.Sprintf("%s/payment/create", baseURL)
	bodyMapFallback := map[string]interface{}{
		"name":        buyerName,
		"email":       buyerEmail,
		"amount":      req.Amount,
		"mobile":      cleanPhone,
		"redirectUrl": redirectURL,
		"description": productDesc,
		"expiredAt":   time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
	}
	jsonBytesFallback, _ := json.Marshal(bodyMapFallback)
	httpReqFallback, _ := http.NewRequest("POST", apiURLFallback, bytes.NewBuffer(jsonBytesFallback))
	httpReqFallback.Header.Set("Content-Type", "application/json")
	httpReqFallback.Header.Set("Authorization", "Bearer "+apiKey)

	respFallback, errFallback := client.Do(httpReqFallback)
	if errFallback == nil {
		defer respFallback.Body.Close()
		respBodyFallback, _ := io.ReadAll(respFallback.Body)
		log.Printf("📩 Mayar /payment/create fallback Status: %d | Body: %s", respFallback.StatusCode, string(respBodyFallback))
		var fallbackResp struct {
			Data struct {
				ID            string `json:"id"`
				TransactionID string `json:"transactionId"`
				Link          string `json:"link"`
				PaymentURL    string `json:"paymentUrl"`
				URL           string `json:"url"`
			} `json:"data"`
		}
		if jsonErr := json.Unmarshal(respBodyFallback, &fallbackResp); jsonErr == nil {
			fbURL := fallbackResp.Data.Link
			if fbURL == "" {
				fbURL = fallbackResp.Data.PaymentURL
			}
			if fbURL == "" {
				fbURL = fallbackResp.Data.URL
			}
			if fbURL != "" {
				fbTransID := fallbackResp.Data.TransactionID
				if fbTransID == "" {
					fbTransID = fallbackResp.Data.ID
				}
				if fbTransID == "" {
					fbTransID = transID
				}
				qrURL := fetchMayarQRIS()
				log.Printf("✅ Mayar Payment URL Generated via fallback: %s", fbURL)
				return fbURL, fbTransID, qrURL, nil
			}
		}
	}

	errMsg := mayarResp.Messages
	if errMsg == "" {
		errMsg = string(respBody)
	}
	return "", "", "", fmt.Errorf("Mayar API error (%d): %s", resp.StatusCode, errMsg)
}

// callIpaymuAPI calls the iPaymu payment API (v2/payment endpoint)
func (s *PaymentService) callIpaymuAPI(paymentID uint, req *CreatePaymentRequest, settings *model.PaymentSettingsConfig) (string, string, error) {
	baseURL := "https://sandbox.ipaymu.com"
	if settings.IpaymuIsProduction {
		baseURL = "https://my.ipaymu.com"
	}

	va := settings.IpaymuVA
	if va == "" {
		va = s.cfg.IpaymuVA
	}
	apiKey := settings.IpaymuAPIKey
	if apiKey == "" {
		apiKey = s.cfg.IpaymuAPIKey
	}

	refID := fmt.Sprintf("KTB-%d", paymentID)
	if req.OrderID != nil && *req.OrderID > 0 {
		refID = fmt.Sprintf("KTB-ORD-%d-%d", *req.OrderID, paymentID)
	} else if req.ProjectID != nil && *req.ProjectID > 0 {
		refID = fmt.Sprintf("KTB-PRJ-%d-%d", *req.ProjectID, paymentID)
	}

	if va == "" || apiKey == "" || va == "your_virtual_account_number" {
		return "", "", errors.New("akun iPaymu (VA / API Key) belum dikonfigurasi di Admin Panel atau .env. Silakan atur iPaymu terlebih dahulu atau pilih metode pembayaran Pakasir / Transfer Bank.")
	}

	apiURL := fmt.Sprintf("%s/api/v2/payment", baseURL)
	log.Printf("💳 Sending iPaymu API Request to %s (VA: %s)", apiURL, va)

	bodyMap := map[string]interface{}{
		"product":     []string{req.ProductName},
		"qty":         []int{1},
		"price":       []int64{req.Amount},
		"returnUrl":   s.cfg.IpaymuReturnURL,
		"cancelUrl":   s.cfg.IpaymuCancelURL,
		"notifyUrl":   s.cfg.IpaymuNotifyURL,
		"referenceId": refID,
		"buyerName":   req.BuyerName,
		"buyerEmail":  req.BuyerEmail,
		"buyerPhone":  req.BuyerPhone,
	}

	jsonBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return "", "", err
	}

	h := sha256.New()
	h.Write(jsonBytes)
	bodyHash := hex.EncodeToString(h.Sum(nil))

	stringToSign := fmt.Sprintf("POST:%s:%s:%s", va, bodyHash, apiKey)

	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("va", va)
	httpReq.Header.Set("signature", signature)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("❌ iPaymu HTTP Error: %v", err)
		return "", "", fmt.Errorf("koneksi ke server iPaymu gagal: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("📩 iPaymu Response Status: %d | Body: %s", resp.StatusCode, string(respBody))

	var ipaymuResp struct {
		Status  int    `json:"Status"`
		Success bool   `json:"Success"`
		Message string `json:"Message"`
		Data    struct {
			Url           string `json:"Url"`
			SessionID     string `json:"SessionId"`
			TransactionID int64  `json:"TransactionId"`
		} `json:"Data"`
	}

	if err := json.Unmarshal(respBody, &ipaymuResp); err != nil {
		log.Printf("❌ Failed to parse iPaymu JSON: %v", err)
		return "", "", fmt.Errorf("respon dari iPaymu tidak valid: %s", string(respBody))
	}

	if ipaymuResp.Data.Url != "" {
		log.Printf("✅ iPaymu Payment URL Generated: %s", ipaymuResp.Data.Url)
		return ipaymuResp.Data.Url, fmt.Sprintf("%d", ipaymuResp.Data.TransactionID), nil
	}

	return "", "", fmt.Errorf("gagal membuat link iPaymu (%d): %s", ipaymuResp.Status, ipaymuResp.Message)
}

// ProcessPaymentSuccess handles post-payment state transitions, automatic project creation, milestone initialization, and email notifications
func (s *PaymentService) ProcessPaymentSuccess(payment *model.Payment) error {
	now := time.Now()
	payment.Status = "success"
	payment.PaidAt = &now

	var customerEmail string
	var customerName string
	var projectTitle string = "Layanan Software Kotban"
	handledProjectID := uint(0)

	// 1. Process Order association if payment was made for an Order
	if payment.OrderID != nil && *payment.OrderID > 0 {
		var order model.Order
		if err := s.db.First(&order, *payment.OrderID).Error; err == nil {
			customerEmail = order.CustomerEmail
			customerName = order.CustomerName
			projectTitle = order.ProjectName

			if payment.PaymentType == "pelunasan" {
				order.Status = "completed"
			} else {
				order.Status = "dp_paid"
			}
			s.db.Save(&order)

			// Find or Auto-Create Project
			var project model.Project
			err := s.db.Where("order_id = ?", order.ID).First(&project).Error
			if err != nil {
				// Project doesn't exist yet! Auto-create project for the client
				var clientUser model.User
				if s.userRepo != nil {
					user, _ := s.userRepo.FindByEmail(order.CustomerEmail)
					if user != nil {
						clientUser = *user
					}
				}
				if clientUser.ID == 0 {
					s.db.Where("email = ?", order.CustomerEmail).First(&clientUser)
				}

				if clientUser.ID > 0 {
					project = model.Project{
						OrderID:         &order.ID,
						ClientUserID:    clientUser.ID,
						Title:           order.ProjectName,
						Description:     order.ProjectDesc,
						Status:          "briefing",
						ProgressPercent: 5,
						TotalAmount:     order.TotalAmount,
						PaidAmount:      payment.Amount,
						Notes:           fmt.Sprintf("Paket: %s | Kategori: %s", order.PackageType, order.ServiceCategory),
					}
					if err := s.db.Create(&project).Error; err == nil {
						payment.ProjectID = &project.ID
						handledProjectID = project.ID

						// Create standard 6 default milestones
						defaultMilestones := []struct {
							title  string
							desc   string
							status string
						}{
							{"1. Briefing & Requirement Gathering", "Diskusi mendalam spesifikasi, fitur, dan workflow aplikasi", "in_progress"},
							{"2. Desain UI/UX & Wireframe", "Perancangan antarmuka pengguna interaktif & user flow", "pending"},
							{"3. Frontend & Backend Development", "Pengembangan kode sistem, database, dan integrasi API", "pending"},
							{"4. Quality Assurance & Testing", "Pengujian keamanan, performa, dan validasi fungsional", "pending"},
							{"5. UAT & Revisi Klien", "Uji coba langsung oleh klien dan penyempurnaan revisi", "pending"},
							{"6. Deployment & Serah Terima", "Peluncuran ke production server dan penyerahan akun & source code", "pending"},
						}

						for i, m := range defaultMilestones {
							s.db.Create(&model.Milestone{
								ProjectID:   project.ID,
								Title:       m.title,
								Description: m.desc,
								Status:      m.status,
								SortOrder:   i + 1,
							})
						}

						// Post initial system comment
						s.db.Create(&model.ProjectComment{
							ProjectID: project.ID,
							UserID:    clientUser.ID,
							Content:   fmt.Sprintf("🎉 Pembayaran DP (%s) sebesar Rp %d telah berhasil diverifikasi via %s. Proyek resmi dimulai pada tahap Briefing!", payment.InvoiceNumber, payment.Amount, strings.ToUpper(payment.PaymentMethod)),
							Type:      "status_update",
						})
						log.Printf("🚀 Auto-created Project #%d with 6 milestones for Order #%s", project.ID, order.OrderNumber)
					}
				}
			} else {
				// Project already exists, increment paid amount
				project.PaidAmount += payment.Amount
				if project.PaidAmount >= project.TotalAmount && project.TotalAmount > 0 {
					project.Status = "completed"
				}
				s.db.Save(&project)
				payment.ProjectID = &project.ID
				handledProjectID = project.ID
			}
		}
	}

	// 2. Process Project association if payment was made directly for a Project (e.g. milestone or direct invoice)
	if payment.ProjectID != nil && *payment.ProjectID > 0 && *payment.ProjectID != handledProjectID {
		var project model.Project
		if err := s.db.Preload("Client").First(&project, *payment.ProjectID).Error; err == nil {
			projectTitle = project.Title
			if customerEmail == "" && project.Client.Email != "" {
				customerEmail = project.Client.Email
				customerName = project.Client.Name
			}

			project.PaidAmount += payment.Amount
			if project.PaidAmount >= project.TotalAmount && project.TotalAmount > 0 {
				project.Status = "completed"
			}
			s.db.Save(&project)

			// Add system comment on project discussion
			s.db.Create(&model.ProjectComment{
				ProjectID: project.ID,
				UserID:    project.ClientUserID,
				Content:   fmt.Sprintf("💳 Pembayaran invoice %s sebesar Rp %d telah berhasil diverifikasi via %s.", payment.InvoiceNumber, payment.Amount, strings.ToUpper(payment.PaymentMethod)),
				Type:      "status_update",
			})
		}
	}

	if err := s.db.Save(payment).Error; err != nil {
		return err
	}

	// 3. Send Email Receipt & Admin Notification asynchronously
	if s.emailSvc != nil && customerEmail != "" {
		go func() {
			_ = s.emailSvc.SendPaymentReceipt(
				customerEmail,
				customerName,
				payment.InvoiceNumber,
				payment.PaymentMethod,
				payment.Amount,
				payment.PaymentType,
				projectTitle,
			)

			_ = s.emailSvc.SendAdminNotification(
				fmt.Sprintf("Pembayaran Diterima #%s - Rp %d", payment.InvoiceNumber, payment.Amount),
				fmt.Sprintf("<p>Pembayaran sebesar <strong>Rp %d</strong> untuk invoice <strong>#%s</strong> telah berhasil diterima via <strong>%s</strong>.</p><p>Pelanggan: %s (%s)</p>", payment.Amount, payment.InvoiceNumber, payment.PaymentMethod, customerName, customerEmail),
			)
		}()
	}

	return nil
}

// HandleMayarNotification processes webhook notifications from Mayar.id
func (s *PaymentService) HandleMayarNotification(data map[string]interface{}) error {
	log.Printf("🔔 Received Mayar webhook: %+v", data)

	event, _ := data["event"].(string)

	var targetID string
	var status string = "success"

	if dataMap, ok := data["data"].(map[string]interface{}); ok {
		if idVal, exists := dataMap["id"].(string); exists && idVal != "" {
			targetID = idVal
		}
		if invVal, exists := dataMap["invoice_id"].(string); exists && invVal != "" && targetID == "" {
			targetID = invVal
		}
		if payVal, exists := dataMap["payment_id"].(string); exists && payVal != "" && targetID == "" {
			targetID = payVal
		}
		if trxVal, exists := dataMap["transactionId"].(string); exists && trxVal != "" && targetID == "" {
			targetID = trxVal
		}
		if stVal, exists := dataMap["status"].(string); exists {
			st := strings.ToLower(strings.TrimSpace(stVal))
			if st == "paid" || st == "settled" || st == "success" {
				status = "success"
			} else if st == "expired" {
				status = "expire"
			} else if st == "cancelled" || st == "cancel" {
				status = "cancel"
			}
		}
	}

	if targetID == "" {
		if idVal, exists := data["id"].(string); exists && idVal != "" {
			targetID = idVal
		}
		if orderIDVal, exists := data["order_id"].(string); exists && orderIDVal != "" {
			targetID = orderIDVal
		}
	}

	var payment model.Payment
	err := s.db.Where("gateway_trans_id = ? OR invoice_number = ?", targetID, targetID).First(&payment).Error
	if err != nil && targetID != "" {
		// Try parsing payment ID from targetID if format is KTB-x
		parts := strings.Split(targetID, "-")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if payID, parseErr := strconv.Atoi(lastPart); parseErr == nil && payID > 0 {
				err = s.db.Where("id = ?", uint(payID)).First(&payment).Error
			}
		}
	}

	if err != nil {
		// Fallback try latest pending mayar payment
		err = s.db.Where("payment_method = ? AND status = ?", "mayar", "pending").Order("created_at DESC").First(&payment).Error
		if err != nil {
			return fmt.Errorf("payment not found for Mayar transaction: %s", targetID)
		}
	}

	rawData, _ := json.Marshal(data)
	payment.GatewayData = string(rawData)

	if event == "payment.received" || event == "payment.paid" || status == "success" {
		payment.PaymentMethod = "mayar"
		return s.ProcessPaymentSuccess(&payment)
	} else if status == "expire" {
		payment.Status = "expire"
		return s.db.Save(&payment).Error
	} else if status == "cancel" {
		payment.Status = "cancel"
		return s.db.Save(&payment).Error
	}

	return s.db.Save(&payment).Error
}

// HandleIpaymuNotification processes webhook notifications from iPaymu
func (s *PaymentService) HandleIpaymuNotification(data map[string]interface{}) error {
	log.Printf("🔔 Received iPaymu webhook: %+v", data)

	transID, _ := data["trx_id"].(string)
	sid, _ := data["sid"].(string)
	refID, _ := data["reference_id"].(string)
	if refID == "" {
		refID, _ = data["referenceId"].(string)
	}
	status, _ := data["status"].(string)
	statusCode, _ := data["status_code"].(string)

	var payment model.Payment
	var err error

	// 1. Try finding by trx_id
	if transID != "" {
		err = s.db.Where("ipaymu_trans_id = ? OR gateway_trans_id = ?", transID, transID).First(&payment).Error
	}
	// 2. Try finding by reference_id / invoice_number
	if err != nil && refID != "" {
		err = s.db.Where("gateway_trans_id = ? OR invoice_number = ?", refID, refID).First(&payment).Error
	}
	// 3. Try parsing payment ID from reference_id (e.g. KTB-ORD-1-2 -> 2, KTB-2 -> 2)
	if err != nil && refID != "" {
		parts := strings.Split(refID, "-")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if payID, parseErr := strconv.Atoi(lastPart); parseErr == nil && payID > 0 {
				err = s.db.Where("id = ?", uint(payID)).First(&payment).Error
			}
		}
	}
	// 4. Try finding by sid
	if err != nil && sid != "" {
		err = s.db.Where("gateway_trans_id = ?", sid).First(&payment).Error
	}
	// 5. Fallback latest pending ipaymu payment
	if err != nil {
		err = s.db.Where("payment_method = ? AND status = ?", "ipaymu", "pending").Order("created_at DESC").First(&payment).Error
		if err != nil {
			return fmt.Errorf("payment not found for iPaymu transaction: %s", transID)
		}
	}

	rawData, _ := json.Marshal(data)
	payment.IpaymuData = string(rawData)
	payment.GatewayData = string(rawData)

	statusLower := strings.ToLower(strings.TrimSpace(status))
	if statusCode == "1" || statusLower == "berhasil" || statusLower == "success" || statusLower == "settlement" {
		payment.PaymentMethod = "ipaymu"
		return s.ProcessPaymentSuccess(&payment)
	} else if statusLower == "expired" || statusLower == "expire" {
		payment.Status = "expire"
		return s.db.Save(&payment).Error
	} else if statusLower == "cancel" || statusLower == "cancelled" {
		payment.Status = "cancel"
		return s.db.Save(&payment).Error
	}

	return s.db.Save(&payment).Error
}

// HandlePakasirNotification processes webhook notifications from Pakasir
func (s *PaymentService) HandlePakasirNotification(data map[string]interface{}) error {
	log.Printf("🔔 Received Pakasir webhook: %+v", data)

	orderID, _ := data["order_id"].(string)
	if orderID == "" {
		if idVal, exists := data["id"].(string); exists {
			orderID = idVal
		}
	}

	status, _ := data["status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))

	if orderID == "" {
		return errors.New("missing order_id in Pakasir payload")
	}

	var payment model.Payment
	err := s.db.Where("gateway_trans_id = ? OR invoice_number = ?", orderID, orderID).First(&payment).Error
	if err != nil {
		// Try parsing payment ID from KTB-PRJ-x-y or KTB-ORD-x-y or KTB-y
		parts := strings.Split(orderID, "-")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if payID, parseErr := strconv.Atoi(lastPart); parseErr == nil && payID > 0 {
				err = s.db.Where("id = ?", uint(payID)).First(&payment).Error
			}
		}
	}

	if err != nil {
		// Fallback: try latest pending pakasir payment
		err = s.db.Where("payment_method = ? AND status = ?", "pakasir", "pending").Order("created_at DESC").First(&payment).Error
		if err != nil {
			return fmt.Errorf("payment not found for Pakasir order: %s", orderID)
		}
	}

	rawData, _ := json.Marshal(data)
	payment.GatewayData = string(rawData)

	if status == "paid" || status == "success" || status == "settlement" || status == "completed" {
		payment.PaymentMethod = "pakasir"
		log.Printf("✅ Payment %s marked as SUCCESS via Pakasir webhook", payment.InvoiceNumber)
		return s.ProcessPaymentSuccess(&payment)
	} else if status == "expired" || status == "expire" {
		payment.Status = "expire"
		return s.db.Save(&payment).Error
	} else if status == "cancelled" || status == "cancel" {
		payment.Status = "cancel"
		return s.db.Save(&payment).Error
	}

	return s.db.Save(&payment).Error
}

// UploadPaymentProof stores client receipt proof for manual transfer
func (s *PaymentService) UploadPaymentProof(paymentID uint, proofURL, bankName, accountHolder, notes string) (*model.Payment, error) {
	var payment model.Payment
	if err := s.db.First(&payment, paymentID).Error; err != nil {
		return nil, fmt.Errorf("payment not found with ID: %d", paymentID)
	}

	payment.ProofURL = proofURL
	if bankName != "" {
		payment.BankName = bankName
	}
	if accountHolder != "" {
		payment.AccountHolder = accountHolder
	}
	if notes != "" {
		payment.Notes = notes
	}
	payment.Status = "waiting_confirmation"

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

// VerifyManualPayment lets admin confirm or reject a manual payment
func (s *PaymentService) VerifyManualPayment(paymentID uint, status, notes string) (*model.Payment, error) {
	var payment model.Payment
	if err := s.db.First(&payment, paymentID).Error; err != nil {
		return nil, fmt.Errorf("payment not found with ID: %d", paymentID)
	}

	payment.Status = status
	if notes != "" {
		payment.Notes = notes
	}

	if status == "success" {
		if err := s.ProcessPaymentSuccess(&payment); err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Save(&payment).Error; err != nil {
			return nil, err
		}
	}

	return &payment, nil
}

// ListAllPayments returns payments with filtering and pagination for admin dashboard
func (s *PaymentService) ListAllPayments(status, paymentMethod string, page, limit int) ([]model.Payment, int64, error) {
	var payments []model.Payment
	var total int64

	query := s.db.Model(&model.Payment{}).Preload("Order").Preload("Project.Client")

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if paymentMethod != "" && paymentMethod != "all" {
		query = query.Where("payment_method = ?", paymentMethod)
	}

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&payments).Error

	return payments, total, err
}

// VerifyDocument checks validity of an invoice or quotation document
func (s *PaymentService) VerifyDocument(docNumber, docType string) (map[string]interface{}, error) {
	if docNumber == "" {
		return nil, errors.New("nomor dokumen wajib diisi")
	}

	docNumber = strings.TrimSpace(docNumber)
	docType = strings.ToLower(strings.TrimSpace(docType))

	docNumberNormalized := strings.ReplaceAll(docNumber, "-", "/")
	docNumberHyphen := strings.ReplaceAll(docNumber, "/", "-")

	// 1. Check Invoice first if requested or if prefix matches INV or default
	if docType == "invoice" || strings.HasPrefix(strings.ToUpper(docNumber), "INV") || docType == "" {
		var payment model.Payment
		err := s.db.Preload("Project.Client").Preload("Order").
			Where("invoice_number = ? OR invoice_number = ? OR invoice_number = ? OR invoice_number LIKE ? OR gateway_trans_id = ?",
				docNumber, docNumberNormalized, docNumberHyphen, "%"+docNumber+"%", docNumber).
			First(&payment).Error

		if err != nil {
			parts := strings.Split(docNumberNormalized, "/")
			if len(parts) > 0 {
				lastPart := strings.TrimLeft(parts[len(parts)-1], "0")
				if lastPart == "" {
					lastPart = "0"
				}
				if id, errConv := strconv.Atoi(lastPart); errConv == nil && id > 0 {
					err = s.db.Preload("Project.Client").Preload("Order").First(&payment, id).Error
				}
			}
			if err != nil {
				cleaned := strings.TrimPrefix(strings.TrimPrefix(docNumber, "KTB-PAY-"), "KTB-")
				if id, errConv := strconv.Atoi(cleaned); errConv == nil && id > 0 {
					err = s.db.Preload("Project.Client").Preload("Order").First(&payment, id).Error
				}
			}
		}

		if err == nil && payment.ID > 0 {
			clientName := "Pelanggan Terdaftar"
			companyName := ""
			projectTitle := "Pengembangan Sistem Perangkat Lunak"

			if payment.Project != nil {
				projectTitle = payment.Project.Title
				if payment.Project.Client.ID > 0 {
					clientName = payment.Project.Client.Name
					companyName = payment.Project.Client.CompanyName
				}
			}
			if payment.Order != nil {
				if projectTitle == "Pengembangan Sistem Perangkat Lunak" || projectTitle == "" {
					projectTitle = payment.Order.ProjectName
				}
				if clientName == "Pelanggan Terdaftar" || clientName == "" {
					clientName = payment.Order.CustomerName
					companyName = payment.Order.CompanyName
				}
			}

			return map[string]interface{}{
				"is_valid":        true,
				"document_type":   "INVOICE",
				"document_number": payment.InvoiceNumber,
				"issuer":          "PT KOTBAN SOLUSI TEKNOLOGI",
				"client_name":     clientName,
				"company_name":    companyName,
				"project_title":   projectTitle,
				"amount":          payment.Amount,
				"total_amount":    payment.Amount,
				"payment_type":    payment.PaymentType,
				"payment_method":  payment.PaymentMethod,
				"status":          payment.Status,
				"issued_at":       payment.CreatedAt,
				"paid_at":         payment.PaidAt,
			}, nil
		}

		// If strictly invoice requested and not found
		if docType == "invoice" {
			return nil, errors.New("dokumen invoice tidak ditemukan atau nomor dokumen tidak sah")
		}
	}

	// 2. Check Quotation
	if docType == "quotation" || strings.HasPrefix(strings.ToUpper(docNumber), "QUO") || docType == "" {
		var q model.Quotation
		err := s.db.Where("id = ?", docNumber).First(&q).Error
		if err != nil {
			parts := strings.Split(docNumberNormalized, "/")
			if len(parts) > 0 {
				lastPart := strings.TrimLeft(parts[len(parts)-1], "0")
				if lastPart == "" {
					lastPart = "0"
				}
				if id, errConv := strconv.Atoi(lastPart); errConv == nil && id > 0 {
					err = s.db.First(&q, id).Error
				}
			}
		}

		if err == nil && q.ID > 0 {
			var project model.Project
			s.db.Preload("Client").First(&project, q.ProjectID)

			clientName := "Klien Terdaftar"
			companyName := ""
			projectTitle := "Layanan Software Kotban"
			if project.ID > 0 {
				projectTitle = project.Title
				if project.Client.ID > 0 {
					clientName = project.Client.Name
					companyName = project.Client.CompanyName
				}
			}

			return map[string]interface{}{
				"is_valid":        true,
				"document_type":   "SURAT PENAWARAN HARGA",
				"document_number": fmt.Sprintf("QUO/%d%02d/%04d", q.CreatedAt.Year(), int(q.CreatedAt.Month()), q.ID),
				"issuer":          "PT KOTBAN SOLUSI TEKNOLOGI",
				"client_name":     clientName,
				"company_name":    companyName,
				"project_title":   projectTitle,
				"total_amount":    q.TotalAmount,
				"amount":          q.TotalAmount,
				"estimated_days":  q.EstimatedDays,
				"status":          q.Status,
				"valid_until":     q.ValidUntil,
				"issued_at":       q.CreatedAt,
			}, nil
		}
	}

	return nil, errors.New("dokumen tidak ditemukan atau nomor dokumen tidak sah di sistem Kotban.com")
}

// SyncPaymentWithGateway checks and synchronizes live payment status from the gateway
func (s *PaymentService) SyncPaymentWithGateway(paymentID uint) (*model.Payment, error) {
	var payment model.Payment
	if err := s.db.Preload("Order").Preload("Project").First(&payment, paymentID).Error; err != nil {
		return nil, fmt.Errorf("payment not found: %d", paymentID)
	}

	// If already marked as success, return directly
	if payment.Status == "success" {
		return &payment, nil
	}

	settings, _ := s.GetAdminSettings()

	switch payment.PaymentMethod {
	case "pakasir":
		slug := settings.PakasirProjectSlug
		if slug == "" {
			slug = s.cfg.PakasirProjectSlug
		}
		if slug == "" {
			slug = "kotban"
		}
		apiKey := settings.PakasirAPIKey
		if apiKey == "" {
			apiKey = s.cfg.PakasirAPIKey
		}

		refID := payment.GatewayTransID
		if refID == "" {
			refID = fmt.Sprintf("KTB-%d", payment.ID)
		}

		apiURL := fmt.Sprintf("https://app.pakasir.com/api/transactiondetail?project=%s&amount=%d&order_id=%s&api_key=%s",
			url.QueryEscape(slug),
			payment.Amount,
			url.QueryEscape(refID),
			url.QueryEscape(apiKey),
		)

		log.Printf("🔍 Checking Pakasir transaction status: %s", apiURL)
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Get(apiURL)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			log.Printf("📩 Pakasir check response: %s", string(body))

			var res struct {
				Status        string `json:"status"`
				Amount        int64  `json:"amount"`
				OrderID       string `json:"order_id"`
				PaymentMethod string `json:"payment_method"`
			}
			if jsonErr := json.Unmarshal(body, &res); jsonErr == nil {
				st := strings.ToLower(strings.TrimSpace(res.Status))
				if st == "completed" || st == "paid" || st == "success" || st == "settlement" {
					payment.PaymentMethod = "pakasir"
					_ = s.ProcessPaymentSuccess(&payment)
					return &payment, nil
				}
			}
		}

	case "mayar":
		apiKey := settings.MayarAPIKey
		if apiKey == "" {
			apiKey = s.cfg.MayarAPIKey
		}
		if apiKey != "" && payment.GatewayTransID != "" {
			baseURL := "https://api.mayar.io/hl/v1"
			if settings.MayarIsProduction || s.cfg.MayarIsProduction {
				baseURL = "https://api.mayar.id/hl/v1"
			}
			// Check Mayar invoice endpoint
			apiURL := fmt.Sprintf("%s/invoice/%s", baseURL, payment.GatewayTransID)
			httpReq, _ := http.NewRequest("GET", apiURL, nil)
			httpReq.Header.Set("Authorization", "Bearer "+apiKey)

			client := &http.Client{Timeout: 8 * time.Second}
			resp, err := client.Do(httpReq)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				log.Printf("📩 Mayar invoice check response: %s", string(body))
				var res struct {
					Data struct {
						Status string `json:"status"`
					} `json:"data"`
				}
				if jsonErr := json.Unmarshal(body, &res); jsonErr == nil {
					st := strings.ToLower(strings.TrimSpace(res.Data.Status))
					if st == "paid" || st == "settled" || st == "success" || st == "completed" {
						payment.PaymentMethod = "mayar"
						_ = s.ProcessPaymentSuccess(&payment)
						return &payment, nil
					}
				}
			}

			// Fallback check Mayar payment endpoint
			apiURLPayment := fmt.Sprintf("%s/payment/%s", baseURL, payment.GatewayTransID)
			httpReqP, _ := http.NewRequest("GET", apiURLPayment, nil)
			httpReqP.Header.Set("Authorization", "Bearer "+apiKey)
			if respP, errP := client.Do(httpReqP); errP == nil {
				defer respP.Body.Close()
				bodyP, _ := io.ReadAll(respP.Body)
				log.Printf("📩 Mayar payment check response: %s", string(bodyP))
				var resP struct {
					Data struct {
						Status string `json:"status"`
					} `json:"data"`
				}
				if jsonErr := json.Unmarshal(bodyP, &resP); jsonErr == nil {
					st := strings.ToLower(strings.TrimSpace(resP.Data.Status))
					if st == "paid" || st == "settled" || st == "success" || st == "completed" {
						payment.PaymentMethod = "mayar"
						_ = s.ProcessPaymentSuccess(&payment)
						return &payment, nil
					}
				}
			}
		}

	case "ipaymu":
		log.Printf("🔍 Checking iPaymu transaction for payment #%d", payment.ID)
	}

	return &payment, nil
}

// SyncPaymentByIdentifier syncs a payment using invoice number, payment id, or order id
func (s *PaymentService) SyncPaymentByIdentifier(identifier string) (*model.Payment, error) {
	var payment model.Payment
	err := s.db.Where("invoice_number = ? OR gateway_trans_id = ?", identifier, identifier).First(&payment).Error
	if err != nil {
		// Try parsing as ID
		cleaned := strings.TrimPrefix(identifier, "KTB-PAY-")
		cleaned = strings.TrimPrefix(cleaned, "KTB-")
		if id, errConv := strconv.Atoi(cleaned); errConv == nil && id > 0 {
			err = s.db.Where("id = ?", uint(id)).First(&payment).Error
		}
	}

	if err != nil {
		// Try latest pending payment
		err = s.db.Where("status = ?", "pending").Order("created_at DESC").First(&payment).Error
	}

	if err != nil || payment.ID == 0 {
		return nil, errors.New("transaksi pembayaran tidak ditemukan")
	}

	return s.SyncPaymentWithGateway(payment.ID)
}
