# Regulatory Compliance Engine — Especificação Técnica

> **Motor de conformidade regulatória para tropicalização de serviços financeiros (EUA → Brasil)**

---

> **Nota de originalidade:** projeto **fictício e original**, desenvolvido do zero para fins de portfólio. Cenário, dados e regras são ilustrativos; não reproduz sistemas, código, arquitetura ou soluções proprietárias de nenhuma empresa. As tecnologias citadas (Go, Fiber, MongoDB, PHP) são de domínio público.

---

## 1. A ideia em uma frase

Assim como um produto importado passa pela **alfândega** antes de entrar no país, um **produto financeiro criado sob as regras dos EUA** precisa passar por uma verificação de conformidade antes de operar no Brasil. Este projeto é essa "alfândega regulatória": recebe uma operação ou a configuração de um produto e responde **"pode operar no Brasil? o que precisa ser adaptado?"**, aplicando regras regulatórias brasileiras de forma configurável.

## 2. Contexto e problema

Empresas de tecnologia que atendem o setor financeiro convivem com um desafio recorrente: **traduzir norma regulatória em software**. Quando um serviço financeiro concebido no exterior (tipicamente nos EUA) é trazido para o Brasil — o processo de *tropicalização* —, ele precisa se adequar a exigências locais que não existiam na origem: tributação específica, limites, obrigações de comunicação a órgãos reguladores, prevenção à lavagem de dinheiro e proteção de dados.

Fazer essa adequação manualmente, produto a produto, é lento e propenso a erro. Este projeto sistematiza a verificação: um serviço central que, dada uma operação, aponta objetivamente as exigências regulatórias brasileiras aplicáveis e as adaptações necessárias.

O projeto demonstra três capacidades:

1. **Entendimento do domínio** financeiro/regulatório brasileiro.
2. **Transformar regra de negócio em software** bem estruturado.
3. **Arquitetura poliglota** com integração entre um sistema legado e uma nova API.

## 3. Arquitetura

Dois microsserviços conteinerizados, orquestrados por Docker Compose, reproduzindo um padrão comum de migração/modernização gradual — a coexistência entre um sistema legado e uma nova API, cada um com o seu banco. O fluxo principal: **proposta no formato legado → gateway PHP → motor Go**, com o legado registrando a proposta no MySQL e o motor registrando a avaliação completa no MongoDB.

![Arquitetura da solução: gateway PHP com MySQL legado e motor Go com MongoDB, orquestrados por Docker Compose; rules.json embutido no motor](arquitetura.png)

**Papéis:**

- **`regulatory-engine` (Go + Fiber):** coração do projeto. Recebe uma operação/produto, devolve o veredito de conformidade e **persiste cada avaliação no MongoDB** (requisição + veredito, como registro de auditoria), expondo o histórico por API. Contém o motor de regras e os domínios regulatórios e lê a parametrização de um arquivo de regras. Não guarda estado em memória entre requisições — todo estado vive no banco —, então escala horizontalmente.
- **`legacy-gateway` (PHP + Slim):** representa o sistema legado de uma instituição financeira, com **vocabulário e formato próprios** (campos em português, valores em centavos, flags `"S"`/`"N"`). Atua como **camada anticorrupção**: traduz a proposta legada para o contrato do motor e a avaliação de volta para o formato legado, sem que um lado contamine o outro (seção 5.2).
- **Dois bancos, cada um com seu dono:** o **MySQL legado** guarda as propostas aceitas no vocabulário do legado; o **MongoDB** guarda as avaliações completas (a auditoria, fonte da verdade). Os dois se ligam pelo `protocolo` — o id da avaliação no motor.

**Decisão de arquitetura-chave:** as regras (limites, alíquotas, gatilhos de reporte) **não são hardcoded** — vivem em parametrização configurável (`rules.json`, evoluível para tabela no banco). É uma boa prática essencial no contexto regulatório, porque norma muda com frequência.

**Validação da parametrização (fail-fast).** Por ser editável sem recompilar, a parametrização também pode ser editada *errada*. Por isso `rules.Load()` valida o arquivo logo após interpretá-lo (`Parameters.Validate()`) e, se algo estiver incoerente com o que as regras assumem, o serviço **não sobe** — mesmo princípio já usado na conexão com o MongoDB. Os problemas são acumulados (`errors.Join`) para o erro de boot listar tudo o que precisa ser corrigido de uma vez. Checagens atuais:

