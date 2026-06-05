import { describe, it, expect } from 'vitest'
import { toTradingViewSymbol, toTradingViewInterval } from './tradingViewUtils'

describe('tradingViewUtils', () => {
  it('toTradingViewSymbol maps to Binance perpetual', () => {
    expect(toTradingViewSymbol('ETH')).toBe('BINANCE:ETHUSDT.P')
    expect(toTradingViewSymbol('BTCUSDT')).toBe('BINANCE:BTCUSDT.P')
  })

  it('toTradingViewInterval maps chart intervals', () => {
    expect(toTradingViewInterval('1h')).toBe('60')
    expect(toTradingViewInterval('4h')).toBe('240')
  })
})
