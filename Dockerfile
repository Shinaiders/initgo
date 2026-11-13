# Multi-stage build para Go + Playwright
FROM golang:bookworm AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

# Dependências Go
COPY go.mod go.sum ./
RUN go mod download

# Código fonte
COPY . .

# Compila o binário
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server cmd/api/main.go

# Estágio final com suporte completo a navegadores headless
FROM mcr.microsoft.com/playwright:v1.50.0-jammy

WORKDIR /app

# Instala Go runtime mínimo se necessário ou copia o binário
COPY --from=builder /app/server /app/server
COPY --from=builder /app/go.mod /app/go.sum ./

# Cria diretórios necessários
RUN mkdir -p /app/storage/screenshots

# Variáveis padrão
ENV PORT=8080
ENV ENV=production
ENV DB_DRIVER=postgres
ENV PLAYWRIGHT_HEADLESS=true

EXPOSE 8080

CMD ["/app/server"]
