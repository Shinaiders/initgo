package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goscrapper/internal/model"
	"goscrapper/internal/service"
	"goscrapper/pkg/response"

	"github.com/gin-gonic/gin"
)

type ScrapeHandler struct {
	service service.ScrapeService
}

func NewScrapeHandler(service service.ScrapeService) *ScrapeHandler {
	return &ScrapeHandler{service: service}
}

// CreateScrape godoc
// @Summary Dispara uma raspagem de página web
// @Tags Scraper
// @Accept json
// @Produce json
// @Param request body model.ScrapeRequest true "Dados da requisição"
// @Success 200 {object} response.APIResponse
// @Router /api/v1/scrapes [post]
func (h *ScrapeHandler) CreateScrape(c *gin.Context) {
	var req model.ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dados da requisição inválidos", err.Error())
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "https://" + req.URL
	}

	job, err := h.service.Execute(c.Request.Context(), &req)
	if err != nil && !req.Async {
		response.Error(c, http.StatusInternalServerError, "Falha durante o scraping", job)
		return
	}

	response.Success(c, http.StatusOK, "Scraping processado com sucesso", job)
}

// GetScrape godoc
// @Summary Consulta o status e detalhes de um scraping por ID
// @Tags Scraper
// @Produce json
// @Param id path int true "ID do Job"
// @Success 200 {object} response.APIResponse
// @Router /api/v1/scrapes/{id} [get]
func (h *ScrapeHandler) GetScrape(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "ID inválido", nil)
		return
	}

	job, err := h.service.GetJob(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "Job de scraping não encontrado", nil)
		return
	}

	response.Success(c, http.StatusOK, "Job encontrado", job)
}

// ListScrapes godoc
// @Summary Lista os scrapings realizados com paginação
// @Tags Scraper
// @Produce json
// @Param page query int false "Página atual" default(1)
// @Param limit query int false "Itens por página" default(10)
// @Success 200 {object} response.APIResponse
// @Router /api/v1/scrapes [get]
func (h *ScrapeHandler) ListScrapes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	jobs, total, err := h.service.ListJobs(page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Erro ao listar scrapings", err.Error())
		return
	}

	response.Paginated(c, jobs, total, page, limit)
}

// DeleteScrape godoc
// @Summary Exclui o registro de um scraping
// @Tags Scraper
// @Produce json
// @Param id path int true "ID do Job"
// @Success 200 {object} response.APIResponse
// @Router /api/v1/scrapes/{id} [delete]
func (h *ScrapeHandler) DeleteScrape(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "ID inválido", nil)
		return
	}

	if err := h.service.DeleteJob(uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, "Erro ao remover job", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Job removido com sucesso", nil)
}

type AnalisarPaginaRequest struct {
	URL    string `json:"url" form:"url"`
	Format string `json:"format" form:"format"` // "image" (padrão) ou "json"
}

// AnalisarPagina abre a página solicitada e retorna o print inteiro (FullPage)
// Suporta tanto GET /analisar-pagina?url=https://... quanto POST com JSON {"url": "https://..."}
func (h *ScrapeHandler) AnalisarPagina(c *gin.Context) {
	var targetURL string
	var format string

	if c.Request.Method == http.MethodPost {
		var req AnalisarPaginaRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.URL != "" {
			targetURL = req.URL
			format = req.Format
		} else if err := c.ShouldBindQuery(&req); err == nil && req.URL != "" {
			targetURL = req.URL
			format = req.Format
		}
	} else {
		targetURL = c.Query("url")
		format = c.Query("format")
	}

	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		response.Error(c, http.StatusBadRequest, "O parâmetro 'url' é obrigatório (ex: /analisar-pagina?url=https://exemplo.com)", nil)
		return
	}

	// Normaliza automaticamente caso o usuário envie "www.google.com" ou "google.com" sem protocolo
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	screenshotBytes, savedPath, err := h.service.CaptureFullPage(c.Request.Context(), targetURL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Falha ao capturar print inteiro da página", err.Error())
		return
	}

	// Se o cliente pediu formato JSON explicitamente
	if format == "json" {
		response.Success(c, http.StatusOK, "Página analisada e print capturado com sucesso", gin.H{
			"url":             targetURL,
			"screenshot_path": savedPath,
			"size_bytes":      len(screenshotBytes),
		})
		return
	}

	// Retorno padrão: A imagem binária PNG inteira (Full Page)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"fullpage_%d.png\"", time.Now().Unix()))
	c.Data(http.StatusOK, "image/png", screenshotBytes)
}
