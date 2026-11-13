#!/usr/bin/env bash
set -euo pipefail

# Script de inicialização para repositórios criados a partir do template initgo (goscrapper)
# Uso: make init MODULE=seu-novo-modulo ou ./scripts/setup-template.sh [novo-modulo]

OLD_MODULE="goscrapper"
NEW_MODULE="${1:-}"

echo "=========================================================="
echo "          🚀 Inicializador de Projeto (initgo)            "
echo "=========================================================="

if [ -z "$NEW_MODULE" ]; then
    printf "Digite o nome do novo módulo (ex: github.com/seu-usuario/meu-crawler ou meu-crawler): "
    read -r NEW_MODULE
fi

if [ -z "$NEW_MODULE" ]; then
    echo "❌ Erro: O nome do módulo não pode ser vazio."
    exit 1
fi

if [ "$NEW_MODULE" = "$OLD_MODULE" ]; then
    echo "ℹ️ O nome do módulo já é '$OLD_MODULE'. Nenhuma substituição necessária."
    exit 0
fi

# Extrair apenas o nome base (caso seja uma URL como github.com/user/meuapp -> meuapp)
BASE_APP_NAME=$(basename "$NEW_MODULE")

echo "📦 Renomeando módulo de '$OLD_MODULE' para '$NEW_MODULE'..."

# 1. Substituir no go.mod
if [ -f "go.mod" ]; then
    echo "  -> Atualizando go.mod..."
    perl -pi -e "s|module $OLD_MODULE|module $NEW_MODULE|g" go.mod
fi

# 2. Substituir em todos os arquivos Go
echo "  -> Atualizando imports nos arquivos Go..."
find cmd internal pkg -type f -name "*.go" -exec perl -pi -e "s|\"$OLD_MODULE/|\"$NEW_MODULE/|g" {} +

# 3. Substituir no Makefile (APP_NAME)
if [ -f "Makefile" ]; then
    echo "  -> Atualizando Makefile..."
    perl -pi -e "s|APP_NAME=$OLD_MODULE|APP_NAME=$BASE_APP_NAME|g" Makefile
fi

# 4. Substituir no docker-compose.yml
if [ -f "docker-compose.yml" ]; then
    echo "  -> Atualizando docker-compose.yml..."
    perl -pi -e "s|${OLD_MODULE}_|${BASE_APP_NAME}_|g" docker-compose.yml
    perl -pi -e "s|${OLD_MODULE}-|${BASE_APP_NAME}-|g" docker-compose.yml
    perl -pi -e "s|${OLD_MODULE}\.db|${BASE_APP_NAME}.db|g" docker-compose.yml
fi

# 5. Configurar .env a partir de .env.example
if [ ! -f ".env" ] && [ -f ".env.example" ]; then
    echo "  -> Criando .env a partir de .env.example..."
    cp .env.example .env
    perl -pi -e "s|${OLD_MODULE}_|${BASE_APP_NAME}_|g" .env
    perl -pi -e "s|${OLD_MODULE}\.db|${BASE_APP_NAME}.db|g" .env
fi

# 6. Atualizar dependências Go
echo "  -> Executando go mod tidy..."
go mod tidy

echo ""
echo "=========================================================="
echo "  ✅ Projeto inicializado com sucesso para: $NEW_MODULE"
echo "=========================================================="
echo "Próximos passos:"
echo "  1. make install-playwright (se precisar do navegador Chromium)"
echo "  2. make deps-up            (sobe postgres, redis, rabbitmq e rustfs)"
echo "  3. make dev                (inicia a aplicação com live-reload)"
echo "=========================================================="