- `pld.reporting_threshold.currency` deve ser `BRL`: o teto é comparado com o valor da operação já convertido para reais; outra moeda seria tratada como BRL sem aviso.
- `fx.usd_brl` deve ser maior que 0: câmbio zero zeraria toda conversão (IOF de R$ 0,00 e nenhuma operação atingindo o teto).
- `pld.reporting_threshold.amount` deve ser maior que 0.
- `iof.international_transfer.rate` deve ser maior que 0 e menor que 1 (100%).

O critério "maior que zero" tem um segundo papel: no JSON, **chave ausente vira zero** (o `decimal` não distingue "não veio" de "veio 0"), então uma chave esquecida ou digitada errado também é barrada. Por isso a alíquota zero não é aceita mesmo existindo operações com IOF 0% na norma: aceitar zero aceitaria esquecimento, e a regra emitiria `adaptation_required` com IOF de R$ 0,00 — veredito incoerente. Alíquota zero real se representa desligando a regra. O limite superior (`< 1`) é só sanidade; valores plausíveis porém errados (ex.: `0.38` no lugar de `0.0038`) exigiriam um teto de domínio, deixado de fora de propósito.

Um teste carrega o `config/rules.json` versionado, de modo que uma parametrização inválida quebra o CI antes de chegar ao boot.

## 4. Domínios regulatórios (escopo inicial)

> **Sobre os valores:** os números abaixo são *ilustrativos* para a estrutura funcionar. Antes de finalizar, confirmar os parâmetros vigentes nas fontes oficiais (Bacen, COAF/UIF, Receita Federal) e ajustar no `rules.json`. A arquitetura configurável existe justamente para isso.

Cada item indica se já está **implementado** (com o código da regra) ou **planejado**.

### 4.1. PLD/FT + COAF (Prevenção à Lavagem de Dinheiro)
- Operações com valor, convertido para BRL, igual ou acima do teto de comunicação → **reportáveis ao COAF**. *Implementado* (`pld.reporting_threshold`).
- Screening contra lista de **PEP / sancionados** (lista mock local). *Implementado* (`pld.pep_screening`).
- Outras hipóteses de comunicação (ex.: espécie acima de teto, fracionamento suspeito). *Planejado.*
- Saída: alertas + indicação de comunicação obrigatória.

### 4.2. Câmbio / IOF (o core da "tropicalização")
- Cálculo de **IOF** na transferência internacional EUA → BR. *Implementado* (`iof.international_transfer`).
- Conversão cambial com taxa parametrizável — hoje apenas USD → BRL. *Implementado.*
- Regras para outros tipos de operação (investimento, empréstimo). *Planejado.*
- Saída: tributos aplicáveis + adaptações necessárias vs. o produto original.

### 4.3. (Roadmap) Limites Bacen/Pix + LGPD
- Fica documentado como próxima fase, para o projeto ter história de evolução.

## 5. Contratos das APIs

### 5.1. Motor (`regulatory-engine`)

```
POST /api/v1/evaluate
  → avalia um produto/operação, PERSISTE e retorna a avaliação criada (201)
    { id, created_at, rules_version, request, report }
GET  /api/v1/evaluations
  → lista as avaliações mais recentes
GET  /api/v1/evaluations/{id}
  → recupera uma avaliação pelo id
GET  /api/v1/health
  → liveness: processo no ar (não consulta o banco)
GET  /api/v1/ready
  → readiness: 200 se o MongoDB responde ao ping, 503 se não
```

**Exemplo de requisição** (`POST /api/v1/evaluate`):
```json
{
  "product": {
    "type": "personal_loan",
    "origin": "US",
    "origin_currency": "USD"
  },
  "operation": {
    "amount": 75000.00,
    "currency": "USD",
    "method": "international_transfer",
    "counterparty": { "name": "John Doe", "pep": false }
  }
}
```

**Exemplo do `report`** (dentro da avaliação retornada pelo `POST`):
```json
{
  "compliant": false,
  "results": [
    {
      "domain": "IOF",
      "status": "adaptation_required",
      "detail": "Entrada convertida de USD para BRL a 5.00; IOF de 0.38% aplicável (produto original não previa tributação).",
      "calculated_amount": "1425.00",
      "reference": "rule:iof.international_transfer"
    },
    {
      "domain": "PLD_COAF",
      "status": "reportable",
      "detail": "Valor acima do teto de comunicação automática ao COAF.",
      "reference": "rule:pld.reporting_threshold"
    }
  ],
  "required_adaptations": [
    "Incluir cálculo e retenção de IOF no fluxo de entrada.",
    "Gerar comunicação ao COAF para a operação."
  ]
}
```

