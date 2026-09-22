# Regulatory Compliance Engine

Motor de conformidade regulatória para a **tropicalização de serviços financeiros** (EUA → Brasil).

Dado um produto ou operação financeira, o sistema responde a duas perguntas: **pode operar no Brasil?** e **o que precisa ser adaptado?** — aplicando regras regulatórias brasileiras (PLD/COAF, Câmbio/IOF) de forma **configurável**.

> Projeto de portfólio, **fictício e original**. Cenário, dados e regras são ilustrativos e não reproduzem sistemas de nenhuma empresa. Detalhes na [especificação técnica](docs/especificacao_tecnica.md).

## Arquitetura

![Arquitetura da solução](docs/arquitetura.png)

- **`regulatory-engine`** (Go + Fiber v3): motor de regras e domínios regulatórios.
- **`legacy-gateway`** (PHP + Slim): gateway que representa o sistema legado _(blocos futuros)_.
- **MongoDB**: persistência de submissões e vereditos.

## Stack

Go 1.25 · Fiber v3 · MongoDB · Docker Compose. _(PHP/Slim, testes e CI/CD nas etapas seguintes.)_

## Como rodar (desenvolvimento)

Pré-requisitos: **Go 1.25+** e **Docker**.

```bash
# sobe o MongoDB (aguarda ficar healthy) e a API com hot reload
make dev
```

Sem `make`, manualmente:

```bash
docker compose up -d --wait mongodb
cd regulatory-engine
go run ./cmd/api
```

Teste o healthcheck:

```bash
curl http://localhost:3000/api/v1/health
# {"service":"regulatory-engine","status":"ok"}
```

## Estrutura

```
regulatory-compliance-engine/
├── docker-compose.yml        # infraestrutura local (MongoDB)
├── Makefile                  # atalhos de desenvolvimento
├── docs/                     # especificação e diagramas
└── regulatory-engine/        # motor de regras (Go)
    ├── cmd/api/              # ponto de entrada
    └── internal/
        ├── config/           # configuração via ambiente
        ├── database/         # conexão com o MongoDB
        └── httpapi/          # rotas e handlers HTTP
```

## Status

Em desenvolvimento, construído em **blocos incrementais**. Veja a [especificação técnica](docs/especificacao_tecnica.md) para o escopo completo e o roadmap.
