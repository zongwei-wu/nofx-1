import { useState, useEffect, useCallback } from 'react'
import { httpClient } from '../../lib/httpClient'

const ACTION_LABELS: Record<string, string> = {
  lead_open: '带单开仓',
  lead_close: '带单平仓',
  copied_open: '已跟单开仓',
  holding: '继续持有',
  ai_rejected: 'AI拒绝',
  skipped: '已跳过',
  already_copied: '已跟过',
  qty_too_small: '数量过小',
  open_failed: '开仓失败',
  fetch_failed: '拉单失败',
  no_exchange: '无交易所',
  no_ai: '无AI模型',
  lead_other: '其他操作',
}

const ACTION_COLORS: Record<string, string> = {
  lead_open: '#F0B90B',
  lead_close: '#848E9C',
  copied_open: '#0ECB81',
  holding: '#5E6673',
  ai_rejected: '#F6465D',
  skipped: '#848E9C',
  open_failed: '#F6465D',
  fetch_failed: '#F6465D',
}

interface MonitorData {
  timer_interval_sec: number
  auto_follow_running: boolean
  last_auto_follow_finished: string
  next_run_estimated_at: string
  last_run: Record<string, unknown> | null
  monitored_traders: Array<{
    config_id: number
    portfolio_id: string
    nickname: string
    auto_follow: boolean
    last_order_time: number
    latest_lead_actions: Array<{
      symbol: string
      action: string
      display: string
      order_time: number
    }>
    our_positions: Array<{
      symbol: string
      status: string
      executed_qty: number
      total_pnl: number
    }>
    last_event: {
      action: string
      symbol: string
      detail: string
      our_status: string
      created_at: string
    } | null
  }>
  recent_runs: Array<Record<string, unknown>>
  recent_events: Array<Record<string, unknown>>
}

function formatAction(action: string) {
  return ACTION_LABELS[action] || action
}

function formatTime(ts: string | number | undefined) {
  if (!ts) return '-'
  const d = typeof ts === 'number' ? new Date(ts) : new Date(ts)
  if (Number.isNaN(d.getTime())) return String(ts)
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function CountdownBar({
  nextAt,
  intervalSec,
}: {
  nextAt: string
  intervalSec: number
}) {
  const [now, setNow] = useState(Date.now())
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(t)
  }, [])
  const target = new Date(nextAt).getTime()
  const remain = Math.max(0, Math.floor((target - now) / 1000))
  const pct = Math.min(100, Math.max(0, ((intervalSec - remain) / intervalSec) * 100))
  const min = Math.floor(remain / 60)
  const sec = remain % 60
  return (
    <div className="mt-3">
      <div className="flex justify-between text-xs mb-1" style={{ color: '#848E9C' }}>
        <span>距下次自动检查</span>
        <span>
          {min}:{String(sec).padStart(2, '0')}
        </span>
      </div>
      <div className="h-1.5 rounded-full overflow-hidden" style={{ background: '#2B3139' }}>
        <div
          className="h-full transition-all duration-1000"
          style={{ width: `${pct}%`, background: '#F0B90B' }}
        />
      </div>
    </div>
  )
}

