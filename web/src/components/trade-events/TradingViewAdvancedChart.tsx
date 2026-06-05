import { memo, useEffect, useRef } from 'react'
import {
  toTradingViewInterval,
  toTradingViewSymbol,
  DEFAULT_TV_CHART_HEIGHT,
  TV_COPYRIGHT_HEIGHT,
  type ChartInterval,
} from './tradingViewUtils'

const WIDGET_SCRIPT =
  'https://s3.tradingview.com/external-embedding/embed-widget-advanced-chart.js'

export interface TradingViewAdvancedChartProps {
  symbol: string
  interval: ChartInterval
  height?: number
}

function enforceChartHeight(root: HTMLElement, chartHeight: number) {
  const widget = root.querySelector('.tradingview-widget-container__widget')
  const iframe = root.querySelector('iframe')
  if (widget instanceof HTMLElement) {
    widget.style.height = `${chartHeight}px`
    widget.style.minHeight = `${chartHeight}px`
  }
  if (iframe instanceof HTMLElement) {
    iframe.style.height = `${chartHeight}px`
    iframe.style.minHeight = `${chartHeight}px`
    iframe.style.width = '100%'
  }
}

function TradingViewAdvancedChartInner({
  symbol,
  interval,
  height = DEFAULT_TV_CHART_HEIGHT,
}: TradingViewAdvancedChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const totalHeight = height + TV_COPYRIGHT_HEIGHT

  useEffect(() => {
    const container = containerRef.current
    if (!container || !symbol) return

    container.innerHTML = ''

    const widgetHost = document.createElement('div')
    widgetHost.className = 'tradingview-widget-container__widget'
    widgetHost.style.height = `${height}px`
    widgetHost.style.minHeight = `${height}px`
    widgetHost.style.width = '100%'

    const copyright = document.createElement('div')
    copyright.className = 'tradingview-widget-copyright'
    copyright.style.height = `${TV_COPYRIGHT_HEIGHT}px`
    copyright.style.lineHeight = `${TV_COPYRIGHT_HEIGHT}px`
    copyright.style.fontSize = '11px'
    copyright.innerHTML =
      '<a href="https://www.tradingview.com/" rel="noopener nofollow" target="_blank">' +
      '<span style="color:#848E9C">TradingView</span></a>'

    const script = document.createElement('script')
    script.type = 'text/javascript'
    script.async = true
    script.src = WIDGET_SCRIPT
    script.textContent = JSON.stringify({
      autosize: false,
      width: '100%',
      height,
      symbol: toTradingViewSymbol(symbol),
      interval: toTradingViewInterval(interval),
      timezone: 'Asia/Shanghai',
      theme: 'dark',
      style: '1',
      locale: 'zh_CN',
      backgroundColor: '#0B0E11',
      gridColor: '#2B3139',
      hide_top_toolbar: false,
      hide_legend: false,
      allow_symbol_change: false,
      save_image: false,
      calendar: false,
      support_host: 'https://www.tradingview.com',
    })

    container.appendChild(widgetHost)
    container.appendChild(copyright)
    container.appendChild(script)

    const applyHeight = () => enforceChartHeight(container, height)
    script.onload = () => {
      applyHeight()
      window.setTimeout(applyHeight, 300)
      window.setTimeout(applyHeight, 1200)
    }

    const observer = new MutationObserver(applyHeight)
    observer.observe(container, { childList: true, subtree: true, attributes: true })

    return () => {
      observer.disconnect()
      container.innerHTML = ''
    }
  }, [symbol, interval, height])

  if (!symbol) {
    return (
      <div
        className="flex items-center justify-center text-sm rounded-lg"
        style={{ height: totalHeight, color: '#5E6673', background: '#0B0E11' }}
      >
        请选择币种
      </div>
    )
  }

  return (
    <div
      style={{
        height: totalHeight,
        minHeight: totalHeight,
        width: '100%',
      }}
    >
      <div
        ref={containerRef}
        className="tradingview-widget-container rounded-lg"
        style={{
          height: totalHeight,
          minHeight: totalHeight,
          width: '100%',
          background: '#0B0E11',
          border: '1px solid #2B3139',
          overflow: 'hidden',
        }}
      />
    </div>
  )
}

export const TradingViewAdvancedChart = memo(TradingViewAdvancedChartInner)
