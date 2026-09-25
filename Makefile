ENGINE = regulatory-engine

.PHONY: help up down mongo-up mongo-logs run dev build test tidy fmt vet

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

test: ## Executa os testes
	cd $(ENGINE) && go test ./...

tidy: ## Ajusta as dependencias
	cd $(ENGINE) && go mod tidy

fmt: ## Formata o codigo
	cd $(ENGINE) && go fmt ./...

vet: ## Analise estatica basica
	cd $(ENGINE) && go vet ./...