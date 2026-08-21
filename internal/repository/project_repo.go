package repository

import (
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type ProjectRepo struct {
	db *gorm.DB
}

func NewProjectRepo(db *gorm.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) Create(project *model.Project) error {
	return r.db.Create(project).Error
}

func (r *ProjectRepo) FindByID(id uint) (*model.Project, error) {
	var project model.Project
	err := r.db.Preload("Client").
		Preload("Admin").
		Preload("Order").
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("milestones.sort_order ASC, milestones.id ASC")
		}).
		First(&project, id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepo) ListByClient(clientID uint, page, limit int) ([]model.Project, int64, error) {
	var projects []model.Project
	var total int64

	query := r.db.Model(&model.Project{}).Where("client_user_id = ?", clientID)
	query.Count(&total)

	err := query.Preload("Milestones").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&projects).Error

	return projects, total, err
}

func (r *ProjectRepo) ListAll(page, limit int, status string) ([]model.Project, int64, error) {
	var projects []model.Project
	var total int64

	query := r.db.Model(&model.Project{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)

	err := query.Preload("Client").
		Preload("Milestones").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&projects).Error

	return projects, total, err
}

func (r *ProjectRepo) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&model.Project{}).Where("id = ?", id).Updates(updates).Error
}

func (r *ProjectRepo) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.Project{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ProjectRepo) UpdateProgress(id uint, progress int) error {
	return r.db.Model(&model.Project{}).Where("id = ?", id).Update("progress_percent", progress).Error
}
