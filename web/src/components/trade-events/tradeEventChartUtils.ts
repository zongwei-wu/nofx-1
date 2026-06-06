import type { TradeEvent, TradeEventType, KlinePoint } from './tradeEventTypes'
import { EVENT_TYPE_COLORS, EVENT_TYPE_LABELS } from './tradeEventTypes'
export const DEFAULT_CHART_SYMBOL = 'BTCUSDT'

/** K 线默认可见时间窗口（小时）— AI 看板 */
export const CHART_DEFAULT_VISIBLE_HOURS = 48

/** 跟单管理 K 线默认可见时间窗口（1 周） */
export const CHART_COPY_TRADE_VISIBLE_HOURS = 7 * 24

export type ChartSymbolMode = 'all' | 'ai_exchange' | 'ai_copy'

const AI_TRADE_ACTIONS = new Set([
  'open_long',
  'open_short',
  'close_long',
  'close_short',
  'partial_close',
])

type PositionLike = { symbol?: string; position_amt?: number; positionAmt?: number }
type DecisionActionLike = { symbol?: string; success?: boolean; action?: string }
type DecisionRecordLike = {
  source?: 'auto_trader' | 'copy_trade'
  positions?: PositionLike[]
  decisions?: DecisionActionLike[]
  copy_trade_meta?: { ai_trader_id?: string; action_taken?: string }
  success?: boolean
}

function isAITradeDecisionAction(action?: string): boolean {
  return Boolean(action && AI_TRADE_ACTIONS.has(action))
}

/** 合并当前持仓与决策日志中的历史持仓/成交币种 */
export function collectChartSymbols(
  currentSymbols: string[],
  decisions: DecisionRecordLike[] = [],
  options?: { mode?: ChartSymbolMode; traderId?: string }
): string[] {
  const mode = options?.mode ?? 'all'
  const traderId = options?.traderId ?? ''
  const merged = new Set<string>()

  if (mode === 'all' || mode === 'ai_exchange') {
    for (const s of currentSymbols) {
      const n = normalizeTradingSymbol(s)
      if (n) merged.add(n)
    }
  }

  for (const rec of decisions) {
    if (mode === 'ai_exchange' && rec.source === 'copy_trade') {
      continue
    }
    if (mode === 'ai_copy') {
      if (rec.source !== 'copy_trade') continue
      if (traderId && rec.copy_trade_meta?.ai_trader_id !== traderId) continue
      if (rec.copy_trade_meta?.action_taken !== 'copied_open' || rec.success === false) {
        continue
      }
      for (const d of rec.decisions ?? []) {
        if (!d.symbol) continue
        const n = normalizeTradingSymbol(d.symbol)
        if (n) merged.add(n)
      }
      continue
    }

    if (mode !== 'ai_copy') {
      for (const pos of rec.positions ?? []) {
        const amt = pos.position_amt ?? pos.positionAmt ?? 0
        if (amt === 0) continue
        const n = normalizeTradingSymbol(pos.symbol)
        if (n) merged.add(n)
      }
    }

    for (const d of rec.decisions ?? []) {
      if (d.success === false || !d.symbol) continue
      if (mode === 'ai_exchange' && !isAITradeDecisionAction(d.action)) {
        continue
      }
      const n = normalizeTradingSymbol(d.symbol)
      if (n) merged.add(n)
    }
  }
  return [...merged]
}

/** 默认选中比特币；若当前选项仍有效则保留用户选择 */
export function pickDefaultChartSymbol(
  available: string[],
  preferred: string[],
  current: string
): string {
  const pool = new Set(
    [...available, ...preferred, DEFAULT_CHART_SYMBOL]
      .map(normalizeTradingSymbol)
      .filter(Boolean)
  )
  const list = [...pool]

  if (current) {
    const norm = normalizeTradingSymbol(current)
    if (pool.has(norm)) return norm
  }

  const btc = normalizeTradingSymbol(DEFAULT_CHART_SYMBOL)
  if (pool.has(btc)) return btc

  for (const s of preferred) {
    const norm = normalizeTradingSymbol(s)
    if (pool.has(norm)) return norm
  }

  return list[0] || btc
}

/** 统一为 XXXUSDT，避免 PUMPBTC 与 PUMPBTCUSDT 过滤不一致 */
export function normalizeTradingSymbol(symbol?: string): string {
  const u = (symbol || '').toUpperCase().trim()
  if (!u) return ''
  return u.endsWith('USDT') ? u : `${u}USDT`
}

export function symbolsMatch(a?: string, b?: string): boolean {
  return normalizeTradingSymbol(a) === normalizeTradingSymbol(b)
}

/** 将事件/订单时间统一为 Unix 秒（兼容秒与毫秒） */
export function msToChartTime(ms: number): number {
  if (!ms || ms <= 0) return 0
  if (ms >= 1e11) return Math.floor(ms / 1000)
  return Math.floor(ms)
}

