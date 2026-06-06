import { useEffect, useRef, useMemo } from 'react'
import {
  createChart,
  ColorType,
  type IChartApi,
  type ISeriesApi,
  type Time,
} from 'lightweight-charts'
import type { KlinePoint, TradeEvent } from './tradeEventTypes'
import {
  mapKlinesToCandlestickData,
  getDefaultVisibleTimeRange,
  CHART_DEFAULT_VISIBLE_HOURS,
  mapKlinesToChartRows,
  mapEventsToScatterPoints,
  mapScatterPointsToClusteredMarkers,
  findClusterNearTime,
  markerClickThresholdSec,
  type LwcMarkerCluster,
} from './tradeEventChartUtils'
import type { ChartInterval } from './tradingViewUtils'
import { DEFAULT_TV_CHART_HEIGHT } from './tradingViewUtils'

export interface LightweightTradeChartProps {
  klines: KlinePoint[]
  events: TradeEvent[]
  interval: ChartInterval
  height?: number
  loading?: boolean
  visibleHours?: number
  selectedEvent: TradeEvent | null
  onSelectEvent: (event: TradeEvent | null) => void
}

export function LightweightTradeChart({
  klines,
  events,
  interval,
  height = DEFAULT_TV_CHART_HEIGHT,
  loading = false,
  visibleHours = CHART_DEFAULT_VISIBLE_HOURS,
  selectedEvent,
  onSelectEvent,
}: LightweightTradeChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const clustersRef = useRef<LwcMarkerCluster[]>([])
  const onSelectRef = useRef(onSelectEvent)
  const lastClusterClickRef = useRef({ time: 0, index: 0 })
  onSelectRef.current = onSelectEvent

  const candleData = useMemo(() => mapKlinesToCandlestickData(klines), [klines])
  const clusters = useMemo(() => {
    const priceRows = mapKlinesToChartRows(klines)
    const scatterPoints = mapEventsToScatterPoints(events, priceRows)
    return mapScatterPointsToClusteredMarkers(scatterPoints, selectedEvent)
  }, [klines, events, selectedEvent])
  clustersRef.current = clusters

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    let disposed = false
    let resizeObserver: ResizeObserver | null = null
    let clickHandler: ((param: { time?: Time }) => void) | null = null
    let rafId = 0

    const destroyChart = () => {
      if (chartRef.current && clickHandler) {
        chartRef.current.unsubscribeClick(clickHandler)
      }
      chartRef.current?.remove()
      chartRef.current = null
      seriesRef.current = null
    }

    const applyChartData = () => {
      const series = seriesRef.current
      const chart = chartRef.current
      if (!series || !chart || candleData.length === 0) return

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
        clusters.map((c) => ({
          time: c.time as Time,
          position: c.position,
          color: c.color,
          shape: c.shape,
          text: c.text,
        }))
      )
      const visibleRange = getDefaultVisibleTimeRange(
        candleData.map((c) => c.time),
        visibleHours
      )
      if (visibleRange) {
        chart.timeScale().setVisibleRange({
          from: visibleRange.from as Time,
          to: visibleRange.to as Time,
        })
      } else {
        chart.timeScale().fitContent()
      }
    }

    if (candleData.length === 0) {
      destroyChart()
      return () => {
        disposed = true
        cancelAnimationFrame(rafId)
      }
    }

    const mountChart = () => {
      if (disposed) return

      const width = container.clientWidth
      if (width <= 0) {
        rafId = requestAnimationFrame(mountChart)
        return
      }

      destroyChart()

      const chart = createChart(container, {
        width,
        height,
        layout: {
          background: { type: ColorType.Solid, color: '#0B0E11' },
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

      const threshold = markerClickThresholdSec(interval)
      clickHandler = (param: { time?: Time }) => {
        if (!param.time) {
          onSelectRef.current(null)
          return
        }
        const cluster = findClusterNearTime(
          clustersRef.current,
          Number(param.time),
          threshold
        )
        if (!cluster) {
          onSelectRef.current(null)
          return
        }
        if (cluster.events.length === 1) {
          lastClusterClickRef.current = { time: cluster.time, index: 0 }
          onSelectRef.current(cluster.events[0])
          return
        }
        const sorted = [...cluster.events].sort((a, b) => b.time - a.time)
        if (lastClusterClickRef.current.time === cluster.time) {
          lastClusterClickRef.current.index =
            (lastClusterClickRef.current.index + 1) % sorted.length
        } else {
          lastClusterClickRef.current = { time: cluster.time, index: 0 }
        }
        onSelectRef.current(sorted[lastClusterClickRef.current.index])
      }
      chart.subscribeClick(clickHandler)

      applyChartData()

      resizeObserver = new ResizeObserver(() => {
        if (disposed || !chartRef.current) return
        const nextWidth = container.clientWidth
        if (nextWidth > 0) {
          chartRef.current.applyOptions({ width: nextWidth })
        }
      })
      resizeObserver.observe(container)
    }

    mountChart()

    return () => {
      disposed = true
      cancelAnimationFrame(rafId)
      resizeObserver?.disconnect()
      destroyChart()
    }
  }, [candleData, clusters, height, interval, visibleHours])

  const showEmpty = !loading && candleData.length === 0

  return (
    <div
      className="relative rounded-lg overflow-hidden"
      style={{
        height,
        width: '100%',
        border: '1px solid #2B3139',
        background: '#0B0E11',
      }}
    >
      <div ref={containerRef} style={{ height: '100%', width: '100%' }} />
      {loading && (
        <div
          className="absolute inset-0 flex items-center justify-center text-sm"
          style={{ color: '#848E9C', background: 'rgba(11, 14, 17, 0.85)' }}
        >
          加载中…
        </div>
      )}
      {showEmpty && (
        <div
          className="absolute inset-0 flex items-center justify-center text-sm"
          style={{ color: '#5E6673', background: '#0B0E11' }}
        >
          暂无 K 线数据
        </div>
      )}
    </div>
  )
}
