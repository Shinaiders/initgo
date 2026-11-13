.PHONY: all build run dev test tidy install-playwright docker-up docker-down docker-logs deps-up deps-down clean init

# Variáveis
APP_NAME=goscrapper
MAIN_FILE=cmd/api/main.go

all: build

## Inicializa o novo projeto a partir deste template renomeando o módulo
init:
	@chmod +x scripts/setup-template.sh
	@./scripts/setup-template.sh $(MODULE)

## Instala os drivers e navegadores do Playwright localmente
install-playwright:
	go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6401.0 install --with-deps

## Organiza as dependências do Go
tidy:
	go mod tidy

## Compila o binário
build: tidy
	go build -o bin/$(APP_NAME) $(MAIN_FILE)

## Executa localmente com Live / Hot Reload (Air) e navegador visível
dev:
	@if command -v air > /dev/null; then \
		PLAYWRIGHT_HEADLESS=false air; \
	elif [ -f $(shell go env GOPATH)/bin/air ]; then \
		PLAYWRIGHT_HEADLESS=false $(shell go env GOPATH)/bin/air; \
	else \
		echo "Instalando Air..."; \
		go install github.com/air-verse/air@latest; \
		PLAYWRIGHT_HEADLESS=false $(shell go env GOPATH)/bin/air; \
	fi

## Executa localmente a aplicação (sem reload)
run:
	go run $(MAIN_FILE)

## Executa os testes
test:
	go test -v ./...

## Sobe apenas as dependências/infra (PostgreSQL, Redis, RabbitMQ, RustFS) sem a API (para testes e dev local)
deps-up:
	docker compose up -d db redis rabbitmq rustfs

## Para apenas as dependências/infra
deps-down:
	docker compose stop db redis rabbitmq rustfs

## Sobe o ambiente completo no Docker (App + Dependências)
docker-up:
	docker compose up --build -d

## Para o ambiente Docker
docker-down:
	docker compose down

## Visualiza logs dos containers
docker-logs:
	docker compose logs -f

## Limpa arquivos temporários de build e screenshots
clean:
	rm -rf bin/ tmp/ storage/screenshots/* *.db build-errors.log