> **Valores monetários** trafegam como **string com 2 casas** (`"1425.00"`), para não perder precisão no cliente. Na entrada, o campo `amount` aceita número ou string. Alíquotas e câmbio (parametrização) usam decimal simples.

**Validação de entrada.** Antes de chegar ao motor, a requisição é validada por `EvaluationRequest.Validate()` (camada de domínio). Sem essa barreira, campos ausentes viram *zero values* do Go (valor `0`, método vazio, nome vazio): as regras "não se aplicam" e a operação sairia **conforme por omissão** — o falso negativo silencioso que um motor de compliance não pode produzir.

| Campo | Regra |
|---|---|
| `product.type` | valor conhecido (`personal_loan`) |
| `product.origin` | código de país ISO 3166-1 alpha-2 (ex.: `US`) |
| `product.origin_currency` | código de moeda ISO 4217 (ex.: `USD`); informativo |
| `operation.amount` | maior que zero, no máximo 2 casas decimais |
| `operation.currency` | código de moeda ISO 4217 e, por ora, somente `USD` |
| `operation.method` | valor conhecido (`international_transfer`) |
| `operation.counterparty.name` | obrigatório (sem ele o screening fica cego) |

Todos os campos inválidos são reportados de uma vez, com `400 Bad Request`, e nada é persistido:

```json
{
  "error": "requisicao invalida",
  "details": [
    { "field": "operation.amount", "message": "deve ser maior que zero" },
    { "field": "operation.method", "message": "metodo de operacao desconhecido" }
  ]
}
```

Decisões: a validação fica no domínio (não em tags de biblioteca) para manter a lista de valores válidos junto das constantes e testável sem HTTP; a origem é obrigatória porque a avaliação é um registro de auditoria, e tornar um campo obrigatório depois quebraria clientes, enquanto afrouxar não. Para os códigos de país e moeda valida-se só o formato, sem consultar a lista oficial.

**Moedas.** Os dois campos de moeda têm papéis diferentes:

- `operation.currency` é a moeda do `amount` e **entra no cálculo**: IOF e teto do COAF convertem o valor para BRL. Como a parametrização só tem o câmbio `usd_brl`, apenas `USD` é aceito; qualquer outra moeda retorna `400` com `"moeda nao suportada (aceita: USD)"`. Recusar é preferível a calcular errado — antes desta restrição, `"BRL"` era tratado como dólar sem aviso.
- `product.origin_currency` **descreve o produto de origem** e não entra em nenhum cálculo. É obrigatório (registro de auditoria completo) e tem o formato validado, mas não é restrito a `USD`: restringir um campo que não afeta o resultado rejeitaria requisições sem motivo técnico. Passa a ter validação semântica quando alguma regra depender dele.

Suportar só USD é **decisão de escopo**, não limitação acidental: o cenário do projeto é a tropicalização EUA → Brasil. Aceitar outra moeda exigiria trocar `fx.usd_brl` por uma tabela de câmbio por moeda e um conversor único usado pelas regras, com a lista de moedas aceitas derivada da parametrização.

### 5.2. Gateway legado (`legacy-gateway`)

```
POST /propostas   → traduz, chama o motor, registra no MySQL legado e responde no formato legado (201)
GET  /health      → liveness
```

**Formato legado.** Entrada e saída no vocabulário do sistema legado:

```json
{ "produto":  { "tipo": "EMPRESTIMO_PESSOAL", "pais_origem": "US", "moeda_origem": "USD" },
  "operacao": { "valor_centavos": 7500000, "moeda": "USD", "modalidade": "TRANSFERENCIA_INTERNACIONAL",
                "contraparte": { "nome": "John Doe", "pep": "N" } } }
```

```json
{ "protocolo": "6abab272d44d8dedcac5a7a6", "situacao": "COMUNICAR_COAF", "iof_centavos": 142500,
  "exigencias": ["Incluir calculo e retencao de IOF...", "Gerar comunicacao ao COAF..."],
  "versao_regras": "sha256:cab75867..." }
```

**Tradução (camada anticorrupção).** `EMPRESTIMO_PESSOAL` → `personal_loan`, `TRANSFERENCIA_INTERNACIONAL` → `international_transfer`, `pep: "S"/"N"` → `true/false`, e `valor_centavos` ↔ `amount` decimal com 2 casas por **aritmética inteira e de string, nunca float**. A `situacao` é a mais grave do relatório: algum resultado `reportable` → `COMUNICAR_COAF`; senão, algum `adaptation_required` → `PENDENTE_ADAPTACAO`; senão → `APROVADA`.

