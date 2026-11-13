package repository

import (
	"goscrapper/internal/model"

	"gorm.io/gorm"
)

type ScrapeRepository interface {
	Create(job *model.ScrapeJob) error
	GetByID(id uint) (*model.ScrapeJob, error)
	Update(job *model.ScrapeJob) error
	List(limit, offset int) ([]model.ScrapeJob, int64, error)
	Delete(id uint) error
}

type scrapeRepository struct {
	db *gorm.DB
}

func NewScrapeRepository(db *gorm.DB) ScrapeRepository {
	return &scrapeRepository{db: db}
}

func (r *scrapeRepository) Create(job *model.ScrapeJob) error {
	return r.db.Create(job).Error
}

func (r *scrapeRepository) GetByID(id uint) (*model.ScrapeJob, error) {
	var job model.ScrapeJob
	if err := r.db.First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *scrapeRepository) Update(job *model.ScrapeJob) error {
	return r.db.Save(job).Error
}

func (r *scrapeRepository) List(limit, offset int) ([]model.ScrapeJob, int64, error) {
	var jobs []model.ScrapeJob
	var total int64

	if err := r.db.Model(&model.ScrapeJob{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Order("id desc").Limit(limit).Offset(offset).Find(&jobs).Error
	if err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (r *scrapeRepository) Delete(id uint) error {
	return r.db.Delete(&model.ScrapeJob{}, id).Error
}
