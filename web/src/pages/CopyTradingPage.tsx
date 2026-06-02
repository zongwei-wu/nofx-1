import { useState, useEffect } from 'react'
import { useLanguage } from '../contexts/LanguageContext'
import { httpClient } from '../lib/httpClient'

interface OrderRecord {
  symbol: string
  baseAsset: string
  quoteAsset: string
  side: 'BUY' | 'SELL'
  type: string
  positionSide: 'LONG' | 'SHORT'
  executedQty: number
  avgPrice: number
  totalPnl: number
  orderTime: number
}

interface TraderData {
  leadPortfolioId: string
  nickname: string
  avatarUrl: string
  currentCopyCount: number
  maxCopyCount: number
  roi: number
  pnl: number
  aum: number
  mdd: number
  winRate: number
  sharpRatio: number | null
  badgeName: string | null
  badgeCopierCount: number | null
  apiKeyTag: string | null
  tradFiTag: string | null
  portfolioType: string
  startTime: number
  chartItems?: Array<{ value: number; dataType: string; dateTime: number }>
  orders?: OrderRecord[]
  ordersLoading?: boolean
}

export function CopyTradingPage() {
  const { language } = useLanguage()
  const [pnlTraders, setPnlTraders] = useState<TraderData[]>([])
  const [roiTraders, setRoiTraders] = useState<TraderData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [selectedTrader, setSelectedTrader] = useState<TraderData | null>(null)
  const [activeTab, setActiveTab] = useState<'pnl' | 'roi'>('pnl')
  const [orderModal, setOrderModal] = useState<{
    visible: boolean
    loading: boolean
    analysis: string
    account: any
    recommendedQty: number
    trade: any
    payload: any
  }>({ visible: false, loading: false, analysis: '', account: null, recommendedQty: 0, trade: null, payload: null })

  useEffect(() => {
    const fetchData = async () => {
      try {
        const token = localStorage.getItem('auth_token')
        const response = await httpClient.get('/api/copy-trading/leaderboard', {
          Authorization: token ? `Bearer ${token}` : '',
        })
        const result = await response.json()
        if (result.code === '000000' && result.data) {
          setPnlTraders(result.data.highestPnlLeads || [])
          setRoiTraders(result.data.highestRoiLeads || [])
          if (result.stale && result.warning) {
            setError(String(result.warning))
          }
        } else {
          setError(result.error || result.warning || '获取数据失败')
        }
      } catch (err) {
        setError('网络错误，请稍后重试')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  const formatNumber = (num: number, decimals = 2) => {
    if (num >= 1000000) return (num / 10000).toFixed(decimals) + '万'
    if (num >= 10000) return (num / 10000).toFixed(decimals) + '万'
    return num.toLocaleString('zh-CN', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
  }

  const formatROI = (roi: number) => {
    const sign = roi >= 0 ? '+' : ''
    return `${sign}${roi.toFixed(2)}%`
  }

  const getBadgeLabel = (badge: string | null) => {
    if (!badge) return null
    const labels: Record<string, string> = {
      MASTER: '大师',
      CHAMPION: '冠军',
      EXPERT: '行家',
      LEGEND: '传奇',
    }
    return labels[badge] || badge
  }

  const getBadgeColor = (badge: string | null) => {
    if (!badge) return ''
    const colors: Record<string, string> = {
      MASTER: '#FFD700',
      CHAMPION: '#FF6B6B',
      EXPERT: '#4ECDC4',
      LEGEND: '#9B59B6',
    }
    return colors[badge] || '#848E9C'
  }

  const fetchOrders = async (trader: TraderData) => {
    // Mark as loading
    const markOrdersLoading = (list: TraderData[], id: string, loading: boolean) =>
      list.map(t => t.leadPortfolioId === id ? { ...t, ordersLoading: loading } : t)

    setPnlTraders(prev => markOrdersLoading(prev, trader.leadPortfolioId, true))
    setRoiTraders(prev => markOrdersLoading(prev, trader.leadPortfolioId, true))

    try {
      const token = localStorage.getItem('auth_token')
      const response = await httpClient.get(
        `/api/copy-trading/orders?portfolio_id=${trader.leadPortfolioId}&page_size=10`,
        { Authorization: token ? `Bearer ${token}` : '' }
      )
      const result = await response.json()
      if (result.code === '000000' && result.data) {
        const orders = result.data.list || []
        const addOrders = (list: TraderData[]) =>
          list.map(t => t.leadPortfolioId === trader.leadPortfolioId ? { ...t, orders, ordersLoading: false } : t)
        setPnlTraders(prev => addOrders(prev))
        setRoiTraders(prev => addOrders(prev))
      } else {
        const clearLoading = (list: TraderData[]) =>
          list.map(t => t.leadPortfolioId === trader.leadPortfolioId ? { ...t, ordersLoading: false } : t)
        setPnlTraders(prev => clearLoading(prev))
        setRoiTraders(prev => clearLoading(prev))
      }
    } catch {
      const clearLoading = (list: TraderData[]) =>
        list.map(t => t.leadPortfolioId === trader.leadPortfolioId ? { ...t, ordersLoading: false } : t)
      setPnlTraders(prev => clearLoading(prev))
      setRoiTraders(prev => clearLoading(prev))
    }
  }

  const handleTraderClick = (trader: TraderData) => {
    if (selectedTrader?.leadPortfolioId === trader.leadPortfolioId) {
      setSelectedTrader(null)
    } else {
      setSelectedTrader(trader)
      if (!trader.orders && !trader.ordersLoading) {
        fetchOrders(trader)
      }
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="text-center">
          <div className="animate-spin w-8 h-8 border-2 border-yellow-500 border-t-transparent rounded-full mx-auto mb-4" />
          <p style={{ color: '#848E9C' }}>加载跟单数据...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-center py-20">
        <p style={{ color: '#F6465D' }}>{error}</p>
        <button
          onClick={() => window.location.reload()}
          className="mt-4 px-4 py-2 rounded text-sm font-semibold"
          style={{ background: '#F0B90B', color: '#0B0E11' }}
        >
          重新加载
        </button>
      </div>
    )
  }

  const displayTraders = activeTab === 'pnl' ? pnlTraders : roiTraders

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: '#EAECEF' }}>
          币安跟单排行榜
        </h1>
        <p className="text-sm mt-1" style={{ color: '#848E9C' }}>
          数据来源: Binance Copy Trading · 实时更新
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 p-1 rounded-lg" style={{ background: '#1E2329' }}>
        <button
          onClick={() => setActiveTab('pnl')}
          className="flex-1 py-2 px-4 rounded text-sm font-semibold transition-all"
          style={{
            background: activeTab === 'pnl' ? '#F0B90B' : 'transparent',
            color: activeTab === 'pnl' ? '#0B0E11' : '#848E9C',
          }}
        >
          高盈亏
        </button>
        <button
          onClick={() => setActiveTab('roi')}
          className="flex-1 py-2 px-4 rounded text-sm font-semibold transition-all"
          style={{
            background: activeTab === 'roi' ? '#F0B90B' : 'transparent',
            color: activeTab === 'roi' ? '#0B0E11' : '#848E9C',
          }}
        >
          高收益
        </button>
      </div>

      {/* Trader List */}
      <div className="space-y-3">
        {displayTraders.map((trader, index) => (
          <div
            key={trader.leadPortfolioId}
            className="rounded-lg overflow-hidden transition-all duration-200 hover:scale-[1.01] cursor-pointer"
            style={{
              background: '#1E2329',
              border: '1px solid #2B3139',
            }}
            onClick={() =>
              handleTraderClick(trader)
            }
          >
            {/* Main Card */}
            <div className="p-4">
              <div className="flex items-center gap-4">
                {/* Rank */}
                <div
                  className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold"
                  style={{
                    background: index < 3 ? '#F0B90B' : '#2B3139',
                    color: index < 3 ? '#0B0E11' : '#848E9C',
                  }}
                >
                  {index + 1}
                </div>

                {/* Avatar */}
                <div
                  className="w-10 h-10 rounded-full flex items-center justify-center text-lg font-bold flex-shrink-0"
                  style={{ background: '#2B3139', color: '#F0B90B' }}
                >
                  {trader.nickname.charAt(0)}
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-semibold text-sm" style={{ color: '#EAECEF' }}>
                      {trader.nickname}
                    </span>
                    {trader.apiKeyTag && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded font-medium" style={{ background: 'rgba(240,185,11,0.15)', color: '#F0B90B' }}>
                        API
                      </span>
                    )}
                    {trader.tradFiTag && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded font-medium" style={{ background: 'rgba(78,205,196,0.15)', color: '#4ECDC4' }}>
                        TradFi
                      </span>
                    )}
                    {getBadgeLabel(trader.badgeName) && (
                      <span
                        className="text-[10px] px-1.5 py-0.5 rounded font-medium"
                        style={{
                          background: `${getBadgeColor(trader.badgeName)}20`,
                          color: getBadgeColor(trader.badgeName),
                        }}
                      >
                        {getBadgeLabel(trader.badgeName)}
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-3 mt-1 text-xs" style={{ color: '#848E9C' }}>
                    <span>{trader.currentCopyCount}/{trader.maxCopyCount} 跟单</span>
                    <span>{Math.floor((Date.now() - trader.startTime) / (1000 * 60 * 60 * 24))} 天</span>
                  </div>
                </div>

                {/* Stats */}
                <div className="text-right">
                  <div className="text-sm font-bold" style={{ color: trader.roi >= 0 ? '#0ECB81' : '#F6465D' }}>
                    {formatROI(trader.roi)}
                  </div>
                  <div className="text-xs mt-0.5" style={{ color: '#EAECEF' }}>
                    ${formatNumber(trader.pnl)}
                  </div>
                </div>

                {/* Expand indicator */}
                <div
                  className="transition-transform duration-200"
                  style={{
                    transform: selectedTrader?.leadPortfolioId === trader.leadPortfolioId ? 'rotate(180deg)' : '',
                    color: '#848E9C',
                  }}
                >
                  ▼
                </div>
              </div>
            </div>

            {/* Expanded Detail */}
            {selectedTrader?.leadPortfolioId === trader.leadPortfolioId && (
              <div className="px-4 pb-4">
                <div className="h-px mb-4" style={{ background: '#2B3139' }} />
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                  <div className="p-3 rounded" style={{ background: '#0B0E11' }}>
                    <div className="text-xs mb-1" style={{ color: '#848E9C' }}>资产管理规模</div>
                    <div className="text-sm font-semibold" style={{ color: '#EAECEF' }}>${formatNumber(trader.aum)}</div>
                  </div>
                  <div className="p-3 rounded" style={{ background: '#0B0E11' }}>
                    <div className="text-xs mb-1" style={{ color: '#848E9C' }}>最大回撤</div>
                    <div className="text-sm font-semibold" style={{ color: '#F6465D' }}>{trader.mdd.toFixed(2)}%</div>
                  </div>
                  <div className="p-3 rounded" style={{ background: '#0B0E11' }}>
                    <div className="text-xs mb-1" style={{ color: '#848E9C' }}>胜率</div>
                    <div className="text-sm font-semibold" style={{ color: '#EAECEF' }}>{trader.winRate.toFixed(2)}%</div>
                  </div>
                  <div className="p-3 rounded" style={{ background: '#0B0E11' }}>
                    <div className="text-xs mb-1" style={{ color: '#848E9C' }}>夏普比率</div>
                    <div className="text-sm font-semibold" style={{ color: '#EAECEF' }}>
                      {trader.sharpRatio !== null ? trader.sharpRatio.toFixed(2) : '-'}
                    </div>
                  </div>
                </div>

                {/* Chart / Operation Records */}
                {trader.chartItems && trader.chartItems.length > 0 && (
                  <div className="mt-4">
                    <div className="text-xs font-semibold mb-2" style={{ color: '#848E9C' }}>
                      近期收益记录（最近30天）
                    </div>
                    <div className="flex items-end gap-0.5 h-20" style={{ minHeight: '80px' }}>
                      {trader.chartItems.slice(-20).map((item, i) => {
                        const isPositive = item.value >= 0
                        const maxAbs = Math.max(
                          ...trader.chartItems!.slice(-20).map((c) => Math.abs(c.value)),
                          1
                        )
                        const height = Math.abs(item.value) / maxAbs * 100
                        return (
                          <div
                            key={i}
                            className="flex-1 rounded-t transition-all hover:opacity-80 relative group"
                            style={{
                              height: `${Math.max(height, 2)}%`,
                              background: isPositive ? '#0ECB81' : '#F6465D',
                              opacity: 0.8,
                            }}
                            title={`${new Date(item.dateTime).toLocaleDateString()}: ${item.value.toFixed(2)}%`}
                          />
                        )
                      })}
                    </div>
                  </div>
                )}

                {/* Latest Orders - Vertical Timeline */}
                <div className="mt-4">
                  <div className="text-xs font-semibold mb-3" style={{ color: '#848E9C' }}>
                    最新操作记录
                  </div>
                  {trader.ordersLoading ? (
                    <div className="text-center py-6 text-xs" style={{ color: '#5E6673' }}>
                      加载中...
                    </div>
                  ) : trader.orders && trader.orders.length > 0 ? (
                    <div className="relative pl-7">
                      {/* Vertical line */}
                      <div className="absolute left-[11px] top-1 bottom-1 w-px" style={{ background: '#2B3139' }} />
                      {trader.orders.map((order: any, i: number) => {
                        const isLong = order.positionSide === 'LONG' || (order.positionSide === 'BOTH' && order.side === 'BUY')
                        const isOpen = (order.positionSide === 'LONG' && order.side === 'BUY') || (order.positionSide === 'SHORT' && order.side === 'SELL') || (order.positionSide === 'BOTH' && order.side === 'SELL')
                        const dotColor = isLong ? '#0ECB81' : '#F6465D'
                        const dt = new Date(order.orderTime)
                        const timeStr = `${String(dt.getHours()).padStart(2,'0')}:${String(dt.getMinutes()).padStart(2,'0')}`
                        const dateStr = `${String(dt.getMonth()+1).padStart(2,'0')}-${String(dt.getDate()).padStart(2,'0')}`
                        return (
                          <div key={i} className="relative pb-4 group">
                            {/* Timeline dot */}
                            <div
                              className="absolute -left-[23px] top-[6px] w-[15px] h-[15px] rounded-full border-[3px] z-10 transition-transform group-hover:scale-125"
                              style={{
                                background: '#0B0E11',
                                borderColor: dotColor,
                                boxShadow: `0 0 6px ${dotColor}44`,
                              }}
                            />
                            {/* Hover glow line */}
                            <div
                              className="absolute left-[-15px] top-0 bottom-0 w-[1px] opacity-0 group-hover:opacity-100 transition-opacity"
                              style={{ background: dotColor, boxShadow: `0 0 4px ${dotColor}` }}
                            />
                            {/* Content card */}
                            <div
                              className="rounded-lg p-3 text-xs transition-all group-hover:scale-[1.01]"
                              style={{ background: '#0B0E11', border: '1px solid #1E2329' }}
                            >
                              {/* Top row: time + symbol + side */}
                              <div className="flex items-center gap-2 mb-1.5">
                                <span className="text-[10px] font-mono whitespace-nowrap" style={{ color: '#5E6673' }}>
                                  {dateStr} {timeStr}
                                </span>
                                <span className="font-bold text-sm" style={{ color: '#EAECEF' }}>
                                  {order.symbol?.replace('USDT', '')}
                                </span>
                                <span
                                  className="px-1.5 py-0.5 rounded text-[10px] font-semibold"
                                  style={{
                                    background: isLong ? 'rgba(14,203,129,0.15)' : 'rgba(246,70,93,0.15)',
                                    color: isLong ? '#0ECB81' : '#F6465D',
                                  }}
                                >
                                  {isOpen ? (isLong ? '开多' : '开空') : (isLong ? '平多' : '平空')}
                                </span>
                                {order.type === 'LIMIT' && (
                                  <span className="text-[10px] px-1 rounded" style={{ background: '#1E2329', color: '#848E9C' }}>限价</span>
                                )}
                              </div>
                              {/* Detail row: qty + price + PnL */}
                              <div className="flex items-center gap-3 text-[11px]" style={{ color: '#848E9C' }}>
                                <span>{order.executedQty} 张</span>
                                <span>${Number(order.avgPrice).toLocaleString()}</span>
                                {order.totalPnl !== undefined && order.totalPnl !== null && (
                                  <span className="font-medium" style={{ color: order.totalPnl >= 0 ? '#0ECB81' : '#F6465D' }}>
                                    {order.totalPnl >= 0 ? '+' : ''}{order.totalPnl.toFixed(2)} USDT
                                  </span>
                                )}
                                <div className="flex-1" />
                                {isOpen && (
                                  <button
                                    onClick={async (e) => {
                                      e.stopPropagation()
                                      const token = localStorage.getItem('auth_token')
                                      const headers = { 'Content-Type': 'application/json', Authorization: token ? `Bearer ${token}` : '' }
                                      const payload = {
                                        portfolio_id: trader.leadPortfolioId,
                                        nickname: trader.nickname,
                                        symbol: order.symbol,
                                        side: order.side,
                                        position_side: order.positionSide,
                                        executed_qty: order.executedQty,
                                        avg_price: order.avgPrice,
                                        order_time: order.orderTime,
                                        skip_ai: false,
                                      }
                                      setOrderModal(prev => ({ ...prev, visible: true, loading: true, payload }))
                                      try {
                                        const res = await fetch('/api/copy-trade/copy-order', { method: 'POST', headers, body: JSON.stringify(payload) })
                                        const data = await res.json()
                                        if (!res.ok) {
                                          setOrderModal(prev => ({ ...prev, loading: false, visible: false }))
                                          alert(`❌ ${data.error || '请求失败'}`)
                                          return
                                        }
                                        if (!data.need_confirm) {
                                          setOrderModal(prev => ({ ...prev, loading: false, visible: false }))
                                          alert(data.message || '跟单成功')
                                          return
                                        }
                                        setOrderModal(prev => ({
                                          ...prev,
                                          loading: false,
                                          analysis: data.ai_analysis || '',
                                          account: data.account,
                                          recommendedQty: data.recommended_qty || order.executedQty,
                                          trade: data.trade,
                                        }))
                                      } catch(e: any) {
                                        setOrderModal(prev => ({ ...prev, loading: false }))
                                        alert(`❌ ${e.message}`)
                                      }
                                    }}
                                    className="px-2 py-1 rounded text-[10px] font-semibold transition-all hover:scale-105"
                                    style={{ background: '#F0B90B', color: '#0B0E11' }}
                                  >
                                    跟单
                                  </button>
                                )}
                              </div>
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  ) : (
                    <div className="text-center py-6 text-xs" style={{ color: '#5E6673' }}>
                      {trader.orders ? '暂无操作记录' : '点击加载操作记录'}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {displayTraders.length === 0 && (
        <div className="text-center py-20" style={{ color: '#848E9C' }}>
          暂无数据
        </div>
      )}

      {/* AI 分析确认弹窗 */}
      {orderModal.visible && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-60 p-4" onClick={() => setOrderModal(prev => ({ ...prev, visible: false }))}>
          <div className="rounded-xl max-w-lg w-full max-h-[80vh] overflow-y-auto" style={{ background: '#1E2329', border: '1px solid #2B3139' }} onClick={e => e.stopPropagation()}>
            {/* Header */}
            <div className="flex items-center justify-between p-4 border-b" style={{ borderColor: '#2B3139' }}>
              <h3 className="text-base font-bold" style={{ color: '#EAECEF' }}>🤖 AI 交易决策</h3>
              <button onClick={() => setOrderModal(prev => ({ ...prev, visible: false }))} className="text-sm" style={{ color: '#848E9C' }}>✕</button>
            </div>

            {orderModal.loading ? (
              <div className="flex items-center justify-center py-12">
                <div className="animate-spin w-6 h-6 border-2 border-yellow-500 border-t-transparent rounded-full" />
                <span className="ml-3 text-sm" style={{ color: '#848E9C' }}>AI 正在分析账户...</span>
              </div>
            ) : (
              <div className="p-4 space-y-4">
                {/* 交易信息 */}
                <div className="p-3 rounded-lg" style={{ background: '#0B0E11' }}>
                  <div className="text-xs mb-2" style={{ color: '#848E9C' }}>跟单请求</div>
                  <div className="flex items-center gap-2">
                    <span className="font-bold text-base" style={{ color: '#F0B90B' }}>{orderModal.trade?.symbol?.replace('USDT', '')}</span>
                    <span className="px-2 py-0.5 rounded text-xs font-semibold" style={{
                      background: orderModal.trade?.positionSide === 'LONG' ? 'rgba(14,203,129,0.15)' : 'rgba(246,70,93,0.15)',
                      color: orderModal.trade?.positionSide === 'LONG' ? '#0ECB81' : '#F6465D',
                    }}>
                      {orderModal.trade?.positionSide === 'LONG' ? '做多' : '做空'}
                    </span>
                    <span className="text-xs" style={{ color: '#848E9C' }}>带单价 ${orderModal.trade?.price?.toLocaleString()}</span>
                  </div>
                </div>

                {/* 账户状态 */}
                <div className="grid grid-cols-3 gap-2">
                  <div className="p-2 rounded text-center" style={{ background: '#0B0E11' }}>
                    <div className="text-[10px]" style={{ color: '#848E9C' }}>总权益</div>
                    <div className="text-sm font-bold" style={{ color: '#EAECEF' }}>${orderModal.account?.total_equity?.toFixed(0) || '?'}</div>
                  </div>
                  <div className="p-2 rounded text-center" style={{ background: '#0B0E11' }}>
                    <div className="text-[10px]" style={{ color: '#848E9C' }}>可用余额</div>
                    <div className="text-sm font-bold" style={{ color: '#EAECEF' }}>${orderModal.account?.available?.toFixed(0) || '?'}</div>
                  </div>
                  <div className="p-2 rounded text-center" style={{ background: '#0B0E11' }}>
                    <div className="text-[10px]" style={{ color: '#848E9C' }}>持仓数</div>
                    <div className="text-sm font-bold" style={{ color: '#EAECEF' }}>{orderModal.account?.open_positions || 0}</div>
                  </div>
                </div>

                {/* AI 分析 */}
                <div className="p-3 rounded-lg" style={{ background: 'rgba(240,185,11,0.05)', border: '1px solid rgba(240,185,11,0.15)' }}>
                  <div className="text-xs font-semibold mb-2" style={{ color: '#F0B90B' }}>💡 AI 分析结论</div>
                  <div className="text-sm whitespace-pre-wrap" style={{ color: '#EAECEF', lineHeight: '1.6' }}>
                    {orderModal.analysis || '分析中...'}
                  </div>
                </div>

                {/* 建议仓位 */}
                <div className="p-3 rounded-lg" style={{ background: '#0B0E11' }}>
                  <div className="flex items-center justify-between">
                    <span className="text-sm" style={{ color: '#848E9C' }}>建议跟单数量</span>
                    <span className="text-lg font-bold" style={{ color: '#0ECB81' }}>
                      {orderModal.recommendedQty?.toFixed(4)} 张
                    </span>
                  </div>
                </div>

                {/* 操作按钮 */}
                <div className="flex gap-3 pt-2">
                  <button
                    onClick={() => setOrderModal(prev => ({ ...prev, visible: false }))}
                    className="flex-1 py-2.5 rounded-lg text-sm font-semibold transition-all"
                    style={{ background: '#2B3139', color: '#848E9C' }}
                  >
                    取消
                  </button>
                  <button
                    onClick={async () => {
                      setOrderModal(prev => ({ ...prev, loading: true }))
                      const token = localStorage.getItem('auth_token')
                      try {
                        const p = { ...orderModal.payload, skip_ai: true, executed_qty: orderModal.recommendedQty }
                        const res = await fetch('/api/copy-trade/copy-order', {
                          method: 'POST',
                          headers: { 'Content-Type': 'application/json', Authorization: token ? `Bearer ${token}` : '' },
                          body: JSON.stringify(p),
                        })
                        const data = await res.json()
                        setOrderModal(prev => ({ ...prev, visible: false, loading: false }))
                        if (res.ok) {
                          alert(`✅ 跟单成功\n${orderModal.trade?.symbol} ${orderModal.trade?.positionSide}\n数量: ${data.qty?.toFixed(4)}张\n价格: $${data.price?.toLocaleString()}`)
                        } else {
                          alert(`❌ ${data.error || '跟单失败'}`)
                        }
                      } catch(e: any) {
                        setOrderModal(prev => ({ ...prev, loading: false }))
                        alert(`❌ ${e.message}`)
                      }
                    }}
                    className="flex-1 py-2.5 rounded-lg text-sm font-semibold transition-all hover:opacity-90"
                    style={{ background: '#F0B90B', color: '#0B0E11' }}
                  >
                    确认跟单
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
