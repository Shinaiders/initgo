package model

import (
	"time"

	"gorm.io/gorm"
)

type JobStatus string

const (
	StatusPending   JobStatus = "PENDING"
	StatusRunning   JobStatus = "RUNNING"
	StatusCompleted JobStatus = "COMPLETED"
	StatusFailed    JobStatus = "FAILED"
)

// ScrapeJob representa uma tarefa e o resultado de scraping salvo no banco
type ScrapeJob struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	URL             string         `gorm:"size:2048;not null;index" json:"url"`
	Status          JobStatus      `gorm:"size:20;not null;default:'PENDING';index" json:"status"`
	PageTitle       string         `gorm:"size:500" json:"page_title,omitempty"`
	HTTPStatusCode  int            `json:"http_status_code,omitempty"`
	ContentText     string         `gorm:"type:text" json:"content_text,omitempty"`
	ScreenshotPath  string         `gorm:"size:1024" json:"screenshot_path,omitempty"`
	ExecutionTimeMs int64          `json:"execution_time_ms,omitempty"`
	ErrorMessage    string         `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// ScrapeRequest DTO de entrada para disparar um scraping
type ScrapeRequest struct {
	URL            string `json:"url" binding:"required"`
	TakeScreenshot bool   `json:"take_screenshot"`
	WaitSelector   string `json:"wait_selector"`
	Async          bool   `json:"async"`
}

// ScrapeResult dados brutos extraídos pelo motor Playwright
type ScrapeResult struct {
	PageTitle       string
	HTTPStatusCode  int
	ContentText     string
	ScreenshotBytes []byte
	ScreenshotPath  string
	ExecutionTimeMs int64
}
