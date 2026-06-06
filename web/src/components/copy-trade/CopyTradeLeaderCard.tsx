import { httpClient } from '../../lib/httpClient'
import type { LeaderboardTrader } from './copyTradeLeaderboardUtils'

export interface CopyConfig {
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

export interface CopyTradeLeaderCardProps {
  trader: LeaderboardTrader
  cfg?: CopyConfig
  onToggle: (trader: LeaderboardTrader) => void
  onConfigUpdated: () => void | Promise<void>
}

export function CopyTradeLeaderCard({
  trader,
  cfg,
  onToggle,
  onConfigUpdated,
}: CopyTradeLeaderCardProps) {
  const enabled = cfg?.enabled || false

  return (
    <div
      className="p-3 rounded-lg transition-all"
      style={{
        background: enabled ? 'rgba(14,203,129,0.08)' : '#1E2329',
        border: `1px solid ${enabled ? 'rgba(14,203,129,0.3)' : '#2B3139'}`,
      }}
    >
      <div
        className="flex items-center justify-between"
        onClick={() => onToggle(trader)}
        style={{ cursor: 'pointer' }}
      >
        <div className="flex items-center gap-3">
          <div
            className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold"
            style={{ background: '#2B3139', color: '#F0B90B' }}
          >
            {trader.nickname?.charAt(0)}
          </div>
          <div>
            <div className="text-sm font-semibold" style={{ color: '#EAECEF' }}>
              {trader.nickname}
            </div>
            <div className="text-xs" style={{ color: '#848E9C' }}>
              盈亏: ${(trader.pnl || 0).toLocaleString()} · ROI:{' '}
              {(trader.roi || 0).toFixed(2)}%
            </div>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {cfg && (
            <span className="text-xs" style={{ color: '#5E6673' }}>
              倍数: {cfg.size_multiplier}x
            </span>
          )}
          <div
            className="w-3 h-3 rounded-full"
            style={{ background: enabled ? '#0ECB81' : '#5E6673' }}
          />
        </div>
      </div>
      {cfg && enabled && (
        <div
          className="mt-2 pt-2 border-t space-y-2"
          style={{ borderColor: '#2B3139' }}
          onClick={(e) => e.stopPropagation()}
        >
          <label className="flex items-start gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={cfg.auto_follow || false}
              disabled={!cfg.enabled}
              onChange={async (e) => {
                const token = localStorage.getItem('auth_token')
                await httpClient.post(
                  '/api/copy-trade/configs',
                  {
                    id: cfg.id,
                    portfolio_id: cfg.portfolio_id,
                    nickname: cfg.nickname,
                    enabled: cfg.enabled,
                    auto_follow: e.target.checked,
                    max_copy_size: cfg.max_copy_size,
                    size_multiplier: cfg.size_multiplier,
                    copy_open_only: cfg.copy_open_only,
                  },
                  {
                    'Content-Type': 'application/json',
                    Authorization: token ? `Bearer ${token}` : '',
                  }
                )
                await onConfigUpdated()
              }}
              className="w-3.5 h-3.5 mt-0.5 rounded shrink-0"
              style={{ accentColor: '#0ECB81' }}
            />
            <span className="min-w-0">
              <span className="text-xs font-medium block" style={{ color: '#EAECEF' }}>
                自动监控
              </span>
              <span className="text-[10px] block mt-0.5" style={{ color: '#5E6673' }}>
                每 5 分钟拉取带单并 AI 分析，通过后自动下单
              </span>
            </span>
          </label>
        </div>
      )}
      {cfg && !enabled && (
        <div
          className="text-[10px] mt-2 pt-2 border-t"
          style={{ borderColor: '#2B3139', color: '#5E6673' }}
        >
          已禁用 · 不参与手动同步与自动监控
        </div>
      )}
    </div>
  )
}
