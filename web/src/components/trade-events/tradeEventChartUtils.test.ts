import { describe, it, expect } from 'vitest'
import { mapKlinesToLineData, mapEventsToMarkers, findEventNearTime } from './tradeEventChartUtils'
import type { TradeEvent, KlinePoint } from './tradeEventTypes'

describe('tradeEventChartUtils', () => {
  it('mapKlinesToLineData', () => {
    const klines: KlinePoint[] = [
      { time: 100, open: 1, high: 2, low: 0.5, close: 1.5 },
      { time: 200, open: 1.5, high: 2, low: 1, close: 1.8 },
    ]
    const data = mapKlinesToLineData(klines)
    expect(data).toHaveLength(2)
    expect(data[0].value).toBe(1.5)
  })

  it('mapEventsToMarkers', () => {
    const events: TradeEvent[] = [
      {
        symbol: 'BTCUSDT',
        time: 150000,
        type: 'open',
        side: 'LONG',
        qty: 1,
        price: 100,
        source: 'copy_trade',
        detail: 'test',
      },
    ]
    const m = mapEventsToMarkers(events)
    expect(m[0].time).toBe(150)
    expect(m[0].text).toBe('开仓')
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