**Erros.**

| Situação | Resposta |
|---|---|
| Corpo não é JSON | `400` |
| Formato legado inválido (tipo, enum, centavos não inteiros) | `422`, campos no vocabulário legado; o motor não é chamado |
| Motor rejeita (`400` com `details`) | `422`, com os campos do motor **traduzidos** (`operation.currency` → `operacao.moeda`) |
| Motor responde `5xx` ou fora do contrato | `502`, sem vazar detalhe interno |
| Motor fora do ar ou timeout (2s conexão, 8s total) | `503` |

**Decisões.**

- **O gateway valida só o que traduz** (tipos e enums do legado). As regras de negócio — ISO, moeda suportada, contraparte obrigatória — ficam só no motor, a fonte da verdade; revalidá-las no gateway criaria duas validações que divergiriam com o tempo.
- **`400` do motor sem `details` vira `502`, não `422`:** se o motor não entendeu o corpo que o *gateway* montou, o defeito é da integração, não da proposta do cliente.
- **Falha ao gravar no MySQL legado responde `201` mesmo assim**, com um evento `gravacao_legado_falhou` (nível `ERROR`, com o `protocolo`) no log. O motor já gravou a avaliação; responder erro levaria o cliente a reenviar, e o motor — que não tem idempotência — criaria uma segunda avaliação. A linha legada é reconciliável pelo protocolo; a solução completa (padrão *outbox*) fica fora do escopo.
- **Só propostas aceitas vão para o MySQL legado.** Rejeições e falhas vão para o log (seção 8.3), com o motivo e **sem dados pessoais**.

## 6. Estrutura de pastas (monorepo)

```
regulatory-compliance-engine/
├── .github/workflows/ci.yml      # CI: motor (Go), gateway (PHP), docker build e E2E
├── docker-compose.yml            # MongoDB + motor, MySQL + gateway
├── scripts/e2e.sh                # E2E: propostas pelo gateway até o MySQL e o MongoDB
├── Makefile                      # atalhos de desenvolvimento
├── README.md
├── docs/
│   ├── especificacao_tecnica.md
│   ├── cenarios_de_teste.md      # catálogo de cenários (entrada → resultado esperado)
│   ├── cenarios_de_teste.http    # os mesmos cenários, executáveis (REST Client)
│   ├── postman/                  # os mesmos cenários como collection do Postman, com testes
│   └── arquitetura.png           # diagrama (fonte: arquitetura.svg)
├── regulatory-engine/            # Go + Fiber
│   ├── cmd/api/main.go           # ponto de entrada (montagem da aplicação)
│   ├── internal/
│   │   ├── config/               # configuração via ambiente
│   │   ├── database/             # conexão com o MongoDB
│   │   ├── model/                # structs do domínio (contrato) + validação de entrada
│   │   ├── money/                # tipo monetário (decimal, sempre 2 casas)
│   │   ├── engine/               # interface Rule + avaliador
│   │   ├── rules/                # regras (IOF, PLD) + carregamento e validação do rules.json
│   │   ├── repository/           # persistência das avaliações + índices
│   │   └── httpapi/              # servidor, rotas e handlers
│   ├── config/rules.json         # parametrização regulatória
│   ├── go.mod
│   └── Dockerfile                # imagem multi-stage (distroless)
└── legacy-gateway/               # PHP 8.3 + Slim (Apache + mod_php)
    ├── public/index.php          # ponto de entrada (DocumentRoot)
    ├── bin/migrate.php           # migração idempotente do MySQL legado, antes do Apache
    ├── src/                      # tradutor, cliente do motor, handler, repositório PDO, log
    ├── tests/                    # PHPUnit com motor, banco e log simulados
    ├── composer.json / .lock     # Slim, Guzzle; dev: PHPUnit, PHPStan
    └── Dockerfile
```

Os índices do MongoDB são criados pelo próprio motor no boot (ver seção 7), sem script de *seed* separado.

## 7. Modelo de dados

### 7.1. MongoDB (motor)

Banco `regulatory`. A avaliação é gravada como **um único documento embedded** na coleção `evaluations` — o modelo idiomático de MongoDB: uma escrita, e uma leitura traz o quadro completo, sem "join".

