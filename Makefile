ENGINE = regulatory-engine
GATEWAY = legacy-gateway

# Roda comandos do gateway em containers (nao exige PHP local). O composer:2
# instala as dependencias; os testes rodam no mesmo PHP da imagem (8.3).
GATEWAY_RUN = docker run --rm -v "$(CURDIR)/$(GATEWAY):/app" -w /app

# Variaveis de ambiente vao por "export" do proprio make, e nao inline
# (VAR=valor comando): no Windows, chamado pelo PowerShell, o make executa as
# receitas no cmd.exe, que nao entende a forma inline.
# MSYS_NO_PATHCONV evita que o Git Bash reescreva os caminhos do docker run.
gateway-install gateway-test postman: export MSYS_NO_PATHCONV := 1
test-integration: export MONGO_TEST_URI := mongodb://admin:admin123@localhost:27017

# O E2E e um script bash. No Windows, o primeiro "bash" do PATH pode ser o do
# WSL, que nao enxerga o Docker Desktop; por isso usa-se o bash do Git for
# Windows, localizado pelo proprio git (sem caminho fixo de instalacao).
ifeq ($(OS),Windows_NT)
BASH = "$(shell git --exec-path)/../../../bin/bash.exe"
else
BASH = bash
endif

.PHONY: help up down reset mongo-up mongo-logs run dev build test test-integration tidy fmt vet gateway-install gateway-test e2e postman

help: ## Mostra esta ajuda (usa grep e awk: no Windows, rode pelo Git Bash)
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

up: ## Sobe a stack completa em containers (MongoDB + motor + gateway)
	docker compose up -d --build --wait

down: ## Para a stack completa
	docker compose down

# Apaga so os volumes DESTE projeto (regulatory-compliance-engine_*): o
# "down -v" do compose e restrito ao projeto. Nao usar "docker volume prune"
# nem "docker system prune", que afetam volumes de todos os projetos.
reset: ## Apaga os dados do MongoDB e do MySQL deste projeto e sobe a stack limpa
	docker compose down -v
	docker compose up -d --build --wait

mongo-up: ## Sobe o MongoDB (espera ficar healthy)
	docker compose up -d --wait mongodb

mongo-logs: ## Acompanha os logs do MongoDB
	docker compose logs -f mongodb

dev: ## Sobe o Mongo e roda a API com hot reload (Air)
	docker compose up -d --wait mongodb
	cd $(ENGINE) && air

run: ## Compila e executa a API (sem hot reload)
	cd $(ENGINE) && go run ./cmd/api

build: ## Compila o binario da API
	cd $(ENGINE) && go build -o bin/regulatory-engine ./cmd/api

test: ## Executa os testes (integracao do repositorio e pulada sem MONGO_TEST_URI)
	cd $(ENGINE) && go test ./...

test-integration: ## Sobe o MongoDB e executa todos os testes, incluindo integracao
	docker compose up -d --wait mongodb
	cd $(ENGINE) && go test -count=1 ./...

tidy: ## Ajusta as dependencias
	cd $(ENGINE) && go mod tidy

fmt: ## Formata o codigo
	cd $(ENGINE) && go fmt ./...

vet: ## Analise estatica basica
	cd $(ENGINE) && go vet ./...

gateway-install: ## Instala as dependencias do gateway (composer.lock)
	$(GATEWAY_RUN) composer:2 composer install --no-interaction --no-progress

e2e: ## Sobe a stack e roda o E2E (propostas pelo gateway ate o MySQL e o MongoDB)
	docker compose up -d --build --wait
	$(BASH) scripts/e2e.sh

postman: ## Roda a collection do Postman (Newman, em container) contra a stack no ar
	docker run --rm --network regulatory-compliance-engine_default -v "$(CURDIR)/docs/postman:/etc/newman" postman/newman:6-alpine run regulatory-compliance-engine.postman_collection.json --env-var baseUrl=http://regulatory-engine:3000/api/v1 --env-var gatewayUrl=http://legacy-gateway --reporter-cli-no-banner

gateway-test: gateway-install ## Analise estatica (PHPStan) e testes (PHPUnit) do gateway
	$(GATEWAY_RUN) php:8.3-cli sh -c "vendor/bin/phpstan analyse --no-progress && vendor/bin/phpunit"