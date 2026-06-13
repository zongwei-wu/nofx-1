package backtest

import (
	"fmt"
	"nofx/indicator"
	"nofx/market"
)

// DataLoader K 线数据加载器
type DataLoader struct {
	client *market.APIClient
}

// NewDataLoader 创建数据加载器
func NewDataLoader() *DataLoader {
	return &DataLoader{client: market.NewAPIClient()}
}

// LoadKlines 加载历史 K 线
func (d *DataLoader) LoadKlines(symbol, interval string, limit int) ([]indicator.Kline, error) {
	symbol = market.Normalize(symbol)
	if limit <= 0 {
		limit = 500
	}
	if limit > 1500 {
		limit = 1500
	}
	mk, err := d.client.GetKlines(symbol, interval, limit)
	if err != nil {
		return nil, fmt.Errorf("加载K线失败: %w", err)
	}
	out := make([]indicator.Kline, len(mk))
	for i, k := range mk {
		out[i] = indicator.Kline{
			OpenTime: k.OpenTime,
			Open:     k.Open,
			High:     k.High,
			Low:      k.Low,
			Close:    k.Close,
			Volume:   k.Volume,
		}
	}
	return out, nil
}
