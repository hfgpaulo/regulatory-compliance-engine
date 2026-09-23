package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config reúne as variáveis de ambiente usadas pela aplicação.
// Centralizar isso em um único lugar evita espalhar os.Getenv pelo código.
type Config struct {
	Port      string // porta HTTP em que a API escuta
	AppEnv    string // ambiente lógico: development, production, etc.
	MongoURI  string // string de conexão com o MongoDB
	MongoDB   string // nome do banco de dados
	RulesPath string // caminho do arquivo de parametrização (rules.json)
}

// Load lê o arquivo .env (se existir) e monta a Config a partir do ambiente.
// O .env é opcional de propósito: em produção as variáveis vêm do ambiente
// ou de um secrets manager, nunca de um arquivo versionado.
func Load() Config {
	_ = godotenv.Load() // ignorar erro: ausência de .env é um caso válido

	return Config{
		Port:      getEnv("PORT", "3000"),
		AppEnv:    getEnv("APP_ENV", "development"),
		MongoURI:  getEnv("MONGO_URI", "mongodb://admin:admin123@localhost:27017"),
		MongoDB:   getEnv("MONGO_DB", "regulatory"),
		RulesPath: getEnv("RULES_PATH", "config/rules.json"),
	}
}

// getEnv retorna o valor da variável de ambiente ou um padrão, quando vazia.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
