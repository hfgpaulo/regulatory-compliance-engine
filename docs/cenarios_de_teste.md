# Cenários de teste da API

Catálogo de cenários para testar o `regulatory-engine` na prática, com a entrada e o resultado esperado de cada um. Os cenários regulatórios (**C01–C10**) são os mesmos do teste automatizado de aceitação (`internal/rules/acceptance_test.go`), que roda no CI contra o `config/rules.json` versionado: o que você vê aqui à mão é o que o CI garante a cada push.

## Como rodar

1. Suba a API: `make up` (stack em containers) ou `make dev` (hot reload).
2. Escolha a forma de disparar as requisições:
   - **VS Code + REST Client** (extensão `humao.rest-client`): abra [`cenarios_de_teste.http`](cenarios_de_teste.http) e clique em **Send Request** acima de cada cenário. É o caminho mais prático.
   - **curl** (Git Bash no Windows — no PowerShell, `curl` é outro comando e as aspas do JSON quebram). Requisição base, usada no C01:

     ```bash
     curl -s -X POST http://localhost:3000/api/v1/evaluate -H "Content-Type: application/json" -d '{"product":{"type":"personal_loan","origin":"US","origin_currency":"USD"},"operation":{"amount":"1000.00","currency":"USD","method":"international_transfer","counterparty":{"name":"John Doe","pep":false}}}'
     ```

     Os demais cenários alteram só os campos indicados na coluna **Entrada** das tabelas abaixo.

## Parametrização assumida

Os valores esperados dependem do `config/rules.json` atual: **IOF 0,38%**, câmbio **USD → BRL 5,00**, teto de comunicação ao COAF **R$ 50.000,00** e sancionados **"Ivan Petrov"** e **"Mara Costa"**. Se a parametrização mudar, o teste de aceitação falha e aponta quais cenários mudaram de veredito — é o comportamento desejado.

> **Nenhum cenário válido sai `compliant: true`.** Toda transferência internacional exige adaptação de IOF, e hoje é o único método suportado. Não é defeito: é o cenário de tropicalização modelado (produto dos EUA que não previa o tributo).

## Operação (liveness e readiness)

| ID | Objetivo | Requisição | Esperado |
|---|---|---|---|
| H01 | Processo no ar | `GET /health` | `200`, `status: "ok"` |
| H02 | Pronto para tráfego | `GET /ready` | `200`, `status: "ready"`; `503` com o MongoDB fora |

## Cenários regulatórios (`POST /evaluate`)

Todos retornam `201` com a avaliação persistida (`id`, `created_at`, `rules_version`, `request`, `report`) e `compliant: false`.

| ID | Objetivo | Entrada (diferença da base) | Resultados esperados no `report` |
|---|---|---|---|
| C01 | Transferência simples, contraparte sem risco | — | IOF `19.00` |
| C02 | Arredondamento do IOF (R$ 6.172,85 × 0,38% = 23,45683) | `amount: "1234.57"` | IOF `23.46` |
| C03 | Logo abaixo do teto do COAF (R$ 49.999,95) | `amount: "9999.99"` | IOF `190.00`; **sem** PLD |
| C04 | Exatamente no teto (R$ 50.000,00 — o teto é inclusivo) | `amount: "10000.00"` | IOF `190.00`; PLD reportável `50000.00` |
| C05 | Acima do teto | `amount: "75000.00"` | IOF `1425.00`; PLD reportável `375000.00` |
| C06 | Contraparte PEP | `pep: true` | IOF `19.00`; screening "pessoa exposta politicamente (PEP)" |
| C07 | Contraparte sancionada | `name: "Ivan Petrov"` | IOF `19.00`; screening "consta em lista de sancionados" |
| C08 | Sancionada com caixa e espaços diferentes | `name: "  ivan PETROV "` | igual ao C07 (nome normalizado) |
| C09 | PEP e sancionada | `name: "Mara Costa"`, `pep: true` | IOF `19.00`; screening "e PEP e consta em lista de sancionados" |
| C10 | Todos os domínios juntos | `amount: "75000.00"`, `name: "Mara Costa"`, `pep: true` | IOF `1425.00`; PLD `375000.00`; screening (PEP + sancionada) — 3 resultados e 3 adaptações |

Os pares **C03/C04** testam a fronteira do teto (um centavo abaixo e exatamente nele) — é onde erros de comparação (`>` vs `>=`) ou de arredondamento apareceriam.

## Validação de entrada (`POST /evaluate`)

Todos retornam `400` e **nada é persistido**. Cobertos automaticamente por `internal/model/validation_test.go` e `internal/httpapi/evaluate_test.go`.

| ID | Objetivo | Entrada | Esperado |
|---|---|---|---|
| V01 | Corpo vazio não vira "conforme" | `{}` | 7 campos em `details` (tipo, origem, moedas, valor, método, contraparte) |
| V02 | JSON malformado | `{ nao e json` | `"corpo da requisicao invalido"` |
| V03 | Moeda válida, mas não suportada | `currency: "BRL"` | `operation.currency`: "moeda nao suportada (aceita: USD)" |
| V04 | Mais de 2 casas decimais | `amount: "1000.005"` | `operation.amount`: "deve ter no maximo 2 casas decimais" |
| V05 | Vários erros de uma vez | `amount: "0"`, `method: "pix"` | `operation.amount` e `operation.method` na mesma resposta |
| V06 | Origem fora do ISO 3166-1 | `origin: "usa"` | `product.origin` |

## Consulta de avaliações

| ID | Objetivo | Requisição | Esperado |
|---|---|---|---|
| P01 | Listagem | `GET /evaluations` | `200`, até 50 avaliações, a mais recente primeiro |
| P02 | Busca por id | `GET /evaluations/{id do C01}` | `200`, a mesma avaliação do C01 (no `.http`, o id é capturado automaticamente) |
| P03 | Id inexistente | `GET /evaluations/nao-existe` | `404`, `"avaliacao nao encontrada"` |

## Cenário de resiliência (opcional)

Com a stack no ar (`make up`), pare o banco com `docker stop regulatory-mongodb`: `GET /health` continua `200`, `GET /ready` passa a `503` e, após cerca de 30s, `docker ps` mostra o motor como `unhealthy`. Suba de novo com `docker start regulatory-mongodb`.
