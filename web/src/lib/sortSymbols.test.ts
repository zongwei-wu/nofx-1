import { describe, it, expect } from 'vitest'
import { sortSymbolsByPreference } from './sortSymbols'

describe('sortSymbolsByPreference', () => {
  it('sorts by notional descending by default', () => {
    const out = sortSymbolsByPreference(['BTCUSDT', 'ETHUSDT', 'SOLUSDT'], {
      notionalBySymbol: {
        BTCUSDT: 100,
        ETHUSDT: 500,
        SOLUSDT: 200,
      },
    })
    expect(out).toEqual(['ETHUSDT', 'SOLUSDT', 'BTCUSDT'])
  })

  it('uses custom order when enabled', () => {
    const out = sortSymbolsByPreference(['BTCUSDT', 'ETHUSDT', 'SOLUSDT'], {
      useCustomOrder: true,
      customOrder: ['SOLUSDT', 'BTCUSDT'],
      notionalBySymbol: { ETHUSDT: 999 },
    })
    expect(out[0]).toBe('SOLUSDT')
    expect(out[1]).toBe('BTCUSDT')
    expect(out[2]).toBe('ETHUSDT')
  })
})
