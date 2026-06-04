import { describe, it, expect } from 'vitest'
import {
  mapKlinesToChartRows,
  mapEventsToScatterPoints,
  findEventNearTime,
  eventTradeAmount,
  scaleDotRadiusByAmount,
  normalizeTradingSymbol,
  symbolsMatch,
} from './tradeEventChartUtils'
import type { TradeEvent, KlinePoint } from './tradeEventTypes'

describe('tradeEventChartUtils', () => {
  it('mapKlinesToChartRows', () => {
    const klines: KlinePoint[] = [
      { time: 100, open: 1, high: 2, low: 0.5, close: 1.5 },
      { time: 200, open: 1.5, high: 2, low: 1, close: 1.8 },
    ]
    const data = mapKlinesToChartRows(klines)
    expect(data).toHaveLength(2)
    expect(data[0].close).toBe(1.5)
  })

  it('mapEventsToScatterPoints', () => {
    const priceRows = mapKlinesToChartRows([
      { time: 100, open: 1, high: 2, low: 0.5, close: 100 },
    ])
    const events: TradeEvent[] = [
      {
        symbol: 'BTCUSDT',
        time: 100000,
        type: 'open',
        side: 'LONG',
        qty: 1,
        price: 0,
        source: 'copy_trade',
        detail: 'test',
      },
    ]
    const pts = mapEventsToScatterPoints(events, priceRows)
    expect(pts[0].price).toBe(100)
    expect(pts[0].label).toBe('开仓')
    expect(pts[0].amount).toBe(100)
    expect(pts[0].dotRadius).toBeGreaterThan(0)
  })

  it('normalizeTradingSymbol', () => {
    expect(normalizeTradingSymbol('pumpbtc')).toBe('PUMPBTCUSDT')
    expect(symbolsMatch('ETH', 'ETHUSDT')).toBe(true)
  })

  it('scaleDotRadiusByAmount — larger amount yields larger radius', () => {
    const small = scaleDotRadiusByAmount(100, 100, 10000, 4, 12)
    const large = scaleDotRadiusByAmount(10000, 100, 10000, 4, 12)
    expect(large).toBeGreaterThan(small)
    expect(eventTradeAmount({ symbol: 'X', time: 1, type: 'open', side: 'LONG', qty: 2, price: 50, source: '', detail: '' })).toBe(100)
  })

  it('findEventNearTime', () => {
    const events: TradeEvent[] = [
      {
        symbol: 'BTCUSDT',
        time: 1000000,
        type: 'open',
        side: 'LONG',
        qty: 1,
        price: 100,
        source: 'ai_trader',
        detail: '',
      },
    ]
    expect(findEventNearTime(events, 1000, 60)?.type).toBe('open')
    expect(findEventNearTime(events, 9999, 10)).toBeNull()
  })
})
