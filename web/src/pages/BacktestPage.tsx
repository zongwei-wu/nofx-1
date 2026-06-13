import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../lib/api'

export function BacktestPage() {
  const { id: strategyId, backtestId } = useParams<{ id: string; backtestId: string }>()
  const navigate = useNavigate()
  const [result, setResult] = useState<Record<string, unknown> | null>(null)
  const [status, setStatus] = useState('pending')

  useEffect(() => {
    if (!backtestId) return
    let cancelled = false
    const poll = async () => {
      const data = await api.getBacktest(backtestId)
      if (cancelled) return
      setStatus(data.status)
      if (data.result) {
        setResult(data.result as Record<string, unknown>)
      }
      if (data.status === 'pending' || data.status === 'running') {
        setTimeout(poll, 2000)
      }
    }
    poll()
    return () => {
      cancelled = true
    }
  }, [backtestId])

  const metrics = (result?.metrics ?? {}) as Record<string, number>
  const trades = (result?.trades ?? []) as Array<Record<string, unknown>>
  const equityCurve = (result?.equity_curve ?? []) as Array<{ time: number; equity: number }>

  return (
    <div>
      <button
        onClick={() => navigate(`/strategies/${strategyId}`)}
        className="text-sm mb-4"
        style={{ color: '#848E9C' }}
      >
        ← 返回策略详情
      </button>

      <h1 className="text-2xl font-bold mb-2" style={{ color: '#EAECEF' }}>
        回测结果
      </h1>
      <p className="text-sm mb-6" style={{ color: '#848E9C' }}>
        状态: {status}
      </p>

      {status === 'done' && result && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            {[
              ['总收益率', `${metrics.total_return_pct?.toFixed(2) ?? '-'}%`],
              ['最大回撤', `${metrics.max_drawdown_pct?.toFixed(2) ?? '-'}%`],
              ['夏普比率', metrics.sharpe_ratio?.toFixed(2) ?? '-'],
              ['胜率', `${metrics.win_rate_pct?.toFixed(2) ?? '-'}%`],
              ['盈亏比', metrics.profit_factor?.toFixed(2) ?? '-'],
              ['总交易', String(metrics.total_trades ?? '-')],
              ['最终权益', String(result.final_equity ?? '-')],
              ['初始资金', String(result.initial_capital ?? '-')],
            ].map(([label, value]) => (
              <div
                key={label}
                className="rounded-lg p-4"
                style={{ background: '#1E2329', border: '1px solid #2B3139' }}
              >
                <p className="text-xs" style={{ color: '#848E9C' }}>{label}</p>
                <p className="text-lg font-semibold mt-1" style={{ color: '#EAECEF' }}>{value}</p>
              </div>
            ))}
          </div>

          {equityCurve.length > 0 && (
            <div className="rounded-lg p-4 mb-6" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <h3 className="font-medium mb-3" style={{ color: '#EAECEF' }}>权益曲线</h3>
              <div className="flex items-end gap-px h-32">
                {equityCurve.slice(-50).map((p, i) => {
                  const max = Math.max(...equityCurve.map((e) => e.equity))
                  const min = Math.min(...equityCurve.map((e) => e.equity))
                  const range = max - min || 1
                  const h = ((p.equity - min) / range) * 100
                  return (
                    <div
                      key={i}
                      className="flex-1"
                      style={{
                        height: `${Math.max(h, 2)}%`,
                        background: p.equity >= (result.initial_capital as number) ? '#0ECB81' : '#F6465D',
                        opacity: 0.8,
                      }}
                    />
                  )
                })}
              </div>
            </div>
          )}

          <div className="rounded-lg p-4" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
            <h3 className="font-medium mb-3" style={{ color: '#EAECEF' }}>交易明细</h3>
            <table className="w-full text-sm">
              <thead>
                <tr style={{ color: '#848E9C' }}>
                  <th className="text-left py-2">方向</th>
                  <th className="text-left">入场价</th>
                  <th className="text-left">出场价</th>
                  <th className="text-left">盈亏</th>
                  <th className="text-left">盈亏%</th>
                </tr>
              </thead>
              <tbody>
                {trades.map((t, i) => (
                  <tr key={i} style={{ color: '#EAECEF', borderTop: '1px solid #2B3139' }}>
                    <td className="py-2">{String(t.side)}</td>
                    <td>{Number(t.entry_price).toFixed(2)}</td>
                    <td>{Number(t.exit_price).toFixed(2)}</td>
                    <td style={{ color: Number(t.pnl) >= 0 ? '#0ECB81' : '#F6465D' }}>
                      {Number(t.pnl).toFixed(2)}
                    </td>
                    <td>{Number(t.pnl_pct).toFixed(2)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {(status === 'pending' || status === 'running') && (
        <p style={{ color: '#F0B90B' }}>回测进行中，请稍候...</p>
      )}
    </div>
  )
}
