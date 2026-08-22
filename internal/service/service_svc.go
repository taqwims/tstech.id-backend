package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
)

type ServiceService interface {
	Create(svc *model.Service) error
	Update(id uint, input *model.Service) (*model.Service, error)
	Delete(id uint) error
	GetByID(id uint) (*model.Service, error)
	GetBySlug(slug string) (*model.Service, error)
	ListActive() ([]model.Service, error)
	ListAdmin() ([]model.Service, error)
	ToggleStatus(id uint) (*model.Service, error)
}

type serviceService struct {
	repo repository.ServiceRepository
}

func NewServiceService(repo repository.ServiceRepository) ServiceService {
	return &serviceService{repo: repo}
}

func (s *serviceService) Create(svc *model.Service) error {
	if strings.TrimSpace(svc.Title) == "" {
		return errors.New("judul layanan wajib diisi")
	}

	if strings.TrimSpace(svc.Slug) == "" {
		svc.Slug = generateSlug(svc.Title)
	} else {
		svc.Slug = generateSlug(svc.Slug)
	}

	// Check slug uniqueness
	existing, _ := s.repo.FindBySlug(svc.Slug)
	if existing != nil {
		svc.Slug = fmt.Sprintf("%s-%d", svc.Slug, len(svc.Title))
	}

	if svc.Icon == "" {
		svc.Icon = "🌐"
	}
	if svc.CtaButtonText == "" {
		svc.CtaButtonText = "Konsultasi Gratis Sekarang"
	}
	if svc.CtaButtonURL == "" {
		svc.CtaButtonURL = "/konsultasi"
	}

	return s.repo.Create(svc)
}

func (s *serviceService) Update(id uint, input *model.Service) (*model.Service, error) {
	svc, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("layanan tidak ditemukan")
	}

	if strings.TrimSpace(input.Title) == "" {
		return nil, errors.New("judul layanan wajib diisi")
	}

	if strings.TrimSpace(input.Slug) != "" {
		newSlug := generateSlug(input.Slug)
		if newSlug != svc.Slug {
			existing, _ := s.repo.FindBySlug(newSlug)
			if existing != nil && existing.ID != id {
				return nil, errors.New("slug sudah digunakan oleh layanan lain")
			}
			svc.Slug = newSlug
		}
	}

	svc.Title = input.Title
	svc.Tagline = input.Tagline
	svc.Badge = input.Badge
	svc.Icon = input.Icon
	svc.OverviewTitle = input.OverviewTitle
	svc.OverviewContent = input.OverviewContent
	svc.OverviewImage = input.OverviewImage
	svc.ProcessTitle = input.ProcessTitle
	svc.ProcessContent = input.ProcessContent
	svc.ProcessImage = input.ProcessImage
	svc.TechTitle = input.TechTitle
	svc.TechContent = input.TechContent
	svc.TechImage = input.TechImage
	svc.CtaTitle = input.CtaTitle
	svc.CtaDescription = input.CtaDescription
	svc.CtaButtonText = input.CtaButtonText
	svc.CtaButtonURL = input.CtaButtonURL
	svc.SortOrder = input.SortOrder
	svc.IsActive = input.IsActive
	svc.MetaTitle = input.MetaTitle
	svc.MetaDescription = input.MetaDescription
	svc.MetaKeywords = input.MetaKeywords

	if err := s.repo.Update(svc); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *serviceService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *serviceService) GetByID(id uint) (*model.Service, error) {
	return s.repo.FindByID(id)
}

func (s *serviceService) GetBySlug(slug string) (*model.Service, error) {
	return s.repo.FindBySlug(slug)
}

func (s *serviceService) ListActive() ([]model.Service, error) {
	return s.repo.ListActive()
}

func (s *serviceService) ListAdmin() ([]model.Service, error) {
	return s.repo.ListAdmin()
}

func (s *serviceService) ToggleStatus(id uint) (*model.Service, error) {
	svc, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("layanan tidak ditemukan")
	}
	svc.IsActive = !svc.IsActive
	if err := s.repo.Update(svc); err != nil {
		return nil, err
	}
	return svc, nil
}

func generateSlug(text string) string {
	slug := strings.ToLower(text)
	reg := regexp.MustCompile("[^a-z0-9]+")
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "service"
	}
	return slug
}
