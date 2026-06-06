import { useEffect, useState, useCallback, useMemo } from 'react'
import { httpClient } from '../../lib/httpClient'
import type { TradeEvent, KlinePoint } from './tradeEventTypes'
import { EVENT_TYPE_LABELS, EVENT_TYPE_COLORS } from './tradeEventTypes'
import {
  formatEventTime,
  eventTradeAmount,
  normalizeTradingSymbol,
  symbolsMatch,
  pickDefaultChartSymbol,
  DEFAULT_CHART_SYMBOL,
} from './tradeEventChartUtils'
import { LightweightTradeChart } from './LightweightTradeChart'
import { DEFAULT_TV_CHART_HEIGHT, type ChartInterval } from './tradingViewUtils'
import { useSymbolPreferences } from '../../contexts/SymbolPreferencesContext'

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
  height = DEFAULT_TV_CHART_HEIGHT,
}: TradeEventPriceChartProps) {
  const [interval, setInterval] = useState<ChartInterval>('1h')
  const [symbol, setSymbol] = useState(DEFAULT_CHART_SYMBOL)
  const [symbols, setSymbols] = useState<string[]>(symbolsProp || [])
  const [events, setEvents] = useState<TradeEvent[]>([])
  const [klines, setKlines] = useState<KlinePoint[]>([])
  const [eventsLoading, setEventsLoading] = useState(false)
  const [klinesLoading, setKlinesLoading] = useState(false)
  const [error, setError] = useState('')
  const [selectedEvent, setSelectedEvent] = useState<TradeEvent | null>(null)
  const { sortSymbols } = useSymbolPreferences()

  const authHeaders = useCallback(() => {
    const token = localStorage.getItem('auth_token')
    return { Authorization: token ? `Bearer ${token}` : '' }
  }, [])

  const loadEvents = useCallback(async () => {
    setEventsLoading(true)
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
    } finally {
      setEventsLoading(false)
    }
  }, [source, traderId, portfolioId, authHeaders, symbolsProp])

  const loadKlines = useCallback(async () => {
    const sym = normalizeTradingSymbol(symbol)
    if (!sym) return
    setKlinesLoading(true)
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
      setKlinesLoading(false)
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

  const symEvents = useMemo(
    () => events.filter((ev) => !symbol || symbolsMatch(ev.symbol, symbol)),
    [events, symbol]
  )

  const sortedEvents = useMemo(
    () => [...symEvents].sort((a, b) => b.time - a.time).slice(0, 50),
    [symEvents]
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

  const selectedDetail = useMemo(() => {
    if (!selectedEvent) return null
    const displayPrice = selectedEvent.price > 0 ? selectedEvent.price : 0
    return {
      displayPrice,
      amount: eventTradeAmount(selectedEvent, displayPrice),
    }
  }, [selectedEvent])

  const chartLoading = eventsLoading || klinesLoading

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
          onChange={(e) => setInterval(e.target.value as ChartInterval)}
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
          disabled={chartLoading}
          className="text-xs px-3 py-1.5 rounded font-semibold"
          style={{ background: '#2B3139', color: '#F0B90B' }}
        >
          {chartLoading ? '刷新中…' : '刷新'}
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

      {symbol ? (
        <LightweightTradeChart
          klines={klines}
          events={symEvents}
          interval={interval}
          height={height}
          loading={klinesLoading}
          selectedEvent={selectedEvent}
          onSelectEvent={setSelectedEvent}
        />
      ) : (
        <div
          className="flex items-center justify-center text-sm rounded-lg"
          style={{
            height,
            color: '#5E6673',
            background: '#0B0E11',
            border: '1px solid #2B3139',
          }}
        >
          请选择币种
        </div>
      )}

      <p className="text-[10px] mt-2" style={{ color: '#5E6673' }}>
        点击 K 线上的圆点标记或下方列表查看开/加/减/平详情
      </p>

      <div
        className="mt-2 rounded-lg overflow-hidden"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        <div
          className="px-3 py-2 text-xs font-semibold"
          style={{ background: '#1E2329', color: '#848E9C', borderBottom: '1px solid #2B3139' }}
        >
          交易事件
          {symEvents.length > 0 && (
            <span className="ml-2" style={{ color: '#F0B90B' }}>
              {symEvents.length} 条
            </span>
          )}
        </div>

        {selectedEvent && selectedDetail && (
          <div
            className="px-3 py-3 text-xs border-b"
            style={{ borderColor: '#2B3139', background: '#1E2329' }}
          >
            <div className="flex flex-wrap items-center gap-2 mb-2">
              <span
                className="font-semibold text-sm"
                style={{
                  color:
                    EVENT_TYPE_COLORS[selectedEvent.type as keyof typeof EVENT_TYPE_COLORS] ||
                    '#848E9C',
                }}
              >
                {EVENT_TYPE_LABELS[selectedEvent.type as keyof typeof EVENT_TYPE_LABELS] ||
                  selectedEvent.type}
              </span>
              <span style={{ color: '#EAECEF' }}>
                {selectedEvent.symbol?.replace(/USDT$/i, '')}
              </span>
              <span style={{ color: '#848E9C' }}>
                {selectedEvent.side === 'SHORT' ? '空' : '多'}
              </span>
            </div>
            <div
              className="grid grid-cols-2 sm:grid-cols-3 gap-x-4 gap-y-1"
              style={{ color: '#848E9C' }}
            >
              <span>时间：{formatEventTime(selectedEvent.time)}</span>
              <span>数量：{selectedEvent.qty}</span>
              <span>
                价格：$
                {selectedDetail.displayPrice > 0
                  ? selectedDetail.displayPrice >= 100
                    ? selectedDetail.displayPrice.toFixed(2)
                    : selectedDetail.displayPrice.toFixed(4)
                  : '—'}
              </span>
              <span>
                金额：$
                {selectedDetail.amount > 0
                  ? selectedDetail.amount >= 100
                    ? selectedDetail.amount.toFixed(2)
                    : selectedDetail.amount.toFixed(4)
                  : '—'}
              </span>
              {selectedEvent.source && <span>来源：{selectedEvent.source}</span>}
            </div>
            {selectedEvent.detail && (
              <p className="pt-2" style={{ color: '#5E6673' }}>
                {selectedEvent.detail}
              </p>
            )}
          </div>
        )}

        <div className="max-h-[200px] overflow-y-auto">
          {sortedEvents.length === 0 ? (
            <p className="text-xs py-6 text-center" style={{ color: '#5E6673' }}>
              当前币种暂无已执行的交易记录
            </p>
          ) : (
            sortedEvents.map((ev, i) => {
              const color =
                EVENT_TYPE_COLORS[ev.type as keyof typeof EVENT_TYPE_COLORS] || '#848E9C'
              const label =
                EVENT_TYPE_LABELS[ev.type as keyof typeof EVENT_TYPE_LABELS] || ev.type
              const active = selectedEvent === ev
              const timeStr = formatEventTime(ev.time)
              return (
                <div
                  key={`${ev.time}-${ev.type}-${i}`}
                  className="flex items-center gap-2 px-3 py-1.5 text-xs cursor-pointer"
                  style={{
                    borderBottom:
                      i < sortedEvents.length - 1 ? '1px solid #2B3139' : 'none',
                    background: active ? 'rgba(240, 185, 11, 0.08)' : 'transparent',
                  }}
                  onClick={() => setSelectedEvent(active ? null : ev)}
                >
                  <span
                    className="w-2 h-2 rounded-full flex-shrink-0"
                    style={{ background: color }}
                  />
                  <span style={{ color: '#848E9C', minWidth: 120 }}>{timeStr}</span>
                  <span style={{ color, fontWeight: 600, minWidth: 32 }}>{label}</span>
                  <span style={{ color: '#848E9C', fontSize: 10 }}>
                    {ev.side === 'SHORT' ? '空' : '多'}
                  </span>
                  <span className="ml-auto" style={{ color: '#EAECEF' }}>
                    {ev.price > 0
                      ? `$${Number(ev.price).toFixed(ev.price >= 100 ? 2 : 4)}`
                      : '—'}
                    <span className="ml-2" style={{ color: '#5E6673' }}>
                      ×{ev.qty}
                    </span>
                  </span>
                </div>
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}
