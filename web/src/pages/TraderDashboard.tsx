import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import useSWR from 'swr'
import { api } from '../lib/api'
import { swrKeys } from '../lib/swrKeys'
import { EquityChart } from '../components/EquityChart'
import { collectChartSymbols } from '../components/trade-events/tradeEventChartUtils'
import AILearning from '../components/AILearning'
import { useLanguage } from '../contexts/LanguageContext'
import { useAuth } from '../contexts/AuthContext'
import { FEATURES } from '../config/features'
import { useSymbolPreferences } from '../contexts/SymbolPreferencesContext'
import { t } from '../i18n/translations'
import { RefreshCw } from 'lucide-react'
import { AccountOverview } from '../features/dashboard/sections/AccountOverview'
import { PositionsTable } from '../features/dashboard/sections/PositionsTable'
import { DecisionsPanel } from '../features/dashboard/sections/DecisionsPanel'
import { TraderHeader } from '../features/dashboard/sections/TraderHeader'
import { TradeEventsPanel } from '../features/dashboard/sections/TradeEventsPanel'
import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  Statistics,
  TraderInfo,
} from '../types'

export default function TraderDashboard() {
  const { language } = useLanguage()
  const { user, token, hasFeature } = useAuth()
  const hasCopyTrade = hasFeature(FEATURES.copy_trade)
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [selectedTraderId, setSelectedTraderId] = useState<string | undefined>(
    searchParams.get('trader') || undefined
  )
  const [lastUpdate, setLastUpdate] = useState<string>('--:--:--')

  // 决策记录数量选择（从 localStorage 读取，默认 5）
  const [decisionLimit, setDecisionLimit] = useState<number>(() => {
    const saved = localStorage.getItem('decisionLimit')
    return saved ? parseInt(saved, 10) : 5
  })

  // 当 limit 变化时保存到 localStorage
  const handleLimitChange = (newLimit: number) => {
    setDecisionLimit(newLimit)
    localStorage.setItem('decisionLimit', newLimit.toString())
  }

  // 获取trader列表（仅在用户登录时）
  const authed = !!(user && token)
  const { data: traders, error: tradersError } = useSWR<TraderInfo[]>(
    swrKeys.traders(authed),
    api.getTraders,
    {
      refreshInterval: 10000,
      shouldRetryOnError: false,
    }
  )

  // 当获取到traders后，设置默认选中第一个
  useEffect(() => {
    if (traders && traders.length > 0 && !selectedTraderId) {
      const firstTraderId = traders[0].trader_id
      setSelectedTraderId(firstTraderId)
      setSearchParams({ trader: firstTraderId })
    }
  }, [traders, selectedTraderId, setSearchParams])

  // 更新URL参数
  const handleTraderSelect = (traderId: string) => {
    setSelectedTraderId(traderId)
    setSearchParams({ trader: traderId })
  }

  // 如果在trader页面，获取该trader的数据
  const { data: status } = useSWR<SystemStatus>(
    authed && selectedTraderId
      ? swrKeys.status(true, selectedTraderId)
      : null,
    () => api.getStatus(selectedTraderId),
    {
      refreshInterval: 15000,
      revalidateOnFocus: false,
      dedupingInterval: 10000,
    }
  )

  const { data: account } = useSWR<AccountInfo>(
    authed && selectedTraderId
      ? swrKeys.account(true, selectedTraderId)
      : null,
    () => api.getAccount(selectedTraderId),
    {
      refreshInterval: 15000,
      revalidateOnFocus: false,
      dedupingInterval: 10000,
    }
  )

  const { data: positions } = useSWR<Position[]>(
    authed && selectedTraderId
      ? swrKeys.positions(true, selectedTraderId)
      : null,
    () => api.getPositions(selectedTraderId),
    {
      refreshInterval: 15000,
      revalidateOnFocus: false,
      dedupingInterval: 10000,
    }
  )

  const { sortSymbols } = useSymbolPreferences()
  const sortedPositions = useMemo(() => {
    if (!positions?.length) return positions ?? []
    const order = sortSymbols(positions.map((p) => p.symbol))
    const rank = new Map(order.map((s, i) => [s, i]))
    return [...positions].sort(
      (a, b) => (rank.get(a.symbol) ?? 999) - (rank.get(b.symbol) ?? 999)
    )
  }, [positions, sortSymbols])

  const { data: decisions } = useSWR<DecisionRecord[]>(
    authed && selectedTraderId
      ? `decisions/latest-${selectedTraderId}-${decisionLimit}`
      : null,
    () => api.getLatestDecisions(selectedTraderId, decisionLimit),
    {
      refreshInterval: 30000,
      revalidateOnFocus: false,
      dedupingInterval: 20000,
    }
  )

  const [eventChartTab, setEventChartTab] = useState<'ai_exchange' | 'ai_copy'>('ai_exchange')

  const exchangeChartSymbols = useMemo(() => {
    const current = (positions ?? []).map((p) => p.symbol).filter(Boolean)
    return sortSymbols(
      collectChartSymbols(current, decisions ?? [], { mode: 'ai_exchange' })
    )
  }, [positions, decisions, sortSymbols])

  const exchangePositionOverlays = useMemo(
    () =>
      (positions ?? []).map((p) => ({
        symbol: p.symbol,
        position_side: p.side,
        entry_price: p.entry_price,
        unrealized_pnl: p.unrealized_pnl,
        qty: Math.abs(p.quantity),
      })),
    [positions]
  )

  const copyChartSymbols = useMemo(() => {
    if (!selectedTraderId) return []
    return sortSymbols(
      collectChartSymbols([], decisions ?? [], {
        mode: 'ai_copy',
        traderId: selectedTraderId,
      })
    )
  }, [decisions, sortSymbols, selectedTraderId])

  const { data: stats } = useSWR<Statistics>(
    authed && selectedTraderId
      ? swrKeys.statistics(true, selectedTraderId)
      : null,
    () => api.getStatistics(selectedTraderId),
    {
      refreshInterval: 30000,
      revalidateOnFocus: false,
      dedupingInterval: 20000,
    }
  )

  // Avoid unused variable warning
  void stats

  useEffect(() => {
    if (account) {
      const now = new Date().toLocaleTimeString()
      setLastUpdate(now)
    }
  }, [account])

  const selectedTrader = traders?.find((t) => t.trader_id === selectedTraderId)

  // 切换交易员时：未运行默认「AI 跟单成交」；无跟单权限仅「本账户成交」
  useEffect(() => {
    if (!selectedTrader) return
    if (!hasCopyTrade) {
      setEventChartTab('ai_exchange')
      return
    }
    setEventChartTab(selectedTrader.is_running ? 'ai_exchange' : 'ai_copy')
  }, [selectedTrader?.trader_id, hasCopyTrade, selectedTrader?.is_running])

  // If API failed with error, show empty state
  if (tradersError) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center max-w-md mx-auto px-6">
          <div
            className="w-24 h-24 mx-auto mb-6 rounded-full flex items-center justify-center"
            style={{
              background: 'rgba(240, 185, 11, 0.1)',
              border: '2px solid rgba(240, 185, 11, 0.3)',
            }}
          >
            <svg
              className="w-12 h-12"
              style={{ color: '#F0B90B' }}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
          </div>
          <h2 className="text-2xl font-bold mb-3" style={{ color: '#EAECEF' }}>
            {t('dashboardEmptyTitle', language)}
          </h2>
          <p className="text-base mb-6" style={{ color: '#848E9C' }}>
            {t('dashboardEmptyDescription', language)}
          </p>
          <button
            onClick={() => navigate('/traders')}
            className="px-6 py-3 rounded-lg font-semibold transition-all hover:scale-105 active:scale-95"
            style={{
              background: 'linear-gradient(135deg, #F0B90B 0%, #FCD535 100%)',
              color: '#0B0E11',
              boxShadow: '0 4px 12px rgba(240, 185, 11, 0.3)',
            }}
          >
            {t('goToTradersPage', language)}
          </button>
        </div>
      </div>
    )
  }

  // If traders is loaded and empty, show empty state
  if (traders && traders.length === 0) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center max-w-md mx-auto px-6">
          <div
            className="w-24 h-24 mx-auto mb-6 rounded-full flex items-center justify-center"
            style={{
              background: 'rgba(240, 185, 11, 0.1)',
              border: '2px solid rgba(240, 185, 11, 0.3)',
            }}
          >
            <svg
              className="w-12 h-12"
              style={{ color: '#F0B90B' }}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
          </div>
          <h2 className="text-2xl font-bold mb-3" style={{ color: '#EAECEF' }}>
            {t('dashboardEmptyTitle', language)}
          </h2>
          <p className="text-base mb-6" style={{ color: '#848E9C' }}>
            {t('dashboardEmptyDescription', language)}
          </p>
          <button
            onClick={() => navigate('/traders')}
            className="px-6 py-3 rounded-lg font-semibold transition-all hover:scale-105 active:scale-95"
            style={{
              background: 'linear-gradient(135deg, #F0B90B 0%, #FCD535 100%)',
              color: '#0B0E11',
              boxShadow: '0 4px 12px rgba(240, 185, 11, 0.3)',
            }}
          >
            {t('goToTradersPage', language)}
          </button>
        </div>
      </div>
    )
  }

  // If traders is still loading or selectedTrader is not ready, show skeleton
  if (!selectedTrader) {
    return (
      <div className="space-y-6">
        <div className="binance-card p-6 animate-pulse">
          <div className="skeleton h-8 w-48 mb-3"></div>
          <div className="flex gap-4">
            <div className="skeleton h-4 w-32"></div>
            <div className="skeleton h-4 w-24"></div>
            <div className="skeleton h-4 w-28"></div>
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="binance-card p-5 animate-pulse">
              <div className="skeleton h-4 w-24 mb-3"></div>
              <div className="skeleton h-8 w-32"></div>
            </div>
          ))}
        </div>
        <div className="binance-card p-6 animate-pulse">
          <div className="skeleton h-6 w-40 mb-4"></div>
          <div className="skeleton h-64 w-full"></div>
        </div>
      </div>
    )
  }

  return (
    <div>
      <TraderHeader
        selectedTrader={selectedTrader}
        traders={traders}
        selectedTraderId={selectedTraderId}
        status={status}
        language={language}
        onTraderSelect={handleTraderSelect}
      />

      {/* Debug Info */}
      {account && (
        <div
          className="mb-4 p-3 rounded text-xs font-mono"
          style={{ background: '#1E2329', border: '1px solid #2B3139' }}
        >
          <div style={{ color: '#848E9C' }}>
            <RefreshCw className="inline w-4 h-4 mr-1 align-text-bottom" />
            Last Update: {lastUpdate} | Initial:{' '}
            {account?.initial_balance?.toFixed(2) || '0.00'} | Total Equity:{' '}
            {account?.total_equity?.toFixed(2) || '0.00'} | Available:{' '}
            {account?.available_balance?.toFixed(2) || '0.00'} | P&L:{' '}
            {account?.total_pnl?.toFixed(2) || '0.00'} (
            {account?.total_pnl_pct?.toFixed(2) || '0.00'}%)
          </div>
        </div>
      )}

      <AccountOverview account={account} language={language} />

      {/* 主要内容区：左右分屏 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* 左侧：持仓 + 交易事件 + 净值曲线 */}
        <div className="space-y-6">
          <PositionsTable
            positions={positions}
            sortedPositions={sortedPositions}
            language={language}
          />

          <TradeEventsPanel
            traderId={selectedTrader.trader_id}
            eventChartTab={eventChartTab}
            hasCopyTrade={hasCopyTrade}
            exchangeChartSymbols={exchangeChartSymbols}
            copyChartSymbols={copyChartSymbols}
            exchangePositionOverlays={exchangePositionOverlays}
            onTabChange={setEventChartTab}
          />

          {/* Equity Chart */}
          <div className="animate-slide-in" style={{ animationDelay: '0.15s' }}>
            <EquityChart traderId={selectedTrader.trader_id} />
          </div>
        </div>

        <DecisionsPanel
          decisions={decisions}
          decisionLimit={decisionLimit}
          onLimitChange={handleLimitChange}
          language={language}
        />
      </div>

      {/* AI Learning & Performance Analysis */}
      <div className="mb-6 animate-slide-in" style={{ animationDelay: '0.3s' }}>
        <AILearning traderId={selectedTrader.trader_id} />
      </div>
    </div>
  )
}
