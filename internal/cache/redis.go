package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"goscrapper/internal/config"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

// InitRedis inicializa o Redis se ENABLE_REDIS estiver habilitado.
// Caso ENABLE_REDIS=true e ocorra falha de conexão, retorna erro obrigatório.
func InitRedis(cfg *config.Config) (*RedisClient, error) {
	if !cfg.EnableRedis {
		log.Println("[REDIS] Integração opcional com Redis está DESABILITADA (ENABLE_REDIS=false)")
		return nil, nil
	}

	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	log.Printf("[REDIS] Conectando ao Redis em %s (ENABLE_REDIS=true)...", addr)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("falha ao conectar ao Redis (obrigatório com ENABLE_REDIS=true): %w", err)
	}

	log.Println("[REDIS] Conexão com Redis estabelecida com sucesso.")
	return &RedisClient{client: client}, nil
}

func (r *RedisClient) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if r == nil || r.client == nil {
		return "", nil
	}
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Close() {
	if r != nil && r.client != nil {
		_ = r.client.Close()
		log.Println("[REDIS] Conexão com Redis encerrada.")
	}
}
