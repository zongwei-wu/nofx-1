import { describe, it, expect } from 'vitest'
import {
  sortTradersByPnl,
  partitionTradersForDashboard,
  type LeaderboardTrader,
} from './copyTradeLeaderboardUtils'

const t = (id: string, nickname: string, pnl: number, roi = 0): LeaderboardTrader => ({
  leadPortfolioId: id,
  nickname,
  pnl,
  roi,
})

describe('copyTradeLeaderboardUtils', () => {
  it('sortTradersByPnl sorts descending by pnl', () => {
    const sorted = sortTradersByPnl([t('a', 'A', 100), t('b', 'B', 500), t('c', 'C', 200)])
    expect(sorted.map((x) => x.leadPortfolioId)).toEqual(['b', 'c', 'a'])
  })

  it('sortTradersByPnl uses roi as tie-breaker', () => {
    const sorted = sortTradersByPnl([
      t('a', 'A', 100, 10),
      t('b', 'B', 100, 30),
      t('c', 'C', 100, 20),
    ])
    expect(sorted.map((x) => x.leadPortfolioId)).toEqual(['b', 'c', 'a'])
  })

  it('partitionTradersForDashboard splits by auto_follow', () => {
    const leaderboard = [t('p1', 'T1', 300), t('p2', 'T2', 500), t('p3', 'T3', 100)]
    const configs = [
      { portfolio_id: 'p1', nickname: 'T1', enabled: true, auto_follow: true },
      { portfolio_id: 'p2', nickname: 'T2', enabled: true, auto_follow: false },
    ]
    const { monitored, unmonitored } = partitionTradersForDashboard(leaderboard, configs)
    expect(monitored.map((x) => x.leadPortfolioId)).toEqual(['p1'])
    expect(unmonitored.map((x) => x.leadPortfolioId)).toEqual(['p2', 'p3'])
  })

  it('partitionTradersForDashboard includes monitored config not on leaderboard', () => {
    const { monitored, unmonitored } = partitionTradersForDashboard(
      [t('p1', 'T1', 100)],
      [{ portfolio_id: 'p9', nickname: 'OffLb', enabled: true, auto_follow: true }]
    )
    expect(monitored).toHaveLength(1)
    expect(monitored[0].leadPortfolioId).toBe('p9')
    expect(monitored[0].pnl).toBe(0)
    expect(unmonitored).toHaveLength(1)
  })

  it('both columns are sorted by pnl desc', () => {
    const leaderboard = [
      t('p1', 'T1', 100),
      t('p2', 'T2', 500),
      t('p3', 'T3', 300),
      t('p4', 'T4', 200),
    ]
    const configs = [
      { portfolio_id: 'p1', nickname: 'T1', enabled: true, auto_follow: true },
      { portfolio_id: 'p3', nickname: 'T3', enabled: true, auto_follow: true },
    ]
    const { monitored, unmonitored } = partitionTradersForDashboard(leaderboard, configs)
    expect(monitored.map((x) => x.leadPortfolioId)).toEqual(['p3', 'p1'])
    expect(unmonitored.map((x) => x.leadPortfolioId)).toEqual(['p2', 'p4'])
  })
})
