import { useEffect, useState, type CSSProperties } from 'react'
import { copyTradeApi } from '../../lib/api/copyTrade'
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
  const [sizeMultiplier, setSizeMultiplier] = useState(cfg?.size_multiplier ?? 0.1)
  const [maxCopySize, setMaxCopySize] = useState(cfg?.max_copy_size ?? 1000)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!cfg) return
    setSizeMultiplier(cfg.size_multiplier)
    setMaxCopySize(cfg.max_copy_size)
  }, [cfg?.id, cfg?.size_multiplier, cfg?.max_copy_size])

  const persistConfig = async (patch: {
    auto_follow?: boolean
    copy_open_only?: boolean
    size_multiplier?: number
    max_copy_size?: number
  }) => {
    if (!cfg || saving) return
    setSaving(true)
    try {
      await copyTradeApi.upsertConfig({
        id: cfg.id,
        portfolio_id: cfg.portfolio_id,
        nickname: cfg.nickname,
        enabled: cfg.enabled,
        auto_follow: patch.auto_follow ?? cfg.auto_follow,
        max_copy_size: patch.max_copy_size ?? cfg.max_copy_size,
        size_multiplier: patch.size_multiplier ?? cfg.size_multiplier,
        copy_open_only: patch.copy_open_only ?? cfg.copy_open_only,
      })
      await onConfigUpdated()
    } finally {
      setSaving(false)
    }
  }

  const saveSizeMultiplier = async () => {
    if (!cfg) return
    const value = Number(sizeMultiplier)
    if (!Number.isFinite(value) || value <= 0 || value > 10) {
      setSizeMultiplier(cfg.size_multiplier)
      return
    }
    if (value === cfg.size_multiplier) return
    await persistConfig({ size_multiplier: value })
  }

  const saveMaxCopySize = async () => {
    if (!cfg) return
    const value = Number(maxCopySize)
    if (!Number.isFinite(value) || value < 0) {
      setMaxCopySize(cfg.max_copy_size)
      return
    }
    if (value === cfg.max_copy_size) return
    await persistConfig({ max_copy_size: value })
  }

  const inputStyle: CSSProperties = {
    background: '#0B0E11',
    border: '1px solid #2B3139',
    color: '#EAECEF',
  }

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
      {cfg && (
        <div
          className="mt-2 pt-2 border-t space-y-2"
          style={{ borderColor: '#2B3139' }}
          onClick={(e) => e.stopPropagation()}
        >
          <div className="grid grid-cols-2 gap-2">
            <label className="block">
              <span className="text-[10px] block mb-1" style={{ color: '#5E6673' }}>
                跟单倍数
              </span>
              <input
                type="number"
                min={0.01}
                max={10}
                step={0.01}
                disabled={saving}
                value={sizeMultiplier}
                onChange={(e) => setSizeMultiplier(Number(e.target.value))}
                onBlur={() => void saveSizeMultiplier()}
                className="w-full px-2 py-1 rounded text-xs tabular-nums"
                style={inputStyle}
              />
            </label>
            <label className="block">
              <span className="text-[10px] block mb-1" style={{ color: '#5E6673' }}>
                最大跟单金额 (USDT)
              </span>
              <input
                type="number"
                min={0}
                step={10}
                disabled={saving}
                value={maxCopySize}
                onChange={(e) => setMaxCopySize(Number(e.target.value))}
                onBlur={() => void saveMaxCopySize()}
                className="w-full px-2 py-1 rounded text-xs tabular-nums"
                style={inputStyle}
              />
              <span className="text-[10px] mt-0.5 block" style={{ color: '#5E6673' }}>
                填 0 表示不限制
              </span>
            </label>
          </div>

          <label className="flex items-start gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={cfg.auto_follow || false}
              disabled={!cfg.enabled || saving}
              onChange={(e) => void persistConfig({ auto_follow: e.target.checked })}
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

          <label className="flex items-start gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={cfg.copy_open_only}
              disabled={saving}
              onChange={(e) => void persistConfig({ copy_open_only: e.target.checked })}
              className="w-3.5 h-3.5 mt-0.5 rounded shrink-0"
              style={{ accentColor: '#F0B90B' }}
            />
            <span className="min-w-0">
              <span className="text-xs font-medium block" style={{ color: '#EAECEF' }}>
                仅跟开仓
              </span>
              <span className="text-[10px] block mt-0.5" style={{ color: '#5E6673' }}>
                开启后只跟带单员开仓，不自动跟平仓
              </span>
            </span>
          </label>

          {!enabled && (
            <div className="text-[10px]" style={{ color: '#5E6673' }}>
              已禁用 · 不参与手动同步与自动监控，参数仍可预设
            </div>
          )}
        </div>
      )}
    </div>
  )
}
