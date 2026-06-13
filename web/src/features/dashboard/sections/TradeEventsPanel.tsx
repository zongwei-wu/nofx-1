import { TrendingUp } from 'lucide-react'
import { TradeEventPriceChart } from '../../../components/trade-events/TradeEventPriceChart'
import { ChartErrorBoundary } from '../../../components/trade-events/ChartErrorBoundary'

type EventChartTab = 'ai_exchange' | 'ai_copy'

interface TradeEventsPanelProps {
  traderId: string
  eventChartTab: EventChartTab
  hasCopyTrade: boolean
  exchangeChartSymbols: string[]
  copyChartSymbols: string[]
  exchangePositionOverlays: Array<{
    symbol: string
    position_side: string
    entry_price: number
    unrealized_pnl: number
    qty: number
  }>
  onTabChange: (tab: EventChartTab) => void
}

export function TradeEventsPanel({
  traderId,
  eventChartTab,
  hasCopyTrade,
  exchangeChartSymbols,
  copyChartSymbols,
  exchangePositionOverlays,
  onTabChange,
}: TradeEventsPanelProps) {
  return (
    <div
      className="binance-card p-6 animate-slide-in"
      style={{ animationDelay: '0.12s' }}
    >
      <h2
        className="text-xl font-bold mb-1 flex items-center gap-2"
        style={{ color: '#EAECEF' }}
      >
        <TrendingUp className="w-5 h-5" style={{ color: '#F0B90B' }} />
        持仓币种价格与交易事件
      </h2>
      <p className="text-xs mb-3" style={{ color: '#5E6673' }}>
        {eventChartTab === 'ai_exchange'
          ? 'AI 交易员在自己绑定交易所的实际成交（开/加/减/平）'
          : '该 AI 交易员作为跟单风控时，成功执行的开/平仓（与跟单管理全量数据不同）'}
      </p>
      <div
        className="flex gap-1 p-1 rounded-lg mb-4"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        <button
          type="button"
          onClick={() => onTabChange('ai_exchange')}
          className={`py-1.5 px-3 rounded text-xs font-semibold ${hasCopyTrade ? 'flex-1' : 'w-full'}`}
          style={{
            background:
              eventChartTab === 'ai_exchange' ? '#2B3139' : 'transparent',
            color: eventChartTab === 'ai_exchange' ? '#F0B90B' : '#848E9C',
          }}
        >
          本账户成交
        </button>
        {hasCopyTrade && (
          <button
            type="button"
            onClick={() => onTabChange('ai_copy')}
            className="flex-1 py-1.5 px-3 rounded text-xs font-semibold"
            style={{
              background:
                eventChartTab === 'ai_copy' ? '#2B3139' : 'transparent',
              color: eventChartTab === 'ai_copy' ? '#F0B90B' : '#848E9C',
            }}
          >
            AI 跟单成交
          </button>
        )}
      </div>
      <ChartErrorBoundary>
        <TradeEventPriceChart
          key={eventChartTab}
          source={
            eventChartTab === 'ai_exchange' ? 'ai_trader' : 'ai_copy_trade'
          }
          traderId={traderId}
          symbols={
            eventChartTab === 'ai_exchange'
              ? exchangeChartSymbols
              : copyChartSymbols
          }
          positionOverlays={exchangePositionOverlays}
        />
      </ChartErrorBoundary>
    </div>
  )
}
