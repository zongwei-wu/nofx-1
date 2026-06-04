import { useEffect, useRef, useState, useCallback } from 'react'
import {
  createChart,
  type IChartApi,
  type ISeriesApi,
  type MouseEventParams,
  type Time,
} from 'lightweight-charts'
import { httpClient } from '../../lib/httpClient'
import type { TradeEvent, KlinePoint } from './tradeEventTypes'
import { EVENT_TYPE_LABELS, EVENT_TYPE_COLORS } from './tradeEventTypes'
import {
  mapKlinesToLineData,
  mapEventsToMarkers,
  findEventNearTime,
  formatEventTime,
} from './tradeEventChartUtils'

export interface TradeEventPriceChartProps {
  source: 'copy_trade' | 'ai_trader'
  traderId?: string
  portfolioId?: string
  symbols?: string[]
  height?: number
}

export function TradeEventPriceChart({
  source,
  traderId,
  portfolioId,
  symbols: symbolsProp,
  height = 360,
}: TradeEventPriceChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null)

  const [interval, setInterval] = useState<'1h' | '4h'>('1h')
  const [symbol, setSymbol] = useState('')
  const [symbols, setSymbols] = useState<string[]>(symbolsProp || [])
  const [events, setEvents] = useState<TradeEvent[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [hoverEvent, setHoverEvent] = useState<TradeEvent | null>(null)

  const authHeaders = useCallback(() => {
    const token = localStorage.getItem('auth_token')
    return { Authorization: token ? `Bearer ${token}` : '' }
  }, [])

  const loadEvents = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const params = new URLSearchParams({ source })
      if (traderId) params.set('trader_id', traderId)
      if (portfolioId) params.set('portfolio_id', portfolioId)
      if (symbol) params.set('symbol', symbol)

      const res = await httpClient.get(`/api/trade-events?${params}`, authHeaders())
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || '加载事件失败')
      }
      const data = await res.json()
      const list: TradeEvent[] = data.events || []
      setEvents(list)
      const syms: string[] = data.symbols || []
      if (!symbol && syms.length > 0) {
        setSymbol(syms[0])
      }
      if (!symbolsProp?.length && syms.length > 0) {
        setSymbols(syms)
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载失败')
      setEvents([])
    } finally {
      setLoading(false)
    }
  }, [source, traderId, portfolioId, symbol, authHeaders, symbolsProp])

  useEffect(() => {
    loadEvents()
  }, [loadEvents])

  useEffect(() => {
    if (symbolsProp?.length) {
      setSymbols(symbolsProp)
      if (!symbol && symbolsProp[0]) {
        setSymbol(symbolsProp[0])
      }
    }
  }, [symbolsProp, symbol])

  useEffect(() => {
    if (!containerRef.current) return

    const chart = createChart(containerRef.current, {
      width: containerRef.current.clientWidth,
      height,
      layout: {
        background: { color: '#1E2329' },
        textColor: '#848E9C',
      },
      grid: {
        vertLines: { color: '#2B3139' },
        horzLines: { color: '#2B3139' },
      },
      crosshair: { mode: 1 },
      timeScale: {
        borderColor: '#2B3139',
        timeVisible: true,
        secondsVisible: false,
      },
      rightPriceScale: { borderColor: '#2B3139' },
    })
    const series = chart.addLineSeries({
      color: '#F0B90B',
      lineWidth: 2,
    })
    chartRef.current = chart
    seriesRef.current = series

    const onCrosshair = (param: MouseEventParams<Time>) => {
      if (param.time === undefined) {
        setHoverEvent(null)
        return
      }
      const t =
        typeof param.time === 'number'
          ? param.time
          : typeof param.time === 'string'
            ? Math.floor(new Date(param.time).getTime() / 1000)
            : 0
      const symEvents = events.filter((ev: TradeEvent) => !symbol || ev.symbol === symbol)
      setHoverEvent(findEventNearTime(symEvents, t))
    }
    chart.subscribeCrosshairMove(onCrosshair)

    const ro = new ResizeObserver(() => {
      if (containerRef.current) {
        chart.applyOptions({ width: containerRef.current.clientWidth })
      }
    })
    ro.observe(containerRef.current)

    return () => {
      ro.disconnect()
      chart.remove()
      chartRef.current = null
      seriesRef.current = null
    }
  }, [height, events, symbol])

  useEffect(() => {
    const chart = chartRef.current
    const series = seriesRef.current
    if (!chart || !series || !symbol) return

    let cancelled = false
    ;(async () => {
      setLoading(true)
      try {
        const params = new URLSearchParams({
          symbol,
          interval,
          limit: '200',
        })
        const kRes = await httpClient.get(`/api/market/klines?${params}`)
        if (!kRes.ok) throw new Error('K线加载失败')
        const kData = await kRes.json()
        const klines: KlinePoint[] = kData.klines || []
        if (cancelled) return

        const lineData = mapKlinesToLineData(klines)
        series.setData(lineData)

        const symEvents = events.filter((ev: TradeEvent) => ev.symbol === symbol)
        series.setMarkers(mapEventsToMarkers(symEvents))
        chart.timeScale().fitContent()
      } catch {
        if (!cancelled) series.setData([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [symbol, interval, events])

  const displayEvent = hoverEvent

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3 mb-3">
        <select
          value={symbol}
          onChange={(e) => setSymbol(e.target.value)}
          className="text-sm px-3 py-1.5 rounded"
          style={{ background: '#0B0E11', color: '#EAECEF', border: '1px solid #2B3139' }}
        >
          <option value="">选择币种</option>
          {(symbols.length > 0 ? symbols : [...new Set(events.map((ev: TradeEvent) => ev.symbol))]).map((s: string) => (
            <option key={s} value={s}>
              {s.replace(/USDT$/i, '')}
            </option>
          ))}
        </select>
        <select
          value={interval}
          onChange={(e) => setInterval(e.target.value as '1h' | '4h')}
          className="text-sm px-3 py-1.5 rounded"
          style={{ background: '#0B0E11', color: '#EAECEF', border: '1px solid #2B3139' }}
        >
          <option value="1h">1小时</option>
          <option value="4h">4小时</option>
        </select>
        <button
          type="button"
          onClick={() => loadEvents()}
          className="text-xs px-3 py-1.5 rounded font-semibold"
          style={{ background: '#2B3139', color: '#F0B90B' }}
        >
          刷新事件
        </button>
        <div className="flex gap-2 text-[10px]" style={{ color: '#848E9C' }}>
          {(['open', 'add', 'reduce', 'close'] as const).map((t) => (
            <span key={t} className="flex items-center gap-1">
              <span
                className="w-2 h-2 rounded-full inline-block"
                style={{ background: EVENT_TYPE_COLORS[t] }}
              />
              {EVENT_TYPE_LABELS[t]}
            </span>
          ))}
        </div>
      </div>

      {error && (
        <p className="text-xs mb-2" style={{ color: '#F6465D' }}>
          {error}
        </p>
      )}

      <div ref={containerRef} style={{ width: '100%', height }} />

      {loading && (
        <p className="text-xs mt-2 text-center" style={{ color: '#848E9C' }}>
          加载中…
        </p>
      )}

      <div
        className="mt-3 p-3 rounded-lg min-h-[72px] text-xs"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        {displayEvent ? (
          <>
            <div
              className="font-semibold mb-1"
              style={{ color: EVENT_TYPE_COLORS[displayEvent.type as keyof typeof EVENT_TYPE_COLORS] }}
            >
              {EVENT_TYPE_LABELS[displayEvent.type as keyof typeof EVENT_TYPE_LABELS]} ·{' '}
              {displayEvent.symbol} · {displayEvent.side}
            </div>
            <div style={{ color: '#848E9C' }}>{formatEventTime(displayEvent.time)}</div>
            <div style={{ color: '#EAECEF' }}>
              数量 {displayEvent.qty.toFixed(4)} · 价格 ${displayEvent.price.toFixed(2)} ·{' '}
              {displayEvent.source}
            </div>
            {displayEvent.detail && (
              <div className="mt-1" style={{ color: '#5E6673' }}>
                {displayEvent.detail}
              </div>
            )}
          </>
        ) : (
          <span style={{ color: '#5E6673' }}>移动十字线到事件点附近查看详情</span>
        )}
      </div>
    </div>
  )
}
