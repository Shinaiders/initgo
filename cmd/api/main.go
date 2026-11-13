package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goscrapper/internal/cache"
	"goscrapper/internal/config"
	"goscrapper/internal/database"
	"goscrapper/internal/handler"
	"goscrapper/internal/queue"
	"goscrapper/internal/repository"
	"goscrapper/internal/scraper"
	"goscrapper/internal/service"
	"goscrapper/internal/storage"
)

func main() {
	log.Println("[BOOT] Carregando configurações...")
	cfg := config.LoadConfig()

	// 1. Inicializa o banco de dados (GORM)
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("[FATAL] Erro ao conectar ao banco de dados: %v", err)
	}

	// 2. Inicializa o armazenamento (RustFS via S3 se ENABLE_RUSTFS=true, ou Local Disk)
	storageProvider, err := storage.InitStorage(cfg)
	if err != nil {
		log.Fatalf("[FATAL] Erro ao inicializar Storage (RustFS obrigatório com ENABLE_RUSTFS=true): %v", err)
	}
	defer storageProvider.Close()

	// 3. Inicializa o motor de automação Playwright
	scraperEngine, err := scraper.NewEngine(cfg, storageProvider)
	if err != nil {
		log.Fatalf("[FATAL] Erro ao inicializar Playwright: %v", err)
	}
	defer scraperEngine.Close()

	// 4. Inicializa o Redis (Opcional, obrigatório se ENABLE_REDIS=true)
	redisClient, err := cache.InitRedis(cfg)
	if err != nil {
		log.Fatalf("[FATAL] Erro ao conectar ao Redis: %v", err)
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	// 5. Inicializa o RabbitMQ (Opcional, obrigatório se ENABLE_RABBITMQ=true)
	rabbitClient, err := queue.InitRabbitMQ(cfg)
	if err != nil {
		log.Fatalf("[FATAL] Erro ao conectar ao RabbitMQ: %v", err)
	}
	if rabbitClient != nil {
		defer rabbitClient.Close()
	}

	// 6. Injeção de dependências (Repository -> Service -> Handler)
	scrapeRepo := repository.NewScrapeRepository(db)
	scrapeService := service.NewScrapeService(scrapeRepo, scraperEngine, redisClient, rabbitClient)
	scrapeHandler := handler.NewScrapeHandler(scrapeService)

	// 7. Contexto global de ciclo de vida da aplicação
	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	// Se RabbitMQ estiver ativo, inicia o worker de consumo assíncrono em background
	if rabbitClient != nil {
		queue.StartWorker(appCtx, rabbitClient, scrapeService)
	}

	// 8. Configuração das rotas Gin
	router := handler.SetupRouter(scrapeHandler)

	// 9. Configuração do HTTP Server com Graceful Shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[SERVER] Servidor HTTP rodando na porta :%s\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Erro no servidor HTTP: %v", err)
		}
	}()

	// Aguarda sinal de interrupção (Ctrl+C ou SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[SHUTDOWN] Desligando aplicação graciosamente...")

	// Cancela contexto global de workers
	cancelApp()

	// Timeout de 15 segundos para encerrar requisições em andamento
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[FATAL] Erro ao forçar desligamento do servidor: %v", err)
	}

	// Fecha conexões com o banco de dados
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}

	log.Println("[SHUTDOWN] Servidor finalizado com sucesso.")
}