- **`evaluations`**: `{ _id, created_at, rules_version, request { ... }, report { ... } }` — a submissão original e o veredito produzido, juntos no mesmo documento, com a versão da parametrização que o produziu.

**Versão da parametrização (`rules_version`).** Um registro de auditoria precisa responder não só *o que* foi decidido, mas *sob quais regras*: se a alíquota de IOF mudar no `rules.json`, um `calculated_amount` antigo só continua explicável se o registro disser qual arquivo estava em vigor. Por isso cada avaliação grava `rules_version = "sha256:" + hash dos bytes do rules.json`, calculado no `rules.Load` e também registrado no log de boot.

- **Por que o hash dos bytes:** qualquer um confere com ferramenta padrão (`sha256sum config/rules.json`), sem depender do código; e o hash não "esquece" de mudar, ao contrário de um número de versão escrito à mão. A normalização de fim de linha para LF (`.gitattributes`) garante o mesmo hash no Windows, no Linux e na imagem Docker.
- **Por que fora do `report`:** é proveniência do registro (como `created_at`), não resultado de regra; o motor continua sem conhecer a versão, que é injetada no servidor.
- **Avaliações anteriores** a este campo ficam sem ele: não há como saber retroativamente qual arquivo estava em uso, então não há migração.

**Índices.** `created_at_desc` (`{ created_at: -1 }`) atende a listagem das avaliações mais recentes: sem ele, a consulta faria *collection scan* com ordenação em memória. O índice é declarado pelo próprio repositório (`EnsureIndexes`) e criado no boot, com fail-fast — quem consulta declara o índice de que precisa, e ele existe em qualquer ambiente, não só no compose. Como o `createIndexes` do MongoDB é idempotente, rodar a cada boot não tem custo. Em coleções grandes, a criação migraria para um passo de migração separado, para não alongar o boot.

**Identificador (`_id` como string).** O id é gerado como ObjectId (`primitive.NewObjectID()`), mas gravado como a sua representação hexadecimal em **string**, e não com o tipo nativo `ObjectId`. É um *tradeoff* consciente:

- **Ganho:** o domínio fica livre de tipos do driver (`model.Evaluation.ID` é `string`, sem importar `primitive`), o JSON expõe o id sem serialização customizada, e o handler repassa o parâmetro da rota direto à consulta — um id malformado simplesmente não encontra nada (`404`), sem etapa de conversão.
- **Custo:** a chave ocupa 24 bytes em vez de 12, o que aumenta o índice `_id`, e perde-se o timestamp embutido do `ObjectId` (`getTimestamp()`). O segundo custo é neutralizado pelo campo explícito `created_at`; a ordem cronológica também se preserva, porque o hexadecimal de tamanho fixo ordena igual ao `ObjectId`.
- **Reversão:** migrar para o tipo nativo depois exigiria reescrever o `_id` de todos os documentos — por isso a decisão fica registrada aqui. No volume atual, a simplicidade compensa o custo de armazenamento.

Valores monetários são gravados como **Decimal128** (o decimal nativo do Mongo), preservando a precisão e permitindo consultas e agregações por valor.

**Decisão de modelagem (embedded vs. referência).** `request` e `report` têm relação 1:1, nascem juntos e são sempre lidos juntos — portanto ficam **embutidos** no mesmo documento (separá-los em duas coleções seria um anti-padrão: duas escritas não-atômicas e um *join* na leitura, sem benefício). Como a avaliação é um **registro de auditoria**, o documento é tratado como um **retrato imutável** do que foi avaliado e do veredito daquele momento. Promover a **contraparte** a entidade própria (coleção `parties`) só se justificaria com uma **identidade estável** — um documento (CPF/CNPJ), não o nome —, o que exigiria *entity resolution* (casamento aproximado contra listas de sanção), um problema à parte e fora do escopo deste projeto.

### 7.2. MySQL (legado)

Banco `legacy`, tabela `propostas`, no estilo do legado — nomes em português e dinheiro em **centavos `BIGINT`** (sem float nem decimal):

```sql
id BIGINT AUTO_INCREMENT PRIMARY KEY, protocolo CHAR(24) NOT NULL UNIQUE,
tipo_produto VARCHAR(40), modalidade VARCHAR(40), valor_centavos BIGINT, moeda CHAR(3),
contraparte_nome VARCHAR(200), contraparte_pep CHAR(1), situacao VARCHAR(30),
iof_centavos BIGINT, versao_regras VARCHAR(80), criado_em DATETIME(3)
```

