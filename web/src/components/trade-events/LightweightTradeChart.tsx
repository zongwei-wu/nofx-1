import { useEffect, useRef } from 'react'
import { createChart, type IChartApi, type ISeriesApi, type Time } from 'lightweight-charts'
import type { KlinePoint, TradeEvent } from './tradeEventTypes'
import {
  mapKlinesToCandlestickData,
  mapKlinesToChartRows,
  mapEventsToScatterPoints,
  mapScatterPointsToMarkers,
  findEventNearTime,
  markerClickThresholdSec,
} from './tradeEventChartUtils'
import type { ChartInterval } from './tradingViewUtils'
import { DEFAULT_TV_CHART_HEIGHT } from './tradingViewUtils'

export interface LightweightTradeChartProps {
  klines: KlinePoint[]
  events: TradeEvent[]
  interval: ChartInterval
  height?: number
  loading?: boolean
  selectedEvent: TradeEvent | null
  onSelectEvent: (event: TradeEvent | null) => void
}

export function LightweightTradeChart({
  klines,
  events,
  interval,
  height = DEFAULT_TV_CHART_HEIGHT,
  loading = false,
  selectedEvent,
  onSelectEvent,
}: LightweightTradeChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const eventsRef = useRef(events)
  eventsRef.current = events

  const priceRows = mapKlinesToChartRows(klines)
  const scatterPoints = mapEventsToScatterPoints(events, priceRows)
  const candleData = mapKlinesToCandlestickData(klines)
  const markers = mapScatterPointsToMarkers(scatterPoints, selectedEvent)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const chart = createChart(container, {
      width: container.clientWidth,
      height,
      layout: {
        background: { color: '#0B0E11' },
        textColor: '#848E9C',
      },
      grid: {
        vertLines: { color: '#2B3139' },
        horzLines: { color: '#2B3139' },
      },
      rightPriceScale: {
        borderColor: '#2B3139',
      },
      timeScale: {
        borderColor: '#2B3139',
        timeVisible: true,
        secondsVisible: false,
      },
      crosshair: {
        vertLine: { color: '#5E6673' },
        horzLine: { color: '#5E6673' },
      },
    })

    const series = chart.addCandlestickSeries({
      upColor: '#0ECB81',
      downColor: '#F6465D',
      borderUpColor: '#0ECB81',
      borderDownColor: '#F6465D',
      wickUpColor: '#0ECB81',
      wickDownColor: '#F6465D',
    })

    chartRef.current = chart
    seriesRef.current = series

    const ro = new ResizeObserver(() => {
      if (container.clientWidth > 0) {
        chart.applyOptions({ width: container.clientWidth })
      }
    })
    ro.observe(container)

    const threshold = markerClickThresholdSec(interval)
    const clickHandler = (param: { time?: Time }) => {
      if (!param.time) {
        onSelectEvent(null)
        return
      }
      const timeSec = Number(param.time)
      const matched = findEventNearTime(eventsRef.current, timeSec, threshold)
      onSelectEvent(matched)
    }
    chart.subscribeClick(clickHandler)

    return () => {
      ro.disconnect()
      chart.unsubscribeClick(clickHandler)
      chart.remove()
      chartRef.current = null
      seriesRef.current = null
    }
  }, [height, interval, onSelectEvent])

  useEffect(() => {
    const series = seriesRef.current
    if (!series) return

    if (candleData.length === 0) {
      series.setData([])
      series.setMarkers([])
      return
    }

    series.setData(
      candleData.map((c) => ({
        time: c.time as Time,
        open: c.open,
        high: c.high,
        low: c.low,
        close: c.close,
      }))
    )

    series.setMarkers(
      markers.map((m) => ({
        time: m.time as Time,
        position: m.position,
        color: m.color,
        shape: m.shape,
        text: m.text,
      }))
    )

    chartRef.current?.timeScale().fitContent()
  }, [candleData, markers])

  if (loading) {
    return (
      <div
        className="flex items-center justify-center text-sm rounded-lg"
        style={{ height, color: '#5E6673', background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        加载中…
      </div>
    )
  }

  if (candleData.length === 0) {
    return (
      <div
        className="flex items-center justify-center text-sm rounded-lg"
        style={{ height, color: '#5E6673', background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        暂无 K 线数据
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      className="rounded-lg overflow-hidden"
      style={{
        height,
        width: '100%',
        border: '1px solid #2B3139',
        background: '#0B0E11',
      }}
    />
  )
}
