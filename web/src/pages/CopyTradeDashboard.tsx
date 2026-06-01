import { useState, useEffect } from 'react'
import { useLanguage } from '../contexts/LanguageContext'
import { httpClient } from '../lib/httpClient'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'

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
  copy_time: string
}

export function CopyTradeDashboard() {
  const { language } = useLanguage()
  const [configs, setConfigs] = useState<CopyConfig[]>([])
  const [records, setRecords] = useState<CopyRecord[]>([])
  const [leaderboard, setLeaderboard] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [activeTab, setActiveTab] = useState<'traders' | 'records' | 'pnl'>('traders')
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
      if (configRes.ok) setConfigs(await configRes.json())

      // Load records
      const recordsRes = await httpClient.get('/api/copy-trade/records', headers)
      if (recordsRes.ok) setRecords(await recordsRes.json())

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
    const existing = configs.find(c => c.portfolio_id === trader.leadPortfolioId)
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

  const activeConfigs = configs.filter(c => c.enabled)

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: '#EAECEF' }}>跟单管理</h1>
          <p className="text-sm mt-1" style={{ color: '#848E9C' }}>
            {activeConfigs.length} 个交易员已启用 · {records.filter(r => r.status === 'OPEN').length} 个持仓
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
          跟单记录 ({records.length})
        </button>
        <button
          onClick={() => setActiveTab('pnl')}
          className="flex-1 py-2 px-4 rounded text-sm font-semibold transition-all"
          style={{ background: activeTab === 'pnl' ? '#F0B90B' : 'transparent', color: activeTab === 'pnl' ? '#0B0E11' : '#848E9C' }}
        >
          收益
        </button>
      </div>

      {activeTab === 'traders' && (
        <div className="space-y-2">
          <div className="text-xs font-semibold mb-2" style={{ color: '#848E9C' }}>
            点击交易员启用/禁用跟单
          </div>
          {leaderboard.map((trader: any) => {
            const cfg = configs.find(c => c.portfolio_id === trader.leadPortfolioId)
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
          {records.length === 0 ? (
            <div className="text-center py-10 text-sm" style={{ color: '#5E6673' }}>暂无跟单记录，请先启用交易员并执行同步</div>
          ) : (
            <div className="space-y-6">
              {/* 持仓中 */}
              {(() => {
                const open = records.filter(r => r.status === 'OPEN')
                return open.length > 0 && <>
                  <div>
                    <div className="text-sm font-semibold mb-2 flex items-center gap-2" style={{ color: '#0ECB81' }}>
                      <span className="w-2 h-2 rounded-full bg-green-500 inline-block" style={{ background: '#0ECB81' }}></span>
                      持仓中 ({open.length})
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
                const closed = records.filter(r => r.status === 'CLOSED')
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
                const failed = records.filter(r => r.status === 'FAILED')
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
          {/* Summary */}
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>总持仓</div>
              <div className="text-2xl font-bold" style={{ color: '#EAECEF' }}>
                {records.filter(r => r.status === 'OPEN').length}
              </div>
            </div>
            <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>总未实现盈亏</div>
              <div className="text-2xl font-bold" style={{
                color: (() => {
                  const total = records.reduce((s, r) => s + (r.total_pnl || 0), 0)
                  return total >= 0 ? '#0ECB81' : '#F6465D'
                })()
              }}>
                ${records.reduce((s, r) => s + (r.total_pnl || 0), 0).toFixed(2)}
              </div>
            </div>
          </div>

          {/* Per-coin PnL Line Chart */}
          <div className="p-4 rounded-lg mb-4" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
            <div className="text-sm font-semibold mb-4" style={{ color: '#EAECEF' }}>各币种盈亏趋势</div>
            {(() => {
              const openTrades = records.filter(r => r.status === 'OPEN')
              if (openTrades.length === 0) {
                return <div className="text-center py-8 text-sm" style={{ color: '#5E6673' }}>暂无持仓</div>
              }
              // Sort by time
              const sorted = [...openTrades].sort((a, b) => (a.lead_order_time || 0) - (b.lead_order_time || 0))
              // Build chart data: each point is a trade, with time on X and PnL on Y
              const chartData = sorted.map(r => ({
                time: new Date(r.lead_order_time).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit' }),
                pnl: r.total_pnl || 0,
                coin: r.symbol?.replace('USDT', '') || r.symbol,
                qty: r.executed_qty,
              }))
              // Group by coin for multiple lines
              const coinColors: Record<string, string> = { BTC: '#F0B90B', ETH: '#627EEA', SOL: '#9945FF', BNB: '#F3BA2F', XRP: '#23292F', ADA: '#0033AD', DOGE: '#C2A633', HYPE: '#FF007A' }
              const coins = [...new Set(sorted.map(r => r.symbol?.replace('USDT', '') || r.symbol))]
              return (
                <>
                  <div style={{ width: '100%', height: 250 }}>
                    <ResponsiveContainer>
                      <LineChart data={chartData} margin={{ top: 5, right: 20, left: 20, bottom: 5 }}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#2B3139" />
                        <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#848E9C' }} />
                        <YAxis tick={{ fontSize: 10, fill: '#848E9C' }} tickFormatter={(v) => `$${v}`} />
                        <Tooltip
                          contentStyle={{ background: '#1E2329', border: '1px solid #2B3139', borderRadius: '8px' }}
                          labelStyle={{ color: '#EAECEF' }}
                          formatter={(value: number, name: string) => [`${value >= 0 ? '+' : ''}$${value.toFixed(2)}`, name]}
                        />
                        <Legend wrapperStyle={{ fontSize: '11px', color: '#848E9C' }} />
                        {coins.map(coin => (
                          <Line
                            key={coin}
                            type="monotone"
                            dataKey="pnl"
                            data={chartData.filter(d => d.coin === coin)}
                            name={coin}
                            stroke={coinColors[coin] || '#848E9C'}
                            strokeWidth={2}
                            dot={{ r: 3 }}
                            connectNulls
                          />
                        ))}
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                  {/* Summary cards below chart */}
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-4">
                    {(() => {
                      const byCoin: Record<string, { pnl: number; qty: number }> = {}
                      openTrades.forEach(r => {
                        const coin = r.symbol?.replace('USDT', '') || r.symbol
                        if (!byCoin[coin]) byCoin[coin] = { pnl: 0, qty: 0 }
                        byCoin[coin].pnl += r.total_pnl || 0
                        byCoin[coin].qty += r.executed_qty || 0
                      })
                      return Object.entries(byCoin).map(([coin, data]) => (
                        <div key={coin} className="p-3 rounded text-center" style={{ background: '#0B0E11' }}>
                          <div className="text-xs" style={{ color: '#848E9C' }}>{coin}</div>
                          <div className="text-sm font-bold mt-1" style={{ color: data.pnl >= 0 ? '#0ECB81' : '#F6465D' }}>
                            {data.pnl >= 0 ? '+' : ''}{data.pnl.toFixed(2)}
                          </div>
                          <div className="text-[10px]" style={{ color: '#5E6673' }}>{data.qty.toFixed(2)}张</div>
                        </div>
                      ))
                    })()}
                  </div>
                </>
              )
            })()}
          </div>

          {/* Per-trade PnL List */}
          <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
            <div className="text-sm font-semibold mb-3" style={{ color: '#EAECEF' }}>逐笔盈亏</div>
            {records.filter(r => r.status === 'OPEN').length === 0 ? (
              <div className="text-center py-4 text-sm" style={{ color: '#5E6673' }}>暂无持仓</div>
            ) : (
              <div className="space-y-1.5">
                {records.filter(r => r.status === 'OPEN').map((rec) => (
                  <div key={rec.id} className="flex items-center justify-between p-2 rounded text-xs" style={{ background: '#0B0E11' }}>
                    <div className="flex items-center gap-2">
                      <span style={{ color: '#F0B90B' }}>{rec.symbol?.replace('USDT', '')}</span>
                      <span className="px-1 py-0.5 rounded font-medium" style={{
                        background: rec.side === 'BUY' ? 'rgba(14,203,129,0.15)' : 'rgba(246,70,93,0.15)',
                        color: rec.side === 'BUY' ? '#0ECB81' : '#F6465D',
                      }}>
                        {rec.side === 'BUY' ? '多' : '空'}
                      </span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span style={{ color: '#848E9C' }}>{rec.executed_qty?.toFixed(4)}张</span>
                      <span style={{ color: '#EAECEF' }}>${rec.avg_price?.toLocaleString()}</span>
                      <span style={{ color: (rec.total_pnl || 0) >= 0 ? '#0ECB81' : '#F6465D' }}>
                        {(rec.total_pnl || 0) >= 0 ? '+' : ''}{(rec.total_pnl || 0).toFixed(2)}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
