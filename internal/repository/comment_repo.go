package repository

import (
	"github.com/kotban/backend/internal/model"
	"gorm.io/gorm"
)

type CommentRepo struct {
	db *gorm.DB
}

func NewCommentRepo(db *gorm.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

func (r *CommentRepo) Create(comment *model.ProjectComment) error {
	return r.db.Create(comment).Error
}

func (r *CommentRepo) ListByProject(projectID uint) ([]model.ProjectComment, error) {
	var comments []model.ProjectComment
	err := r.db.Preload("User").
		Where("project_id = ?", projectID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}
