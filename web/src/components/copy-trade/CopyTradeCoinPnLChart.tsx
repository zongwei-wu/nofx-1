import { useMemo } from 'react'
import { useSymbolPreferences } from '../../contexts/SymbolPreferencesContext'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
  ReferenceLine,
} from 'recharts'
import type { CopyTradePnLRecord } from './CopyTradePnLList'

const COIN_COLORS = [
  '#F0B90B',
  '#0ECB81',
  '#6366F1',
  '#F6465D',
  '#38BDF8',
  '#A78BFA',
  '#FB923C',
  '#14B8A6',
]

const MAX_HOURS = 48

function coinFromSymbol(symbol?: string) {
  return symbol?.replace(/USDT$/i, '') || symbol || 'UNKNOWN'
}

function parseRecordTime(ts: number | string | undefined): Date | null {
  if (ts === undefined || ts === null || ts === '') return null
  const d = typeof ts === 'number' ? new Date(ts) : new Date(ts)
  return Number.isNaN(d.getTime()) ? null : d
}

function floorHour(d: Date): Date {
  const x = new Date(d)
  x.setMinutes(0, 0, 0)
  return x
}

function formatHourLabel(d: Date) {
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function buildHourlyCoinPnlSeries(records: CopyTradePnLRecord[]) {
  const relevant = records.filter(
    (r) => (r.status === 'OPEN' || r.status === 'CLOSED') && r.symbol
  )
  if (relevant.length === 0) {
    return { chartData: [] as Record<string, string | number>[], coins: [] as string[] }
  }

  const now = floorHour(new Date())
  let startHour = new Date(now)
  startHour.setHours(startHour.getHours() - MAX_HOURS + 1)

  for (const r of relevant) {
    const openAt = parseRecordTime(r.copy_time) ?? parseRecordTime(r.lead_order_time)
    const closeAt = r.status === 'CLOSED' ? parseRecordTime(r.close_time) : null
    for (const t of [openAt, closeAt]) {
      if (t) {
        const h = floorHour(t)
        if (h < startHour) startHour = h
      }
    }
  }

  const hours: Date[] = []
  for (let h = new Date(startHour); h <= now; h.setHours(h.getHours() + 1)) {
    hours.push(new Date(h))
  }

  const coins = [...new Set(relevant.map((r) => coinFromSymbol(r.symbol)))].sort()

  const chartData = hours.map((hourStart) => {
    const hourEnd = hourStart.getTime() + 3600_000 - 1
    const point: Record<string, string | number> = {
      hour: formatHourLabel(hourStart),
      hourTs: hourStart.getTime(),
    }

    for (const coin of coins) {
      let pnl = 0
      for (const r of relevant) {
        if (coinFromSymbol(r.symbol) !== coin) continue

        if (r.status === 'CLOSED') {
          const closedAt = parseRecordTime(r.close_time)
          if (closedAt && closedAt.getTime() <= hourEnd) {
            pnl += r.total_pnl || 0
          }
          continue
        }

        const openedAt =
          parseRecordTime(r.copy_time) ?? parseRecordTime(r.lead_order_time)
        if (openedAt && openedAt.getTime() <= hourEnd) {
          pnl += r.total_pnl || 0
        }
      }
      point[coin] = Math.round(pnl * 100) / 100
    }
    return point
  })

  return { chartData, coins }
}

function CoinTooltip({
  active,
  payload,
  label,
}: {
  active?: boolean
  payload?: { name: string; value: number; color: string }[]
  label?: string
}) {
  if (!active || !payload?.length) return null
  return (
    <div
      className="rounded-lg p-3 text-xs"
      style={{ background: '#1E2329', border: '1px solid #2B3139' }}
    >
      <div className="font-semibold mb-2" style={{ color: '#EAECEF' }}>
        {label}
      </div>
      {payload.map((entry) => (
        <div key={entry.name} className="flex justify-between gap-4 mb-1">
          <span style={{ color: entry.color }}>{entry.name}</span>
          <span
            className="font-mono font-semibold"
            style={{ color: entry.value >= 0 ? '#0ECB81' : '#F6465D' }}
          >
            {entry.value >= 0 ? '+' : ''}${Number(entry.value).toFixed(2)}
          </span>
        </div>
      ))}
    </div>
  )
}

interface CopyTradeCoinPnLChartProps {
  records: CopyTradePnLRecord[]
}

export function CopyTradeCoinPnLChart({ records }: CopyTradeCoinPnLChartProps) {
  const { sortSymbols } = useSymbolPreferences()
  const { chartData, coins } = useMemo(
    () => buildHourlyCoinPnlSeries(records),
    [records]
  )

  const sortedCoins = useMemo(() => {
    const withUsdt = coins.map((c) => (c.endsWith('USDT') ? c : `${c}USDT`))
    return sortSymbols(withUsdt).map((s) => s.replace(/USDT$/i, ''))
  }, [coins, sortSymbols])

  const latestByCoin = useMemo(() => {
    if (chartData.length === 0) return []
    const last = chartData[chartData.length - 1]
    return sortedCoins.map((coin) => ({
      coin,
      pnl: Number(last[coin] ?? 0),
    }))
  }, [chartData, sortedCoins])

  if (chartData.length === 0 || sortedCoins.length === 0) {
    return (
      <div className="text-center py-8 text-sm" style={{ color: '#5E6673' }}>
        暂无盈亏数据
      </div>
    )
  }

  const interval = Math.max(0, Math.floor(chartData.length / 8) - 1)

  return (
    <>
      <div style={{ width: '100%', height: 280 }}>
        <ResponsiveContainer>
          <LineChart data={chartData} margin={{ top: 8, right: 24, left: 8, bottom: 8 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#2B3139" />
            <XAxis
              dataKey="hour"
              tick={{ fontSize: 10, fill: '#848E9C' }}
              stroke="#474D57"
              interval={interval}
              angle={-20}
              textAnchor="end"
              height={56}
            />
            <YAxis
              tick={{ fontSize: 10, fill: '#848E9C' }}
              stroke="#474D57"
              tickFormatter={(v) => `$${v}`}
              width={56}
            />
            <Tooltip content={<CoinTooltip />} />
            <ReferenceLine y={0} stroke="#474D57" strokeDasharray="4 4" />
            <Legend
              iconType="line"
              wrapperStyle={{ fontSize: 12, color: '#848E9C' }}
            />
            {sortedCoins.map((coin, i) => (
              <Line
                key={coin}
                type="monotone"
                dataKey={coin}
                name={coin}
                stroke={COIN_COLORS[i % COIN_COLORS.length]}
                strokeWidth={2}
                dot={chartData.length <= 24 ? { r: 2 } : false}
                activeDot={{ r: 4 }}
                connectNulls
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-4">
        {latestByCoin.map((data, i) => (
          <div key={data.coin} className="p-3 rounded text-center" style={{ background: '#0B0E11' }}>
            <div className="flex items-center justify-center gap-1.5 text-xs" style={{ color: '#848E9C' }}>
              <span
                className="inline-block w-2 h-2 rounded-full"
                style={{ background: COIN_COLORS[i % COIN_COLORS.length] }}
              />
              {data.coin}
            </div>
            <div
              className="text-sm font-bold mt-1"
              style={{ color: data.pnl >= 0 ? '#0ECB81' : '#F6465D' }}
            >
              {data.pnl >= 0 ? '+' : ''}
              {data.pnl.toFixed(2)}
            </div>
          </div>
        ))}
      </div>
    </>
  )
}
