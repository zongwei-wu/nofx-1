import { useState, useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useLanguage } from '../contexts/LanguageContext'
import { httpClient } from '../lib/httpClient'
import { api } from '../lib/api'
import type { TraderInfo } from '../types'
import { CopyTradeMonitorTab } from '../components/copy-trade/CopyTradeMonitorTab'
import { CopyTradeCoinPnLChart } from '../components/copy-trade/CopyTradeCoinPnLChart'
import { CopyTradePnLList } from '../components/copy-trade/CopyTradePnLList'
import { TradeEventPriceChart } from '../components/trade-events/TradeEventPriceChart'
import { ChartErrorBoundary } from '../components/trade-events/ChartErrorBoundary'
import {
  aggregateOpenPositions,
  countUniqueOpenPositions,
} from '../components/copy-trade/copyTradePositionUtils'
import { CopyTradeLeaderCard } from '../components/copy-trade/CopyTradeLeaderCard'
import {
  partitionTradersForDashboard,
  type LeaderboardTrader,
} from '../components/copy-trade/copyTradeLeaderboardUtils'

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

type TraderRecordGroup = {
  key: string
  nickname: string
  portfolio_id: string
  records: CopyRecord[]
}

function groupRecordsByTrader(records: CopyRecord[]): TraderRecordGroup[] {
  const map = new Map<string, TraderRecordGroup>()
  for (const rec of records) {
    const key = rec.portfolio_id || rec.nickname || `id-${rec.id}`
    if (!map.has(key)) {
      map.set(key, {
        key,
        nickname: rec.nickname || '未知交易员',
        portfolio_id: rec.portfolio_id || '',
        records: [],
      })
    }
    map.get(key)!.records.push(rec)
  }
  for (const group of map.values()) {
    group.records.sort((a, b) => (b.lead_order_time || 0) - (a.lead_order_time || 0))
  }
  return Array.from(map.values()).sort((a, b) => a.nickname.localeCompare(b.nickname, 'zh-CN'))
}

const RECORDS_PREVIEW_COUNT = 5

