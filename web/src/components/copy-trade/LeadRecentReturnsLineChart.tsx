import { useMemo } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts'

export type LeadChartItem = {
  value: number
  dataType: string
  dateTime: number
}

function formatDateLabel(ts: number) {
  const d = new Date(ts)
  return `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function RecentReturnsTooltip({
  active,
  payload,
}: {
  active?: boolean
  payload?: Array<{ payload?: { dateLabel: string; value: number } }>
}) {
  if (!active || !payload?.length) return null
  const row = payload[0].payload
  if (!row) return null
  const color = row.value >= 0 ? '#0ECB81' : '#F6465D'
  return (
    <div
      className="rounded px-2 py-1 text-xs"
      style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
    >
      <div style={{ color: '#848E9C' }}>{row.dateLabel}</div>
      <div style={{ color, fontWeight: 600 }}>
        {row.value >= 0 ? '+' : ''}
        {row.value.toFixed(2)}%
      </div>
    </div>
  )
}

export function LeadRecentReturnsLineChart({
  chartItems,
  maxPoints = 20,
}: {
  chartItems: LeadChartItem[]
  maxPoints?: number
}) {
  const { chartData, strokeColor } = useMemo(() => {
    const sorted = [...chartItems]
      .sort((a, b) => a.dateTime - b.dateTime)
      .slice(-maxPoints)
    const data = sorted.map((item) => ({
      dateLabel: formatDateLabel(item.dateTime),
      dateTime: item.dateTime,
      value: item.value,
    }))
    const last = data[data.length - 1]?.value ?? 0
    return {
      chartData: data,
      strokeColor: last >= 0 ? '#0ECB81' : '#F6465D',
    }
  }, [chartItems, maxPoints])

  if (chartData.length === 0) return null

  return (
    <div className="w-full" style={{ height: 100 }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#2B3139" vertical={false} />
          <XAxis
            dataKey="dateLabel"
            stroke="#5E6673"
            tick={{ fill: '#848E9C', fontSize: 10 }}
            tickLine={false}
            axisLine={{ stroke: '#2B3139' }}
            interval="preserveStartEnd"
          />
          <YAxis
            stroke="#5E6673"
            tick={{ fill: '#848E9C', fontSize: 10 }}
            tickLine={false}
            axisLine={false}
            width={40}
            tickFormatter={(v) => `${Number(v).toFixed(0)}%`}
          />
          <Tooltip content={<RecentReturnsTooltip />} />
          <ReferenceLine y={0} stroke="#474D57" strokeDasharray="3 3" />
          <Line
            type="monotone"
            dataKey="value"
            stroke={strokeColor}
            strokeWidth={2}
            dot={chartData.length > 12 ? false : { r: 2, fill: strokeColor }}
            activeDot={{ r: 4, fill: strokeColor, stroke: '#0B0E11', strokeWidth: 2 }}
            connectNulls
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
