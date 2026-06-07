import { describe, it, expect } from 'vitest'
import {
  mapKlinesToChartRows,
  mapKlinesToCandlestickData,
  getDefaultVisibleTimeRange,
  CHART_COPY_TRADE_VISIBLE_HOURS,
  mapEventsToScatterPoints,
  mapScatterPointsToMarkers,
  mapScatterPointsToClusteredMarkers,
  thinScatterPointsForDensity,
  findClusterNearTime,
  countEventsAtSameKline,
  findEventNearTime,
  eventTradeAmount,
  scaleDotRadiusByAmount,
  normalizeTradingSymbol,
  symbolsMatch,
  pickDefaultChartSymbol,
  collectChartSymbols,
  DEFAULT_CHART_SYMBOL,
  filterOverlaysForSymbol,
  formatPositionLineTitle,
  positionOverlayLineColor,
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

  it('collectChartSymbols merges current and historical positions', () => {
    const syms = collectChartSymbols(['BTCUSDT'], [
      {
        positions: [{ symbol: 'ETHUSDT', position_amt: 1 }],
        decisions: [{ symbol: 'SOLUSDT', success: true }],
      },
    ])
    expect(syms).toContain('BTCUSDT')
    expect(syms).toContain('ETHUSDT')
    expect(syms).toContain('SOLUSDT')
  })

  it('collectChartSymbols ai_exchange excludes copy trade and non-trade actions', () => {
    const syms = collectChartSymbols(['BTCUSDT'], [
      {
        source: 'copy_trade',
        success: true,
        copy_trade_meta: { ai_trader_id: 't1', action_taken: 'copied_open' },
        decisions: [{ symbol: 'ETHUSDT', success: true }],
      },
      {
        source: 'auto_trader',
        decisions: [
          { symbol: 'SOLUSDT', success: true, action: 'hold' },
          { symbol: 'XRPUSDT', success: true, action: 'open_long' },
        ],
      },
    ], { mode: 'ai_exchange' })
    expect(syms).toContain('BTCUSDT')
    expect(syms).toContain('XRPUSDT')
    expect(syms).not.toContain('ETHUSDT')
    expect(syms).not.toContain('SOLUSDT')
  })

  it('collectChartSymbols ai_copy filters by trader and copied_open', () => {
    const syms = collectChartSymbols([], [
      {
        source: 'copy_trade',
        success: true,
        copy_trade_meta: { ai_trader_id: 't1', action_taken: 'copied_open' },
        decisions: [{ symbol: 'BTCUSDT', success: true }],
      },
      {
        source: 'copy_trade',
        success: true,
        copy_trade_meta: { ai_trader_id: 't2', action_taken: 'copied_open' },
        decisions: [{ symbol: 'ETHUSDT', success: true }],
      },
      {
        source: 'copy_trade',
        success: false,
        copy_trade_meta: { ai_trader_id: 't1', action_taken: 'open_failed' },
        decisions: [{ symbol: 'SOLUSDT', success: false }],
      },
    ], { mode: 'ai_copy', traderId: 't1' })
    expect(syms).toEqual(['BTCUSDT'])
  })

  it('pickDefaultChartSymbol prefers first in preferred order', () => {
    expect(
      pickDefaultChartSymbol(['ETHUSDT', 'SOLUSDT'], ['SOLUSDT', 'ETHUSDT'], '')
    ).toBe('SOLUSDT')
    expect(
      pickDefaultChartSymbol(['ETHUSDT'], ['ETHUSDT'], 'ETHUSDT')
    ).toBe('ETHUSDT')
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
        time: 1000,
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

  it('getDefaultVisibleTimeRange uses last 48 hours', () => {
    const base = 1_700_000_000
    const times = Array.from({ length: 100 }, (_, i) => base + i * 3600)
    const range = getDefaultVisibleTimeRange(times, 48)
    expect(range).not.toBeNull()
    expect(range!.to).toBe(base + 99 * 3600)
    expect(range!.from).toBe(range!.to - 48 * 3600)
  })

  it('getDefaultVisibleTimeRange supports 1 week window for copy trade', () => {
    const base = 1_700_000_000
    const times = Array.from({ length: 200 }, (_, i) => base + i * 3600)
    const range = getDefaultVisibleTimeRange(times, CHART_COPY_TRADE_VISIBLE_HOURS)
    expect(range!.to).toBe(base + 199 * 3600)
    expect(range!.from).toBe(range!.to - CHART_COPY_TRADE_VISIBLE_HOURS * 3600)
  })

  it('getDefaultVisibleTimeRange clamps when data shorter than window', () => {
    const range = getDefaultVisibleTimeRange([1000, 2000, 3000], 48)
    expect(range!.from).toBe(1000)
    expect(range!.to).toBe(3000)
  })

  it('mapKlinesToCandlestickData', () => {
    const data = mapKlinesToCandlestickData([
      { time: 200, open: 1, high: 2, low: 0.5, close: 1.5 },
      { time: 100, open: 1, high: 2, low: 0.5, close: 1.2 },
    ])
    expect(data).toHaveLength(2)
    expect(data[0].time).toBe(100)
    expect(data[1].close).toBe(1.5)
  })

  it('mapScatterPointsToClusteredMarkers merges same kline', () => {
    const priceRows = mapKlinesToChartRows([
      { time: 100, open: 1, high: 2, low: 0.5, close: 100 },
    ])
    const events: TradeEvent[] = [
      {
        symbol: 'BTCUSDT',
        time: 100,
        type: 'open',
        side: 'LONG',
        qty: 1,
        price: 100,
        source: '',
        detail: '',
      },
      {
        symbol: 'BTCUSDT',
        time: 101,
        type: 'add',
        side: 'LONG',
        qty: 1,
        price: 100,
        source: '',
        detail: '',
      },
      {
        symbol: 'BTCUSDT',
        time: 102,
        type: 'close',
        side: 'LONG',
        qty: 1,
        price: 100,
        source: '',
        detail: '',
      },
    ]
    const points = mapEventsToScatterPoints(events, priceRows)
    const clusters = mapScatterPointsToClusteredMarkers(points, null)
    expect(clusters).toHaveLength(1)
    expect(clusters[0].events).toHaveLength(3)
    expect(clusters[0].text).toBe('3')
    expect(clusters[0].primaryEvent.type).toBe('close')
  })

  it('thinScatterPointsForDensity keeps open/close when crowded', () => {
    const points = Array.from({ length: 30 }, (_, i) => ({
      timeSec: 100 + i,
      rawTimeSec: 100 + i,
      price: 100,
      color: '#0ECB81',
      label: '加仓',
      amount: 100,
      dotRadius: 6,
      event: {
        symbol: 'BTCUSDT',
        time: 100 + i,
        type: (i === 0 ? 'open' : i === 29 ? 'close' : i % 2 === 0 ? 'add' : 'reduce') as TradeEvent['type'],
        side: 'LONG' as const,
        qty: 1,
        price: 100,
        source: '',
        detail: '',
      },
    }))
    const thinned = thinScatterPointsForDensity(points)
    expect(thinned).toHaveLength(2)
    expect(thinned.map((p) => p.event.type)).toEqual(['open', 'close'])
  })

  it('findClusterNearTime and countEventsAtSameKline', () => {
    const priceRows = mapKlinesToChartRows([
      { time: 100, open: 1, high: 2, low: 0.5, close: 100 },
    ])
    const events: TradeEvent[] = [
      {
        symbol: 'BTCUSDT',
        time: 100,
        type: 'open',
        side: 'LONG',
        qty: 1,
        price: 100,
        source: '',
        detail: '',
      },
      {
        symbol: 'BTCUSDT',
        time: 101,
        type: 'add',
        side: 'LONG',
        qty: 1,
        price: 100,
        source: '',
        detail: '',
      },
    ]
    const points = mapEventsToScatterPoints(events, priceRows)
    const clusters = mapScatterPointsToClusteredMarkers(points, null)
    expect(findClusterNearTime(clusters, 100, 60)?.events).toHaveLength(2)
    expect(countEventsAtSameKline(points, events[0])).toBe(2)
  })

  it('mapScatterPointsToMarkers highlights selected', () => {
    const event: TradeEvent = {
      symbol: 'BTCUSDT',
      time: 100,
      type: 'open',
      side: 'LONG',
      qty: 1,
      price: 100,
      source: '',
      detail: '',
    }
    const points = mapEventsToScatterPoints([event], mapKlinesToChartRows([
      { time: 100, open: 1, high: 2, low: 0.5, close: 100 },
    ]))
    const markers = mapScatterPointsToMarkers(points, event)
    expect(markers[0].color).toBe('#EAECEF')
    expect(markers[0].text).toBe('开仓')
  })

  it('filterOverlaysForSymbol matches normalized symbol', () => {
    const overlays = [
      {
        symbol: 'BTCUSDT',
        position_side: 'LONG',
        entry_price: 100000,
        unrealized_pnl: 10,
      },
      {
        symbol: 'ETHUSDT',
        position_side: 'SHORT',
        entry_price: 3000,
        unrealized_pnl: -5,
      },
      {
        symbol: 'SOLUSDT',
        position_side: 'LONG',
        entry_price: 0,
        unrealized_pnl: 0,
      },
    ]
    const filtered = filterOverlaysForSymbol(overlays, 'BTC')
    expect(filtered).toHaveLength(1)
    expect(filtered[0].symbol).toBe('BTCUSDT')
  })

  it('formatPositionLineTitle includes direction price and pnl', () => {
    const title = formatPositionLineTitle({
      symbol: 'BTCUSDT',
      position_side: 'LONG',
      entry_price: 96500.5,
      unrealized_pnl: 12.34,
    })
    expect(title).toContain('入场')
    expect(title).toContain('多')
    expect(title).toContain('+12.34')
  })

  it('positionOverlayLineColor uses fixed accent color', () => {
    expect(positionOverlayLineColor(1)).toBe('#22D3EE')
    expect(positionOverlayLineColor(-1)).toBe('#22D3EE')
  })
})
