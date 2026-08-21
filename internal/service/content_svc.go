package service

import (
	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/repository"
)

type ContentService struct {
	repo *repository.SiteContentRepo
}

func NewContentService(repo *repository.SiteContentRepo) *ContentService {
	return &ContentService{repo: repo}
}

func (s *ContentService) GetAll() (map[string]string, error) {
	contents, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, c := range contents {
		result[c.Key] = c.Value
	}
	return result, nil
}

func (s *ContentService) GetByGroup(group string) (map[string]string, error) {
	contents, err := s.repo.GetByGroup(group)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, c := range contents {
		result[c.Key] = c.Value
	}
	return result, nil
}

func (s *ContentService) GetAllRaw() ([]model.SiteContent, error) {
	return s.repo.GetAll()
}

func (s *ContentService) UpdateContent(key, value, contentType, group string) error {
	content := &model.SiteContent{
		Key:   key,
		Value: value,
		Type:  contentType,
		Group: group,
	}
	return s.repo.Upsert(content)
}

func (s *ContentService) BatchUpdate(items []model.SiteContent) error {
	for _, item := range items {
		if err := s.repo.Upsert(&item); err != nil {
			return err
		}
	}
	return nil
}
