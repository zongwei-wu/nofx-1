import type { CopyTradePnLRecord } from './copyTradePositionUtils'
import {
  formatRecordDirection,
  formatRecordTimeShort,
  recordStatusLabel,
  sortCopyTradeRecords,
} from './copyTradeRecordUtils'

const DESKTOP_GRID =
  'minmax(72px,1fr) 40px 44px 64px 72px 72px 88px 44px minmax(80px,1fr)'

function StatusBadge({ status }: { status: string }) {
  const isOpen = status === 'OPEN'
  const isClosed = status === 'CLOSED'
  return (
    <span
      className="px-1 py-0.5 rounded text-[10px] font-medium whitespace-nowrap"
      style={{
        background: isOpen
          ? 'rgba(14,203,129,0.15)'
          : isClosed
            ? 'rgba(142,142,147,0.15)'
            : 'rgba(246,70,93,0.15)',
        color: isOpen ? '#0ECB81' : isClosed ? '#8E8E93' : '#F6465D',
      }}
    >
      {recordStatusLabel(status)}
    </span>
  )
}

function DirectionBadge({ side, positionSide }: { side: string; positionSide: string }) {
  return (
    <span
      className="px-1 py-0.5 rounded font-medium text-center whitespace-nowrap text-[10px]"
      style={{
        background: side === 'BUY' ? 'rgba(14,203,129,0.15)' : 'rgba(246,70,93,0.15)',
        color: side === 'BUY' ? '#0ECB81' : '#F6465D',
      }}
    >
      {formatRecordDirection(side, positionSide)}
    </span>
  )
}

function RecordRow({ rec }: { rec: CopyTradePnLRecord }) {
  const coin = rec.symbol?.replace('USDT', '') || rec.symbol
  const showError = rec.status === 'FAILED' && rec.error_message

  return (
    <>
      {/* Desktop */}
      <div
        className="hidden md:grid gap-2 py-1.5 px-2 text-xs items-center"
        style={{ gridTemplateColumns: DESKTOP_GRID }}
      >
        <span className="truncate font-medium" style={{ color: '#EAECEF' }} title={rec.nickname}>
          {rec.nickname || '未知'}
        </span>
        <span className="font-mono" style={{ color: '#F0B90B' }}>
          {coin}
        </span>
        <DirectionBadge side={rec.side} positionSide={rec.position_side} />
        <span className="text-right tabular-nums" style={{ color: '#848E9C' }}>
          {Number(rec.executed_qty).toFixed(4)}
        </span>
        <span className="font-mono text-right tabular-nums" style={{ color: '#EAECEF' }}>
          ${Number(rec.avg_price).toLocaleString(undefined, { maximumFractionDigits: 2 })}
        </span>
        <span className="font-mono text-right tabular-nums" style={{ color: '#5E6673' }}>
          {rec.status === 'CLOSED'
            ? `$${Number(rec.close_price || 0).toLocaleString(undefined, { maximumFractionDigits: 2 })}`
            : '-'}
        </span>
        <span className="text-right tabular-nums" style={{ color: '#5E6673' }} title="带单员下单时间">
          {formatRecordTimeShort(rec.lead_order_time)}
        </span>
        <span className="text-center">
          <StatusBadge status={rec.status} />
        </span>
        <span
          className="truncate text-[10px]"
          style={{ color: '#F6465D' }}
          title={showError ? rec.error_message : undefined}
        >
          {showError ? rec.error_message : ''}
        </span>
      </div>

      {/* Mobile */}
      <div
        className="md:hidden py-1.5 px-2 text-xs space-y-0.5"
        style={{ borderBottom: '1px solid #1E2329' }}
      >
        <div className="flex items-center justify-between gap-2 min-w-0">
          <div className="flex items-center gap-1.5 min-w-0 flex-1">
            <span className="truncate font-medium shrink" style={{ color: '#EAECEF' }}>
              {rec.nickname || '未知'}
            </span>
            <span className="font-mono shrink-0" style={{ color: '#F0B90B' }}>
              {coin}
            </span>
            <DirectionBadge side={rec.side} positionSide={rec.position_side} />
          </div>
          <StatusBadge status={rec.status} />
        </div>
        <div className="flex items-center justify-between gap-2 text-[10px] tabular-nums">
          <span style={{ color: '#848E9C' }}>
            {Number(rec.executed_qty).toFixed(4)}张 · 开$
            {Number(rec.avg_price).toLocaleString(undefined, { maximumFractionDigits: 0 })}
            {rec.status === 'CLOSED' && (
              <>
                {' '}
                → 平$
                {Number(rec.close_price || 0).toLocaleString(undefined, { maximumFractionDigits: 0 })}
              </>
            )}
          </span>
          <span style={{ color: '#5E6673' }}>{formatRecordTimeShort(rec.lead_order_time)}</span>
        </div>
        {showError && (
          <div className="text-[10px] truncate" style={{ color: '#F6465D' }} title={rec.error_message}>
            {rec.error_message}
          </div>
        )}
      </div>
    </>
  )
}

export interface CopyTradeRecordsTableProps {
  records: CopyTradePnLRecord[]
  emptyMessage?: string
}

export function CopyTradeRecordsTable({
  records,
  emptyMessage = '该分类下暂无记录',
}: CopyTradeRecordsTableProps) {
  const sorted = sortCopyTradeRecords(records)

  if (sorted.length === 0) {
    return (
      <div className="text-center py-6 text-sm" style={{ color: '#5E6673' }}>
        {emptyMessage}
      </div>
    )
  }

  return (
    <div
      className="rounded-lg overflow-hidden"
      style={{ background: '#1E2329', border: '1px solid #2B3139' }}
    >
      <div
        className="hidden md:grid gap-2 px-2 py-1.5 text-[10px] font-semibold uppercase tracking-wide"
        style={{ color: '#5E6673', gridTemplateColumns: DESKTOP_GRID, background: '#0B0E11' }}
      >
        <span>带单员</span>
        <span>币种</span>
        <span>方向</span>
        <span className="text-right">数量</span>
        <span className="text-right">开仓价</span>
        <span className="text-right">平仓价</span>
        <span className="text-right">带单时间</span>
        <span className="text-center">状态</span>
        <span>失败原因</span>
      </div>
      <div className="divide-y divide-[#2B3139] md:divide-[#1E2329]">
        {sorted.map((rec) => (
          <RecordRow key={rec.id} rec={rec} />
        ))}
      </div>
    </div>
  )
}
