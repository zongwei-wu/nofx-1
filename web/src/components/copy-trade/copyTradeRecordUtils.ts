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

export function sortCopyTradeRecords(records: CopyTradePnLRecord[]) {
  return [...records].sort((a, b) => {
    const nameCmp = (a.nickname || '').localeCompare(b.nickname || '', 'zh-CN')
    if (nameCmp !== 0) return nameCmp
    return (b.lead_order_time || 0) - (a.lead_order_time || 0)
  })
}
