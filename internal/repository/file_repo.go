package repository

import (
	"github.com/kotban/backend/internal/model"
	"gorm.io/gorm"
)

type FileRepo struct {
	db *gorm.DB
}

func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{db: db}
}

func (r *FileRepo) Create(file *model.ProjectFile) error {
	return r.db.Create(file).Error
}

func (r *FileRepo) ListByProject(projectID uint) ([]model.ProjectFile, error) {
	var files []model.ProjectFile
	err := r.db.Preload("Uploader").
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

func (r *FileRepo) FindByID(id uint) (*model.ProjectFile, error) {
	var f model.ProjectFile
	err := r.db.First(&f, id).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FileRepo) Delete(id uint) error {
	return r.db.Delete(&model.ProjectFile{}, id).Error
}
