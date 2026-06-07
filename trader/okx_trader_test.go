package trader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertSymbolToOKX(t *testing.T) {
	assert.Equal(t, "BTC-USDT-SWAP", convertSymbolToOKX("BTCUSDT"))
	assert.Equal(t, "ETH-USDT-SWAP", convertSymbolToOKX("ethusdt"))
	assert.Equal(t, "BTC-USDT-SWAP", convertSymbolToOKX("BTC-USDT-SWAP"))
}

func TestConvertOKXToSymbol(t *testing.T) {
	assert.Equal(t, "BTCUSDT", convertOKXToSymbol("BTC-USDT-SWAP"))
	assert.Equal(t, "ETHUSDT", convertOKXToSymbol("ETH-USDT-SWAP"))
}

func newOKXTestTrader(t *testing.T, handler http.HandlerFunc) *OKXTrader {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &OKXTrader{
		apiKey:      "test-key",
		secretKey:   "test-secret",
		passphrase:  "test-pass",
		testnet:     true,
		isCross:     true,
		baseURL:     server.URL,
		instruments: make(map[string]OKXInstrument),
		client:      server.Client(),
	}
}

func okxOK(data interface{}) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"code": "0",
		"msg":  "",
		"data": data,
	})
	return b
}

func TestOKXTrader_GetBalance(t *testing.T) {
	tr := newOKXTestTrader(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/account/set-position-mode"):
			w.Write(okxOK([]map[string]string{{"posMode": "long_short_mode"}}))
		case strings.Contains(r.URL.Path, "/account/balance"):
			w.Write(okxOK([]map[string]interface{}{
				{
					"totalEq": "10000",
					"upl":     "100.5",
					"details": []map[string]string{
						{"ccy": "USDT", "eq": "10000", "availEq": "8000", "cashBal": "9899.5", "upl": "100.5"},
					},
				},
			}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	bal, err := tr.GetBalance()
	require.NoError(t, err)
	assert.Equal(t, 9899.5, bal["totalWalletBalance"])
	assert.Equal(t, 8000.0, bal["availableBalance"])
	assert.Equal(t, 100.5, bal["totalUnrealizedProfit"])
}

func TestOKXTrader_GetPositions(t *testing.T) {
	tr := newOKXTestTrader(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/account/set-position-mode"):
			w.Write(okxOK([]map[string]string{}))
		case strings.Contains(r.URL.Path, "/public/instruments"):
			w.Write(okxOK([]map[string]string{
				{"instId": "BTC-USDT-SWAP", "ctVal": "0.01", "lotSz": "1", "minSz": "1", "tickSz": "0.1"},
			}))
		case strings.Contains(r.URL.Path, "/account/positions"):
			w.Write(okxOK([]map[string]string{
				{
					"instId": "BTC-USDT-SWAP", "pos": "10", "posSide": "long",
					"avgPx": "50000", "markPx": "50500", "upl": "50", "lever": "10", "liqPx": "40000",
				},
			}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	positions, err := tr.GetPositions()
	require.NoError(t, err)
	require.Len(t, positions, 1)
	assert.Equal(t, "BTCUSDT", positions[0]["symbol"])
	assert.Equal(t, "long", positions[0]["side"])
	assert.Equal(t, 0.1, positions[0]["positionAmt"])
}

func TestOKXTrader_GetMarketPrice(t *testing.T) {
	tr := newOKXTestTrader(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/account/set-position-mode"):
			w.Write(okxOK([]map[string]string{}))
		case strings.Contains(r.URL.Path, "/market/ticker"):
			w.Write(okxOK([]map[string]string{{"last": "50000.5"}}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	price, err := tr.GetMarketPrice("BTCUSDT")
	require.NoError(t, err)
	assert.Equal(t, 50000.5, price)
}

func TestOKXTrader_GetLotStepSize(t *testing.T) {
	tr := newOKXTestTrader(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/account/set-position-mode"):
			w.Write(okxOK([]map[string]string{}))
		case strings.Contains(r.URL.Path, "/public/instruments"):
			w.Write(okxOK([]map[string]string{
				{"instId": "ETH-USDT-SWAP", "ctVal": "0.1", "lotSz": "1", "minSz": "1", "tickSz": "0.01"},
			}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	step, err := tr.GetLotStepSize("ETHUSDT")
	require.NoError(t, err)
	assert.Equal(t, 0.1, step)
}

func TestOKXTrader_BaseToContracts(t *testing.T) {
	tr := newOKXTestTrader(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/account/set-position-mode"):
			w.Write(okxOK([]map[string]string{}))
		case strings.Contains(r.URL.Path, "/public/instruments"):
			w.Write(okxOK([]map[string]string{
				{"instId": "BTC-USDT-SWAP", "ctVal": "0.01", "lotSz": "1", "minSz": "1", "tickSz": "0.1"},
			}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	contracts, err := tr.baseToContracts("BTCUSDT", 0.05)
	require.NoError(t, err)
	assert.Equal(t, 5.0, contracts)
}
