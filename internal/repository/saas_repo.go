package repository

import (
	"strings"

	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type SaaSRepository interface {
	// SaaS Products & Plans
	GetActiveProducts() ([]model.SaaSProduct, error)
	GetProductBySlug(slug string) (*model.SaaSProduct, error)
	GetProductByID(id uint) (*model.SaaSProduct, error)
	GetPlanByID(id uint) (*model.SaaSPlan, error)
	GetAllProductsAdmin() ([]model.SaaSProduct, error)
	CreateProduct(prod *model.SaaSProduct) error
	UpdateProduct(prod *model.SaaSProduct) error
	DeleteProduct(id uint) error
	CreatePlan(plan *model.SaaSPlan) error
	UpdatePlan(plan *model.SaaSPlan) error
	DeletePlan(id uint) error
	GetSaaSStats() (map[string]interface{}, error)

	// Subscriptions
	CheckSubdomainAvailable(fullSubdomain string) (bool, error)
	GetSubscriptionBySubdomain(domainOrSlug string) (*model.SaaSSubscription, error)
	CreateSubscription(sub *model.SaaSSubscription) error
	GetSubscriptionByID(id uint) (*model.SaaSSubscription, error)
	GetSubscriptionByNumber(subNum string) (*model.SaaSSubscription, error)
	GetUserSubscriptions(userID uint) ([]model.SaaSSubscription, error)
	GetAllSubscriptions(status, search string, page, limit int) ([]model.SaaSSubscription, int64, error)
	UpdateSubscription(sub *model.SaaSSubscription) error

	// Invoices
	CreateInvoice(inv *model.Invoice) error
	GetInvoiceByID(id uint) (*model.Invoice, error)
	GetInvoiceByNumber(invNum string) (*model.Invoice, error)
	GetUserInvoices(userID uint) ([]model.Invoice, error)
	GetAllInvoices(status, search string, page, limit int) ([]model.Invoice, int64, error)
	UpdateInvoice(inv *model.Invoice) error
}

type saasRepository struct {
	db *gorm.DB
}

func NewSaaSRepository(db *gorm.DB) SaaSRepository {
	return &saasRepository{db: db}
}

func (r *saasRepository) GetActiveProducts() ([]model.SaaSProduct, error) {
	var products []model.SaaSProduct
	err := r.db.Preload("Plans", func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true).Order("sort_order asc")
	}).Where("is_active = ?", true).Order("sort_order asc").Find(&products).Error
	return products, err
}

func (r *saasRepository) GetProductBySlug(slug string) (*model.SaaSProduct, error) {
	var product model.SaaSProduct
	err := r.db.Preload("Plans", func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true).Order("sort_order asc")
	}).Where("slug = ?", slug).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *saasRepository) GetProductByID(id uint) (*model.SaaSProduct, error) {
	var product model.SaaSProduct
	err := r.db.Preload("Plans").First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *saasRepository) GetPlanByID(id uint) (*model.SaaSPlan, error) {
	var plan model.SaaSPlan
	err := r.db.First(&plan, id).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *saasRepository) GetAllProductsAdmin() ([]model.SaaSProduct, error) {
	var products []model.SaaSProduct
	err := r.db.Preload("Plans", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order asc")
	}).Order("sort_order asc, id asc").Find(&products).Error
	return products, err
}

func (r *saasRepository) CreateProduct(prod *model.SaaSProduct) error {
	return r.db.Create(prod).Error
}

func (r *saasRepository) UpdateProduct(prod *model.SaaSProduct) error {
	return r.db.Save(prod).Error
}

func (r *saasRepository) DeleteProduct(id uint) error {
	return r.db.Delete(&model.SaaSProduct{}, id).Error
}

func (r *saasRepository) CreatePlan(plan *model.SaaSPlan) error {
	return r.db.Create(plan).Error
}

func (r *saasRepository) UpdatePlan(plan *model.SaaSPlan) error {
	return r.db.Save(plan).Error
}

func (r *saasRepository) DeletePlan(id uint) error {
	return r.db.Delete(&model.SaaSPlan{}, id).Error
}

func (r *saasRepository) GetSaaSStats() (map[string]interface{}, error) {
	var totalTenants int64
	var activeTenants int64
	var expiringSoonTenants int64
	var expiredTenants int64
	var totalInvoices int64
	var paidInvoices int64
	var unpaidInvoices int64
	var totalRevenue float64

	r.db.Model(&model.SaaSSubscription{}).Count(&totalTenants)
	r.db.Model(&model.SaaSSubscription{}).Where("status = ?", "active").Count(&activeTenants)
	r.db.Model(&model.SaaSSubscription{}).Where("status = ?", "expired").Count(&expiredTenants)
	r.db.Model(&model.SaaSSubscription{}).Where("status = ?", "expiring_soon").Count(&expiringSoonTenants)

	r.db.Model(&model.Invoice{}).Count(&totalInvoices)
	r.db.Model(&model.Invoice{}).Where("status = ?", "paid").Count(&paidInvoices)
	r.db.Model(&model.Invoice{}).Where("status = ?", "unpaid").Count(&unpaidInvoices)

	r.db.Model(&model.Invoice{}).Where("status = ?", "paid").Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	// Calculate MRR estimate from active subscriptions
	var activeSubs []model.SaaSSubscription
	r.db.Where("status = ?", "active").Find(&activeSubs)
	var estimatedMRR float64
	for _, s := range activeSubs {
		if s.BillingCycle == "yearly" {
			estimatedMRR += (s.PriceAmount / 12)
		} else {
			estimatedMRR += s.PriceAmount
		}
	}

	return map[string]interface{}{
		"total_tenants":         totalTenants,
		"active_tenants":        activeTenants,
		"expiring_soon_tenants": expiringSoonTenants,
		"expired_tenants":       expiredTenants,
		"total_invoices":        totalInvoices,
		"paid_invoices":         paidInvoices,
		"unpaid_invoices":       unpaidInvoices,
		"total_revenue":         totalRevenue,
		"estimated_mrr":         estimatedMRR,
		"estimated_arr":         estimatedMRR * 12,
	}, nil
}

