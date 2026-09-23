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

## Endpoints

| Método | Rota | Descrição |
|---|---|---|
| `GET`  | `/api/v1/health`            | Healthcheck |
| `POST` | `/api/v1/evaluate`          | Avalia um produto/operação, **persiste** e retorna a avaliação criada (201) |
| `GET`  | `/api/v1/evaluations`       | Lista as avaliações mais recentes |
| `GET`  | `/api/v1/evaluations/{id}`  | Recupera uma avaliação pelo id |

Exemplo de avaliação:

```bash
curl -X POST http://localhost:3000/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
        "product":   { "type": "personal_loan", "origin": "US", "origin_currency": "USD" },
        "operation": { "amount": "75000.00", "currency": "USD", "method": "international_transfer",
                       "counterparty": { "name": "John Doe", "pep": false } }
      }'
```

Resposta (`201 Created`) — a avaliação persistida (`id`, `created_at`, `request` e o `report`):

```json
{
  "id": "665f...c3",
  "created_at": "2026-09-23T16:00:00Z",
  "request": { "product": { "...": "..." }, "operation": { "...": "..." } },
  "report": {
    "compliant": false,
    "results": [
    {
      "domain": "IOF",
      "status": "adaptation_required",
      "detail": "Entrada convertida de USD para BRL a 5.00; IOF de 0.38% aplicavel...",
      "calculated_amount": "1425.00",
      "reference": "rule:iof.international_transfer"
    },
    {
      "domain": "PLD_COAF",
      "status": "reportable",
      "detail": "Valor de 375000.00 BRL atinge o teto de comunicacao (50000.00 BRL); operacao reportavel ao COAF.",
      "calculated_amount": "375000.00",
      "reference": "rule:pld.reporting_threshold"
    }
  ],
    "required_adaptations": [
      "Incluir calculo e retencao de IOF no fluxo de entrada.",
      "Gerar comunicacao ao COAF para a operacao."
    ]
  }
}
```

> Os valores regulatórios (alíquotas, limites, câmbio) vivem em `regulatory-engine/config/rules.json` e são **configuráveis** — mudar a norma não exige recompilar a lógica.

## Estrutura

```
regulatory-compliance-engine/
├── docker-compose.yml        # infraestrutura local (MongoDB)
├── Makefile                  # atalhos de desenvolvimento
├── docs/                     # especificação e diagramas
└── regulatory-engine/        # motor de regras (Go)
    ├── cmd/api/              # ponto de entrada
    ├── config/rules.json     # parametrização regulatória (alíquotas, limites, câmbio)
    └── internal/
        ├── config/           # configuração via ambiente
        ├── database/         # conexão com o MongoDB
        ├── model/            # structs do domínio (contrato)
        ├── money/            # tipo monetário (decimal, sempre 2 casas)
        ├── engine/           # interface Rule + avaliador
        ├── rules/            # regras (ex.: IOF) + carregamento do rules.json
        └── httpapi/          # servidor, rotas e handlers HTTP
```

## Status

Em desenvolvimento, construído em **blocos incrementais**. Veja a [especificação técnica](docs/especificacao_tecnica.md) para o escopo completo.

## Roadmap

- **Gateway PHP + Slim** — serviço legado que consome o motor Go (integração legado ↔ novo).
- **Identidade de contraparte (`parties`)** — promover a contraparte a entidade própria, identificada por documento (CPF/CNPJ), com *entity resolution* para screening.
- **Novos domínios** — Limites Bacen/Pix, LGPD, SCR.
- **Observabilidade completa** — Prometheus + Grafana + Loki.
- **Regras versionadas em banco** — histórico de vigência das normas.

Detalhes na [especificação técnica](docs/especificacao_tecnica.md#9-roadmap-evolução-futura).
