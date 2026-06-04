import type { TradeEvent, TradeEventType, KlinePoint } from './tradeEventTypes'
import { EVENT_TYPE_COLORS, EVENT_TYPE_LABELS } from './tradeEventTypes'

export function msToChartTime(ms: number): number {
  return ms < 1e12 ? ms : Math.floor(ms / 1000)
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
  timeLabel: string
  price: number
  event: TradeEvent
  color: string
  label: string
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

export function mapEventsToScatterPoints(
  events: TradeEvent[],
  priceRows: PriceChartRow[]
): EventScatterPoint[] {
  return events
    .filter((e) => e.time > 0)
    .map((e) => {
      const timeSec = msToChartTime(e.time)
      const type = e.type as TradeEventType
      return {
        timeSec,
        timeLabel: formatChartTimeLabel(timeSec),
        price: resolveEventPrice(e, priceRows),
        event: e,
        color: EVENT_TYPE_COLORS[type] || '#848E9C',
        label: EVENT_TYPE_LABELS[type] || e.type,
      }
    })
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
