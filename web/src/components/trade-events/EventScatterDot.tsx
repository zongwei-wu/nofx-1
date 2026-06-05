import { ReferenceDot } from 'recharts'
import type { EventScatterPoint } from './tradeEventChartUtils'
import type { TradeEvent } from './tradeEventTypes'

type EventReferenceDotProps = {
  point: EventScatterPoint
  selected: TradeEvent | null
  onSelect: (ev: TradeEvent | null, current: TradeEvent | null) => void
}

export function EventReferenceDot({ point, selected, onSelect }: EventReferenceDotProps) {
  const active = selected === point.event
  const r = active ? point.dotRadius + 2 : point.dotRadius
  return (
    <ReferenceDot
      x={point.timeSec}
      y={point.price}
      r={r}
      fill={point.color}
      stroke={active ? '#EAECEF' : '#1E2329'}
      strokeWidth={active ? 2 : 1}
      ifOverflow="visible"
      isFront
      shape={(props: { cx?: number; cy?: number }) => {
        if (props.cx == null || props.cy == null) return <g />
        return (
          <circle
            cx={props.cx}
            cy={props.cy}
            r={r}
            fill={point.color}
            stroke={active ? '#EAECEF' : '#1E2329'}
            strokeWidth={active ? 2 : 1}
            style={{ cursor: 'pointer', pointerEvents: 'all' }}
            onClick={(e) => {
              e.stopPropagation()
              onSelect(point.event, selected)
            }}
          />
        )
      }}
    />
  )
}
