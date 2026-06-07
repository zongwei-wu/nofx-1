export type TradeEventType = 'open' | 'add' | 'reduce' | 'close'

export interface TradeEvent {
  symbol: string
  time: number
  type: TradeEventType
  side: string
  qty: number
  price: number
  source: string
  detail: string
}

export interface KlinePoint {
  time: number
  open: number
  high: number
  low: number
  close: number
}

/** K 线持仓叠加：入场价水平线 + 未实现盈亏 */
export interface ChartPositionOverlay {
  symbol: string
  position_side: string
  entry_price: number
  unrealized_pnl: number
  qty?: number
  label?: string
}

export const EVENT_TYPE_LABELS: Record<TradeEventType, string> = {
  open: '开仓',
  add: '加仓',
  reduce: '减仓',
  close: '平仓',
}

export const EVENT_TYPE_COLORS: Record<TradeEventType, string> = {
  open: '#0ECB81',
  add: '#38BDF8',
  reduce: '#F0B90B',
  close: '#F6465D',
}
