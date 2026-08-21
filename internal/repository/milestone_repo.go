package repository

import (
	"time"

	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type MilestoneRepo struct {
	db *gorm.DB
}

func NewMilestoneRepo(db *gorm.DB) *MilestoneRepo {
	return &MilestoneRepo{db: db}
}

func (r *MilestoneRepo) Create(milestone *model.Milestone) error {
	return r.db.Create(milestone).Error
}

func (r *MilestoneRepo) FindByID(id uint) (*model.Milestone, error) {
	var m model.Milestone
	err := r.db.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MilestoneRepo) ListByProject(projectID uint) ([]model.Milestone, error) {
	var milestones []model.Milestone
	err := r.db.Where("project_id = ?", projectID).
		Order("sort_order ASC, id ASC").
		Find(&milestones).Error
	return milestones, err
}

func (r *MilestoneRepo) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&model.Milestone{}).Where("id = ?", id).Updates(updates).Error
}

func (r *MilestoneRepo) Complete(id uint) error {
	now := time.Now()
	return r.db.Model(&model.Milestone{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       "completed",
		"completed_at": &now,
	}).Error
}

func (r *MilestoneRepo) Delete(id uint) error {
	return r.db.Delete(&model.Milestone{}, id).Error
}
