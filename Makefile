ENGINE = regulatory-engine

.PHONY: help up down mongo-up mongo-logs run dev build test test-integration tidy fmt vet

help: ## Mostra esta ajuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

up: ## Sobe a stack completa em containers (MongoDB + motor)
	docker compose up -d --build --wait

down: ## Para a stack completa
	docker compose down

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
	cd $(ENGINE) && MONGO_TEST_URI=mongodb://admin:admin123@localhost:27017 go test -count=1 ./...

tidy: ## Ajusta as dependencias
	cd $(ENGINE) && go mod tidy

fmt: ## Formata o codigo
	cd $(ENGINE) && go fmt ./...

vet: ## Analise estatica basica
	cd $(ENGINE) && go vet ./...