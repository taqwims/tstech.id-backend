package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
)

type ProductService interface {
	Create(product *model.Product) error
	Update(id uint, input *model.Product) (*model.Product, error)
	Delete(id uint) error
	GetByID(id uint) (*model.Product, error)
	GetBySlug(slug string) (*model.Product, error)
	ListActive(category string, featuredOnly bool) ([]model.Product, error)
	ListAdmin(category string, search string) ([]model.Product, error)
	ToggleStatus(id uint) (*model.Product, error)
	ToggleFeatured(id uint) (*model.Product, error)
	CreatePlan(productId uint, plan *model.SaaSPlan) (*model.SaaSPlan, error)
	UpdatePlan(planId uint, plan *model.SaaSPlan) (*model.SaaSPlan, error)
	DeletePlan(planId uint) error
}

type productService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) ProductService {
	return &productService{repo: repo}
}

func (s *productService) Create(product *model.Product) error {
	if strings.TrimSpace(product.Title) == "" {
		return errors.New("nama produk wajib diisi")
	}

	if strings.TrimSpace(product.Slug) == "" {
		product.Slug = generateProductSlug(product.Title)
	} else {
		product.Slug = generateProductSlug(product.Slug)
	}

	// Check slug uniqueness
	existing, _ := s.repo.FindBySlug(product.Slug)
	if existing != nil {
		product.Slug = fmt.Sprintf("%s-%d", product.Slug, len(product.Title))
	}

	if product.Icon == "" {
		product.Icon = "📦"
	}
	if product.PriceType == "" {
		product.PriceType = "one_time"
	}

	// Default SaaS pattern if IsSaaS is true
	if product.IsSaaS {
		if product.BaseDomain == "" {
			product.BaseDomain = fmt.Sprintf("%s.tstech.id", product.Slug)
		}
		if product.SubdomainPattern == "" {
			product.SubdomainPattern = fmt.Sprintf("{tenant}.%s", product.BaseDomain)
		}
		if product.APISecretKey == "" {
			product.APISecretKey = fmt.Sprintf("sec_%s_%s", product.Slug, generateRandomJTI()[:12])
		}
	}

	return s.repo.Create(product)
}

func (s *productService) Update(id uint, input *model.Product) (*model.Product, error) {
	prod, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("produk tidak ditemukan")
	}

	if strings.TrimSpace(input.Title) == "" {
		return nil, errors.New("nama produk wajib diisi")
	}

	if strings.TrimSpace(input.Slug) != "" {
		newSlug := generateProductSlug(input.Slug)
		if newSlug != prod.Slug {
			existing, _ := s.repo.FindBySlug(newSlug)
			if existing != nil && existing.ID != id {
				return nil, errors.New("slug sudah digunakan oleh produk lain")
			}
			prod.Slug = newSlug
		}
	}

	prod.Title = input.Title
	prod.Tagline = input.Tagline
	prod.Badge = input.Badge
	prod.Category = input.Category
	prod.Icon = input.Icon
	prod.Thumbnail = input.Thumbnail
	prod.Images = input.Images
	prod.Price = input.Price
	prod.PriceDiscount = input.PriceDiscount
	prod.PriceType = input.PriceType
	prod.DemoURL = input.DemoURL
	prod.DocURL = input.DocURL
	prod.Features = input.Features
	prod.TechStack = input.TechStack
	prod.Overview = input.Overview
	prod.IsFeatured = input.IsFeatured
	prod.IsActive = input.IsActive
	prod.SortOrder = input.SortOrder
	prod.MetaTitle = input.MetaTitle
	prod.MetaDescription = input.MetaDescription
	prod.MetaKeywords = input.MetaKeywords

	// Unified SaaS fields
	prod.IsSaaS = input.IsSaaS
	prod.SubdomainPattern = input.SubdomainPattern
	prod.BaseDomain = input.BaseDomain
	prod.WebhookURL = input.WebhookURL
	if input.APISecretKey != "" {
		prod.APISecretKey = input.APISecretKey
	} else if prod.IsSaaS && prod.APISecretKey == "" {
		prod.APISecretKey = fmt.Sprintf("sec_%s_%s", prod.Slug, generateRandomJTI()[:12])
	}

	if err := s.repo.Update(prod); err != nil {
		return nil, err
	}

	return prod, nil
}

func (s *productService) CreatePlan(productId uint, plan *model.SaaSPlan) (*model.SaaSPlan, error) {
	prod, err := s.repo.FindByID(productId)
	if err != nil {
		return nil, errors.New("produk tidak ditemukan")
	}

	plan.ProductID = &prod.ID
	plan.SaaSProductID = prod.ID
	if plan.Code == "" {
		plan.Code = fmt.Sprintf("%s_plan_%d", prod.Slug, time.Now().Unix()%1000)
	}

	if err := s.repo.CreatePlan(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *productService) UpdatePlan(planId uint, plan *model.SaaSPlan) (*model.SaaSPlan, error) {
	plan.ID = planId
	if err := s.repo.UpdatePlan(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *productService) DeletePlan(planId uint) error {
	return s.repo.DeletePlan(planId)
}

func (s *productService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *productService) GetByID(id uint) (*model.Product, error) {
	return s.repo.FindByID(id)
}

func (s *productService) GetBySlug(slug string) (*model.Product, error) {
	return s.repo.FindBySlug(slug)
}

func (s *productService) ListActive(category string, featuredOnly bool) ([]model.Product, error) {
	return s.repo.ListActive(category, featuredOnly)
}

func (s *productService) ListAdmin(category string, search string) ([]model.Product, error) {
	return s.repo.ListAdmin(category, search)
}

func (s *productService) ToggleStatus(id uint) (*model.Product, error) {
	prod, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("produk tidak ditemukan")
	}
	prod.IsActive = !prod.IsActive
	if err := s.repo.Update(prod); err != nil {
		return nil, err
	}
	return prod, nil
}

func (s *productService) ToggleFeatured(id uint) (*model.Product, error) {
	prod, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("produk tidak ditemukan")
	}
	prod.IsFeatured = !prod.IsFeatured
	if err := s.repo.Update(prod); err != nil {
		return nil, err
	}
	return prod, nil
}

func generateProductSlug(text string) string {
	slug := strings.ToLower(text)
	reg := regexp.MustCompile("[^a-z0-9]+")
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "product"
	}
	return slug
}
