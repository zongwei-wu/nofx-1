import { useEffect, useState, useCallback, useMemo } from 'react'
import {
  ComposedChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { httpClient } from '../../lib/httpClient'
import type { TradeEvent, KlinePoint } from './tradeEventTypes'
import { EVENT_TYPE_LABELS, EVENT_TYPE_COLORS } from './tradeEventTypes'
import {
  mapKlinesToChartRows,
  mapEventsToScatterPoints,
  formatEventTime,
  eventTradeAmount,
  normalizeTradingSymbol,
  symbolsMatch,
  pickDefaultChartSymbol,
  DEFAULT_CHART_SYMBOL,
} from './tradeEventChartUtils'
import { useSymbolPreferences } from '../../contexts/SymbolPreferencesContext'
import { EventReferenceDot } from './EventScatterDot'

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
  const [interval, setInterval] = useState<'1h' | '4h'>('1h')
  const [symbol, setSymbol] = useState(DEFAULT_CHART_SYMBOL)
  const [symbols, setSymbols] = useState<string[]>(symbolsProp || [])
  const [events, setEvents] = useState<TradeEvent[]>([])
  const [klines, setKlines] = useState<KlinePoint[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [selectedEvent, setSelectedEvent] = useState<TradeEvent | null>(null)
  const { sortSymbols } = useSymbolPreferences()

  const authHeaders = useCallback(() => {
    const token = localStorage.getItem('auth_token')
    return { Authorization: token ? `Bearer ${token}` : '' }
  }, [])

  const loadEvents = useCallback(async () => {
    setError('')
    try {
      const params = new URLSearchParams({ source })
      if (traderId) params.set('trader_id', traderId)
      if (portfolioId) params.set('portfolio_id', portfolioId)

      const res = await httpClient.get(`/api/trade-events?${params}`, authHeaders())
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || '加载事件失败')
      }
      const data = await res.json()
      const list: TradeEvent[] = (data.events || []).map((e: TradeEvent) => ({
        ...e,
        symbol: normalizeTradingSymbol(e.symbol),
      }))
      setEvents(list)
      const syms: string[] = (data.symbols || []).map((s: string) => normalizeTradingSymbol(s))
      const preferred = [
        ...(symbolsProp || []).map(normalizeTradingSymbol),
        ...syms,
      ]
      setSymbols((prev) => {
        const merged = new Set([...preferred, ...prev, ...list.map((e) => e.symbol)])
        return [...merged].filter(Boolean)
      })
      setSymbol((prev) =>
        pickDefaultChartSymbol(
          [...syms, ...list.map((e) => e.symbol)],
          preferred,
          prev
        )
      )
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载失败')
      setEvents([])
    }
  }, [source, traderId, portfolioId, authHeaders, symbolsProp])

  const loadKlines = useCallback(async () => {
    const sym = normalizeTradingSymbol(symbol)
    if (!sym) return
    setLoading(true)
    setError('')
    try {
      const params = new URLSearchParams({ symbol: sym, interval, limit: '500' })
      const kRes = await httpClient.get(`/api/market/klines?${params}`)
      if (!kRes.ok) throw new Error('K线加载失败')
      const kData = await kRes.json()
      setKlines(kData.klines || [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'K线加载失败')
      setKlines([])
    } finally {
      setLoading(false)
    }
  }, [symbol, interval])

  useEffect(() => {
    loadEvents()
  }, [loadEvents])

  useEffect(() => {
    loadKlines()
  }, [loadKlines])

  useEffect(() => {
    if (!symbolsProp?.length) return
    const normalized = symbolsProp.map(normalizeTradingSymbol).filter(Boolean)
    setSymbols((prev) => [...new Set([...normalized, ...prev])])
    setSymbol((prev) =>
      pickDefaultChartSymbol(
        [...normalized, ...events.map((e) => e.symbol)],
        normalized,
        prev
      )
    )
  }, [symbolsProp, events])

  useEffect(() => {
    setSelectedEvent(null)
  }, [symbol, interval])

  const priceRows = useMemo(() => mapKlinesToChartRows(klines), [klines])

  const symEvents = useMemo(
    () => events.filter((ev) => !symbol || symbolsMatch(ev.symbol, symbol)),
    [events, symbol]
  )

  const scatterPoints = useMemo(
    () => mapEventsToScatterPoints(symEvents, priceRows),
    [symEvents, priceRows]
  )

  const symbolOptions = useMemo(() => {
    const merged = new Set<string>([DEFAULT_CHART_SYMBOL])
    for (const s of symbols) {
      const n = normalizeTradingSymbol(s)
      if (n) merged.add(n)
    }
    for (const e of events) {
      if (e.symbol) merged.add(e.symbol)
    }
    return sortSymbols([...merged])
  }, [symbols, events, sortSymbols])

  const yDomain = useMemo((): [number, number] | undefined => {
    const prices = [
      ...priceRows.map((r) => r.close),
      ...scatterPoints.map((p) => p.price),
    ].filter((p) => p > 0)
    if (prices.length === 0) return undefined
    const min = Math.min(...prices)
    const max = Math.max(...prices)
    const pad = (max - min) * 0.05 || max * 0.01
    return [min - pad, max + pad]
  }, [priceRows, scatterPoints])

  const selectedDetail = useMemo(() => {
    if (!selectedEvent) return null
    const chartPrice = scatterPoints.find((p) => p.event === selectedEvent)?.price ?? 0
    const displayPrice = selectedEvent.price > 0 ? selectedEvent.price : chartPrice
    return {
      displayPrice,
      amount: eventTradeAmount(selectedEvent, chartPrice),
    }
  }, [selectedEvent, scatterPoints])

  const handleSelectEvent = useCallback(
    (ev: TradeEvent | null, current: TradeEvent | null) => {
      setSelectedEvent(current === ev ? null : ev)
    },
    []
  )

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3 mb-3">
        <select
          value={symbol}
          onChange={(e) => setSymbol(normalizeTradingSymbol(e.target.value))}
          className="text-sm px-3 py-1.5 rounded"
          style={{ background: '#0B0E11', color: '#EAECEF', border: '1px solid #2B3139' }}
        >
          <option value="">选择币种</option>
          {symbolOptions.map((s) => (
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
          onClick={() => {
            loadEvents()
            loadKlines()
          }}
          className="text-xs px-3 py-1.5 rounded font-semibold"
          style={{ background: '#2B3139', color: '#F0B90B' }}
        >
          刷新
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

      {symbol && priceRows.length > 0 ? (
        <>
          <ResponsiveContainer width="100%" height={height}>
            <ComposedChart margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#2B3139" />
              <XAxis
                dataKey="timeSec"
                type="number"
                domain={['dataMin', 'dataMax']}
                tick={{ fill: '#848E9C', fontSize: 10 }}
                tickFormatter={(v: number) =>
                  new Date(v * 1000).toLocaleString('zh-CN', {
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                  })
                }
              />
              <YAxis
                domain={yDomain ?? ['auto', 'auto']}
                tick={{ fill: '#848E9C', fontSize: 10 }}
                width={72}
                tickFormatter={(v: number) => `$${Number(v).toLocaleString()}`}
              />
              <Tooltip
                contentStyle={{
                  background: '#1E2329',
                  border: '1px solid #2B3139',
                  borderRadius: 8,
                  fontSize: 12,
                }}
                labelStyle={{ color: '#848E9C' }}
                formatter={(value: number | string) => [
                  `$${Number(value).toFixed(2)}`,
                  '价格',
                ]}
                labelFormatter={(label) => formatEventTime(Number(label) * 1000)}
              />
              <Line
                data={priceRows}
                type="monotone"
                dataKey="close"
                stroke="#F0B90B"
                strokeWidth={2}
                dot={false}
                isAnimationActive={false}
              />
              {scatterPoints.map((p, i) => (
                <EventReferenceDot
                  key={`${p.event.time}-${p.event.type}-${i}`}
                  point={p}
                  selected={selectedEvent}
                  onSelect={handleSelectEvent}
                />
              ))}
            </ComposedChart>
          </ResponsiveContainer>
          {scatterPoints.length === 0 && (
            <p className="text-xs mt-2 text-center" style={{ color: '#5E6673' }}>
              当前币种暂无已执行的开仓/加仓/减仓/平仓记录（仅展示成功成交的 AI 决策）
            </p>
          )}
        </>
      ) : (
        <div
          className="flex items-center justify-center text-sm"
          style={{ height, color: '#5E6673', background: '#0B0E11', borderRadius: 8 }}
        >
          {loading ? '加载中…' : symbol ? '暂无K线数据' : '请选择币种'}
        </div>
      )}

      <div
        className="mt-3 p-3 rounded-lg min-h-[72px] text-xs"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        {selectedEvent && selectedDetail ? (
          <>
            <div
              className="font-semibold mb-1"
              style={{
                color: EVENT_TYPE_COLORS[selectedEvent.type as keyof typeof EVENT_TYPE_COLORS],
              }}
            >
              {EVENT_TYPE_LABELS[selectedEvent.type as keyof typeof EVENT_TYPE_LABELS]} ·{' '}
              {selectedEvent.symbol} · {selectedEvent.side}
            </div>
            <div style={{ color: '#848E9C' }}>{formatEventTime(selectedEvent.time)}</div>
            <div style={{ color: '#EAECEF' }}>
              数量 {selectedEvent.qty.toFixed(4)} · 价格 $
              {selectedDetail.displayPrice.toFixed(2)} · 金额 $
              {selectedDetail.amount.toLocaleString('zh-CN', { maximumFractionDigits: 2 })} ·{' '}
              {selectedEvent.source}
            </div>
            {selectedEvent.detail && (
              <div className="mt-1" style={{ color: '#5E6673' }}>
                {selectedEvent.detail}
              </div>
            )}
          </>
        ) : (
          <span style={{ color: '#5E6673' }}>点击 K 线上的圆点查看交易事件详情</span>
        )}
      </div>
    </div>
  )
}
