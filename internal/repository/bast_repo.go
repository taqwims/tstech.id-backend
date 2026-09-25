package repository

import (
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type BastRepo struct {
	db *gorm.DB
}

func NewBastRepo(db *gorm.DB) *BastRepo {
	return &BastRepo{db: db}
}

func (r *BastRepo) Save(bast *model.Bast) error {
	var existing model.Bast
	if err := r.db.Where("project_id = ?", bast.ProjectID).First(&existing).Error; err == nil {
		bast.ID = existing.ID
		return r.db.Save(bast).Error
	}
	return r.db.Create(bast).Error
}

func (r *BastRepo) FindByProjectID(projectID uint) (*model.Bast, error) {
	var bast model.Bast
	err := r.db.Preload("Project.Client").Where("project_id = ?", projectID).First(&bast).Error
	if err != nil {
		return nil, err
	}
	return &bast, nil
}

func (r *BastRepo) FindByBastNumber(number string) (*model.Bast, error) {
	var bast model.Bast
	err := r.db.Preload("Project.Client").
		Where("bast_number = ? OR bast_number LIKE ?", number, "%"+number+"%").
		First(&bast).Error
	if err != nil {
		return nil, err
	}
	return &bast, nil
}
