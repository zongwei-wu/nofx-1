import { memo, useEffect, useRef } from 'react'
import {
  toTradingViewInterval,
  toTradingViewSymbol,
  type ChartInterval,
} from './tradingViewUtils'

const WIDGET_SCRIPT =
  'https://s3.tradingview.com/external-embedding/embed-widget-advanced-chart.js'

export interface TradingViewAdvancedChartProps {
  symbol: string
  interval: ChartInterval
  height?: number
}

function TradingViewAdvancedChartInner({
  symbol,
  interval,
  height = 360,
}: TradingViewAdvancedChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = containerRef.current
    if (!container || !symbol) return

    container.innerHTML = ''

    const widgetHost = document.createElement('div')
    widgetHost.className = 'tradingview-widget-container__widget'
    widgetHost.style.height = 'calc(100% - 32px)'
    widgetHost.style.width = '100%'

    const copyright = document.createElement('div')
    copyright.className = 'tradingview-widget-copyright'
    copyright.innerHTML =
      '<a href="https://www.tradingview.com/" rel="noopener nofollow" target="_blank">' +
      '<span style="color:#848E9C">TradingView</span></a>'

    const script = document.createElement('script')
    script.type = 'text/javascript'
    script.async = true
    script.src = WIDGET_SCRIPT
    script.textContent = JSON.stringify({
      autosize: true,
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

    return () => {
      container.innerHTML = ''
    }
  }, [symbol, interval])

  if (!symbol) {
    return (
      <div
        className="flex items-center justify-center text-sm rounded-lg"
        style={{ height, color: '#5E6673', background: '#0B0E11' }}
      >
        请选择币种
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      className="tradingview-widget-container rounded-lg overflow-hidden"
      style={{
        height,
        width: '100%',
        background: '#0B0E11',
        border: '1px solid #2B3139',
      }}
    />
  )
}

export const TradingViewAdvancedChart = memo(TradingViewAdvancedChartInner)
