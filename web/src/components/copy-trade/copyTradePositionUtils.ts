export interface CopyTradePnLRecord {
  id: number
  portfolio_id: string
  nickname: string
  order_id?: string
  symbol: string
  side: string
  position_side: string
  executed_qty: number
  avg_price: number
  close_price?: number
  total_pnl: number
  pnl_kind?: 'unrealized' | 'realized'
  status: string
  error_message?: string
  lead_order_time: number
  copy_time?: string
  close_time?: string
}

export function openPositionKey(symbol: string, positionSide?: string) {
  return `${symbol}|${positionSide || ''}`
}

/** 按 symbol + position_side 聚合 OPEN 持仓（与交易所实际持仓口径一致） */
export function aggregateOpenPositions(
  records: CopyTradePnLRecord[]
): CopyTradePnLRecord[] {
  const open = records.filter((r) => r.status === 'OPEN' && r.symbol)
  const groups = new Map<string, CopyTradePnLRecord[]>()

  for (const rec of open) {
    const key = openPositionKey(rec.symbol, rec.position_side)
    const list = groups.get(key) ?? []
    list.push(rec)
    groups.set(key, list)
  }

  const aggregated: CopyTradePnLRecord[] = []
  for (const [, items] of groups) {
    const first = items[0]
    let totalQty = 0
    let weightedPrice = 0
    let totalPnl = 0
    for (const item of items) {
      const qty = item.executed_qty || 0
      totalQty += qty
      weightedPrice += (item.avg_price || 0) * qty
      totalPnl += item.total_pnl || 0
    }
    const nicknames = [...new Set(items.map((i) => i.nickname).filter(Boolean))]
    aggregated.push({
      ...first,
      id: first.id,
      nickname:
        nicknames.length > 1
          ? `${nicknames.length}位交易员`
          : nicknames[0] || first.nickname,
      executed_qty: totalQty,
      avg_price: totalQty > 0 ? weightedPrice / totalQty : first.avg_price,
      total_pnl: totalPnl,
    })
  }

  return aggregated.sort(
    (a, b) =>
      openPositionKey(a.symbol, a.position_side).localeCompare(
        openPositionKey(b.symbol, b.position_side)
      )
  )
}

export function countUniqueOpenPositions(records: CopyTradePnLRecord[]) {
  const keys = new Set<string>()
  for (const rec of records) {
    if (rec.status === 'OPEN' && rec.symbol) {
      keys.add(openPositionKey(rec.symbol, rec.position_side))
    }
  }
  return keys.size
}
