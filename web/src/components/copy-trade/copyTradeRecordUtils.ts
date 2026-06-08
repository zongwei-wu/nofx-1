import type { CopyTradePnLRecord } from './copyTradePositionUtils'

export function formatRecordDirection(side: string, positionSide: string) {
  const action = side === 'BUY' ? '买' : '卖'
  const dir = positionSide === 'LONG' ? '多' : '空'
  return `${action}${dir}`
}

export function formatRecordTimeShort(ts: number | string | undefined) {
  if (!ts) return '-'
  const d = typeof ts === 'number' ? new Date(ts) : new Date(ts)
  if (Number.isNaN(d.getTime())) return String(ts)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${mm}/${dd} ${hh}:${mi}`
}

export function recordStatusLabel(status: string) {
  switch (status) {
    case 'OPEN':
      return '持仓'
    case 'CLOSED':
      return '已平'
    case 'FAILED':
      return '失败'
    default:
      return status
  }
}

function parseRecordTimeMs(value: number | string | undefined): number | null {
  if (value == null || value === '') return null
  if (typeof value === 'number') {
    return value > 0 ? value : null
  }
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? null : parsed
}

/** 本账户成交记录排序用时间：已平仓优先 close_time，否则 copy_time，再回退 lead_order_time */
export function recordEventTimeMs(rec: CopyTradePnLRecord): number {
  if (rec.status === 'CLOSED') {
    const closeMs = parseRecordTimeMs(rec.close_time)
    if (closeMs != null) return closeMs
  }
  const copyMs = parseRecordTimeMs(rec.copy_time)
  if (copyMs != null) return copyMs
  return rec.lead_order_time || 0
}

export function sortCopyTradeRecords(records: CopyTradePnLRecord[]) {
  return [...records].sort(
    (a, b) => recordEventTimeMs(b) - recordEventTimeMs(a)
  )
}
