package model

import (
	"time"

	"gorm.io/gorm"
)

// SaaSProduct represents a standalone SaaS product offered by TsTech (e.g. Schola, RestoPOS, Medika)
type SaaSProduct struct {
	ID               uint           `json:"id" gorm:"primarykey"`
	Slug             string         `json:"slug" gorm:"size:100;uniqueIndex;not null"` // e.g. "schola", "restopos", "medika"
	Name             string         `json:"name" gorm:"size:200;not null"`             // e.g. "Schola LMS & Sistem Sekolah"
	Tagline          string         `json:"tagline" gorm:"type:text"`
	Icon             string         `json:"icon" gorm:"size:50;default:'🏫'"`
	Category         string         `json:"category" gorm:"size:100"`                  // Pendidikan, FnB, Kesehatan, ERP, dll.
	SubdomainPattern string         `json:"subdomain_pattern" gorm:"size:255"`         // e.g. "{tenant}.schola.tstech.id"
	BaseDomain       string         `json:"base_domain" gorm:"size:255"`               // e.g. "schola.tstech.id"
	DemoURL          string         `json:"demo_url" gorm:"size:255"`
	DocURL           string         `json:"doc_url" gorm:"size:255"`
	Thumbnail        string         `json:"thumbnail" gorm:"type:text"`
	Images           string         `json:"images" gorm:"type:text"` // JSON array
	Features         string         `json:"features" gorm:"type:text"`
	TechStack        string         `json:"tech_stack" gorm:"type:text"`
	Overview         string         `json:"overview" gorm:"type:text"`
	WebhookURL       string         `json:"webhook_url" gorm:"size:255"` // Endpoint to notify remote SaaS tenant creation
	APISecretKey     string         `json:"-" gorm:"size:255"`           // Secret for signing SSO & webhooks
	IsActive         bool           `json:"is_active" gorm:"default:true;index"`
	SortOrder        int            `json:"sort_order" gorm:"default:0"`
	Plans            []SaaSPlan     `json:"plans,omitempty" gorm:"foreignKey:SaaSProductID"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// SaaSPlan represents pricing & packaging for a SaaS product (e.g. Starter, Pro, Enterprise)
type SaaSPlan struct {
	ID            uint           `json:"id" gorm:"primarykey"`
	ProductID     *uint          `json:"product_id,omitempty" gorm:"index"`
	SaaSProductID uint           `json:"saas_product_id" gorm:"index;default:0"`
	Name          string         `json:"name" gorm:"size:100;not null"`              // e.g. "Paket Sekolah Dasar", "Pro Bulanan"
	Code          string         `json:"code" gorm:"size:50;not null"`               // e.g. "starter_monthly", "pro_yearly"
	Interval      string         `json:"interval" gorm:"size:30;default:'monthly'"`  // "monthly", "yearly", "lifetime"
	PriceMonthly  float64        `json:"price_monthly" gorm:"default:0"`
	PriceYearly   float64        `json:"price_yearly" gorm:"default:0"`
	DiscountPct    int            `json:"discount_pct" gorm:"default:0"`
	Features       string         `json:"features" gorm:"type:text"`                  // JSON list of bullet features
	FeatureModules string         `json:"feature_modules" gorm:"type:text"`          // JSON array of technical module slugs: ["billing","ppdb",...]
	AllowedUnits   string         `json:"allowed_units" gorm:"type:text"`            // JSON array of allowed education units: ["sdit","mts","ma"]
	MaxUsers       int            `json:"max_users" gorm:"default:0"`                 // 0 = unlimited
	MaxStorageGB   int            `json:"max_storage_gb" gorm:"default:5"`
	IsPopular      bool           `json:"is_popular" gorm:"default:false"`
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	SortOrder      int            `json:"sort_order" gorm:"default:0"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// SaaSSubscription represents a client's subscribed SaaS tenant/instance
type SaaSSubscription struct {
	ID                 uint           `json:"id" gorm:"primarykey"`
	SubscriptionNumber string         `json:"subscription_number" gorm:"size:50;uniqueIndex;not null"` // e.g. "SUB-2026-0001"
	UserID             uint           `json:"user_id" gorm:"index;not null"`
	User               *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	ProductID          *uint          `json:"product_id,omitempty" gorm:"index"`
	Product            *Product       `json:"product,omitempty" gorm:"foreignKey:ProductID"`
	SaaSProductID      uint           `json:"saas_product_id" gorm:"index;not null"`
	SaaSProduct        *SaaSProduct   `json:"saas_product,omitempty" gorm:"foreignKey:SaaSProductID"`
	SaaSPlanID         uint           `json:"saas_plan_id" gorm:"index;not null"`
	SaaSPlan           *SaaSPlan      `json:"saas_plan,omitempty" gorm:"foreignKey:SaaSPlanID"`
	TenantName         string         `json:"tenant_name" gorm:"size:200;not null"`                      // e.g. "SMK Negeri 1 Surabaya"
	SubdomainSlug      string         `json:"subdomain_slug" gorm:"size:100;index;not null"`             // e.g. "smkn1"
	FullSubdomain      string         `json:"full_subdomain" gorm:"size:255;uniqueIndex;not null"`       // e.g. "smkn1.schola.tstech.id"
	CustomModules      string         `json:"custom_modules" gorm:"type:text"`                           // JSON array override of active modules
	ActiveUnits        string         `json:"active_units" gorm:"type:text"`                             // JSON array of active school units: ["sdit","mts"]
	Status             string         `json:"status" gorm:"size:50;default:'active';index"`             // "trial", "active", "expiring_soon", "past_due", "expired", "suspended", "cancelled"
	BillingCycle       string         `json:"billing_cycle" gorm:"size:30;default:'monthly'"`            // "monthly", "yearly"
	PriceAmount        float64        `json:"price_amount" gorm:"default:0"`
	StartDate          time.Time      `json:"start_date"`
	EndDate            time.Time      `json:"end_date" gorm:"index"`
	TrialEndsAt        *time.Time     `json:"trial_ends_at"`
	AutoRenew          bool           `json:"auto_renew" gorm:"default:true"`
	RemoteTenantID     string         `json:"remote_tenant_id" gorm:"size:100"`                          // ID in satellite SaaS DB
	CustomDomain       string         `json:"custom_domain" gorm:"size:255"`                             // e.g. "lms.smkn1sby.sch.id"
	AdminNotes         string         `json:"admin_notes" gorm:"type:text"`
	Invoices           []Invoice      `json:"invoices,omitempty" gorm:"foreignKey:SubscriptionID"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

// Invoice represents a billing invoice for SaaS subscription or service
type Invoice struct {
	ID             uint           `json:"id" gorm:"primarykey"`
	InvoiceNumber  string         `json:"invoice_number" gorm:"size:50;uniqueIndex;not null"` // e.g. "INV-2026-0901-0001"
	UserID         uint           `json:"user_id" gorm:"index;not null"`
	User           *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	SubscriptionID *uint          `json:"subscription_id" gorm:"index"`
	Subscription   *SaaSSubscription `json:"subscription,omitempty" gorm:"foreignKey:SubscriptionID"`
	Title          string         `json:"title" gorm:"size:255;not null"`                     // e.g. "Langganan Schola LMS - SMK Negeri 1 (1 Bulan)"
	Description    string         `json:"description" gorm:"type:text"`
	Amount         float64        `json:"amount" gorm:"not null"`
	DiscountAmount float64        `json:"discount_amount" gorm:"default:0"`
	TaxAmount      float64        `json:"tax_amount" gorm:"default:0"`
	TotalAmount    float64        `json:"total_amount" gorm:"not null"`
	Status         string         `json:"status" gorm:"size:50;default:'unpaid';index"`       // "unpaid", "paid", "expired", "cancelled", "refunded"
	PaymentMethod  string         `json:"payment_method" gorm:"size:50"`                      // "mayar", "manual_transfer", "qris", "va"
	PaymentChannel string         `json:"payment_channel" gorm:"size:50"`                     // "bca", "mandiri", "qris", "mayar_checkout"
	PaymentURL     string         `json:"payment_url" gorm:"type:text"`                       // URL to Mayar checkout
	QRString       string         `json:"qr_string" gorm:"type:text"`
	QRCodeURL      string         `json:"qr_code_url" gorm:"type:text"`
	ProofURL       string         `json:"proof_url" gorm:"type:text"`                         // Manual transfer receipt upload
	BankName       string         `json:"bank_name" gorm:"size:100"`
	AccountHolder  string         `json:"account_holder" gorm:"size:150"`
	AccountNumber  string         `json:"account_number" gorm:"size:100"`
	GatewayTransID string         `json:"gateway_trans_id" gorm:"size:255"`                   // Mayar transaction / payment link ID
	DueDate        time.Time      `json:"due_date"`
	PaidAt         *time.Time     `json:"paid_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}
