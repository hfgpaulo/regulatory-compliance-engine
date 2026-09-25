package main

import (
	"net/http"
	"time"
)

// healthcheck consulta o readiness do próprio serviço e devolve o código de
// saída que o HEALTHCHECK do Docker espera: 0 saudável, 1 não saudável.
// Existe porque a imagem distroless não tem curl nem wget.
func healthcheck(url string) int {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
