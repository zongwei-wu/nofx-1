package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockQtyConverter struct {
	step float64
}

func (m *mockQtyConverter) GetLotStepSize(symbol string) (float64, error) {
	return m.step, nil
}

func TestLeadContractsToBaseQtyGeneric(t *testing.T) {
	conv := &mockQtyConverter{step: 0.1}
	got, err := leadContractsToBaseQtyGeneric(conv, "ETHUSDT", 10)
	assert.NoError(t, err)
	assert.Equal(t, 1.0, got)
}

func TestExchangeOrderQtyLabel(t *testing.T) {
	assert.Equal(t, "OKX 合约", exchangeOrderQtyLabel("okx"))
	assert.Equal(t, "币安合约", exchangeOrderQtyLabel("binance"))
}