function formatRecordTime(ts: number | string) {
  const d = typeof ts === 'number' ? new Date(ts) : new Date(ts)
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function CopyRecordRow({ rec }: { rec: CopyRecord }) {
  return (
    <div
      className="flex flex-col gap-1 p-2.5 rounded text-xs"
      style={{ background: '#0B0E11', border: '1px solid #1E2329' }}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-mono" style={{ color: '#F0B90B' }}>
            {rec.symbol?.replace('USDT', '')}
          </span>
          <span
            className="px-1 py-0.5 rounded font-medium whitespace-nowrap"
            style={{
              background:
                rec.side === 'BUY' ? 'rgba(14,203,129,0.15)' : 'rgba(246,70,93,0.15)',
              color: rec.side === 'BUY' ? '#0ECB81' : '#F6465D',
            }}
          >
            {rec.side === 'BUY' ? '买入' : '卖出'}
            {rec.position_side === 'LONG' ? '多' : '空'}
          </span>
        </div>
        <div className="flex items-center gap-3 whitespace-nowrap">
          <span style={{ color: '#848E9C' }}>{Number(rec.executed_qty).toFixed(4)}张</span>
          <span className="font-mono" style={{ color: '#EAECEF' }}>
            开${Number(rec.avg_price).toLocaleString()}
          </span>
          {rec.status === 'CLOSED' && (
            <span className="font-mono" style={{ color: '#5E6673' }}>
              平${Number(rec.close_price || 0).toLocaleString()}
            </span>
          )}
          <span
            className="px-1.5 py-0.5 rounded text-[10px] font-medium"
            style={{
              background:
                rec.status === 'OPEN'
                  ? 'rgba(14,203,129,0.15)'
                  : rec.status === 'CLOSED'
                    ? 'rgba(142,142,147,0.15)'
                    : 'rgba(246,70,93,0.15)',
              color:
                rec.status === 'OPEN'
                  ? '#0ECB81'
                  : rec.status === 'CLOSED'
                    ? '#8E8E93'
                    : '#F6465D',
            }}
          >
            {rec.status === 'OPEN' ? '持仓' : rec.status === 'CLOSED' ? '已平' : '失败'}
          </span>
          <span style={{ color: '#5E6673' }}>{formatRecordTime(rec.lead_order_time)}</span>
        </div>
      </div>
      {rec.status === 'FAILED' && rec.error_message && (
        <div className="text-[10px]" style={{ color: '#F6465D' }}>
          {rec.error_message}
        </div>
      )}
    </div>
  )
}

function TraderRecordGroupCard({
  nickname,
  records,
}: {
  nickname: string
  records: CopyRecord[]
}) {
  const [expanded, setExpanded] = useState(false)
  const hasMore = records.length > RECORDS_PREVIEW_COUNT
  const visible = expanded ? records : records.slice(0, RECORDS_PREVIEW_COUNT)
  const hiddenCount = records.length - RECORDS_PREVIEW_COUNT

  return (
    <div
      className="rounded-lg overflow-hidden"
      style={{ background: '#1E2329', border: '1px solid #2B3139' }}
    >
      <div
        className="px-3 py-2 flex items-center justify-between gap-2"
        style={{ background: '#0B0E11', borderBottom: '1px solid #2B3139' }}
      >
        <span className="text-sm font-semibold truncate" style={{ color: '#EAECEF' }}>
          {nickname}
        </span>
        <span className="text-[10px] shrink-0" style={{ color: '#5E6673' }}>
          {records.length} 笔
        </span>
      </div>
      <div className="p-2 space-y-1.5">
        {visible.map((rec) => (
          <CopyRecordRow key={rec.id} rec={rec} />
        ))}
        {hasMore && (
          <button
            type="button"
            onClick={() => setExpanded((v) => !v)}
            className="w-full py-2 text-xs font-medium rounded transition-colors"
            style={{
              background: '#0B0E11',
              border: '1px solid #2B3139',
              color: '#F0B90B',
            }}
          >
            {expanded ? '收起' : `展开其余 ${hiddenCount} 笔`}
          </button>
        )}
      </div>
    </div>
  )
}

function RecordsByTraderSection({
  records,
  status,
  title,
  dotColor,
  titleColor,
}: {
  records: CopyRecord[]
  status: string
  title: string
  dotColor: string
  titleColor: string
}) {
  const filtered = records.filter((r) => r.status === status)
  if (filtered.length === 0) return null
  const groups = groupRecordsByTrader(filtered)
  const uniqueOpen =
    status === 'OPEN' ? countUniqueOpenPositions(filtered) : filtered.length
  const countLabel =
    status === 'OPEN'
      ? `${uniqueOpen} 个持仓 · ${filtered.length} 笔`
      : `${filtered.length} 笔`

  return (
    <div>
      <div
        className="text-sm font-semibold mb-3 flex items-center gap-2"
        style={{ color: titleColor }}
      >
        <span className="w-2 h-2 rounded-full inline-block" style={{ background: dotColor }} />
        {title} ({countLabel})
      </div>
      <div className="space-y-4">
        {groups.map((group) => {
          const inGroup = group.records.filter((r) => r.status === status)
          if (inGroup.length === 0) return null
          return (
            <TraderRecordGroupCard
              key={`${status}-${group.key}`}
              nickname={group.nickname}
              records={inGroup}
            />
          )
        })}
      </div>
    </div>
  )
}

interface CopyTradeSettings {
  ai_trader_id: string
  ai_trader_name?: string
  ai_model_name?: string
  fallback_used?: boolean
}

export function CopyTradeDashboard() {
  void useLanguage()
  const navigate = useNavigate()
  const [configs, setConfigs] = useState<CopyConfig[]>([])
  const [records, setRecords] = useState<CopyRecord[]>([])
  const [leaderboard, setLeaderboard] = useState<LeaderboardTrader[]>([])
  const [aiTraders, setAiTraders] = useState<TraderInfo[]>([])
  const [copyTradeSettings, setCopyTradeSettings] = useState<CopyTradeSettings>({ ai_trader_id: '' })
  const [savingAiTrader, setSavingAiTrader] = useState(false)
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [refreshingPnl, setRefreshingPnl] = useState(false)
  const [activeTab, setActiveTab] = useState<'traders' | 'records' | 'pnl' | 'monitor'>('traders')
  const [pnlChartTab, setPnlChartTab] = useState<'price' | 'pnl'>('price')
  const [message, setMessage] = useState('')
  const [manualSyncTrigger, setManualSyncTrigger] = useState(false)

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
          const unique = new Map<string, LeaderboardTrader>()
          all.forEach((raw: LeaderboardTrader) => {
            if (!raw?.leadPortfolioId) return
            unique.set(raw.leadPortfolioId, {
              ...raw,
              leadPortfolioId: raw.leadPortfolioId,
              nickname: raw.nickname || '',
              pnl: Number(raw.pnl) || 0,
              roi: Number(raw.roi) || 0,
            })
          })
          setLeaderboard(Array.from(unique.values()))
        }
      }

      const [settingsRes, tradersList] = await Promise.all([
        httpClient.get('/api/copy-trade/settings', headers),
        api.getTraders().catch(() => [] as TraderInfo[]),
      ])
      if (settingsRes.ok) {
        setCopyTradeSettings(await settingsRes.json())
      }
      setAiTraders(tradersList)
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  const toggleTrader = async (trader: LeaderboardTrader) => {
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
        auto_follow: existing.auto_follow || false,
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
        auto_follow: false,
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

  const safeRecords = records ?? []
  const safeConfigs = configs ?? []

  const { monitored: monitoredTraders, unmonitored: unmonitoredTraders } = useMemo(
    () => partitionTradersForDashboard(leaderboard, safeConfigs),
    [leaderboard, safeConfigs]
  )

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin w-8 h-8 border-2 border-yellow-500 border-t-transparent rounded-full" />
      </div>
    )
  }

  const activeConfigs = safeConfigs.filter(c => c.enabled)
  const hasAutoFollow = safeConfigs.some(c => c.enabled && c.auto_follow)
  const showAiTraderWarning = hasAutoFollow && !copyTradeSettings.ai_trader_id
  const uniqueOpenCount = countUniqueOpenPositions(safeRecords)
  const openAggregated = aggregateOpenPositions(safeRecords)

  const handleSaveAiTrader = async (aiTraderId: string) => {
    setSavingAiTrader(true)
    try {
      await api.updateCopyTradeSettings(aiTraderId)
      const settings = await api.getCopyTradeSettings()
      setCopyTradeSettings(settings)
      setMessage(aiTraderId ? 'AI 交易员设置已保存' : '已清除 AI 交易员设置，将使用默认模型')
    } catch (e: any) {
      setMessage('保存失败: ' + (e.message || '未知错误'))
    } finally {
      setSavingAiTrader(false)
    }
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: '#EAECEF' }}>跟单管理</h1>
        <p className="text-sm mt-1" style={{ color: '#848E9C' }}>
          {activeConfigs.length} 个带单员已启用 · {uniqueOpenCount} 个持仓
        </p>
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
          带单员配置
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
        <div className="space-y-3">
          <div
            className="p-3 rounded-lg text-xs space-y-3"
            style={{ background: '#1E2329', border: '1px solid #2B3139' }}
          >
            <div className="font-semibold text-sm" style={{ color: '#EAECEF' }}>
              跟单 AI 分析交易员
            </div>
            <p style={{ color: '#848E9C' }}>
              自动监控、排行榜单笔跟单开单时，由所选 AI 交易员绑定的模型进行风控分析。
            </p>
            {aiTraders.length === 0 ? (
              <div className="space-y-2">
                <p style={{ color: '#5E6673' }}>尚未创建 AI 交易员，请先前往配置页创建。</p>
                <button
                  type="button"
                  onClick={() => navigate('/traders')}
                  className="px-3 py-1.5 rounded text-xs font-semibold"
                  style={{ background: '#F0B90B', color: '#0B0E11' }}
                >
                  前往创建 AI 交易员
                </button>
              </div>
            ) : (
              <div className="flex flex-wrap items-center gap-2">
                <select
                  value={copyTradeSettings.ai_trader_id}
                  disabled={savingAiTrader}
                  onChange={(e) => handleSaveAiTrader(e.target.value)}
                  className="flex-1 min-w-[200px] px-3 py-2 rounded text-sm"
                  style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                >
                  <option value="">未选择（使用默认已启用模型）</option>
                  {aiTraders.map((t) => (
                    <option key={t.trader_id} value={t.trader_id}>
                      {t.trader_name} · {t.ai_model.split('_').pop() || t.ai_model}
                    </option>
                  ))}
                </select>
                {savingAiTrader && (
                  <span style={{ color: '#F0B90B' }}>保存中…</span>
                )}
              </div>
            )}
            {copyTradeSettings.ai_trader_id && copyTradeSettings.ai_model_name && (
              <p style={{ color: '#5E6673' }}>
                当前模型：{copyTradeSettings.ai_model_name}
              </p>
            )}
            {!copyTradeSettings.ai_trader_id && copyTradeSettings.fallback_used && copyTradeSettings.ai_model_name && (
              <p style={{ color: '#5E6673' }}>
                当前回退使用：{copyTradeSettings.ai_model_name}
              </p>
            )}
          </div>

          {showAiTraderWarning && (
            <div
              className="p-3 rounded-lg text-xs"
              style={{ background: 'rgba(240,185,11,0.1)', border: '1px solid rgba(240,185,11,0.2)', color: '#F0B90B' }}
            >
              已开启自动监控但未选择 AI 交易员，将回退使用第一个已启用的 AI 模型。建议明确选择用于跟单分析的交易员。
            </div>
          )}

          <div
            className="p-3 rounded-lg text-xs space-y-2"
            style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#848E9C' }}
          >
            <div className="font-semibold" style={{ color: '#EAECEF' }}>
              操作说明
            </div>
            <ul className="list-disc list-inside space-y-1 leading-relaxed">
              <li>
                <span style={{ color: '#EAECEF' }}>点击带单员行</span>
                ：加入或移出跟单名单（启用 / 禁用），不会自动下单
              </li>
              <li>
                <span style={{ color: '#EAECEF' }}>自动监控</span>
                ：仅对已启用带单员生效，每 5 分钟由所选 AI 交易员分析并自动跟单
              </li>
              <li>
                <span style={{ color: '#EAECEF' }}>手动跟单同步</span>
                ：立即对所有已启用带单员执行一次同步（拉带单、对齐交易所持仓与记录）
              </li>
            </ul>
          </div>

          <label
            className={`flex items-start gap-3 p-3 rounded-lg cursor-pointer transition-opacity ${
              syncing || activeConfigs.length === 0 ? 'opacity-50 cursor-not-allowed' : ''
            }`}
            style={{ background: '#1E2329', border: '1px solid #2B3139' }}
          >
            <input
              type="checkbox"
              checked={manualSyncTrigger || syncing}
              disabled={syncing || activeConfigs.length === 0}
              onChange={async (e) => {
                if (!e.target.checked || syncing) return
                setManualSyncTrigger(true)
                try {
                  await handleSync()
                } finally {
                  setManualSyncTrigger(false)
                }
              }}
              className="w-4 h-4 mt-0.5 rounded shrink-0"
              style={{ accentColor: '#F0B90B' }}
            />
            <div className="min-w-0">
              <div className="text-sm font-semibold" style={{ color: '#EAECEF' }}>
                手动跟单同步
                {syncing && (
                  <span className="ml-2 text-xs font-normal" style={{ color: '#F0B90B' }}>
                    同步中…
                  </span>
                )}
              </div>
              <div className="text-[11px] mt-1 leading-relaxed" style={{ color: '#5E6673' }}>
                勾选后立即执行一次，处理当前所有已启用带单员；与「自动监控」无关，不会定时重复。
                {activeConfigs.length === 0 && '（请先启用至少一名带单员）'}
              </div>
            </div>
          </label>

          <div className="text-xs font-semibold pt-1" style={{ color: '#848E9C' }}>
            带单交易员 · 点击行启用 / 禁用 · 按盈亏金额排序
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div className="space-y-2 order-2 lg:order-1">
              <div
                className="text-xs font-semibold px-1 sticky top-0 py-1"
                style={{ color: '#848E9C' }}
              >
                未监控 · {unmonitoredTraders.length}
              </div>
              {unmonitoredTraders.length === 0 ? (
                <p className="text-xs py-4 text-center rounded-lg" style={{ color: '#5E6673', background: '#1E2329' }}>
                  排行榜带单员均已加入监控或未加载数据
                </p>
              ) : (
                unmonitoredTraders.map((trader) => (
                  <CopyTradeLeaderCard
                    key={trader.leadPortfolioId}
                    trader={trader}
                    cfg={safeConfigs.find((c) => c.portfolio_id === trader.leadPortfolioId)}
                    onToggle={toggleTrader}
                    onConfigUpdated={loadData}
                  />
                ))
              )}
            </div>
            <div className="space-y-2 order-1 lg:order-2">
              <div
                className="text-xs font-semibold px-1 sticky top-0 py-1"
                style={{ color: '#0ECB81' }}
              >
                监控中 · {monitoredTraders.length}
              </div>
              {monitoredTraders.length === 0 ? (
                <p
                  className="text-xs py-4 text-center rounded-lg"
                  style={{
                    color: '#5E6673',
                    background: 'rgba(14,203,129,0.05)',
                    border: '1px solid rgba(14,203,129,0.15)',
                  }}
                >
                  暂无自动监控中的带单员，启用后勾选「自动监控」即可出现在此栏
                </p>
              ) : (
                monitoredTraders.map((trader) => (
                  <CopyTradeLeaderCard
                    key={trader.leadPortfolioId}
                    trader={trader}
                    cfg={safeConfigs.find((c) => c.portfolio_id === trader.leadPortfolioId)}
                    onToggle={toggleTrader}
                    onConfigUpdated={loadData}
                  />
                ))
              )}
            </div>
          </div>
        </div>
      )}

      {activeTab === 'records' && (
        <div>
          {safeRecords.length === 0 ? (
            <div className="text-center py-10 text-sm" style={{ color: '#5E6673' }}>
              暂无跟单记录，请先在「带单员配置」启用带单员并勾选「手动跟单同步」
            </div>
          ) : (
            <div className="space-y-8">
              <RecordsByTraderSection
                records={safeRecords}
                status="OPEN"
                title="持仓中"
                dotColor="#0ECB81"
                titleColor="#0ECB81"
              />
              <RecordsByTraderSection
                records={safeRecords}
                status="CLOSED"
                title="已平仓"
                dotColor="#848E9C"
                titleColor="#848E9C"
              />
              <RecordsByTraderSection
                records={safeRecords}
                status="FAILED"
                title="失败"
                dotColor="#F6465D"
                titleColor="#F6465D"
              />
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

          <div className="flex gap-1 mb-4 p-1 rounded-lg max-w-md" style={{ background: '#0B0E11' }}>
            <button
              type="button"
              onClick={() => setPnlChartTab('price')}
              className="flex-1 py-1.5 px-3 rounded text-xs font-semibold"
              style={{
                background: pnlChartTab === 'price' ? '#2B3139' : 'transparent',
                color: pnlChartTab === 'price' ? '#F0B90B' : '#848E9C',
              }}
            >
              价格与交易事件
            </button>
            <button
              type="button"
              onClick={() => setPnlChartTab('pnl')}
              className="flex-1 py-1.5 px-3 rounded text-xs font-semibold"
              style={{
                background: pnlChartTab === 'pnl' ? '#2B3139' : 'transparent',
                color: pnlChartTab === 'pnl' ? '#F0B90B' : '#848E9C',
              }}
            >
              盈亏走势
            </button>
          </div>

          {pnlChartTab === 'price' ? (
            <div className="p-4 rounded-lg mb-4" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-sm font-semibold mb-1" style={{ color: '#EAECEF' }}>
                币种价格与跟单事件
              </div>
              <div className="text-xs mb-4" style={{ color: '#5E6673' }}>
                含跟单记录与带单员订单推断的开/加/减/平 · 与 AI 看板「本账户成交 / AI 跟单成交」数据源不同
              </div>
              <ChartErrorBoundary>
                <TradeEventPriceChart
                  source="copy_trade"
                  symbols={[
                    ...new Set(
                      safeRecords
                        .filter((r) => r.status === 'OPEN' || r.status === 'CLOSED')
                        .map((r) => r.symbol)
                        .filter(Boolean)
                    ),
                  ]}
                />
              </ChartErrorBoundary>
            </div>
          ) : (
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
          )}

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
