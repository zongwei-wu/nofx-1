import type { TradeEvent, TradeEventType, KlinePoint } from './tradeEventTypes'
import { EVENT_TYPE_COLORS, EVENT_TYPE_LABELS } from './tradeEventTypes'

export function msToChartTime(ms: number): number {
  return Math.floor(ms / 1000)
}

export function mapKlinesToLineData(klines: KlinePoint[]): { time: number; value: number }[] {
  return klines
    .filter((k) => k.time > 0 && k.close > 0)
    .map((k) => ({
      time: k.time,
      value: k.close,
    }))
    .sort((a, b) => a.time - b.time)
}

export function mapEventsToMarkers(events: TradeEvent[]) {
  return events
    .filter((e) => e.time > 0)
    .map((e) => {
      const t = msToChartTime(e.time)
      const color = EVENT_TYPE_COLORS[e.type as TradeEventType] || '#848E9C'
      const label = EVENT_TYPE_LABELS[e.type as TradeEventType] || e.type
      return {
        time: t as import('lightweight-charts').UTCTimestamp,
        position: (e.type === 'close' || e.type === 'reduce' ? 'aboveBar' : 'belowBar') as
          | 'aboveBar'
          | 'belowBar',
        color,
        shape: 'circle' as const,
        text: label,
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
  return new Date(ms).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