O `protocolo` (único) é o id da avaliação no motor e liga os dois bancos: a linha legada diz o que o legado precisa saber; o documento no MongoDB tem a auditoria completa.

**Schema pelo código, não por script de inicialização.** O `bin/migrate.php` roda `CREATE TABLE IF NOT EXISTS` a cada subida do container, antes do Apache aceitar tráfego — o mesmo princípio do `EnsureIndexes` do motor. Um script em `docker-entrypoint-initdb.d` só rodaria na **primeira** criação do volume. Se a migração falhar, o container não sobe (fail-fast).

**Conexão preguiçosa.** O gateway só abre conexão com o MySQL na primeira gravação: o `/health` e as requisições rejeitadas não dependem do banco legado, e um MySQL fora do ar não interrompe o fluxo proposta → motor.

## 8. Testes e qualidade

A garantia de qualidade do `regulatory-engine` se apoia em três pilares implementados: uma suíte de testes automatizados, um pipeline de integração contínua que a executa a cada mudança, e logs estruturados para observabilidade.

### 8.1. Estratégia de testes

A suíte usa `testify` e cobre as três camadas do motor de forma independente:

- **Testes unitários (table-driven).** As regras de negócio — cálculo de IOF, teto de comunicação ao COAF, screening de PEP/sancionados —, a validação de entrada e o tipo monetário são testados com tabelas de casos (entrada esperada vs. saída), o padrão idiomático em Go. Cada regra é verificada isoladamente: quando se aplica, o valor calculado, e quando **não** se aplica (retorno nulo).
- **Teste de agregação do motor com dublê.** O avaliador (`engine`) é testado contra uma **regra falsa** (`fakeRule`) controlada pelo teste, não contra as regras reais. Assim se verifica apenas a responsabilidade do motor — agregar resultados, ignorar regras que não se aplicam, decidir conformidade e propagar erro — sem acoplamento ao comportamento de IOF ou PLD.
- **Teste de handler HTTP sem banco.** Os handlers dependem da **interface** `EvaluationStore`, não do repositório concreto. Nos testes, injeta-se um **store falso em memória** (`fakeStore`) e exercitam-se as rotas de ponta a ponta com `app.Test` (sem abrir porta de rede nem exigir MongoDB): `POST /evaluate` retornando `201` e persistindo, corpo malformado ou requisição incompleta retornando `400` (com a lista de campos inválidos e sem persistir), e busca inexistente retornando `404`. Um teste adicional verifica, nas três rotas que acessam o banco, que o contexto recebido pelo store **deriva do contexto da requisição** (um valor anexado por *middleware* chega ao store) e carrega o timeout do handler. O readiness é testado com um `Pinger` falso (banco acessível → `200`, indisponível → `503`), e um teste garante que o liveness continua `200` com o banco fora. O subcomando `healthcheck` é testado contra servidores HTTP de teste (`httptest`).
- **Cenários regulatórios de aceitação.** Os testes acima isolam cada peça; nenhum deles junta as regras reais **com o `config/rules.json` versionado**. O teste de aceitação faz isso: monta o motor com a mesma função usada em produção (`rules.NewEngine`, único ponto de montagem — o `main` não monta regras por conta própria, então teste e produção não divergem) e verifica, para dez cenários (C01–C10), o veredito completo: domínios, status, valores calculados e detalhes. Inclui as fronteiras que mais escondem erro — arredondamento do IOF e um centavo abaixo / exatamente no teto do COAF. Se alguém alterar a parametrização, o teste falha e aponta quais vereditos mudaram (verificado alterando a alíquota: os dez cenários falham). Os mesmos cenários estão em [`cenarios_de_teste.md`](cenarios_de_teste.md) e [`cenarios_de_teste.http`](cenarios_de_teste.http), para execução manual contra a API.
- **Integração do repositório com MongoDB real.** Os dublês garantem que os handlers chamam o store corretamente, mas não que o MongoDB grava e devolve o que o código espera. Os testes do pacote `repository` rodam contra um Mongo de verdade e cobrem: ida e volta de uma avaliação completa (Decimal128, resultado sem valor calculado, `rules_version`), `FindByID` devolvendo `nil` para id inexistente, `List` ordenado do mais recente e limitado, e `EnsureIndexes` idempotente. Rodam só quando `MONGO_TEST_URI` está definida — no CI, por um *service container*; localmente, com `make test-integration` — e se pulam caso contrário, para o `go test` comum continuar sem Docker. Cada teste usa um banco de nome único, apagado ao final. Optou-se por essa forma em vez de *testcontainers*: mesma cobertura, sem dependência nova.

  O teste de ida e volta revelou uma inconsistência real: o `created_at` da resposta do `POST` tinha precisão de nanossegundos (relógio do Go), mas o MongoDB armazena milissegundos, então um `GET` do mesmo recurso devolvia outro valor. O repositório passou a truncar a data em milissegundos antes de gravar.

