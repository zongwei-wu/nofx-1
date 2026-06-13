import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { api } from '../lib/api'

const DEFAULT_CONFIG = {
  name: '新策略',
  exchange_id: 'binance',
  symbol: 'BTCUSDT',
  timeframe: '1h',
  direction: 'long_only',
  entry_indicator: 'RSI14',
  entry_operator: '<',
  entry_value: 30,
  take_profit_pct: 3,
  stop_loss_pct: 1.5,
  max_positions: 3,
  leverage: 5,
  risk_per_trade: 0.02,
}

function buildPayload(form: typeof DEFAULT_CONFIG) {
  return {
    name: form.name,
    exchange_id: form.exchange_id,
    symbol: form.symbol,
    timeframe: form.timeframe,
    direction: form.direction,
    entry_conditions: [
      [{ indicator: form.entry_indicator, operator: form.entry_operator, value: form.entry_value }],
    ],
    exit_rules: {
      take_profit_pct: form.take_profit_pct,
      stop_loss_pct: form.stop_loss_pct,
    },
    risk: {
      max_positions: form.max_positions,
      leverage: form.leverage,
      risk_per_trade: form.risk_per_trade,
    },
  }
}

function parseConfigToForm(cfg: Record<string, unknown>, name: string) {
  const entryGroup = (cfg.entry_conditions as Array<Array<Record<string, unknown>>> | undefined)?.[0]
  const entry = entryGroup?.[0] ?? {}
  const exitRules = (cfg.exit_rules ?? {}) as Record<string, number>
  const risk = (cfg.risk ?? {}) as Record<string, number>
  return {
    name: (cfg.name as string) || name,
    exchange_id: (cfg.exchange_id as string) || 'binance',
    symbol: (cfg.symbol as string) || 'BTCUSDT',
    timeframe: (cfg.timeframe as string) || '1h',
    direction: (cfg.direction as string) || 'long_only',
    entry_indicator: (entry.indicator as string) || 'RSI14',
    entry_operator: (entry.operator as string) || '<',
    entry_value: Number(entry.value ?? 30),
    take_profit_pct: Number(exitRules.take_profit_pct ?? 3),
    stop_loss_pct: Number(exitRules.stop_loss_pct ?? 1.5),
    max_positions: Number(risk.max_positions ?? 3),
    leverage: Number(risk.leverage ?? 5),
    risk_per_trade: Number(risk.risk_per_trade ?? 0.02),
  }
}

export function StrategyEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isNew = !id || id === 'new'
  const [form, setForm] = useState(DEFAULT_CONFIG)
  const [loading, setLoading] = useState(!isNew)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (isNew) return
    let cancelled = false
    ;(async () => {
      try {
        const strategy = await api.getStrategy(id!)
        if (cancelled) return
        const cfg = JSON.parse(strategy.config) as Record<string, unknown>
        setForm(parseConfigToForm(cfg, strategy.name))
      } catch (err) {
        toast.error(err instanceof Error ? err.message : '加载策略失败')
        navigate('/strategies')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [id, isNew, navigate])

  const update = (key: keyof typeof DEFAULT_CONFIG, value: string | number) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const payload = buildPayload(form)
      if (isNew) {
        const strategy = await api.createStrategy(payload)
        toast.success('策略创建成功')
        navigate(`/strategies/${strategy.id}`)
      } else {
        await api.updateStrategy(id!, payload)
        toast.success('策略已保存')
        navigate(`/strategies/${id}`)
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p style={{ color: '#848E9C' }}>加载中...</p>
  }

  const fieldStyle = {
    background: '#1E2329',
    border: '1px solid #2B3139',
    color: '#EAECEF',
  }

  return (
    <div>
      <button
        onClick={() => navigate(isNew ? '/strategies' : `/strategies/${id}`)}
        className="text-sm mb-4"
        style={{ color: '#848E9C' }}
      >
        ← 返回
      </button>

      <h1 className="text-2xl font-bold mb-6" style={{ color: '#EAECEF' }}>
        {isNew ? '新建策略' : '编辑策略'}
      </h1>

      <div className="grid gap-4 max-w-xl">
        {([
          ['name', '策略名称', 'text'],
          ['exchange_id', '交易所', 'text'],
          ['symbol', '交易对', 'text'],
          ['timeframe', '周期', 'text'],
        ] as const).map(([key, label]) => (
          <label key={key} className="block text-sm" style={{ color: '#848E9C' }}>
            {label}
            <input
              className="mt-1 w-full px-3 py-2 rounded"
              style={fieldStyle}
              value={String(form[key])}
              onChange={(e) => update(key, e.target.value)}
            />
          </label>
        ))}

        <label className="block text-sm" style={{ color: '#848E9C' }}>
          方向
          <select
            className="mt-1 w-full px-3 py-2 rounded"
            style={fieldStyle}
            value={form.direction}
            onChange={(e) => update('direction', e.target.value)}
          >
            <option value="long_only">仅做多</option>
            <option value="short_only">仅做空</option>
            <option value="both">双向</option>
          </select>
        </label>

        <div className="rounded-lg p-4" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
          <h3 className="font-medium mb-3" style={{ color: '#EAECEF' }}>入场条件</h3>
          <div className="grid grid-cols-3 gap-3">
            <input
              className="px-3 py-2 rounded text-sm"
              style={fieldStyle}
              placeholder="指标"
              value={form.entry_indicator}
              onChange={(e) => update('entry_indicator', e.target.value)}
            />
            <select
              className="px-3 py-2 rounded text-sm"
              style={fieldStyle}
              value={form.entry_operator}
              onChange={(e) => update('entry_operator', e.target.value)}
            >
              <option value=">">&gt;</option>
              <option value="<">&lt;</option>
              <option value=">=">&gt;=</option>
              <option value="<=">&lt;=</option>
              <option value="cross_above">金叉</option>
              <option value="cross_below">死叉</option>
            </select>
            <input
              type="number"
              className="px-3 py-2 rounded text-sm"
              style={fieldStyle}
              value={form.entry_value}
              onChange={(e) => update('entry_value', Number(e.target.value))}
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          {([
            ['take_profit_pct', '止盈 %'],
            ['stop_loss_pct', '止损 %'],
            ['leverage', '杠杆'],
            ['max_positions', '最大持仓'],
            ['risk_per_trade', '单笔风险比例'],
          ] as const).map(([key, label]) => (
            <label key={key} className="block text-sm" style={{ color: '#848E9C' }}>
              {label}
              <input
                type="number"
                step="any"
                className="mt-1 w-full px-3 py-2 rounded"
                style={fieldStyle}
                value={form[key]}
                onChange={(e) => update(key, Number(e.target.value))}
              />
            </label>
          ))}
        </div>

        <button
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-2 rounded text-sm font-medium mt-2"
          style={{ background: '#F0B90B', color: '#0B0E11', opacity: saving ? 0.6 : 1 }}
        >
          {saving ? '保存中...' : '保存策略'}
        </button>
      </div>
    </div>
  )
}
