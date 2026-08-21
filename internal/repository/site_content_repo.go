package repository

import (
	"time"

	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SiteContentRepo struct {
	db *gorm.DB
}

func NewSiteContentRepo(db *gorm.DB) *SiteContentRepo {
	return &SiteContentRepo{db: db}
}

func (r *SiteContentRepo) Upsert(content *model.SiteContent) error {
	content.UpdatedAt = time.Now()
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "type", "group", "updated_at"}),
	}).Create(content).Error
}

func (r *SiteContentRepo) GetByKey(key string) (*model.SiteContent, error) {
	var c model.SiteContent
	err := r.db.Where("key = ?", key).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SiteContentRepo) GetByGroup(group string) ([]model.SiteContent, error) {
	var contents []model.SiteContent
	err := r.db.Where("`group` = ? OR \"group\" = ?", group, group).Find(&contents).Error
	return contents, err
}

func (r *SiteContentRepo) GetAll() ([]model.SiteContent, error) {
	var contents []model.SiteContent
	err := r.db.Order("`group` ASC, key ASC").Find(&contents).Error
	if err != nil {
		// Try fallback if sqlite/postgres escaping differences occur
		err = r.db.Order("\"group\" ASC, key ASC").Find(&contents).Error
	}
	return contents, err
}
