# Estágio de build: compila um binário estático (sem cgo), que roda em
# qualquer imagem Linux mínima.
FROM golang:1.25-alpine AS build
WORKDIR /src

# Dependências primeiro: a camada fica em cache enquanto go.mod/go.sum não mudam.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/regulatory-engine ./cmd/api

# Estágio final: distroless sem shell nem gerenciador de pacotes, rodando
# como usuário sem privilégio. As regras vão embutidas: cada versão da imagem
# carrega um conjunto de regras imutável e rastreável.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/regulatory-engine /app/regulatory-engine
COPY config/rules.json /app/config/rules.json
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/app/regulatory-engine"]
