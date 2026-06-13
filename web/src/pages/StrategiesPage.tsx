import { useNavigate } from 'react-router-dom'
import useSWR from 'swr'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

const STATUS_COLORS: Record<string, string> = {
  draft: '#848E9C',
  active: '#0ECB81',
  paused: '#F0B90B',
  archived: '#5E6673',
}

export function StrategiesPage() {
  const { user, token } = useAuth()
  const navigate = useNavigate()
  const { data: strategies, mutate } = useSWR(
    user && token ? 'strategies' : null,
    api.getStrategies
  )

  const handleCreate = async () => {
    const strategy = await api.createStrategy({
      name: '新策略',
      exchange_id: 'binance',
      symbol: 'BTCUSDT',
      timeframe: '1h',
      direction: 'long_only',
      entry_conditions: [[{ indicator: 'RSI14', operator: '<', value: 30 }]],
      exit_rules: { take_profit_pct: 3, stop_loss_pct: 1.5 },
      risk: { max_positions: 3, leverage: 5, risk_per_trade: 0.02 },
    })
    navigate(`/strategies/${strategy.id}`)
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold" style={{ color: '#EAECEF' }}>
          策略管理
        </h1>
        <button
          onClick={handleCreate}
          className="px-4 py-2 rounded text-sm font-medium"
          style={{ background: '#F0B90B', color: '#0B0E11' }}
        >
          新建策略
        </button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {(strategies ?? []).map((s) => {
          let cfg: { symbol?: string; timeframe?: string } = {}
          try {
            cfg = JSON.parse(s.config)
          } catch {
            /* ignore */
          }
          return (
            <div
              key={s.id}
              className="rounded-lg p-4 cursor-pointer transition hover:opacity-90"
              style={{ background: '#1E2329', border: '1px solid #2B3139' }}
              onClick={() => navigate(`/strategies/${s.id}`)}
            >
              <div className="flex items-center justify-between mb-2">
                <h3 className="font-semibold" style={{ color: '#EAECEF' }}>
                  {s.name}
                </h3>
                <span
                  className="text-xs px-2 py-0.5 rounded"
                  style={{
                    background: `${STATUS_COLORS[s.status] ?? '#848E9C'}22`,
                    color: STATUS_COLORS[s.status] ?? '#848E9C',
                  }}
                >
                  {s.status}
                </span>
              </div>
              <p className="text-sm" style={{ color: '#848E9C' }}>
                {cfg.symbol ?? '-'} · {cfg.timeframe ?? '-'}
              </p>
              <div className="flex gap-2 mt-3" onClick={(e) => e.stopPropagation()}>
                {s.status !== 'active' && (
                  <button
                    className="text-xs px-2 py-1 rounded"
                    style={{ background: '#0ECB8122', color: '#0ECB81' }}
                    onClick={async () => {
                      await api.activateStrategy(s.id)
                      mutate()
                    }}
                  >
                    激活
                  </button>
                )}
                {s.status === 'active' && (
                  <button
                    className="text-xs px-2 py-1 rounded"
                    style={{ background: '#F0B90B22', color: '#F0B90B' }}
                    onClick={async () => {
                      await api.pauseStrategy(s.id)
                      mutate()
                    }}
                  >
                    暂停
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {strategies?.length === 0 && (
        <p className="text-center py-12" style={{ color: '#848E9C' }}>
          暂无策略，点击「新建策略」开始
        </p>
      )}
    </div>
  )
}