func (r *saasRepository) CheckSubdomainAvailable(fullSubdomain string) (bool, error) {
	fullSubdomain = strings.ToLower(strings.TrimSpace(fullSubdomain))
	var count int64
	err := r.db.Model(&model.SaaSSubscription{}).
		Where("LOWER(full_subdomain) = ?", fullSubdomain).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (r *saasRepository) GetSubscriptionBySubdomain(domainOrSlug string) (*model.SaaSSubscription, error) {
	val := strings.ToLower(strings.TrimSpace(domainOrSlug))
	var sub model.SaaSSubscription
	err := r.db.Preload("Product").
		Preload("SaaSProduct").
		Preload("SaaSPlan").
		Preload("User").
		Where("LOWER(full_subdomain) = ? OR LOWER(subdomain_slug) = ? OR LOWER(custom_domain) = ?", val, val, val).
		Order("created_at desc").
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *saasRepository) CreateSubscription(sub *model.SaaSSubscription) error {
	return r.db.Create(sub).Error
}

func (r *saasRepository) GetSubscriptionByID(id uint) (*model.SaaSSubscription, error) {
	var sub model.SaaSSubscription
	err := r.db.Preload("Product").
		Preload("SaaSProduct").
		Preload("SaaSPlan").
		Preload("User").
		Preload("Invoices", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at desc")
		}).
		First(&sub, id).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *saasRepository) GetSubscriptionByNumber(subNum string) (*model.SaaSSubscription, error) {
	var sub model.SaaSSubscription
	err := r.db.Preload("Product").
		Preload("SaaSProduct").
		Preload("SaaSPlan").
		Preload("User").
		Where("subscription_number = ?", subNum).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *saasRepository) GetUserSubscriptions(userID uint) ([]model.SaaSSubscription, error) {
	var subs []model.SaaSSubscription
	err := r.db.Preload("Product").
		Preload("SaaSProduct").
		Preload("SaaSPlan").
		Preload("Invoices", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at desc").Limit(3)
		}).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&subs).Error
	return subs, err
}

func (r *saasRepository) GetAllSubscriptions(status, search string, page, limit int) ([]model.SaaSSubscription, int64, error) {
	var subs []model.SaaSSubscription
	var total int64

	db := r.db.Model(&model.SaaSSubscription{}).
		Preload("SaaSProduct").
		Preload("SaaSPlan").
		Preload("User")

	if status != "" && status != "all" {
		db = db.Where("status = ?", status)
	}

	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		db = db.Where("LOWER(tenant_name) LIKE ? OR LOWER(full_subdomain) LIKE ? OR LOWER(subscription_number) LIKE ?", s, s, s)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := db.Order("created_at desc").Offset(offset).Limit(limit).Find(&subs).Error
	return subs, total, err
}

func (r *saasRepository) UpdateSubscription(sub *model.SaaSSubscription) error {
	return r.db.Save(sub).Error
}

func (r *saasRepository) CreateInvoice(inv *model.Invoice) error {
	return r.db.Create(inv).Error
}

func (r *saasRepository) GetInvoiceByID(id uint) (*model.Invoice, error) {
	var inv model.Invoice
	err := r.db.Preload("User").
		Preload("Subscription").
		Preload("Subscription.SaaSProduct").
		Preload("Subscription.SaaSPlan").
		First(&inv, id).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *saasRepository) GetInvoiceByNumber(invNum string) (*model.Invoice, error) {
	var inv model.Invoice
	err := r.db.Preload("User").
		Preload("Subscription").
		Preload("Subscription.SaaSProduct").
		Preload("Subscription.SaaSPlan").
		Where("invoice_number = ?", invNum).
		First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *saasRepository) GetUserInvoices(userID uint) ([]model.Invoice, error) {
	var invoices []model.Invoice
	err := r.db.Preload("Subscription").
		Preload("Subscription.SaaSProduct").
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&invoices).Error
	return invoices, err
}

func (r *saasRepository) GetAllInvoices(status, search string, page, limit int) ([]model.Invoice, int64, error) {
	var invoices []model.Invoice
	var total int64

	db := r.db.Model(&model.Invoice{}).
		Preload("User").
		Preload("Subscription").
		Preload("Subscription.SaaSProduct")

	if status != "" && status != "all" {
		db = db.Where("status = ?", status)
	}

	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		db = db.Where("LOWER(invoice_number) LIKE ? OR LOWER(title) LIKE ?", s, s)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := db.Order("created_at desc").Offset(offset).Limit(limit).Find(&invoices).Error
	return invoices, total, err
}

func (r *saasRepository) UpdateInvoice(inv *model.Invoice) error {
	return r.db.Save(inv).Error
}
