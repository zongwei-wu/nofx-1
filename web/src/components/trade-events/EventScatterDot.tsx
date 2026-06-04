import type { EventScatterPoint } from './tradeEventChartUtils'

type ScatterDotProps = {
  cx?: number
  cy?: number
  payload?: EventScatterPoint
  active?: boolean
  onSelect?: (event: EventScatterPoint['event'] | null, current: EventScatterPoint['event']) => void
  selected?: EventScatterPoint['event'] | null
}

export function EventScatterDot({
  cx,
  cy,
  payload,
  onSelect,
  selected,
}: ScatterDotProps) {
  if (cx == null || cy == null || !payload) return null
  const active = selected === payload.event
  const r = active ? payload.dotRadius + 2 : payload.dotRadius
  return (
    <circle
      cx={cx}
      cy={cy}
      r={r}
      fill={payload.color}
      stroke={active ? '#EAECEF' : '#1E2329'}
      strokeWidth={active ? 2 : 1}
      style={{ cursor: 'pointer', pointerEvents: 'all' }}
      onClick={(e) => {
        e.stopPropagation()
        onSelect?.(payload.event, selected ?? null)
      }}
    />
  )
}
