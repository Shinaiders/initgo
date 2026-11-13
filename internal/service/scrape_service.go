package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"goscrapper/internal/cache"
	"goscrapper/internal/model"
	"goscrapper/internal/queue"
	"goscrapper/internal/repository"
	"goscrapper/internal/scraper"
)

type ScrapeMessage struct {
	JobID   uint                `json:"job_id"`
	Request model.ScrapeRequest `json:"request"`
}

type ScrapeService interface {
	Execute(ctx context.Context, req *model.ScrapeRequest) (*model.ScrapeJob, error)
	RunJob(ctx context.Context, jobID uint, req *model.ScrapeRequest) (*model.ScrapeJob, error)
	GetJob(id uint) (*model.ScrapeJob, error)
	ListJobs(page, limit int) ([]model.ScrapeJob, int64, error)
	DeleteJob(id uint) error
	CaptureFullPage(ctx context.Context, targetURL string) ([]byte, string, error)
}

type scrapeService struct {
	repo    repository.ScrapeRepository
	scraper *scraper.Engine
	redis   *cache.RedisClient
	rabbit  *queue.RabbitMQClient
}

func NewScrapeService(
	repo repository.ScrapeRepository,
	scraper *scraper.Engine,
	redis *cache.RedisClient,
	rabbit *queue.RabbitMQClient,
) ScrapeService {
	return &scrapeService{
		repo:    repo,
		scraper: scraper,
		redis:   redis,
		rabbit:  rabbit,
	}
}

func (s *scrapeService) Execute(ctx context.Context, req *model.ScrapeRequest) (*model.ScrapeJob, error) {
	// Cria o registro com status PENDING no banco
	job := &model.ScrapeJob{
		URL:    req.URL,
		Status: model.StatusPending,
	}

	if err := s.repo.Create(job); err != nil {
		return nil, err
	}

	if req.Async {
		// Se RabbitMQ estiver ativo, publica a tarefa na fila
		if s.rabbit != nil {
			msgPayload, err := json.Marshal(ScrapeMessage{
				JobID:   job.ID,
				Request: *req,
			})
			if err != nil {
				return nil, fmt.Errorf("erro ao serializar mensagem para RabbitMQ: %w", err)
			}

			if err := s.rabbit.Publish(ctx, msgPayload); err != nil {
				return nil, fmt.Errorf("erro ao publicar tarefa no RabbitMQ: %w", err)
			}
			log.Printf("[SERVICE] Tarefa %d enviada para a fila RabbitMQ com sucesso.", job.ID)
			return job, nil
		}

		// Fallback: se RabbitMQ não estiver ativado, processa em goroutine local
		log.Printf("[SERVICE] RabbitMQ desabilitado. Processando job %d via goroutine local.", job.ID)
		go s.RunJob(context.Background(), job.ID, req)
		return job, nil
	}

	// Execução síncrona
	return s.RunJob(ctx, job.ID, req)
}

func (s *scrapeService) RunJob(ctx context.Context, jobID uint, req *model.ScrapeRequest) (*model.ScrapeJob, error) {
	job, err := s.repo.GetByID(jobID)
	if err != nil {
		log.Printf("[SERVICE ERROR] Job %d não encontrado: %v", jobID, err)
		return nil, err
	}

	job.Status = model.StatusRunning
	_ = s.repo.Update(job)

	result, err := s.scraper.Scrape(ctx, req)
	if err != nil {
		job.Status = model.StatusFailed
		job.ErrorMessage = err.Error()
		_ = s.repo.Update(job)
		return job, err
	}

	job.Status = model.StatusCompleted
	job.PageTitle = result.PageTitle
	job.HTTPStatusCode = result.HTTPStatusCode
	job.ContentText = result.ContentText
	job.ScreenshotPath = result.ScreenshotPath
	job.ExecutionTimeMs = result.ExecutionTimeMs
	job.ErrorMessage = ""

	if err := s.repo.Update(job); err != nil {
		return nil, err
	}

	// Cache no Redis se habilitado
	if s.redis != nil {
		cacheKey := fmt.Sprintf("scrape:job:%d", job.ID)
		data, err := json.Marshal(job)
		if err == nil {
			_ = s.redis.Set(ctx, cacheKey, string(data), 1*time.Hour)
		}
	}

	return job, nil
}

func (s *scrapeService) GetJob(id uint) (*model.ScrapeJob, error) {
	cacheKey := fmt.Sprintf("scrape:job:%d", id)

	// Consulta o Redis primeiro caso esteja ativado
	if s.redis != nil {
		val, err := s.redis.Get(context.Background(), cacheKey)
		if err == nil && val != "" {
			var cachedJob model.ScrapeJob
			if err := json.Unmarshal([]byte(val), &cachedJob); err == nil {
				return &cachedJob, nil
			}
		}
	}

	job, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Popula o cache do Redis
	if s.redis != nil {
		data, err := json.Marshal(job)
		if err == nil {
			_ = s.redis.Set(context.Background(), cacheKey, string(data), 1*time.Hour)
		}
	}

	return job, nil
}

func (s *scrapeService) ListJobs(page, limit int) ([]model.ScrapeJob, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.repo.List(limit, offset)
}

func (s *scrapeService) DeleteJob(id uint) error {
	return s.repo.Delete(id)
}

func (s *scrapeService) CaptureFullPage(ctx context.Context, targetURL string) ([]byte, string, error) {
	bytes, savedPath, err := s.scraper.CaptureFullPageScreenshot(ctx, targetURL)
	if err != nil {
		return nil, "", err
	}

	// Registra histórico no banco
	job := &model.ScrapeJob{
		URL:            targetURL,
		Status:         model.StatusCompleted,
		ScreenshotPath: savedPath,
	}
	_ = s.repo.Create(job)

	return bytes, savedPath, nil
}
