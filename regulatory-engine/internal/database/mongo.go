package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Connect abre uma conexão com o MongoDB, valida com um ping e devolve o
// cliente e o handle do banco de dados.
//
// A estratégia é "fail-fast": se não for possível conectar dentro do prazo,
// a função retorna erro e a aplicação não sobe. Isso é intencional — o
// serviço depende do banco para operar, então subir sem banco só esconderia
// o problema para mais tarde.
func Connect(ctx context.Context, uri, dbName string) (*mongo.Client, *mongo.Database, error) {
	connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao conectar no MongoDB: %w", err)
	}

	// O Ping confirma que o banco está de fato acessível, e não apenas que a
	// string de conexão é válida.
	if err := client.Ping(connCtx, readpref.Primary()); err != nil {
		return nil, nil, fmt.Errorf("erro ao pingar o MongoDB: %w", err)
	}

	return client, client.Database(dbName), nil
}
