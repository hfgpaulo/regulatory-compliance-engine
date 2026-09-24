package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// EvaluationRepository isola o acesso à coleção "evaluations" no MongoDB.
// O resto do código fala com esta abstração, não com o driver do Mongo —
// isso mantém a lógica de negócio livre de detalhes de persistência.
type EvaluationRepository struct {
	col *mongo.Collection
}

// NewEvaluationRepository cria o repositório sobre a coleção "evaluations".
func NewEvaluationRepository(db *mongo.Database) *EvaluationRepository {
	return &EvaluationRepository{col: db.Collection("evaluations")}
}

// EnsureIndexes cria os índices de que as consultas do repositório dependem.
// O createIndexes do MongoDB é idempotente: se o índice já existe com a mesma
// especificação, nada acontece — por isso pode rodar a cada boot.
func (r *EvaluationRepository) EnsureIndexes(ctx context.Context) error {
	// List ordena por created_at desc; sem índice seria collection scan com
	// ordenação em memória.
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "created_at", Value: -1}},
		Options: options.Index().SetName("created_at_desc"),
	})
	if err != nil {
		return fmt.Errorf("erro ao criar indice created_at_desc: %w", err)
	}
	return nil
}

// Save grava uma avaliação, preenchendo id e data quando ausentes.
func (r *EvaluationRepository) Save(ctx context.Context, eval *model.Evaluation) error {
	if eval.ID == "" {
		eval.ID = primitive.NewObjectID().Hex()
	}
	if eval.CreatedAt.IsZero() {
		eval.CreatedAt = time.Now().UTC()
	}
	_, err := r.col.InsertOne(ctx, eval)
	return err
}

// FindByID busca uma avaliação pelo id. Retorna (nil, nil) quando não existe,
// deixando o handler decidir o 404 — o repositório não conhece HTTP.
func (r *EvaluationRepository) FindByID(ctx context.Context, id string) (*model.Evaluation, error) {
	var eval model.Evaluation
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&eval)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &eval, nil
}

// List devolve as avaliações mais recentes, limitadas por limit.
func (r *EvaluationRepository) List(ctx context.Context, limit int64) ([]model.Evaluation, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit)

	cursor, err := r.col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	evaluations := make([]model.Evaluation, 0)
	if err := cursor.All(ctx, &evaluations); err != nil {
		return nil, err
	}
	return evaluations, nil
}
