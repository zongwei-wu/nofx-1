import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useLanguage } from '../contexts/LanguageContext'
import { copyTradeApi, type CopyOrderPayload } from '../lib/api/copyTrade'
import { LeadRecentReturnsLineChart } from '../components/copy-trade/LeadRecentReturnsLineChart'

type CopyResultModalState = {
  visible: boolean
  success: boolean
  error?: string
  nickname?: string
  symbol?: string
  positionSide?: string
  qty?: number
  qtyUnit?: string
  price?: number
}

const initialCopyResultModal: CopyResultModalState = {
  visible: false,
  success: true,
}

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

function positionSideLabel(side?: string) {
  return side === 'LONG' ? '做多' : side === 'SHORT' ? '做空' : side || '—'
}

export function CopyTradingPage() {
  void useLanguage()
  const navigate = useNavigate()
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
    qtyUnit: string
    leadQtyContracts: number
    leadQtyBase: number
    trade: any
    payload: any
  }>({
    visible: false,
    loading: false,
    analysis: '',
    account: null,
    recommendedQty: 0,
    qtyUnit: '',
    leadQtyContracts: 0,
    leadQtyBase: 0,
    trade: null,
    payload: null,
  })
  const [copyResultModal, setCopyResultModal] =
    useState<CopyResultModalState>(initialCopyResultModal)

  const showCopyResult = (result: Omit<CopyResultModalState, 'visible'> & { visible?: boolean }) => {
    setCopyResultModal({ ...result, visible: true })
  }

  useEffect(() => {
    const fetchData = async () => {
      try {
        const result = await copyTradeApi.getLeaderboardRaw()
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
      const result = await copyTradeApi.getLeaderboardOrders(
        trader.leadPortfolioId,
        10
      )
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
                    <LeadRecentReturnsLineChart chartItems={trader.chartItems} />
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
                                      const payload: CopyOrderPayload = {
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
                                        const res = await copyTradeApi.copyOrder(payload)
                                        const data = await res.json()
                                        if (!res.ok) {
                                          setOrderModal(prev => ({ ...prev, loading: false, visible: false }))
                                          showCopyResult({
                                            success: false,
                                            error: data.error || '请求失败',
                                            nickname: trader.nickname,
                                            symbol: order.symbol,
                                            positionSide: order.positionSide,
                                          })
                                          return
                                        }
                                        if (!data.need_confirm) {
                                          setOrderModal(prev => ({ ...prev, loading: false, visible: false }))
                                          const qtyUnit =
                                            data.qty_unit || order.symbol?.replace('USDT', '') || ''
                                          showCopyResult({
                                            success: true,
                                            nickname: trader.nickname,
                                            symbol: data.symbol || order.symbol,
                                            positionSide: data.position || order.positionSide,
                                            qty: data.qty,
                                            qtyUnit,
                                            price: data.price,
                                          })
                                          return
                                        }
                                        const qtyUnit =
                                          data.qty_unit ||
                                          order.symbol?.replace('USDT', '') ||
                                          order.baseAsset ||
                                          ''
                                        setOrderModal(prev => ({
                                          ...prev,
                                          loading: false,
                                          analysis: data.ai_analysis || '',
                                          account: data.account,
                                          recommendedQty: data.recommended_qty ?? 0,
                                          qtyUnit,
                                          leadQtyContracts:
                                            data.lead_qty_contracts ?? order.executedQty,
                                          leadQtyBase: data.lead_qty_base ?? 0,
                                          trade: data.trade,
                                        }))
                                      } catch (e: unknown) {
                                        setOrderModal(prev => ({ ...prev, loading: false, visible: false }))
                                        showCopyResult({
                                          success: false,
                                          error: e instanceof Error ? e.message : '网络错误',
                                          nickname: trader.nickname,
                                          symbol: order.symbol,
                                          positionSide: order.positionSide,
                                        })
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
                  {orderModal.leadQtyContracts > 0 && (
                    <div className="text-[11px] mt-2" style={{ color: '#848E9C' }}>
                      带单数量 {orderModal.leadQtyContracts} 张
                      {orderModal.leadQtyBase > 0 && orderModal.qtyUnit
                        ? `（≈ ${orderModal.leadQtyBase.toFixed(4)} ${orderModal.qtyUnit}）`
                        : ''}
                    </div>
                  )}
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
                <div className="p-3 rounded-lg space-y-2" style={{ background: '#0B0E11' }}>
                  <div className="text-sm" style={{ color: '#848E9C' }}>
                    建议跟单数量（币安下单口径，可修改）
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min={0}
                      step={0.001}
                      value={orderModal.recommendedQty}
                      onChange={(e) =>
                        setOrderModal((prev) => ({
                          ...prev,
                          recommendedQty: Number(e.target.value),
                        }))
                      }
                      className="flex-1 px-3 py-2 rounded text-sm font-bold tabular-nums"
                      style={{
                        background: '#1E2329',
                        border: '1px solid #2B3139',
                        color: '#0ECB81',
                      }}
                    />
                    <span className="text-sm font-semibold shrink-0" style={{ color: '#EAECEF' }}>
                      {orderModal.qtyUnit || '币'}
                    </span>
                  </div>
                  {orderModal.trade?.price > 0 && orderModal.recommendedQty > 0 && (
                    <div className="text-[11px]" style={{ color: '#5E6673' }}>
                      约 {(orderModal.recommendedQty * orderModal.trade.price).toFixed(2)} USDT
                    </div>
                  )}
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
                      try {
                        const p: CopyOrderPayload = {
                          ...orderModal.payload,
                          skip_ai: true,
                          executed_qty: orderModal.recommendedQty,
                        }
                        const res = await copyTradeApi.copyOrder(p)
                        const data = await res.json()
                        const snapshot = { ...orderModal }
                        setOrderModal(prev => ({ ...prev, visible: false, loading: false }))
                        if (res.ok) {
                          showCopyResult({
                            success: true,
                            nickname: snapshot.payload?.nickname,
                            symbol: data.symbol || snapshot.trade?.symbol,
                            positionSide: data.position || snapshot.trade?.positionSide,
                            qty: data.qty,
                            qtyUnit: snapshot.qtyUnit,
                            price: data.price,
                          })
                        } else {
                          showCopyResult({
                            success: false,
                            error: data.error || '跟单失败',
                            nickname: snapshot.payload?.nickname,
                            symbol: snapshot.trade?.symbol,
                            positionSide: snapshot.trade?.positionSide,
                          })
                        }
                      } catch (e: unknown) {
                        setOrderModal(prev => ({ ...prev, loading: false }))
                        showCopyResult({
                          success: false,
                          error: e instanceof Error ? e.message : '网络错误',
                          nickname: orderModal.payload?.nickname,
                          symbol: orderModal.trade?.symbol,
                          positionSide: orderModal.trade?.positionSide,
                        })
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

      {/* 跟单结果弹窗 */}
      {copyResultModal.visible && (
        <div
          className="fixed inset-0 z-[60] flex items-center justify-center bg-black/70 p-4"
          onClick={() => setCopyResultModal(initialCopyResultModal)}
        >
          <div
            className="rounded-xl max-w-md w-full overflow-hidden"
            style={{ background: '#1E2329', border: '1px solid #2B3139' }}
            onClick={(e) => e.stopPropagation()}
          >
            <div
              className="px-5 pt-6 pb-4 text-center"
              style={{
                background: copyResultModal.success
                  ? 'linear-gradient(180deg, rgba(14,203,129,0.12) 0%, transparent 100%)'
                  : 'linear-gradient(180deg, rgba(246,70,93,0.12) 0%, transparent 100%)',
              }}
            >
              <div
                className="w-14 h-14 mx-auto mb-3 rounded-full flex items-center justify-center text-2xl"
                style={{
                  background: copyResultModal.success
                    ? 'rgba(14,203,129,0.15)'
                    : 'rgba(246,70,93,0.15)',
                  border: `1px solid ${copyResultModal.success ? 'rgba(14,203,129,0.35)' : 'rgba(246,70,93,0.35)'}`,
                }}
              >
                {copyResultModal.success ? '✓' : '✕'}
              </div>
              <h3
                className="text-lg font-bold"
                style={{ color: copyResultModal.success ? '#0ECB81' : '#F6465D' }}
              >
                {copyResultModal.success ? '跟单成功' : '跟单失败'}
              </h3>
              {copyResultModal.nickname && (
                <p className="text-xs mt-1" style={{ color: '#848E9C' }}>
                  带单员 · {copyResultModal.nickname}
                </p>
              )}
            </div>

            <div className="px-5 pb-5 space-y-3">
              {copyResultModal.success ? (
                <div className="rounded-lg p-3 space-y-2.5" style={{ background: '#0B0E11' }}>
                  <div className="flex items-center justify-between">
                    <span className="text-xs" style={{ color: '#848E9C' }}>交易对</span>
                    <span className="text-sm font-bold" style={{ color: '#F0B90B' }}>
                      {copyResultModal.symbol?.replace('USDT', '') || '—'}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs" style={{ color: '#848E9C' }}>方向</span>
                    <span
                      className="px-2 py-0.5 rounded text-xs font-semibold"
                      style={{
                        background:
                          copyResultModal.positionSide === 'LONG'
                            ? 'rgba(14,203,129,0.15)'
                            : 'rgba(246,70,93,0.15)',
                        color:
                          copyResultModal.positionSide === 'LONG' ? '#0ECB81' : '#F6465D',
                      }}
                    >
                      {positionSideLabel(copyResultModal.positionSide)}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs" style={{ color: '#848E9C' }}>成交数量</span>
                    <span className="text-sm font-bold tabular-nums" style={{ color: '#EAECEF' }}>
                      {copyResultModal.qty != null
                        ? `${copyResultModal.qty.toFixed(4)} ${copyResultModal.qtyUnit || ''}`
                        : '—'}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs" style={{ color: '#848E9C' }}>成交价格</span>
                    <span className="text-sm font-bold tabular-nums" style={{ color: '#EAECEF' }}>
                      {copyResultModal.price != null
                        ? `$${copyResultModal.price.toLocaleString(undefined, { maximumFractionDigits: 2 })}`
                        : '—'}
                    </span>
                  </div>
                  {copyResultModal.qty != null &&
                    copyResultModal.price != null &&
                    copyResultModal.qty > 0 && (
                      <div
                        className="flex items-center justify-between pt-2 border-t"
                        style={{ borderColor: '#2B3139' }}
                      >
                        <span className="text-xs" style={{ color: '#848E9C' }}>名义价值</span>
                        <span className="text-sm font-semibold tabular-nums" style={{ color: '#38BDF8' }}>
                          ≈ {(copyResultModal.qty * copyResultModal.price).toFixed(2)} USDT
                        </span>
                      </div>
                    )}
                </div>
              ) : (
                <div
                  className="rounded-lg p-3 text-sm leading-relaxed"
                  style={{
                    background: 'rgba(246,70,93,0.08)',
                    border: '1px solid rgba(246,70,93,0.2)',
                    color: '#F6465D',
                  }}
                >
                  {copyResultModal.error || '未知错误'}
                </div>
              )}

              <div className="flex gap-3 pt-1">
                {copyResultModal.success && (
                  <button
                    type="button"
                    onClick={() => {
                      setCopyResultModal(initialCopyResultModal)
                      navigate('/copy-trade')
                    }}
                    className="flex-1 py-2.5 rounded-lg text-sm font-semibold"
                    style={{ background: '#2B3139', color: '#EAECEF' }}
                  >
                    查看跟单记录
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setCopyResultModal(initialCopyResultModal)}
                  className="flex-1 py-2.5 rounded-lg text-sm font-semibold"
                  style={{
                    background: copyResultModal.success ? '#F0B90B' : '#2B3139',
                    color: copyResultModal.success ? '#0B0E11' : '#EAECEF',
                  }}
                >
                  {copyResultModal.success ? '完成' : '关闭'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
