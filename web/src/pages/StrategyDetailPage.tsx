import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import useSWR from 'swr'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

export function StrategyDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user, token } = useAuth()
  const [validating, setValidating] = useState(false)
  const [signal, setSignal] = useState<Record<string, unknown> | null>(null)
  const [backtestId, setBacktestId] = useState<string | null>(null)

  const { data: strategy } = useSWR(
    user && token && id ? `strategy-${id}` : null,
    () => api.getStrategy(id!)
  )
  const { data: signals } = useSWR(
    user && token && id ? `signals-${id}` : null,
    () => api.getStrategySignals(id!)
  )

  let cfg: Record<string, unknown> = {}
  if (strategy?.config) {
    try {
      cfg = JSON.parse(strategy.config)
    } catch {
      /* ignore */
    }
  }

  const handleValidate = async () => {
    if (!id) return
    setValidating(true)
    try {
      const result = await api.validateStrategy(id)
      setSignal(result.signal as Record<string, unknown>)
    } finally {
      setValidating(false)
    }
  }

  const handleBacktest = async () => {
    if (!id) return
    const result = await api.startBacktest(id, {
      symbol: cfg.symbol as string,
      timeframe: cfg.timeframe as string,
    })
    setBacktestId(result.backtest_id)
    navigate(`/strategies/${id}/backtest/${result.backtest_id}`)
  }

  if (!strategy) {
    return <p style={{ color: '#848E9C' }}>加载中...</p>
  }

  return (
    <div>
      <button
        onClick={() => navigate('/strategies')}
        className="text-sm mb-4"
        style={{ color: '#848E9C' }}
      >
        ← 返回策略列表
      </button>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: '#EAECEF' }}>
            {strategy.name}
          </h1>
          <p className="text-sm mt-1" style={{ color: '#848E9C' }}>
            {String(cfg.symbol ?? '')} · {String(cfg.timeframe ?? '')} · {strategy.status}
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleValidate}
            disabled={validating}
            className="px-3 py-1.5 rounded text-sm"
            style={{ background: '#2B3139', color: '#EAECEF' }}
          >
            {validating ? '验证中...' : '验证信号'}
          </button>
          <button
            onClick={handleBacktest}
            className="px-3 py-1.5 rounded text-sm"
            style={{ background: '#F0B90B', color: '#0B0E11' }}
          >
            运行回测
          </button>
        </div>
      </div>

      {signal && (
        <div className="rounded-lg p-4 mb-6" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
          <h3 className="font-medium mb-2" style={{ color: '#EAECEF' }}>当前信号</h3>
          <pre className="text-sm overflow-auto" style={{ color: '#848E9C' }}>
            {JSON.stringify(signal, null, 2)}
          </pre>
        </div>
      )}

      <div className="rounded-lg p-4 mb-6" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
        <h3 className="font-medium mb-2" style={{ color: '#EAECEF' }}>策略配置</h3>
        <pre className="text-sm overflow-auto max-h-64" style={{ color: '#848E9C' }}>
          {JSON.stringify(cfg, null, 2)}
        </pre>
      </div>

      <div className="rounded-lg p-4" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
        <h3 className="font-medium mb-3" style={{ color: '#EAECEF' }}>信号历史</h3>
        {(signals ?? []).length === 0 ? (
          <p className="text-sm" style={{ color: '#848E9C' }}>暂无信号记录</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr style={{ color: '#848E9C' }}>
                <th className="text-left py-2">时间</th>
                <th className="text-left">信号</th>
                <th className="text-left">价格</th>
                <th className="text-left">币种</th>
              </tr>
            </thead>
            <tbody>
              {signals!.map((s) => (
                <tr key={s.id} style={{ color: '#EAECEF', borderTop: '1px solid #2B3139' }}>
                  <td className="py-2">{new Date(s.created_at).toLocaleString()}</td>
                  <td>{s.signal}</td>
                  <td>{s.price?.toFixed(2)}</td>
                  <td>{s.symbol}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {backtestId && (
        <p className="text-sm mt-4" style={{ color: '#0ECB81' }}>
          回测已启动: {backtestId}
        </p>
      )}
    </div>
  )
}