O ponto de projeto que torna isso possível é a **injeção de dependência** adotada nos blocos anteriores: como o servidor recebe o motor e o store por interface, ambos podem ser substituídos por dublês nos testes. Testes rápidos, determinísticos e que rodam em qualquer máquina limpa — inclusive no CI, sem infraestrutura.

**Gateway (PHP).** PHPUnit com o mesmo padrão: o app é montado por um único `AppFactory` (usado pelo `index.php` e pelos testes), e as dependências externas chegam por parâmetro — o motor é simulado pelo `MockHandler` do Guzzle, o banco legado e o log por implementações em memória. Cobertos: conversão de centavos (fronteiras, sem float), tradução nos dois sentidos, precedência da `situacao`, mapeamento de erros (`400`/`422`/`502`/`503`), gravação só das aceitas, falha no MySQL respondendo `201` com evento de log, e um teste garantindo que **o nome da contraparte não aparece no log**. PHPStan (nível 8) faz a análise estática.

**E2E: o contrato entre PHP e Go.** Cada lado testa contra um dublê do outro; nada disso garante que os dois **concordam** — se o motor renomear um campo, as duas suítes continuam verdes. O `scripts/e2e.sh` sobe a stack inteira e passa propostas reais pelo gateway (cenários C01, C05 e C10 aceitos, uma rejeição do gateway e uma do motor), conferindo a resposta legada, a linha no MySQL e a avaliação no MongoDB pelo mesmo protocolo, e que rejeitadas não são gravadas. Roda com `make e2e` e a cada push no CI. Verificado que falha quando deve: com o motor parado, todas as checagens das aceitas falham.

### 8.2. Integração contínua (CI)

Um workflow de **GitHub Actions** roda a cada `push` na `main` e em todo *pull request*, numa máquina limpa do runner. A sequência reproduz a verificação local:

1. **`gofmt`** — falha o build se houver código fora do padrão de formatação da linguagem.
2. **`go build ./...`** — garante que todo o módulo compila.
3. **`go vet ./...`** — análise estática de problemas comuns.
4. **`go test ./...`** — executa a suíte descrita acima, incluindo os testes de integração do repositório: o job sobe um MongoDB 7 como *service container* e define `MONGO_TEST_URI`.

Em paralelo, três outros jobs: a construção da imagem Docker do motor (`docker build`), para que uma quebra no Dockerfile seja detectada no mesmo push, e não só no deploy; o **gateway** (PHP 8.3: PHPStan, PHPUnit e build da imagem); e o **E2E**, que sobe a stack completa, roda o `scripts/e2e.sh` e a collection do Postman (Newman), mostrando os logs dos containers se falhar.

O ambiente é fixado em Go 1.25 com cache de módulos. O valor concreto: a verificação deixa de depender da disciplina manual do desenvolvedor — um arquivo esquecido no commit, um `go.sum` inconsistente ou código desformatado são barrados antes de entrar na `main`. O estado do pipeline é exposto por um *badge* no README.

### 8.3. Observabilidade

Os dois serviços emitem **logs estruturados em JSON** (via `slog` no motor Go). Um *middleware* de requisição registra método, rota, status e latência de cada chamada em formato de campo — pronto para ser filtrado por um agregador (Loki, ELK, CloudWatch) sem *parsing* de texto livre.

No gateway, cada rejeição e falha vira um evento JSON (`proposta_rejeitada` com `origem` gateway/motor e os campos inválidos; `motor_indisponivel`, `falha_no_motor` e `gravacao_legado_falhou` com nível `ERROR`, para alertas por nível). Dois cuidados:

- **Sem dados pessoais:** o evento leva os campos e as mensagens, nunca os dados da proposta — o nome da contraparte é dado pessoal (LGPD), e log não tem o controle de acesso nem a retenção de um banco.
- **JSON puro no `stderr`:** o `error_log` do PHP, sob o Apache, prefixa a linha e escapa aspas, e o agregador deixaria de ler o JSON; por isso o evento é escrito direto no `stderr` do processo.

