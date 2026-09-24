package money

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMoney_JSON(t *testing.T) {
	// saída: sempre string com 2 casas
	b, err := json.Marshal(New(decimal.RequireFromString("1425")))
	require.NoError(t, err)
	assert.Equal(t, `"1425.00"`, string(b))

	// entrada: aceita string e número
	var fromString Money
	require.NoError(t, json.Unmarshal([]byte(`"75000.00"`), &fromString))
	assert.Equal(t, "75000.00", fromString.StringFixed(2))

	var fromNumber Money
	require.NoError(t, json.Unmarshal([]byte(`75000`), &fromNumber))
	assert.Equal(t, "75000.00", fromNumber.StringFixed(2))
}

func TestMoney_BSON_RoundTrip(t *testing.T) {
	type doc struct {
		Amount Money `bson:"amount"`
	}

	raw, err := bson.Marshal(doc{Amount: New(decimal.RequireFromString("1425.00"))})
	require.NoError(t, err)

	var back doc
	require.NoError(t, bson.Unmarshal(raw, &back))
	assert.Equal(t, "1425.00", back.Amount.StringFixed(2))
}
