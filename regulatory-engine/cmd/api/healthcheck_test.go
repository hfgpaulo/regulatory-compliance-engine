package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthcheck(t *testing.T) {
	respond := func(status int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))
	}

	ready := respond(http.StatusOK)
	defer ready.Close()
	unavailable := respond(http.StatusServiceUnavailable)
	defer unavailable.Close()

	assert.Equal(t, 0, healthcheck(ready.URL), "200 deve ser saudavel")
	assert.Equal(t, 1, healthcheck(unavailable.URL), "503 deve ser nao saudavel")
	assert.Equal(t, 1, healthcheck("http://127.0.0.1:1/api/v1/ready"), "servico fora do ar deve ser nao saudavel")
}
