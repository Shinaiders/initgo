package database

import (
	"fmt"
	"log"
	"time"

	"goscrapper/internal/config"
	"goscrapper/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB inicializa a conexão com o banco de dados via GORM
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	logLevel := logger.Warn
	if cfg.Env == "development" {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	switch cfg.DBDriver {
	case "postgres":
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
		)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(cfg.DBPath)
	default:
		return nil, fmt.Errorf("driver de banco de dados não suportado: %s", cfg.DBDriver)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no banco de dados (%s): %w", cfg.DBDriver, err)
	}

	// Configuração do pool de conexões SQL
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter sql.DB subjacente: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Printf("[DATABASE] Conexão estabelecida com sucesso usando driver: %s\n", cfg.DBDriver)

	// Executa AutoMigrate
	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}

func runMigrations(db *gorm.DB) error {
	log.Println("[DATABASE] Executando auto-migrations...")
	if err := db.AutoMigrate(
		&model.ScrapeJob{},
		&model.User{},
	); err != nil {
		return fmt.Errorf("falha ao rodar migrations: %w", err)
	}
	log.Println("[DATABASE] Migrations aplicadas com sucesso.")
	return nil
}
