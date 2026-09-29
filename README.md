# Regulatory Compliance Engine

![CI](https://github.com/hfgpaulo/regulatory-compliance-engine/actions/workflows/ci.yml/badge.svg)

Motor de conformidade regulatória para a **tropicalização de serviços financeiros** (EUA → Brasil).

Dado um produto ou operação financeira, o sistema responde a duas perguntas: **pode operar no Brasil?** e **o que precisa ser adaptado?** — aplicando regras regulatórias brasileiras (PLD/COAF, Câmbio/IOF) de forma **configurável**.

> Projeto de portfólio, **fictício e original**. Cenário, dados e regras são ilustrativos e não reproduzem sistemas de nenhuma empresa. Detalhes na [especificação técnica](docs/especificacao_tecnica.md).

## Arquitetura

![Arquitetura da solução](docs/arquitetura.png)

Fluxo principal: **proposta no formato legado → gateway PHP → motor Go**.

- **`regulatory-engine`** (Go + Fiber v3): motor de regras e domínios regulatórios. Grava cada avaliação no **MongoDB** (auditoria, fonte da verdade).
- **`legacy-gateway`** (PHP + Slim): representa o sistema legado. Atua como **camada anticorrupção** — traduz o formato legado (campos em português, valores em centavos, `"S"`/`"N"`) para o contrato do motor e de volta — e registra as propostas aceitas no **MySQL legado**.
- Os dois bancos se ligam pelo `protocolo` (o id da avaliação no motor).

## Stack

Go 1.25 · Fiber v3 · MongoDB · PHP 8.3 · Slim 4 · MySQL 8.4 · Docker Compose · GitHub Actions.

## Como rodar

Com **Docker** apenas (sem Go nem PHP instalados), a stack completa — MongoDB, motor (porta 3000), MySQL legado (porta 3307) e gateway (porta 8080) — sobe em containers, cada serviço só depois de suas dependências ficarem saudáveis:

```bash
docker compose up -d --build --wait
```

Ou `make up` / `make down`. A imagem do motor é multi-stage (binário estático sobre distroless, usuário sem privilégio) e leva o `rules.json` embutido; a do gateway é `php:8.3-apache` e cria a tabela do banco legado ao subir.

Confira que os serviços respondem:

```bash
curl http://localhost:3000/api/v1/ready
curl http://localhost:8080/health
```

> **Windows:** no PowerShell, `curl` é um apelido de `Invoke-WebRequest`, outro comando. Use `curl.exe` para as chamadas simples acima. Para os exemplos com corpo JSON (`-d '{...}'`), use o **Git Bash**, porque o PowerShell altera as aspas, ou o arquivo [`docs/cenarios_de_teste.http`](docs/cenarios_de_teste.http) no VS Code.

### Consultar os bancos

| Banco | Ferramenta | Conexão |
|---|---|---|
| MongoDB (motor) — coleção `evaluations` | MongoDB Compass ou `mongosh` | `mongodb://admin:admin123@localhost:27017`, banco `regulatory` |
| MySQL (legado) — tabela `propostas` | DBeaver (driver MySQL) | host `localhost`, porta `3307`, banco `legacy`, usuário `legacy`, senha `legacy123` |

Credenciais de desenvolvimento, não secretas. No DBeaver, se aparecer "Public Key Retrieval is not allowed", defina `allowPublicKeyRetrieval=true` nas propriedades do driver.

### Desenvolvimento (hot reload)

Pré-requisitos: **Go 1.25+**, **Docker** e [Air](https://github.com/air-verse/air). Suba só o MongoDB e rode a API localmente — não use junto com `make up`, pois ambos ocupam a porta 3000.

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

## Testes

Todos os comandos abaixo rodam a partir da **raiz do repositório**, onde fica o `Makefile`.

| Comando | O que roda | Requer |
|---|---|---|
| `go -C regulatory-engine test ./...` | Motor: unitários, handlers (dublê em memória, sem MongoDB) e aceitação (cenários regulatórios com o `rules.json` real) | Go 1.25 |
| `make test-integration` | O mesmo, mais a integração do repositório contra um MongoDB real (sobe o Mongo do compose) | Go 1.25, Docker, `make` |
| `make gateway-test` | Gateway: PHPStan e PHPUnit, em container, sem PHP local | Docker, `make` |
| `make e2e` | Contrato entre gateway e motor: sobe a stack e passa propostas reais pelo gateway, conferindo a resposta, o MySQL e o MongoDB | Docker, `make`, bash (no Windows, Git for Windows) |

Sem `MONGO_TEST_URI`, os testes de integração do repositório são **pulados**, e o `go test` mostra o pacote como `ok` mesmo assim. Use `-v` para ver o `SKIP`, ou rode o `make test-integration`.

> **Windows:** os alvos precisam do **GNU make**, que não vem instalado (ex.: `choco install make` ou `scoop install make`; o MinGW instala como `mingw32-make`). Confira com `make --version`: se não aparecer "GNU Make", outro programa chamado `make` está antes no PATH. Todos os alvos funcionam no PowerShell e no Git Bash, exceto o `help`, que usa `grep` e `awk` e precisa do Git Bash. O `e2e` é um script bash: no Windows, o `make` usa automaticamente o bash do **Git for Windows**, que precisa estar instalado, e não o do WSL, que não enxerga o Docker Desktop.

No CI, a cada push/PR: motor (`gofmt`, `build`, `vet`, `test` com integração), gateway (PHPStan, PHPUnit), build das imagens e E2E.

Para testar a API na prática, há um catálogo de cenários com entrada e resultado esperado em [`docs/cenarios_de_teste.md`](docs/cenarios_de_teste.md), prontos para executar pelo VS Code (REST Client) em [`docs/cenarios_de_teste.http`](docs/cenarios_de_teste.http). Os cenários regulatórios são os mesmos do teste automatizado de aceitação.

## Endpoints

### Gateway legado (porta 8080)

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/propostas` | Recebe a proposta no formato legado, avalia no motor, registra no MySQL legado e responde no formato legado (201) |
| `GET`  | `/health`    | Liveness |

```bash
curl -X POST http://localhost:8080/propostas \
  -H "Content-Type: application/json" \
  -d '{
        "produto":  { "tipo": "EMPRESTIMO_PESSOAL", "pais_origem": "US", "moeda_origem": "USD" },
        "operacao": { "valor_centavos": 7500000, "moeda": "USD", "modalidade": "TRANSFERENCIA_INTERNACIONAL",
                      "contraparte": { "nome": "John Doe", "pep": "N" } }
      }'
```

```json
{ "protocolo": "6abab272d44d8dedcac5a7a6", "situacao": "COMUNICAR_COAF", "iof_centavos": 142500,
  "exigencias": ["Incluir calculo e retencao de IOF no fluxo de entrada.", "Gerar comunicacao ao COAF para a operacao."],
  "versao_regras": "sha256:cab75867...2dba4f" }
```

Proposta inválida retorna `422` com os campos no vocabulário legado — inclusive os que o motor rejeitou, traduzidos; motor indisponível retorna `503`.

### Motor (porta 3000)

| Método | Rota | Descrição |
|---|---|---|
| `GET`  | `/api/v1/health`            | Liveness: processo no ar |
| `GET`  | `/api/v1/ready`             | Readiness: `200` se o MongoDB responde, `503` se não |
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

Resposta (`201 Created`) — a avaliação persistida (`id`, `created_at`, `rules_version` — hash da parametrização usada —, `request` e o `report`):

```json
{
  "id": "665f...c3",
  "created_at": "2026-09-23T16:00:00Z",
  "rules_version": "sha256:cab75867...2dba4f",
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

Requisição inválida (campo ausente, valor não positivo ou com mais de 2 casas, tipo/método desconhecido, código de país/moeda fora do padrão ISO, moeda da operação diferente de `USD`) retorna `400` com todos os campos inválidos, e nada é persistido:

```json
{
  "error": "requisicao invalida",
  "details": [{ "field": "operation.amount", "message": "deve ser maior que zero" }]
}
```

As regras completas dos dois contratos estão na [especificação técnica](docs/especificacao_tecnica.md#5-contratos-das-apis).

> Os valores regulatórios (alíquotas, limites, câmbio) vivem em `regulatory-engine/config/rules.json` e são **configuráveis** — mudar a norma não exige recompilar a lógica. O arquivo é validado no boot: parametrização incoerente impede o serviço de subir.

## Estrutura

```
regulatory-compliance-engine/
├── docker-compose.yml        # MongoDB + motor, MySQL + gateway
├── Makefile                  # atalhos de desenvolvimento
├── scripts/e2e.sh            # E2E: gateway -> motor -> MySQL e MongoDB
├── docs/                     # especificação, diagramas e cenários de teste
├── regulatory-engine/        # motor de regras (Go)
│   ├── cmd/api/              # ponto de entrada
│   ├── config/rules.json     # parametrização regulatória (alíquotas, limites, câmbio)
│   └── internal/
│       ├── config/           # configuração via ambiente
│       ├── database/         # conexão com o MongoDB
│       ├── model/            # structs do domínio (contrato) + validação
│       ├── money/            # tipo monetário (decimal, sempre 2 casas)
│       ├── engine/           # interface Rule + avaliador
│       ├── rules/            # regras (IOF, PLD) + carregamento do rules.json
│       ├── repository/       # persistência das avaliações
│       └── httpapi/          # servidor, rotas e handlers HTTP
└── legacy-gateway/           # sistema legado (PHP + Slim)
    ├── public/index.php      # ponto de entrada
    ├── bin/migrate.php       # cria a tabela do MySQL legado ao subir
    ├── src/                  # tradução, cliente do motor, handler, repositório
    └── tests/                # PHPUnit
```

## Status

Funcional, construído em **blocos incrementais**. Já entrega:

- **Regras regulatórias** de IOF (transferência internacional) e PLD/COAF (teto de comunicação e screening de PEP/sancionados), parametrizadas em `rules.json`.
- **Parametrização validada no boot** (fail-fast): valores incoerentes impedem o serviço de subir.
- **Validação de entrada**: requisição incompleta retorna `400` com todos os campos inválidos, em vez de sair "conforme" por omissão.
- **Auditoria**: cada avaliação é persistida com o pedido, o veredito e o `rules_version` (hash da parametrização que a produziu).
- **Operação**: readiness (`/ready`) separado de liveness (`/health`), healthcheck no container e encerramento gracioso.
- **Integração com o legado**: gateway PHP como camada anticorrupção, com banco legado próprio (MySQL) e logs de rejeição sem dados pessoais.
- **Entrega**: imagens Docker, testes automatizados (Go e PHP), E2E do contrato entre os serviços e CI.

Veja a [especificação técnica](docs/especificacao_tecnica.md) para o escopo completo e as decisões de projeto.

## Roadmap

- **Novos domínios** — Limites Bacen/Pix, LGPD, SCR.
- **Regras em banco com vigência** — hoje cada avaliação já registra o hash da parametrização usada; a evolução é manter o histórico de versões com datas de vigência, para avaliar uma operação pelas regras válidas na data dela.

Detalhes na [especificação técnica](docs/especificacao_tecnica.md#9-roadmap-evolução-futura).
