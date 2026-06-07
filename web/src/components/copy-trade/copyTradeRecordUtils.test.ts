import { describe, expect, it } from 'vitest'
import {
  formatRecordDirection,
  formatRecordTimeShort,
  recordStatusLabel,
  sortCopyTradeRecords,
} from './copyTradeRecordUtils'
import type { CopyTradePnLRecord } from './copyTradePositionUtils'

function base(overrides: Partial<CopyTradePnLRecord> = {}): CopyTradePnLRecord {
  return {
    id: 1,
    portfolio_id: 'p1',
    nickname: 'Alice',
    symbol: 'BTCUSDT',
    side: 'BUY',
    position_side: 'LONG',
    executed_qty: 1,
    avg_price: 100,
    total_pnl: 0,
    status: 'OPEN',
    lead_order_time: 1000,
    ...overrides,
  }
}

describe('copyTradeRecordUtils', () => {
  it('formatRecordDirection', () => {
    expect(formatRecordDirection('BUY', 'LONG')).toBe('买多')
    expect(formatRecordDirection('SELL', 'SHORT')).toBe('卖空')
  })

  it('formatRecordTimeShort', () => {
    const ts = new Date('2026-06-04T15:30:00').getTime()
    expect(formatRecordTimeShort(ts)).toMatch(/06\/04 15:30/)
  })

  it('recordStatusLabel', () => {
    expect(recordStatusLabel('OPEN')).toBe('持仓')
    expect(recordStatusLabel('CLOSED')).toBe('已平')
    expect(recordStatusLabel('FAILED')).toBe('失败')
  })

  it('sortCopyTradeRecords by nickname then lead_order_time desc', () => {
    const sorted = sortCopyTradeRecords([
      base({ id: 1, nickname: 'Bob', lead_order_time: 100 }),
      base({ id: 2, nickname: 'Alice', lead_order_time: 200 }),
      base({ id: 3, nickname: 'Alice', lead_order_time: 300 }),
    ])
    expect(sorted.map((r) => r.id)).toEqual([3, 2, 1])
  })
})
