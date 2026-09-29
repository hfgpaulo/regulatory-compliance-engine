# Cenários de teste da API

Catálogo de cenários para testar o sistema na prática, com a entrada e o resultado esperado de cada um. Este arquivo é a **fonte da verdade**: o `.http` e a collection do Postman executam os mesmos cenários — direto no motor (`regulatory-engine`, porta 3000) e pelo sistema legado (`legacy-gateway`, porta 8080). Os cenários regulatórios (**C01–C10**) são os mesmos do teste automatizado de aceitação (`internal/rules/acceptance_test.go`), e os do gateway (**G01–G04**) rodam no E2E (`scripts/e2e.sh`): o que você vê aqui à mão é o que o CI garante a cada push.

## Como rodar

1. Suba a stack: `make up` (todos os serviços em containers). O `make dev` sobe só o motor, sem o gateway.
2. Escolha a forma de disparar as requisições:
   - **Postman**: *Import* → [`postman/regulatory-compliance-engine.postman_collection.json`](postman/regulatory-compliance-engine.postman_collection.json). Os cenários vêm com os mesmos IDs e com testes em cada requisição (aba *Test Results*); *Run* na collection roda todos. Os testes conferem os valores originais: para experimentar outros valores, use a pasta **Livre (experimente)**.
   - **VS Code + REST Client** (extensão `humao.rest-client`): abra [`cenarios_de_teste.http`](cenarios_de_teste.http) e clique em **Send Request** acima de cada cenário. O resultado esperado está no título de cada um; para outros valores, use a seção **Livre**, no fim do arquivo.
   - **curl** (Git Bash no Windows — no PowerShell, `curl` é outro comando e as aspas do JSON quebram). Requisição base, usada no C01:

     ```bash
     curl -s -X POST http://localhost:3000/api/v1/evaluate -H "Content-Type: application/json" -d '{"product":{"type":"personal_loan","origin":"US","origin_currency":"USD"},"operation":{"amount":"1000.00","currency":"USD","method":"international_transfer","counterparty":{"name":"John Doe","pep":false}}}'
     ```

     Pelo gateway, no formato legado (G01):

     ```bash
     curl -s -X POST http://localhost:8080/propostas -H "Content-Type: application/json" -d '{"produto":{"tipo":"EMPRESTIMO_PESSOAL","pais_origem":"US","moeda_origem":"USD"},"operacao":{"valor_centavos":100000,"moeda":"USD","modalidade":"TRANSFERENCIA_INTERNACIONAL","contraparte":{"nome":"John Doe","pep":"N"}}}'
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

## Gateway legado (`POST /propostas`, porta 8080)

O mesmo fluxo, entrando pelo sistema legado: formato legado na entrada e na saída, com o motor por trás. Aceitas aparecem na tabela `propostas` do MySQL legado (DBeaver, porta 3307 — ver README) e, com o mesmo `protocolo`, na coleção `evaluations` do MongoDB. Os cenários G01–G04 também rodam automaticamente no E2E (`make e2e`).

| ID | Objetivo | Entrada (formato legado) | Esperado |
|---|---|---|---|
| G01 | Proposta aceita (equivale ao C01) | `valor_centavos: 100000`, `pep: "N"` | `201`, `situacao: "PENDENTE_ADAPTACAO"`, `iof_centavos: 1900`; linha nova no MySQL |
| G02 | Todos os domínios (equivale ao C10) | `valor_centavos: 7500000`, `nome: "Mara Costa"`, `pep: "S"` | `201`, `situacao: "COMUNICAR_COAF"`, `iof_centavos: 142500`, 3 exigências |
| G03 | Formato legado inválido — rejeitado pelo **gateway** | `valor_centavos: 75000.5`, `pep: "talvez"` | `422` com `operacao.valor_centavos` e `operacao.contraparte.pep`; o motor não é chamado; nada no MySQL |
| G04 | Regra de negócio — rejeitada pelo **motor**, erro traduzido | `moeda: "BRL"` | `422` com `operacao.moeda` ("moeda nao suportada"); nada no MySQL |
| G05 | Corpo não é JSON | `{ nao e json` | `400` |
| G06 | Liveness do gateway | `GET /health` | `200`, `service: "legacy-gateway"` |

As rejeições (G03–G05) ficam no log do gateway (`docker logs legacy-gateway`) como eventos JSON `proposta_rejeitada`, com o motivo e **sem o nome da contraparte**.

## Cenários de resiliência (opcional)

Com a stack no ar (`make up`):

- **MySQL legado parado** (`docker stop legacy-mysql`): uma proposta válida continua retornando `201` (a avaliação foi gravada pelo motor), e o log do gateway mostra `gravacao_legado_falhou` com nível `ERROR` e o `protocolo`. A resposta demora cerca de 8s (limitação conhecida, ver spec 8.3). Suba de novo com `docker start legacy-mysql`.
- **Motor parado** (`docker stop regulatory-engine`): o gateway responde `503` "motor de conformidade indisponivel". Suba de novo com `docker start regulatory-engine`.

### Motor com o MongoDB fora do ar

Com a stack no ar (`make up`), pare o banco com `docker stop regulatory-mongodb`: `GET /health` continua `200`, `GET /ready` passa a `503` e, após cerca de 30s, `docker ps` mostra o motor como `unhealthy`. Suba de novo com `docker start regulatory-mongodb`.
