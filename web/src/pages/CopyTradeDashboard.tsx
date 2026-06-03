import { useState, useEffect } from 'react'
import { useLanguage } from '../contexts/LanguageContext'
import { httpClient } from '../lib/httpClient'
import { CopyTradeMonitorTab } from '../components/copy-trade/CopyTradeMonitorTab'
import { CopyTradeCoinPnLChart } from '../components/copy-trade/CopyTradeCoinPnLChart'
import { CopyTradePnLList } from '../components/copy-trade/CopyTradePnLList'
import {
  aggregateOpenPositions,
  countUniqueOpenPositions,
} from '../components/copy-trade/copyTradePositionUtils'

interface CopyConfig {
  id: number
  portfolio_id: string
  nickname: string
  enabled: boolean
  auto_follow: boolean
  max_copy_size: number
  size_multiplier: number
  copy_open_only: boolean
  last_order_time: number
}

interface CopyRecord {
  id: number
  portfolio_id: string
  nickname: string
  order_id?: string
  symbol: string
  side: string
  position_side: string
  executed_qty: number
  avg_price: number
  close_price?: number
  total_pnl: number
  status: string
  error_message?: string
  lead_order_time: number
  copy_time?: string
  close_time?: string
}

export function CopyTradeDashboard() {
  const { language } = useLanguage()
  const [configs, setConfigs] = useState<CopyConfig[]>([])
  const [records, setRecords] = useState<CopyRecord[]>([])
  const [leaderboard, setLeaderboard] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [refreshingPnl, setRefreshingPnl] = useState(false)
  const [activeTab, setActiveTab] = useState<'traders' | 'records' | 'pnl' | 'monitor'>('traders')
  const [message, setMessage] = useState('')

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const token = localStorage.getItem('auth_token')
      const headers = { Authorization: token ? `Bearer ${token}` : '' }

      // Load copy configs
      const configRes = await httpClient.get('/api/copy-trade/configs', headers)
      if (configRes.ok) {
        const configData = await configRes.json()
        setConfigs(Array.isArray(configData) ? configData : [])
      }

      // Load records
      const recordsRes = await httpClient.get('/api/copy-trade/records', headers)
      if (recordsRes.ok) {
        const recordsData = await recordsRes.json()
        setRecords(Array.isArray(recordsData) ? recordsData : [])
      }

      // Load leaderboard for available traders
      const lbRes = await httpClient.get('/api/copy-trading/leaderboard')
      if (lbRes.ok) {
        const lbData = await lbRes.json()
        if (lbData.code === '000000' && lbData.data) {
          const all = [...(lbData.data.highestPnlLeads || []), ...(lbData.data.highestRoiLeads || [])]
          const unique = new Map()
          all.forEach((t: any) => unique.set(t.leadPortfolioId, t))
          setLeaderboard(Array.from(unique.values()))
        }
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  const toggleTrader = async (trader: any) => {
    const existing = (configs ?? []).find(c => c.portfolio_id === trader.leadPortfolioId)
    const token = localStorage.getItem('auth_token')
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Authorization: token ? `Bearer ${token}` : '',
    }

    if (existing) {
      // Toggle enabled/disabled
      await httpClient.post('/api/copy-trade/configs', {
        id: existing.id,
        portfolio_id: existing.portfolio_id,
        nickname: existing.nickname,
        enabled: !existing.enabled,
        max_copy_size: existing.max_copy_size,
        size_multiplier: existing.size_multiplier,
        copy_open_only: existing.copy_open_only,
      }, headers)
    } else {
      // Add new config
      await httpClient.post('/api/copy-trade/configs', {
        portfolio_id: trader.leadPortfolioId,
        nickname: trader.nickname,
        enabled: true,
        max_copy_size: 1000,
        size_multiplier: 0.1,
        copy_open_only: true,
      }, headers)
    }
    await loadData()
  }

  const handleSync = async () => {
    setSyncing(true)
    setMessage('')
    try {
      const token = localStorage.getItem('auth_token')
      const res = await httpClient.post('/api/copy-trade/sync', undefined, {
        Authorization: token ? `Bearer ${token}` : '',
      })
      const data = await res.json()
      setMessage(`同步完成，复制了 ${data.copied || 0} 条操作`)
      await loadData()
    } catch (e: any) {
      setMessage('同步失败: ' + (e.message || '未知错误'))
    } finally {
      setSyncing(false)
    }
  }

  const handleRefreshPnL = async () => {
    setRefreshingPnl(true)
    setMessage('')
    try {
      const token = localStorage.getItem('auth_token')
      const res = await httpClient.post('/api/copy-trade/refresh-pnl', undefined, {
        Authorization: token ? `Bearer ${token}` : '',
      })
      const data = await res.json()
      if (!res.ok) {
        setMessage('刷新盈亏失败: ' + (data.error || '未知错误'))
        return
      }
      setMessage(`盈亏已刷新，更新 ${data.updated || 0} 条记录`)
      await loadData()
    } catch (e: any) {
      setMessage('刷新盈亏失败: ' + (e.message || '未知错误'))
    } finally {
      setRefreshingPnl(false)
    }
  }

  const formatTime = (ts: number | string) => {
    const d = typeof ts === 'number' ? new Date(ts) : new Date(ts)
    return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  }

  // 单条记录组件
  const RecordRow = ({ rec }: { rec: CopyRecord }) => (
    <div className="flex items-center justify-between p-2.5 rounded text-xs" style={{ background: '#0B0E11', border: '1px solid #1E2329' }}>
      <div className="flex items-center gap-2 min-w-0">
        <span className="font-semibold truncate" style={{ color: '#EAECEF' }}>{rec.nickname}</span>
        <span className="font-mono" style={{ color: '#F0B90B' }}>{rec.symbol?.replace('USDT', '')}</span>
        <span className="px-1 py-0.5 rounded font-medium whitespace-nowrap" style={{
          background: rec.side === 'BUY' ? 'rgba(14,203,129,0.15)' : 'rgba(246,70,93,0.15)',
          color: rec.side === 'BUY' ? '#0ECB81' : '#F6465D',
        }}>
          {rec.side === 'BUY' ? '买入' : '卖出'}{rec.position_side === 'LONG' ? '多' : '空'}
        </span>
      </div>
      <div className="flex items-center gap-3 whitespace-nowrap">
        <span style={{ color: '#848E9C' }}>{Number(rec.executed_qty).toFixed(4)}张</span>
        <span className="font-mono" style={{ color: '#EAECEF' }}>开${Number(rec.avg_price).toLocaleString()}</span>
        {rec.status === 'CLOSED' && (
          <span className="font-mono" style={{ color: '#5E6673' }}>平${Number(rec.close_price || 0).toLocaleString()}</span>
        )}
        <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${rec.status === 'OPEN' ? '' : ''}`} style={{
          background: rec.status === 'OPEN' ? 'rgba(14,203,129,0.15)' : rec.status === 'CLOSED' ? 'rgba(142,142,147,0.15)' : 'rgba(246,70,93,0.15)',
          color: rec.status === 'OPEN' ? '#0ECB81' : rec.status === 'CLOSED' ? '#8E8E93' : '#F6465D',
        }}>
          {rec.status === 'OPEN' ? '持仓' : rec.status === 'CLOSED' ? '已平' : '失败'}
        </span>
        <span style={{ color: '#5E6673' }}>{formatTime(rec.lead_order_time)}</span>
      </div>
      {rec.status === 'FAILED' && rec.error_message && (
        <div className="text-[10px] mt-1" style={{ color: '#F6465D' }}>{rec.error_message}</div>
      )}
    </div>
  )

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin w-8 h-8 border-2 border-yellow-500 border-t-transparent rounded-full" />
      </div>
    )
  }

  const safeRecords = records ?? []
  const safeConfigs = configs ?? []
  const activeConfigs = safeConfigs.filter(c => c.enabled)
  const uniqueOpenCount = countUniqueOpenPositions(safeRecords)
  const openAggregated = aggregateOpenPositions(safeRecords)

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: '#EAECEF' }}>跟单管理</h1>
          <p className="text-sm mt-1" style={{ color: '#848E9C' }}>
            {activeConfigs.length} 个交易员已启用 · {uniqueOpenCount} 个持仓
          </p>
        </div>
        <button
          onClick={handleSync}
          disabled={syncing}
          className="px-4 py-2 rounded text-sm font-semibold transition-all disabled:opacity-50"
          style={{ background: '#F0B90B', color: '#0B0E11' }}
        >
          {syncing ? '同步中...' : '⏳ 执行跟单同步'}
        </button>
      </div>

      {message && (
        <div className="mb-4 p-3 rounded text-sm" style={{ background: 'rgba(240,185,11,0.1)', border: '1px solid rgba(240,185,11,0.2)', color: '#F0B90B' }}>
          {message}
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 mb-6 p-1 rounded-lg" style={{ background: '#1E2329' }}>
        <button
          onClick={() => setActiveTab('traders')}
          className="flex-1 py-2 px-4 rounded text-sm font-semibold transition-all"
          style={{ background: activeTab === 'traders' ? '#F0B90B' : 'transparent', color: activeTab === 'traders' ? '#0B0E11' : '#848E9C' }}
        >
          交易员配置
        </button>
        <button
          onClick={() => setActiveTab('records')}
          className="flex-1 py-2 px-4 rounded text-sm font-semibold transition-all"
          style={{ background: activeTab === 'records' ? '#F0B90B' : 'transparent', color: activeTab === 'records' ? '#0B0E11' : '#848E9C' }}
        >
          跟单记录 ({safeRecords.length})
        </button>
        <button
          onClick={() => setActiveTab('pnl')}
          className="flex-1 py-2 px-4 rounded text-sm font-semibold transition-all"
          style={{ background: activeTab === 'pnl' ? '#F0B90B' : 'transparent', color: activeTab === 'pnl' ? '#0B0E11' : '#848E9C' }}
        >
          收益
        </button>
        <button
          onClick={() => setActiveTab('monitor')}
          className="flex-1 py-2 px-4 rounded text-sm font-semibold transition-all"
          style={{ background: activeTab === 'monitor' ? '#F0B90B' : 'transparent', color: activeTab === 'monitor' ? '#0B0E11' : '#848E9C' }}
        >
          自动监控
        </button>
      </div>

      {activeTab === 'traders' && (
        <div className="space-y-2">
          <div className="text-xs font-semibold mb-2" style={{ color: '#848E9C' }}>
            点击交易员启用/禁用跟单
          </div>
          {(leaderboard ?? []).map((trader: any) => {
            const cfg = safeConfigs.find(c => c.portfolio_id === trader.leadPortfolioId)
            const enabled = cfg?.enabled || false
            return (
              <div
                key={trader.leadPortfolioId}
                className="p-3 rounded-lg transition-all"
                style={{ background: enabled ? 'rgba(14,203,129,0.08)' : '#1E2329', border: `1px solid ${enabled ? 'rgba(14,203,129,0.3)' : '#2B3139'}` }}
              >
                <div className="flex items-center justify-between" onClick={() => toggleTrader(trader)} style={{cursor:'pointer'}}>
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold" style={{ background: '#2B3139', color: '#F0B90B' }}>
                      {trader.nickname?.charAt(0)}
                    </div>
                    <div>
                      <div className="text-sm font-semibold" style={{ color: '#EAECEF' }}>{trader.nickname}</div>
                      <div className="text-xs" style={{ color: '#848E9C' }}>
                        盈亏: ${(trader.pnl || 0).toLocaleString()} · ROI: {(trader.roi || 0).toFixed(2)}%
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    {cfg && (
                      <span className="text-xs" style={{ color: '#5E6673' }}>
                        倍数: {cfg.size_multiplier}x
                      </span>
                    )}
                    <div className={`w-3 h-3 rounded-full ${enabled ? 'bg-green-500' : 'bg-gray-500'}`}
                      style={{ background: enabled ? '#0ECB81' : '#5E6673' }}
                    />
                  </div>
                </div>
                {cfg && (
                  <div className="flex items-center gap-4 mt-2 pt-2 border-t" style={{ borderColor: '#2B3139' }}>
                    <label className="flex items-center gap-1.5 text-xs cursor-pointer" style={{ color: '#848E9C' }}>
                      <input
                        type="checkbox"
                        checked={cfg.auto_follow || false}
                        onChange={async (e) => {
                          e.stopPropagation()
                          const token = localStorage.getItem('auth_token')
                          await httpClient.post('/api/copy-trade/configs', {
                            id: cfg.id,
                            portfolio_id: cfg.portfolio_id,
                            nickname: cfg.nickname,
                            enabled: cfg.enabled,
                            auto_follow: e.target.checked,
                            max_copy_size: cfg.max_copy_size,
                            size_multiplier: cfg.size_multiplier,
                            copy_open_only: cfg.copy_open_only,
                          }, { 'Content-Type': 'application/json', Authorization: token ? `Bearer ${token}` : '' })
                          await loadData()
                        }}
                        className="w-3 h-3 rounded"
                        style={{ accentColor: '#0ECB81' }}
                      />
                      自动监控
                    </label>
                    <span className="text-[10px]" style={{ color: '#5E6673' }}>每5分钟AI分析+自动跟单</span>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {activeTab === 'records' && (
        <div>
          {safeRecords.length === 0 ? (
            <div className="text-center py-10 text-sm" style={{ color: '#5E6673' }}>暂无跟单记录，请先启用交易员并执行同步</div>
          ) : (
            <div className="space-y-6">
              {/* 持仓中 */}
              {(() => {
                const open = safeRecords.filter(r => r.status === 'OPEN')
                const openUnique = countUniqueOpenPositions(open)
                return open.length > 0 && <>
                  <div>
                    <div className="text-sm font-semibold mb-2 flex items-center gap-2" style={{ color: '#0ECB81' }}>
                      <span className="w-2 h-2 rounded-full bg-green-500 inline-block" style={{ background: '#0ECB81' }}></span>
                      持仓中 ({openUnique} 个持仓 · {open.length} 笔)
                    </div>
                    <div className="space-y-1.5">
                      {open.map((rec) => (
                        <RecordRow key={rec.id} rec={rec} />
                      ))}
                    </div>
                  </div>
                </>
              })()}

              {/* 已平仓 */}
              {(() => {
                const closed = safeRecords.filter(r => r.status === 'CLOSED')
                return closed.length > 0 && <>
                  <div>
                    <div className="text-sm font-semibold mb-2 flex items-center gap-2" style={{ color: '#848E9C' }}>
                      <span className="w-2 h-2 rounded-full inline-block" style={{ background: '#848E9C' }}></span>
                      已平仓 ({closed.length})
                    </div>
                    <div className="space-y-1.5">
                      {closed.map((rec) => (
                        <RecordRow key={rec.id} rec={rec} />
                      ))}
                    </div>
                  </div>
                </>
              })()}

              {/* 失败/错误 */}
              {(() => {
                const failed = safeRecords.filter(r => r.status === 'FAILED')
                return failed.length > 0 && <>
                  <div>
                    <div className="text-sm font-semibold mb-2 flex items-center gap-2" style={{ color: '#F6465D' }}>
                      <span className="w-2 h-2 rounded-full inline-block" style={{ background: '#F6465D' }}></span>
                      失败 ({failed.length})
                    </div>
                    <div className="space-y-1.5">
                      {failed.map((rec) => (
                        <RecordRow key={rec.id} rec={rec} />
                      ))}
                    </div>
                  </div>
                </>
              })()}
            </div>
          )}
        </div>
      )}

      {activeTab === 'pnl' && (
        <div>
          <div className="flex items-center justify-between mb-4 gap-3 flex-wrap">
            <div className="text-sm" style={{ color: '#848E9C' }}>
              点击刷新从交易所拉取最新未实现盈亏
            </div>
            <button
              onClick={handleRefreshPnL}
              disabled={refreshingPnl}
              className="px-4 py-2 rounded text-sm font-semibold transition-all disabled:opacity-50"
              style={{ background: '#2B3139', color: '#EAECEF', border: '1px solid #474D57' }}
            >
              {refreshingPnl ? '刷新中...' : '刷新盈亏'}
            </button>
          </div>

          {/* Summary */}
          {(() => {
            const openRecords = safeRecords.filter(r => r.status === 'OPEN')
            const closedRecords = safeRecords.filter(r => r.status === 'CLOSED')
            const unrealized = openRecords.reduce((s, r) => s + (r.total_pnl || 0), 0)
            const realized = closedRecords.reduce((s, r) => s + (r.total_pnl || 0), 0)
            const total = unrealized + realized
            return (
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6">
                <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>总持仓</div>
                  <div className="text-2xl font-bold" style={{ color: '#EAECEF' }}>
                    {openAggregated.length}
                  </div>
                  {openRecords.length > openAggregated.length && (
                    <div className="text-[10px] mt-1" style={{ color: '#5E6673' }}>
                      共 {openRecords.length} 笔跟单记录
                    </div>
                  )}
                </div>
                <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>未实现盈亏</div>
                  <div className="text-2xl font-bold" style={{ color: unrealized >= 0 ? '#0ECB81' : '#F6465D' }}>
                    ${unrealized.toFixed(2)}
                  </div>
                </div>
                <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>已实现盈亏</div>
                  <div className="text-2xl font-bold" style={{ color: realized >= 0 ? '#0ECB81' : '#F6465D' }}>
                    ${realized.toFixed(2)}
                  </div>
                </div>
                <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>总盈亏</div>
                  <div className="text-2xl font-bold" style={{ color: total >= 0 ? '#0ECB81' : '#F6465D' }}>
                    ${total.toFixed(2)}
                  </div>
                </div>
              </div>
            )
          })()}

          {/* Per-coin hourly PnL line chart */}
          <div className="p-4 rounded-lg mb-4" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
            <div className="text-sm font-semibold mb-1" style={{ color: '#EAECEF' }}>
              各币种盈亏走势
            </div>
            <div className="text-xs mb-4" style={{ color: '#5E6673' }}>
              横轴：每小时 · 纵轴：盈亏金额（USDT）· 每条线代表一个币种
            </div>
            <CopyTradeCoinPnLChart
              records={safeRecords.filter((r) => r.status === 'OPEN' || r.status === 'CLOSED')}
            />
          </div>

          {/* Per-trade PnL List */}
          <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
            <div className="text-sm font-semibold mb-3" style={{ color: '#EAECEF' }}>逐笔盈亏</div>
            <CopyTradePnLList records={safeRecords.filter(r => r.status === 'OPEN' || r.status === 'CLOSED')} />
          </div>
        </div>
      )}

      {activeTab === 'monitor' && <CopyTradeMonitorTab />}
    </div>
  )
}
