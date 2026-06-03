import {
  aggregateOpenPositions,
  openPositionKey,
  type CopyTradePnLRecord,
} from './copyTradePositionUtils'

export type { CopyTradePnLRecord }

function formatTime(ts: number | string | undefined) {
  if (!ts) return '-'
  const d = typeof ts === 'number' ? new Date(ts) : new Date(ts)
  if (Number.isNaN(d.getTime())) return String(ts)
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatDirection(side: string, positionSide: string) {
  const action = side === 'BUY' ? '买入' : '卖出'
  const dir = positionSide === 'LONG' ? '多' : '空'
  return `${action}${dir}`
}

function pnlRate(rec: CopyTradePnLRecord) {
  const notional = rec.avg_price * rec.executed_qty
  if (!notional) return null
  return (rec.total_pnl / notional) * 100
}

function PnlCell({ value }: { value: number }) {
  const color = value >= 0 ? '#0ECB81' : '#F6465D'
  return (
    <span className="font-semibold" style={{ color }}>
      {value >= 0 ? '+' : ''}${value.toFixed(2)}
    </span>
  )
}

function PnlRow({ rec }: { rec: CopyTradePnLRecord }) {
  const coin = rec.symbol?.replace('USDT', '') || rec.symbol
  const rate = pnlRate(rec)
  const notional = rec.avg_price * rec.executed_qty
  const timeLabel = rec.status === 'CLOSED' ? formatTime(rec.close_time) : formatTime(rec.copy_time || rec.lead_order_time)

  return (
    <>
      {/* Desktop table row */}
      <div
        className="hidden md:grid gap-2 p-2.5 rounded text-xs items-center"
        style={{
          background: '#0B0E11',
          gridTemplateColumns: 'minmax(72px,1fr) 48px 56px 72px 88px 88px 88px 72px 72px 88px 56px',
        }}
      >
        <span className="truncate font-semibold" style={{ color: '#EAECEF' }} title={rec.nickname}>
          {rec.nickname}
        </span>
        <span className="font-mono" style={{ color: '#F0B90B' }}>
          {coin}
        </span>
        <span
          className="px-1 py-0.5 rounded font-medium text-center whitespace-nowrap"
          style={{
            background: rec.side === 'BUY' ? 'rgba(14,203,129,0.15)' : 'rgba(246,70,93,0.15)',
            color: rec.side === 'BUY' ? '#0ECB81' : '#F6465D',
          }}
        >
          {formatDirection(rec.side, rec.position_side)}
        </span>
        <span className="text-right" style={{ color: '#848E9C' }}>
          {Number(rec.executed_qty).toFixed(4)}
        </span>
        <span className="font-mono text-right" style={{ color: '#EAECEF' }}>
          ${Number(rec.avg_price).toLocaleString()}
        </span>
        <span className="font-mono text-right" style={{ color: '#5E6673' }}>
          {rec.status === 'CLOSED' ? `$${Number(rec.close_price || 0).toLocaleString()}` : '-'}
        </span>
        <span className="text-right">
          <PnlCell value={rec.total_pnl || 0} />
        </span>
        <span className="text-right" style={{ color: rate !== null && rate >= 0 ? '#0ECB81' : '#F6465D' }}>
          {rate !== null ? `${rate >= 0 ? '+' : ''}${rate.toFixed(2)}%` : '-'}
        </span>
        <span className="text-right" style={{ color: '#848E9C' }}>
          ${notional.toLocaleString(undefined, { maximumFractionDigits: 0 })}
        </span>
        <span className="text-right" style={{ color: '#5E6673' }}>
          {timeLabel}
        </span>
        <span
          className="text-center px-1 py-0.5 rounded text-[10px] font-medium"
          style={{
            background: rec.status === 'OPEN' ? 'rgba(14,203,129,0.15)' : 'rgba(142,142,147,0.15)',
            color: rec.status === 'OPEN' ? '#0ECB81' : '#8E8E93',
          }}
        >
          {rec.status === 'OPEN' ? '持仓' : '已平'}
        </span>
      </div>

      {/* Mobile card */}
      <div className="md:hidden p-3 rounded text-xs space-y-2" style={{ background: '#0B0E11' }}>
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2 min-w-0">
            <span className="font-semibold truncate" style={{ color: '#EAECEF' }}>
              {rec.nickname}
            </span>
            <span className="font-mono" style={{ color: '#F0B90B' }}>
              {coin}
            </span>
            <span
              className="px-1 py-0.5 rounded font-medium whitespace-nowrap"
              style={{
                background: rec.side === 'BUY' ? 'rgba(14,203,129,0.15)' : 'rgba(246,70,93,0.15)',
                color: rec.side === 'BUY' ? '#0ECB81' : '#F6465D',
              }}
            >
              {formatDirection(rec.side, rec.position_side)}
            </span>
          </div>
          <span
            className="px-1.5 py-0.5 rounded text-[10px] font-medium shrink-0"
            style={{
              background: rec.status === 'OPEN' ? 'rgba(14,203,129,0.15)' : 'rgba(142,142,147,0.15)',
              color: rec.status === 'OPEN' ? '#0ECB81' : '#8E8E93',
            }}
          >
            {rec.status === 'OPEN' ? '持仓' : '已平'}
          </span>
        </div>
        <div className="grid grid-cols-2 gap-x-3 gap-y-1">
          <span style={{ color: '#5E6673' }}>数量</span>
          <span className="text-right" style={{ color: '#848E9C' }}>
            {Number(rec.executed_qty).toFixed(4)} 张
          </span>
          <span style={{ color: '#5E6673' }}>开仓价</span>
          <span className="text-right font-mono" style={{ color: '#EAECEF' }}>
            ${Number(rec.avg_price).toLocaleString()}
          </span>
          {rec.status === 'CLOSED' && (
            <>
              <span style={{ color: '#5E6673' }}>平仓价</span>
              <span className="text-right font-mono" style={{ color: '#5E6673' }}>
                ${Number(rec.close_price || 0).toLocaleString()}
              </span>
            </>
          )}
          <span style={{ color: '#5E6673' }}>名义价值</span>
          <span className="text-right" style={{ color: '#848E9C' }}>
            ${notional.toLocaleString(undefined, { maximumFractionDigits: 0 })}
          </span>
          <span style={{ color: '#5E6673' }}>盈亏率</span>
          <span className="text-right" style={{ color: rate !== null && rate >= 0 ? '#0ECB81' : '#F6465D' }}>
            {rate !== null ? `${rate >= 0 ? '+' : ''}${rate.toFixed(2)}%` : '-'}
          </span>
          <span style={{ color: '#5E6673' }}>{rec.status === 'CLOSED' ? '平仓时间' : '跟单时间'}</span>
          <span className="text-right" style={{ color: '#5E6673' }}>
            {timeLabel}
          </span>
        </div>
        <div className="flex items-center justify-between pt-1 border-t" style={{ borderColor: '#1E2329' }}>
          <span style={{ color: '#848E9C' }}>盈亏</span>
          <PnlCell value={rec.total_pnl || 0} />
        </div>
      </div>
    </>
  )
}

function SectionHeader({ label, count, color }: { label: string; count: number; color: string }) {
  return (
    <div className="text-sm font-semibold mb-2 flex items-center gap-2" style={{ color: '#848E9C' }}>
      <span className="w-2 h-2 rounded-full inline-block" style={{ background: color }} />
      {label} ({count})
    </div>
  )
}

export function CopyTradePnLList({ records }: { records?: CopyTradePnLRecord[] | null }) {
  const list = records ?? []
  const openAggregated = aggregateOpenPositions(list)
  const closedRecords = list.filter((r) => r.status === 'CLOSED')

  if (openAggregated.length === 0 && closedRecords.length === 0) {
    return <div className="text-center py-4 text-sm" style={{ color: '#5E6673' }}>暂无盈亏记录</div>
  }

  return (
    <div className="space-y-4">
      {/* Desktop header */}
      <div
        className="hidden md:grid gap-2 px-2.5 pb-1 text-[10px] font-semibold uppercase tracking-wide"
        style={{
          color: '#5E6673',
          gridTemplateColumns: 'minmax(72px,1fr) 48px 56px 72px 88px 88px 88px 72px 72px 88px 56px',
        }}
      >
        <span>交易员</span>
        <span>币种</span>
        <span>方向</span>
        <span className="text-right">数量</span>
        <span className="text-right">开仓价</span>
        <span className="text-right">平仓价</span>
        <span className="text-right">盈亏</span>
        <span className="text-right">盈亏率</span>
        <span className="text-right">名义价值</span>
        <span className="text-right">时间</span>
        <span className="text-center">状态</span>
      </div>

      {openAggregated.length > 0 && (
        <div>
          <SectionHeader label="当前持仓" count={openAggregated.length} color="#0ECB81" />
          <div className="space-y-1.5">
            {openAggregated.map((rec) => (
              <PnlRow key={openPositionKey(rec.symbol, rec.position_side)} rec={rec} />
            ))}
          </div>
        </div>
      )}

      {closedRecords.length > 0 && (
        <div>
          <SectionHeader label="已平仓" count={closedRecords.length} color="#848E9C" />
          <div className="space-y-1.5">
            {closedRecords.map((rec) => (
              <PnlRow key={rec.id} rec={rec} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