export function formatChartTimeLabel(timeSec: number): string {
  return new Date(timeSec * 1000).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export interface PriceChartRow {
  timeSec: number
  timeLabel: string
  close: number
}

export function mapKlinesToChartRows(klines: KlinePoint[]): PriceChartRow[] {
  return klines
    .filter((k) => k.time > 0 && k.close > 0)
    .map((k) => {
      const timeSec = k.time < 1e12 ? k.time : Math.floor(k.time / 1000)
      return {
        timeSec,
        timeLabel: formatChartTimeLabel(timeSec),
        close: k.close,
      }
    })
    .sort((a, b) => a.timeSec - b.timeSec)
}

export interface EventScatterPoint {
  timeSec: number
  rawTimeSec: number
  timeLabel: string
  price: number
  /** 成交金额（USDT 名义价值）= qty × price */
  amount: number
  dotRadius: number
  event: TradeEvent
  color: string
  label: string
}

/** 单笔事件成交金额（开仓/加仓等为 qty×price） */
export function eventTradeAmount(event: TradeEvent, chartPrice = 0): number {
  const price = event.price > 0 ? event.price : chartPrice
  if (price <= 0 || event.qty <= 0) return 0
  return event.qty * price
}

export function scaleDotRadiusByAmount(
  amount: number,
  minAmount: number,
  maxAmount: number,
  minR: number,
  maxR: number
): number {
  if (amount <= 0) return minR
  if (maxAmount <= minAmount) return (minR + maxR) / 2
  const t = Math.min(1, Math.max(0, (amount - minAmount) / (maxAmount - minAmount)))
  return minR + Math.sqrt(t) * (maxR - minR)
}

export function dotRadiusBounds(pointCount: number): { minR: number; maxR: number } {
  if (pointCount > 40) return { minR: 3, maxR: 8 }
  if (pointCount > 20) return { minR: 3, maxR: 10 }
  return { minR: 4, maxR: 12 }
}

/** 将事件时间对齐到最近一根 K 线，便于在折线图上落点 */
export function snapTimeToNearestKline(
  timeSec: number,
  priceRows: PriceChartRow[]
): number {
  if (priceRows.length === 0) return timeSec
  let best = priceRows[0].timeSec
  let bestDiff = Math.abs(best - timeSec)
  for (const row of priceRows) {
    const diff = Math.abs(row.timeSec - timeSec)
    if (diff < bestDiff) {
      bestDiff = diff
      best = row.timeSec
    }
  }
  return best
}

export function resolveEventPrice(
  event: TradeEvent,
  priceRows: PriceChartRow[]
): number {
  if (event.price > 0) return event.price
  const t = msToChartTime(event.time)
  let best = priceRows[0]?.close ?? 0
  let bestDiff = Infinity
  for (const row of priceRows) {
    const diff = Math.abs(row.timeSec - t)
    if (diff < bestDiff) {
      bestDiff = diff
      best = row.close
    }
  }
  return best
}

export interface LwcMarkerPoint {
  time: number
  position: 'aboveBar' | 'belowBar'
  color: string
  shape: 'circle'
  text?: string
  event: TradeEvent
}

export interface LwcMarkerCluster {
  time: number
  position: 'aboveBar' | 'belowBar'
  color: string
  shape: 'circle'
  text?: string
  events: TradeEvent[]
  primaryEvent: TradeEvent
}

const MARKER_TYPE_PRIORITY: Record<TradeEventType, number> = {
  close: 4,
  open: 3,
  reduce: 2,
  add: 1,
}

/** 计算 K 线默认可见区间（最近 N 小时，秒级 Unix 时间戳） */
export function getDefaultVisibleTimeRange(
  candleTimes: number[],
  visibleHours = CHART_DEFAULT_VISIBLE_HOURS
): { from: number; to: number } | null {
  if (candleTimes.length === 0) return null
  const sorted = [...candleTimes].sort((a, b) => a - b)
  const to = sorted[sorted.length - 1]
  const from = Math.max(sorted[0], to - visibleHours * 3600)
  return { from, to }
}

export function mapKlinesToCandlestickData(klines: KlinePoint[]) {
  const byTime = new Map<
    number,
    { time: number; open: number; high: number; low: number; close: number }
  >()
  for (const k of klines) {
    if (k.time <= 0 || k.close <= 0) continue
    const timeSec = k.time < 1e12 ? k.time : Math.floor(k.time / 1000)
    byTime.set(timeSec, {
      time: timeSec,
      open: k.open > 0 ? k.open : k.close,
      high: k.high > 0 ? k.high : k.close,
      low: k.low > 0 ? k.low : k.close,
      close: k.close,
    })
  }
  return [...byTime.values()].sort((a, b) => a.time - b.time)
}

/** 高密度时仅保留开/平仓，减少中间加仓/减仓标记 */
export function thinScatterPointsForDensity(points: EventScatterPoint[]): EventScatterPoint[] {
  if (points.length <= 24) return points
  const major = points.filter((p) => p.event.type === 'open' || p.event.type === 'close')
  return major.length > 0 ? major : points
}

/** 同一根 K 线上的事件合并为一个标记，显示数量 */
export function mapScatterPointsToClusteredMarkers(
  points: EventScatterPoint[],
  selectedEvent: TradeEvent | null
): LwcMarkerCluster[] {
  const thinned = thinScatterPointsForDensity(points)
  const byTime = new Map<number, EventScatterPoint[]>()

  for (const p of thinned) {
    const list = byTime.get(p.timeSec) ?? []
    list.push(p)
    byTime.set(p.timeSec, list)
  }

  const clusters: LwcMarkerCluster[] = []
  for (const [timeSec, group] of byTime) {
    const sorted = [...group].sort((a, b) => {
      const pa = MARKER_TYPE_PRIORITY[a.event.type as TradeEventType] ?? 0
      const pb = MARKER_TYPE_PRIORITY[b.event.type as TradeEventType] ?? 0
      if (pb !== pa) return pb - pa
      return b.rawTimeSec - a.rawTimeSec
    })
    const primary = sorted[0]
    const events = group.map((g) => g.event)
    const selectedInGroup = selectedEvent && events.includes(selectedEvent)
    const shortCount = group.filter((g) => g.event.side === 'SHORT').length
    const position: 'aboveBar' | 'belowBar' =
      shortCount > group.length / 2 ? 'aboveBar' : 'belowBar'
    const count = group.length
    const active = Boolean(selectedInGroup)

    let text: string | undefined
    if (count > 1) {
      text = active ? `${count}笔` : String(count)
    } else if (active) {
      text = primary.label
    }

    clusters.push({
      time: timeSec,
      position,
      color: active ? '#EAECEF' : primary.color,
      shape: 'circle',
      text,
      events,
      primaryEvent: selectedInGroup && selectedEvent ? selectedEvent : primary.event,
    })
  }

  return clusters.sort((a, b) => a.time - b.time)
}

export function mapScatterPointsToMarkers(
  points: EventScatterPoint[],
  selectedEvent: TradeEvent | null
): LwcMarkerPoint[] {
  return mapScatterPointsToClusteredMarkers(points, selectedEvent).map((c) => ({
    time: c.time,
    position: c.position,
    color: c.color,
    shape: c.shape,
    text: c.text,
    event: c.primaryEvent,
  }))
}

export function findClusterNearTime(
  clusters: LwcMarkerCluster[],
  chartTimeSec: number,
  thresholdSec: number
): LwcMarkerCluster | null {
  let best: LwcMarkerCluster | null = null
  let bestDiff = thresholdSec + 1
  for (const c of clusters) {
    const diff = Math.abs(c.time - chartTimeSec)
    if (diff < bestDiff) {
      bestDiff = diff
      best = c
    }
  }
  return bestDiff <= thresholdSec ? best : null
}

export function countEventsAtSameKline(
  points: EventScatterPoint[],
  event: TradeEvent
): number {
  const target = points.find((p) => p.event === event)
  if (!target) return 1
  return points.filter((p) => p.timeSec === target.timeSec).length
}

export function markerClickThresholdSec(interval: '1h' | '4h'): number {
  return interval === '4h' ? 14400 : 3600
}

export function mapEventsToScatterPoints(
  events: TradeEvent[],
  priceRows: PriceChartRow[]
): EventScatterPoint[] {
  const filtered = events.filter((e) => e.time > 0)
  const bounds = dotRadiusBounds(filtered.length)

  const draft = filtered.map((e) => {
    const rawTimeSec = msToChartTime(e.time)
    const timeSec = snapTimeToNearestKline(rawTimeSec, priceRows)
    const type = e.type as TradeEventType
    const price = resolveEventPrice(e, priceRows)
    return {
      timeSec,
      rawTimeSec,
      timeLabel: formatChartTimeLabel(timeSec),
      price,
      amount: eventTradeAmount(e, price),
      dotRadius: bounds.minR,
      event: e,
      color: EVENT_TYPE_COLORS[type] || '#848E9C',
      label: EVENT_TYPE_LABELS[type] || e.type,
    }
  })

  const positiveAmounts = draft.map((p) => p.amount).filter((a) => a > 0)
  const minAmount = positiveAmounts.length ? Math.min(...positiveAmounts) : 0
  const maxAmount = positiveAmounts.length ? Math.max(...positiveAmounts) : 0

  return draft
    .map((p) => ({
      ...p,
      dotRadius: scaleDotRadiusByAmount(p.amount, minAmount, maxAmount, bounds.minR, bounds.maxR),
    }))
    .filter((p) => p.timeSec > 0 && p.price > 0)
}

export function findEventNearTime(
  events: TradeEvent[],
  chartTimeSec: number,
  thresholdSec = 3600
): TradeEvent | null {
  let best: TradeEvent | null = null
  let bestDiff = thresholdSec + 1
  for (const e of events) {
    const t = msToChartTime(e.time)
    const diff = Math.abs(t - chartTimeSec)
    if (diff < bestDiff) {
      bestDiff = diff
      best = e
    }
  }
  return bestDiff <= thresholdSec ? best : null
}

export function formatEventTime(ms: number): string {
  return new Date(ms < 1e12 ? ms * 1000 : ms).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
