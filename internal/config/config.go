package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	Env             string
	DBDriver        string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	DBPath          string // For SQLite
	BrowserHeadless bool
	BrowserTimeout  int // in seconds

	// Opcionais: Redis, RabbitMQ e RustFS
	EnableRedis   bool
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	EnableRabbitMQ bool
	RabbitMQURL    string
	RabbitMQQueue  string

	EnableRustFS     bool
	RustFSEndpoint   string
	RustFSAccessKey  string
	RustFSSecretKey  string
	RustFSBucket     string
	RustFSUseSSL     bool
	RustFSPublicURL  string
}

func LoadConfig() *Config {
	// Carrega .env se existir, ignora erro se não encontrar (usa env vars do sistema)
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] Nenhum arquivo .env encontrado, utilizando variáveis de ambiente do sistema")
	}

	env := getEnv("ENV", "development")

	// No modo desenvolvimento, o padrão é Headless: false (abre o navegador visualmente)
	// No modo produção (ex: Docker), o padrão é Headless: true
	defaultHeadless := "true"
	if env == "development" {
		defaultHeadless = "false"
	}

	headless, err := strconv.ParseBool(getEnv("PLAYWRIGHT_HEADLESS", defaultHeadless))
	if err != nil {
		headless = (env != "development")
	}

	timeout, err := strconv.Atoi(getEnv("PLAYWRIGHT_TIMEOUT_SEC", "30"))
	if err != nil {
		timeout = 30
	}

	enableRedis, _ := strconv.ParseBool(getEnv("ENABLE_REDIS", "false"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	enableRabbitMQ, _ := strconv.ParseBool(getEnv("ENABLE_RABBITMQ", "false"))

	enableRustFS, _ := strconv.ParseBool(getEnv("ENABLE_RUSTFS", "false"))
	rustFSUseSSL, _ := strconv.ParseBool(getEnv("RUSTFS_USE_SSL", "false"))

	return &Config{
		Port:            getEnv("PORT", "8080"),
		Env:             env,
		DBDriver:        getEnv("DB_DRIVER", "sqlite"), // "sqlite" ou "postgres"
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "postgres"),
		DBName:          getEnv("DB_NAME", "goscrapper_db"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		DBPath:          getEnv("DB_PATH", "goscrapper.db"),
		BrowserHeadless: headless,
		BrowserTimeout:  timeout,

		// Redis
		EnableRedis:   enableRedis,
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,

		// RabbitMQ
		EnableRabbitMQ: enableRabbitMQ,
		RabbitMQURL:    getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitMQQueue:  getEnv("RABBITMQ_QUEUE", "scrape_tasks"),

		// RustFS (S3-compatible Object Storage)
		EnableRustFS:    enableRustFS,
		RustFSEndpoint:  getEnv("RUSTFS_ENDPOINT", "localhost:9000"),
		RustFSAccessKey: getEnv("RUSTFS_ACCESS_KEY", "rustfsadmin"),
		RustFSSecretKey: getEnv("RUSTFS_SECRET_KEY", "rustfsadmin"),
		RustFSBucket:    getEnv("RUSTFS_BUCKET", "goscrapper-screenshots"),
		RustFSUseSSL:    rustFSUseSSL,
		RustFSPublicURL: getEnv("RUSTFS_PUBLIC_URL", "http://localhost:9000"),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
