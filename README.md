# initgo 🚀

[![CI](https://github.com/Shinaiders/initgo/actions/workflows/ci.yml/badge.svg)](https://github.com/Shinaiders/initgo/actions/workflows/ci.yml)
[![Template](https://img.shields.io/badge/Template-GitHub-2ea44f?logo=github)](https://github.com/Shinaiders/initgo)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)](https://go.dev/)
[![npm](https://img.shields.io/badge/npm-goinitshinaider-CB3837?logo=npm)](https://www.npmjs.com/package/create-goinitshinaider)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Template / Boilerplate para projetos em **Go (Golang)** de alta performance estruturado com **Clean Architecture**, integrando a API web **Gin**, motor de automação e raspagem **Playwright**, ORM **GORM** (PostgreSQL/SQLite), além de suporte para **Redis**, **RabbitMQ** e **RustFS** (S3-compatible object storage).

---

## ⚡ Como Criar um Novo Projeto com Este Template

### Opção 1: Via `npx` / `npm init` (Mais Rápido e Automático 🚀)
Basta rodar um único comando no seu terminal:

```bash
# Via npm registry (recomendado):
npx goinitshinaider meu-novo-projeto
# ou
npm init goinitshinaider meu-novo-projeto

# Ou diretamente pelo repositório GitHub (sem precisar do npm):
npx github:Shinaiders/initgo meu-novo-projeto
```
O utilitário clona o template, renomeia todos os módulos e imports Go, configura o `.env`, inicializa o git e executa o `go mod tidy` automaticamente.

---

### Opção 2: Via GitHub CLI
```bash
# 1. Crie seu novo repositório a partir deste template e clone-o
gh repo create meu-novo-projeto --template Shinaiders/initgo --clone --public

# 2. Acesse a pasta do projeto
cd meu-novo-projeto

# 3. Inicialize o projeto renomeando o módulo e configurando dependências
make init MODULE=meu-novo-projeto
```

### Opção 3: Pela Interface do GitHub
1. Clique no botão verde **"Use this template"** > **"Create a new repository"** no topo desta página.
2. Dê um nome ao seu novo repositório e clone-o na sua máquina:
   ```bash
   git clone https://github.com/SEU_USUARIO/SEU_PROJETO.git
   cd SEU_PROJETO
   ```
3. Execute a inicialização para configurar o novo módulo:
   ```bash
   make init MODULE=github.com/SEU_USUARIO/SEU_PROJETO
   ```

---

## 🏛️ Arquitetura e Estrutura de Pastas

```
goscrapper/
├── cmd/
│   └── api/
│       └── main.go              # Entrypoint da aplicação, DI e Graceful Shutdown
├── internal/
│   ├── cache/                   # Cliente Redis opcional (cache de jobs e responses)
│   │   └── redis.go
│   ├── config/                  # Carregamento e validação de variáveis de ambiente
│   │   └── config.go
│   ├── database/                # Conexão GORM (PostgreSQL/SQLite) e AutoMigrations
│   │   └── database.go
│   ├── handler/                 # Controllers HTTP (Gin)
│   │   ├── router.go            # Registro de rotas e middlewares
│   │   └── scrape_handler.go    # Endpoints de scraping
│   ├── middleware/              # Middlewares Gin (CORS, Logger, Recovery)
│   │   └── cors.go
│   ├── model/                   # Entidades GORM e DTOs de entrada/saída
│   │   └── scrape_job.go
│   ├── queue/                   # Cliente RabbitMQ opcional e Worker de consumo
│   │   ├── rabbitmq.go
│   │   └── worker.go
│   ├── repository/              # Camada de persistência e queries GORM
│   │   └── scrape_repository.go
│   ├── scraper/                 # Motor de automação com Playwright
│   │   └── engine.go            # Gerenciamento de Browser, Contexts e Pages
│   ├── service/                 # Regras de negócio e orquestração (sync/async/rabbit)
│   │   └── scrape_service.go
│   └── storage/                 # Provedores de Storage (Local Disk vs RustFS / S3)
│       └── storage.go
├── pkg/
│   └── response/                # Padronização de respostas JSON e paginação
│       └── response.go
├── storage/
│   └── screenshots/             # Armazenamento local de capturas de tela (fallback)
├── .env.example                 # Exemplo de variáveis de ambiente
├── docker-compose.yml           # App + PostgreSQL + Redis + RabbitMQ + RustFS
├── Dockerfile                   # Build multi-stage e runtime com Playwright
├── Makefile                     # Comandos rápidos de automação
├── go.mod
└── go.sum
```

---

## ⚡ Integrações Opcionais vs. Obrigatórias (Feature Flags)

Você pode habilitar ou desabilitar cada serviço alterando apenas as variáveis booleanas no arquivo `.env`:

| Serviço      | Variável de Ambiente | Padrão  | Quando `false` (Opcional)                                   | Quando `true` (Obrigatório)                                                                |
| :----------- | :------------------- | :------ | :---------------------------------------------------------- | :----------------------------------------------------------------------------------------- |
| **Redis**    | `ENABLE_REDIS`       | `false` | A aplicação ignora o Redis e busca diretamente do banco     | **Obrigatório**. Se o Redis não responder no boot, a aplicação falha e encerra             |
| **RabbitMQ** | `ENABLE_RABBITMQ`    | `false` | Tarefas assíncronas rodam via Goroutines locais             | **Obrigatório**. Tarefas vão para fila `scrape_tasks` e o worker interno consome           |
| **RustFS**   | `ENABLE_RUSTFS`      | `false` | Screenshots são salvos localmente em `storage/screenshots/` | **Obrigatório**. Screenshots sobem para o bucket do RustFS via S3 e retornam a URL pública |

---

## ⚙️ Pré-requisitos

1. **Go 1.22+** instalado.
2. **Playwright drivers e navegadores instalados**:
   ```bash
   make install-playwright
   ```

---

## 🚀 Como Executar

### Opção 1: Localmente (Desenvolvimento Rápido)

1. Clone o repositório e configure as variáveis:

   ```bash
   cp .env.example .env
   ```

2. Instale os navegadores do Playwright:

   ```bash
   make install-playwright
   ```

3. Inicie o servidor com **Live / Hot Reload** (reinicia sozinho ao salvar código):
   ```bash
   make dev
   ```
   _(Ou sem reload: `make run`)_

O servidor iniciará em `http://localhost:8080`. Toda vez que você salvar qualquer arquivo `.go`, o **Air** recompilará e reiniciará a aplicação instantaneamente.

---

### Opção 2: Subir Apenas as Dependências (PostgreSQL, Redis, RabbitMQ, RustFS)

Ideal para desenvolvimento local e ambiente de testes com `make dev`, sem subir o container da API Go:

```bash
make deps-up
```

As dependências ficarão expostas no seu `localhost`:

- **PostgreSQL:** `localhost:5432`
- **Redis:** `localhost:6379`
- **RabbitMQ:** `localhost:5672` (Painel Web: `http://localhost:15672`)
- **RustFS (S3):** `localhost:9000` (Console Web: `http://localhost:9001`)

Para parar apenas as dependências:

```bash
make deps-down
```

---

### Opção 3: Com Docker e Docker Compose (Stack Completa com API)

O arquivo `docker-compose.yml` inclui **API + PostgreSQL + Redis + RabbitMQ + RustFS**:

```bash
make docker-up
```

Painéis web disponíveis no Docker Compose:

- **RabbitMQ Management:** `http://localhost:15672` (login: `guest` / `guest`)
- **RustFS Console:** `http://localhost:9001` (login: `rustfsadmin` / `rustfsadmin`)
- **API GoScrapper:** `http://localhost:8080`

Para parar os containers:

```bash
make docker-down
```

---

## 📡 Endpoints da API

### 1. Analisar Página com Print Inteiro (Full Page Screenshot) 📸

Abre a URL solicitada, aguarda a renderização e retorna a captura de tela **inteira** do site.

- **Endpoints:** `/analisar-pagina` e `/api/v1/analisar-pagina`
- **Métodos:** `GET` e `POST`

#### Opção A: Visualizar/Baixar a Imagem Diretamente (Padrão)

Via Navegador ou cURL:

```bash
curl -o site_inteiro.png "http://localhost:8080/analisar-pagina?url=https://github.com"
```

Ou via POST:

```bash
curl -X POST http://localhost:8080/analisar-pagina \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com"}' \
  --output print_completo.png
```

#### Opção B: Retorno em JSON (com metadados e caminho salvo)

Adicione `format=json`:

```bash
curl -X POST http://localhost:8080/analisar-pagina \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com", "format": "json"}'
```

Resposta:

```json
{
  "success": true,
  "message": "Página analisada e print capturado com sucesso",
  "data": {
    "url": "https://github.com",
    "screenshot_path": "storage/screenshots/fullpage_1731454800000.png",
    "size_bytes": 1425320
  }
}
```

---

### 2. Healthcheck

- **GET** `/health`

```bash
curl -X GET http://localhost:8080/health
```

---

### 2. Disparar Scraping (Síncrono ou Assíncrono)

- **POST** `/api/v1/scrapes`

**Body:**

```json
{
  "url": "https://example.com",
  "take_screenshot": true,
  "wait_selector": "h1",
  "async": false
}
```

Exemplo com cURL:

```bash
curl -X POST http://localhost:8080/api/v1/scrapes \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "take_screenshot": true,
    "async": false
  }'
```

---

### 3. Listar Histórico de Scrapings (Paginado)

- **GET** `/api/v1/scrapes?page=1&limit=10`

```bash
curl -X GET "http://localhost:8080/api/v1/scrapes?page=1&limit=10"
```

---

### 4. Consultar Scraping por ID (com cache Redis se ativo)

- **GET** `/api/v1/scrapes/:id`

```bash
curl -X GET http://localhost:8080/api/v1/scrapes/1
```

---

### 5. Remover Registro de Scraping

- **DELETE** `/api/v1/scrapes/:id`

```bash
curl -X DELETE http://localhost:8080/api/v1/scrapes/1
```
