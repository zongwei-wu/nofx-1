import { describe, expect, it } from 'vitest'
import {
  aggregateOpenPositions,
  countUniqueOpenPositions,
} from './copyTradePositionUtils'
import type { CopyTradePnLRecord } from './copyTradePositionUtils'

const base = (over: Partial<CopyTradePnLRecord>): CopyTradePnLRecord => ({
  id: 1,
  portfolio_id: 'p1',
  nickname: 'A',
  symbol: 'BTCUSDT',
  side: 'BUY',
  position_side: 'LONG',
  executed_qty: 1,
  avg_price: 100,
  total_pnl: 10,
  status: 'OPEN',
  lead_order_time: 1,
  ...over,
})

describe('copyTradePositionUtils', () => {
  it('counts unique symbol+position_side', () => {
    const records = [
      base({ id: 1 }),
      base({ id: 2, nickname: 'B' }),
      base({ id: 3, symbol: 'ETHUSDT', position_side: 'SHORT', side: 'SELL' }),
    ]
    expect(countUniqueOpenPositions(records)).toBe(2)
  })

  it('aggregates qty and pnl for same position', () => {
    const records = [
      base({ id: 1, executed_qty: 1, avg_price: 100, total_pnl: 5 }),
      base({ id: 2, executed_qty: 2, avg_price: 110, total_pnl: 7 }),
    ]
    const agg = aggregateOpenPositions(records)
    expect(agg).toHaveLength(1)
    expect(agg[0].executed_qty).toBe(3)
    expect(agg[0].total_pnl).toBe(12)
    expect(agg[0].avg_price).toBeCloseTo(106.666, 2)
  })
})
