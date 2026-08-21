package service

import (
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
)

type ConsultationService struct {
	repo *repository.ConsultationRepo
}

func NewConsultationService(repo *repository.ConsultationRepo) *ConsultationService {
	return &ConsultationService{repo: repo}
}

func (s *ConsultationService) Create(c *model.Consultation) error {
	c.Status = "pending"
	return s.repo.Create(c)
}

func (s *ConsultationService) GetAll(page, perPage int) ([]model.Consultation, int64, error) {
	return s.repo.FindAll(page, perPage)
}
