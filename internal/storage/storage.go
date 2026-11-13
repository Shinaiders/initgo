package storage

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"goscrapper/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageProvider interface {
	Save(ctx context.Context, filename string, data []byte, contentType string) (string, error)
	Close()
}

// LocalStorage salva os arquivos diretamente no disco local
type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	_ = os.MkdirAll(baseDir, 0755)
	return &LocalStorage{baseDir: baseDir}
}

func (l *LocalStorage) Save(ctx context.Context, filename string, data []byte, contentType string) (string, error) {
	filePath := filepath.Join(l.baseDir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("falha ao salvar arquivo localmente: %w", err)
	}
	return filePath, nil
}

func (l *LocalStorage) Close() {}

// RustFSStorage utiliza o RustFS (API compatível com S3) para armazenar objetos
type RustFSStorage struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func NewRustFSStorage(cfg *config.Config) (*RustFSStorage, error) {
	log.Printf("[RUSTFS] Conectando ao RustFS (S3) em %s (ENABLE_RUSTFS=true)...", cfg.RustFSEndpoint)

	client, err := minio.New(cfg.RustFSEndpoint, &minio.Options{
		Creds:  credentials.NewStatic(cfg.RustFSAccessKey, cfg.RustFSSecretKey, "", credentials.SignatureDefault),
		Secure: cfg.RustFSUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar cliente RustFS: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.RustFSBucket)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ou verificar bucket no RustFS (obrigatório com ENABLE_RUSTFS=true): %w", err)
	}

	if !exists {
		log.Printf("[RUSTFS] Criando bucket %s no RustFS...", cfg.RustFSBucket)
		err = client.MakeBucket(ctx, cfg.RustFSBucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("falha ao criar bucket %s no RustFS: %w", cfg.RustFSBucket, err)
		}
	}

	log.Printf("[RUSTFS] Conexão com RustFS estabelecida. Bucket pronto: %s\n", cfg.RustFSBucket)
	return &RustFSStorage{
		client:    client,
		bucket:    cfg.RustFSBucket,
		publicURL: cfg.RustFSPublicURL,
	}, nil
}

func (r *RustFSStorage) Save(ctx context.Context, filename string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)
	objectSize := int64(len(data))

	_, err := r.client.PutObject(ctx, r.bucket, filename, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("falha ao fazer upload de objeto no RustFS: %w", err)
	}

	// Retorna a URL pública ou caminho do objeto no RustFS
	fileURL := fmt.Sprintf("%s/%s/%s", r.publicURL, r.bucket, filename)
	return fileURL, nil
}

func (r *RustFSStorage) Close() {}

// InitStorage resolve se usará RustFS ou LocalStorage com base em ENABLE_RUSTFS
func InitStorage(cfg *config.Config) (StorageProvider, error) {
	if cfg.EnableRustFS {
		rustfs, err := NewRustFSStorage(cfg)
		if err != nil {
			return nil, err
		}
		return rustfs, nil
	}

	log.Println("[STORAGE] Integração com RustFS está DESABILITADA (ENABLE_RUSTFS=false). Utilizando armazenamento em disco local.")
	return NewLocalStorage(filepath.Join("storage", "screenshots")), nil
}