Limitação conhecida: com o container do MySQL **parado**, a proposta leva cerca de 8s para ser respondida (ainda com `201`), porque a resolução de nome no DNS do Docker demora a desistir e o timeout de conexão do PDO não cobre essa etapa. Com o MySQL no ar mas recusando conexão, a falha é imediata.

**Propagação de contexto.** Os handlers derivam o contexto das chamadas ao MongoDB do contexto da requisição (`c.Context()`), com timeout de 5s, em vez de partir de `context.Background()`. A derivação fica num helper único (`newRequestContext`, com o prazo na constante `storeTimeout`), para que nenhum handler novo parta do contexto errado nem repita o prazo. O ganho é um ponto único de propagação: quando entrar um *middleware* de request-id, tracing (OpenTelemetry) ou prazo por requisição, o que ele anexar chega ao banco sem alterar os handlers, e os *spans* do Mongo ficam ligados à requisição que os originou.

Limitação conhecida: isso **não** cancela a query quando o cliente desconecta. O Fiber roda sobre o fasthttp, que por desempenho não sinaliza desconexão durante o handler (o `Done()` da requisição só fecha no desligamento do servidor). Cancelamento por desconexão exigiria um servidor baseado em `net/http`, sem demanda que justifique hoje; o timeout de 5s é o limite efetivo.

**Liveness e readiness.** São duas perguntas diferentes, com dois endpoints:

- `GET /api/v1/health` (**liveness**) responde se o processo está vivo e **não consulta o banco**. Quem reage a uma falha de liveness reinicia o processo — e reiniciar não conserta um banco fora do ar, só geraria um ciclo de reinícios.
- `GET /api/v1/ready` (**readiness**) pinga o MongoDB (prazo de 2s) e responde `503` se ele não responder. Falhar aqui tira o serviço do tráfego sem reiniciá-lo.

No compose, o motor tem `healthcheck` baseado no readiness — é o que permite ao gateway declarar `depends_on: condition: service_healthy` e só subir com o motor pronto. Como a imagem distroless não tem `curl` nem `wget`, o próprio binário oferece o subcomando `regulatory-engine healthcheck`, que consulta o `/ready` local e sai com `0` ou `1`.

**Encerramento gracioso.** Ao receber `SIGTERM` (enviado por `docker stop` e orquestradores) ou `SIGINT`, o motor para de aceitar conexões, espera as requisições em andamento terminarem (até 10s, `shutdownTimeout`) e só então desconecta o MongoDB. Dois cuidados:

- O `Listen` roda numa goroutine e o encerramento usa `ShutdownWithTimeout`, que **bloqueia** até as requisições terminarem. O `GracefulContext` do Fiber não foi usado porque nele o `Listen` retorna assim que o listener fecha, antes das requisições em andamento — o banco seria desconectado no meio delas.
- A janela entre `SIGTERM` e `SIGKILL` precisa caber o **pior caso** do encerramento, senão o orquestrador mata o processo antes: 10s esperando requisições + 5s de prazo para desconectar o MongoDB = 15s. O compose define `stop_grace_period: 20s` (o padrão do Docker, 10s, não caberia). O pior caso foi observado na prática: com o Mongo fora do ar, o `Disconnect` consome o prazo inteiro.

Verificação manual (Docker): com o Mongo de pé, `docker stop` encerra em menos de 1s; com uma requisição em andamento, ela recebe a resposta completa antes do processo sair, e o container termina com código `0` (e não `137`, de `SIGKILL`).

### 8.4. Demonstração

- **README** com contexto de negócio, diagrama, como rodar (`docker compose up`, sem Go nem PHP instalados), exemplos de chamada em curl, como consultar os dois bancos e o **roadmap**.
- **Catálogo de cenários** ([`cenarios_de_teste.md`](cenarios_de_teste.md), a fonte da verdade) executável pelo VS Code ([`.http`](cenarios_de_teste.http)) e pelo **Postman** ([collection](postman/regulatory-compliance-engine.postman_collection.json), com testes em cada requisição e uma pasta "Livre" para experimentar valores). A collection roda no CI via Newman (`make postman`), o que impede que ela fique desatualizada em relação à API.

## 9. Roadmap (evolução futura)

1. Domínios adicionais: Limites Bacen/Pix, LGPD, SCR.
2. **Regras em banco com vigência**: hoje cada avaliação já registra o hash da parametrização usada (`rules_version`, seção 7); a evolução é manter o histórico de versões com datas de vigência, para avaliar uma operação pelas regras válidas na data dela.