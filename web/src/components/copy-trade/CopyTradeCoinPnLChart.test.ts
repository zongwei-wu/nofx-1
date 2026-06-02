import { describe, expect, it } from 'vitest'
import { buildHourlyCoinPnlSeries } from './CopyTradeCoinPnLChart'
import type { CopyTradePnLRecord } from './CopyTradePnLList'

describe('buildHourlyCoinPnlSeries', () => {
  it('aggregates closed pnl by hour and coin', () => {
    const closedAt = new Date()
    closedAt.setMinutes(0, 0, 0)
    const records: CopyTradePnLRecord[] = [
      {
        id: 1,
        portfolio_id: 'p1',
        nickname: 't',
        symbol: 'BTCUSDT',
        side: 'BUY',
        position_side: 'LONG',
        executed_qty: 1,
        avg_price: 100,
        total_pnl: 10,
        status: 'CLOSED',
        lead_order_time: closedAt.getTime() - 3600_000,
        close_time: closedAt.toISOString(),
      },
    ]
    const { chartData, coins } = buildHourlyCoinPnlSeries(records)
    expect(coins).toEqual(['BTC'])
    expect(chartData.length).toBeGreaterThan(0)
    const last = chartData[chartData.length - 1]
    expect(last.BTC).toBe(10)
  })
})