export function CopyTradeMonitorTab() {
  const [data, setData] = useState<MonitorData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [expandedRunId, setExpandedRunId] = useState<number | null>(null)

  const load = useCallback(async () => {
    try {
      const token = localStorage.getItem('auth_token')
      const res = await httpClient.get('/api/copy-trade/monitor', {
        Authorization: token ? `Bearer ${token}` : '',
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        setError((err as { error?: string }).error || '加载失败')
        return
      }
      setData(await res.json())
      setError('')
    } catch (e) {
      setError('网络错误')
      console.error(e)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
    const id = setInterval(load, 30000)
    return () => clearInterval(id)
  }, [load])

  if (loading && !data) {
    return (
      <div className="flex justify-center py-16">
        <div className="animate-spin w-8 h-8 border-2 border-yellow-500 border-t-transparent rounded-full" />
      </div>
    )
  }

  if (error && !data) {
    return (
      <div className="text-center py-10 text-sm" style={{ color: '#F6465D' }}>
        {error}
      </div>
    )
  }

  if (!data) return null

  const monitoredTraders = data.monitored_traders ?? []
  const recentRuns = data.recent_runs ?? []
  const recentEvents = data.recent_events ?? []

  const lastRun = data.last_run as Record<string, unknown> | null

  return (
    <div className="space-y-6">
      {/* 执行摘要 */}
      <div className="p-4 rounded-lg" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
        <div className="flex items-center justify-between flex-wrap gap-2">
          <h2 className="text-sm font-bold" style={{ color: '#EAECEF' }}>
            定时任务状态
            {data.auto_follow_running && (
              <span className="ml-2 text-xs font-normal px-2 py-0.5 rounded" style={{ background: 'rgba(240,185,11,0.2)', color: '#F0B90B' }}>
                执行中
              </span>
            )}
          </h2>
          <span className="text-xs" style={{ color: '#5E6673' }}>
            间隔 {data.timer_interval_sec / 60} 分钟
          </span>
        </div>
        {lastRun ? (
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-3 text-xs">
            <div className="p-2 rounded" style={{ background: '#0B0E11' }}>
              <div style={{ color: '#848E9C' }}>最近执行</div>
              <div className="font-mono mt-0.5" style={{ color: '#EAECEF' }}>
                {formatTime(lastRun.started_at as string)}
              </div>
            </div>
            <div className="p-2 rounded" style={{ background: '#0B0E11' }}>
              <div style={{ color: '#848E9C' }}>状态</div>
              <div className="mt-0.5 font-semibold" style={{ color: lastRun.status === 'success' ? '#0ECB81' : lastRun.status === 'failed' ? '#F6465D' : '#F0B90B' }}>
                {String(lastRun.status || '-')}
              </div>
            </div>
            <div className="p-2 rounded" style={{ background: '#0B0E11' }}>
              <div style={{ color: '#848E9C' }}>检查 / 开仓</div>
              <div className="mt-0.5" style={{ color: '#EAECEF' }}>
                {String(lastRun.traders_checked ?? 0)} / {String(lastRun.actions_opened ?? 0)}
              </div>
            </div>
            <div className="p-2 rounded" style={{ background: '#0B0E11' }}>
              <div style={{ color: '#848E9C' }}>跳过 / 失败</div>
              <div className="mt-0.5" style={{ color: '#EAECEF' }}>
                {String(lastRun.actions_skipped ?? 0)} / {String(lastRun.actions_failed ?? 0)}
              </div>
            </div>
          </div>
        ) : (
          <p className="text-xs mt-2" style={{ color: '#5E6673' }}>
            尚无执行记录，请开启「自动监控」并等待定时任务运行
          </p>
        )}
        {lastRun?.message ? (
          <p className="text-xs mt-2" style={{ color: '#848E9C' }}>
            {String(lastRun.message)}
          </p>
        ) : null}
        <CountdownBar nextAt={data.next_run_estimated_at} intervalSec={data.timer_interval_sec} />
      </div>

      {/* 监控中的交易员 */}
      <div>
        <h2 className="text-sm font-semibold mb-3" style={{ color: '#848E9C' }}>
          监控中的交易员 ({monitoredTraders.length})
        </h2>
        {monitoredTraders.length === 0 ? (
          <div className="text-center py-8 text-sm rounded-lg" style={{ background: '#1E2329', color: '#5E6673' }}>
            暂无自动监控的交易员。请在「交易员配置」中勾选「自动监控」。
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {monitoredTraders.map((t) => (
              <div
                key={t.portfolio_id}
                className="p-3 rounded-lg"
                style={{ background: '#1E2329', border: '1px solid #2B3139' }}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="font-semibold text-sm" style={{ color: '#EAECEF' }}>
                    {t.nickname}
                  </span>
                  {t.last_event && (
                    <span
                      className="text-[10px] px-1.5 py-0.5 rounded font-medium"
                      style={{
                        background: `${ACTION_COLORS[t.last_event.action] || '#5E6673'}22`,
                        color: ACTION_COLORS[t.last_event.action] || '#848E9C',
                      }}
                    >
                      {formatAction(t.last_event.action)}
                    </span>
                  )}
                </div>
                {t.last_event && (
                  <p className="text-[11px] mb-2" style={{ color: '#848E9C' }}>
                    {t.last_event.symbol && (
                      <span className="font-mono mr-1" style={{ color: '#F0B90B' }}>
                        {t.last_event.symbol.replace('USDT', '')}
                      </span>
                    )}
                    {t.last_event.detail || '-'}
                    <span className="ml-2" style={{ color: '#5E6673' }}>
                      {formatTime(t.last_event.created_at)}
                    </span>
                  </p>
                )}
                <div className="text-[10px] mb-1" style={{ color: '#5E6673' }}>
                  带单最近操作
                </div>
                <div className="flex flex-wrap gap-1 mb-2">
                  {t.latest_lead_actions?.length ? (
                    t.latest_lead_actions.map((a, i) => (
                      <span
                        key={i}
                        className="px-1.5 py-0.5 rounded text-[10px]"
                        style={{ background: '#0B0E11', color: '#EAECEF' }}
                      >
                        {a.symbol?.replace('USDT', '')} {a.display || a.action}
                      </span>
                    ))
                  ) : (
                    <span style={{ color: '#5E6673' }}>暂无</span>
                  )}
                </div>
                <div className="text-[10px] mb-1" style={{ color: '#5E6673' }}>
                  我方持仓
                </div>
                <div className="flex flex-wrap gap-1">
                  {t.our_positions?.length ? (
                    t.our_positions.map((p, i) => (
                      <span
                        key={i}
                        className="px-1.5 py-0.5 rounded text-[10px] font-medium inline-flex items-center gap-1"
                        style={{
                          background:
                            p.status === 'OPEN' ? 'rgba(14,203,129,0.15)' : 'rgba(142,142,147,0.15)',
                          color: p.status === 'OPEN' ? '#0ECB81' : '#8E8E93',
                        }}
                      >
                        <span>
                          {p.symbol?.replace('USDT', '')}{' '}
                          {p.status === 'OPEN' ? '持仓' : p.status === 'CLOSED' ? '已平' : p.status}
                        </span>
                        {(p.status === 'OPEN' || (p.total_pnl || 0) !== 0) && (
                          <span style={{ color: (p.total_pnl || 0) >= 0 ? '#0ECB81' : '#F6465D' }}>
                            {(p.total_pnl || 0) >= 0 ? '+' : ''}{(p.total_pnl || 0).toFixed(2)}
                          </span>
                        )}
                      </span>
                    ))
                  ) : (
                    <span style={{ color: '#5E6673' }}>未跟单</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 执行历史 */}
      <div>
        <h2 className="text-sm font-semibold mb-3" style={{ color: '#848E9C' }}>
          执行历史
        </h2>
        <div className="space-y-2">
          {recentRuns.slice(0, 20).map((run) => {
            const id = run.id as number
            const isOpen = expandedRunId === id
            const events = recentEvents.filter((e) => e.run_id === id)
            return (
              <div key={id} className="rounded-lg overflow-hidden" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
                <button
                  type="button"
                  className="w-full p-3 text-left flex items-center justify-between gap-2"
                  onClick={() => setExpandedRunId(isOpen ? null : id)}
                >
                  <div className="text-xs" style={{ color: '#EAECEF' }}>
                    <span className="font-mono">{formatTime(run.started_at as string)}</span>
                    <span className="mx-2" style={{ color: '#5E6673' }}>
                      {run.trigger === 'manual' ? '手动同步' : '自动'}
                    </span>
                    <span
                      style={{
                        color:
                          run.status === 'success' ? '#0ECB81' : run.status === 'failed' ? '#F6465D' : '#F0B90B',
                      }}
                    >
                      {String(run.status)}
                    </span>
                  </div>
                  <div className="text-[10px] whitespace-nowrap" style={{ color: '#848E9C' }}>
                    开{String(run.actions_opened)} 跳{String(run.actions_skipped)} 败{String(run.actions_failed)} ·{' '}
                    {events.length} 条事件 {isOpen ? '▲' : '▼'}
                  </div>
                </button>
                {isOpen && events.length > 0 && (
                  <div className="px-3 pb-3 space-y-1 border-t" style={{ borderColor: '#2B3139' }}>
                    {events.map((ev) => (
                      <div
                        key={String(ev.id)}
                        className="flex items-start gap-2 text-[11px] py-1.5 px-2 rounded"
                        style={{ background: '#0B0E11' }}
                      >
                        <span className="font-semibold shrink-0" style={{ color: '#848E9C' }}>
                          {String(ev.nickname)}
                        </span>
                        <span
                          className="shrink-0 px-1 rounded text-[10px]"
                          style={{
                            color: ACTION_COLORS[String(ev.action)] || '#EAECEF',
                            background: `${ACTION_COLORS[String(ev.action)] || '#5E6673'}18`,
                          }}
                        >
                          {formatAction(String(ev.action))}
                        </span>
                        {ev.symbol ? (
                          <span className="font-mono shrink-0" style={{ color: '#F0B90B' }}>
                            {String(ev.symbol).replace('USDT', '')}
                          </span>
                        ) : null}
                        <span className="flex-1 min-w-0 truncate" style={{ color: '#848E9C' }}>
                          {String(ev.detail || '')}
                        </span>
                        <span className="shrink-0" style={{ color: '#5E6673' }}>
                          {formatTime(ev.created_at as string)}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
                {isOpen && events.length === 0 && (
                  <div className="px-3 pb-3 text-xs" style={{ color: '#5E6673' }}>
                    本次无明细事件
                  </div>
                )}
              </div>
            )
          })}
          {recentRuns.length === 0 && (
            <div className="text-center py-8 text-sm" style={{ color: '#5E6673' }}>
              暂无执行历史
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
