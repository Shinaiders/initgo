package handler

import (
	"net/http"

	"goscrapper/internal/middleware"
	"goscrapper/pkg/response"

	"github.com/gin-gonic/gin"
)

func SetupRouter(scrapeHandler *ScrapeHandler) *gin.Engine {
	r := gin.New()

	// Middlewares padrão
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())

	// Servir pasta de screenshots publicamente
	r.Static("/storage", "./storage")

	// Healthcheck
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "Serviço operacional", gin.H{
			"status": "healthy",
		})
	})

	// Rota direta solicitada: /analisar-pagina (GET e POST)
	r.GET("/analisar-pagina", scrapeHandler.AnalisarPagina)
	r.POST("/analisar-pagina", scrapeHandler.AnalisarPagina)

	// Grupo de rotas API v1
	v1 := r.Group("/api/v1")
	{
		v1.GET("/analisar-pagina", scrapeHandler.AnalisarPagina)
		v1.POST("/analisar-pagina", scrapeHandler.AnalisarPagina)

		scrapes := v1.Group("/scrapes")
		{
			scrapes.POST("", scrapeHandler.CreateScrape)
			scrapes.GET("", scrapeHandler.ListScrapes)
			scrapes.GET("/:id", scrapeHandler.GetScrape)
			scrapes.DELETE("/:id", scrapeHandler.DeleteScrape)
		}
	}

	return r
}
