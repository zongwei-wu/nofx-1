export type LeaderboardTrader = {
  leadPortfolioId: string
  nickname: string
  pnl: number
  roi: number
  [key: string]: unknown
}

export type CopyConfigLike = {
  portfolio_id: string
  nickname: string
  enabled: boolean
  auto_follow: boolean
}

function traderPnl(t: LeaderboardTrader): number {
  return Number(t.pnl) || 0
}

function traderRoi(t: LeaderboardTrader): number {
  return Number(t.roi) || 0
}

export function sortTradersByPnl(traders: LeaderboardTrader[]): LeaderboardTrader[] {
  return [...traders].sort((a, b) => {
    const pnlDiff = traderPnl(b) - traderPnl(a)
    if (pnlDiff !== 0) return pnlDiff
    return traderRoi(b) - traderRoi(a)
  })
}

function isMonitored(cfg: CopyConfigLike | undefined): boolean {
  return Boolean(cfg?.enabled && cfg.auto_follow)
}

export function partitionTradersForDashboard(
  leaderboard: LeaderboardTrader[],
  configs: CopyConfigLike[]
): { monitored: LeaderboardTrader[]; unmonitored: LeaderboardTrader[] } {
  const byPortfolio = new Map<string, LeaderboardTrader>()
  for (const t of leaderboard) {
    if (t.leadPortfolioId) {
      byPortfolio.set(t.leadPortfolioId, t)
    }
  }

  const monitored: LeaderboardTrader[] = []
  const monitoredIds = new Set<string>()

  for (const cfg of configs) {
    if (!cfg.enabled || !cfg.auto_follow) continue
    monitoredIds.add(cfg.portfolio_id)
    const fromLb = byPortfolio.get(cfg.portfolio_id)
    if (fromLb) {
      monitored.push(fromLb)
    } else {
      monitored.push({
        leadPortfolioId: cfg.portfolio_id,
        nickname: cfg.nickname || '未知交易员',
        pnl: 0,
        roi: 0,
      })
    }
  }

  const unmonitored: LeaderboardTrader[] = []
  for (const t of leaderboard) {
    const cfg = configs.find((c) => c.portfolio_id === t.leadPortfolioId)
    if (!isMonitored(cfg)) {
      unmonitored.push(t)
    }
  }

  return {
    monitored: sortTradersByPnl(monitored),
    unmonitored: sortTradersByPnl(unmonitored),
  }
}
