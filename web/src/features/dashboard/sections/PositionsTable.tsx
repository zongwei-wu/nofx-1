import { PieChart, TrendingUp } from 'lucide-react'
import type { Position } from '../../../types'
import { t, type Language } from '../../../i18n/translations'

interface PositionsTableProps {
  positions?: Position[]
  sortedPositions: Position[]
  language: Language
}

export function PositionsTable({
  positions,
  sortedPositions,
  language,
}: PositionsTableProps) {
  return (
    <div
      className="binance-card p-6 animate-slide-in"
      style={{ animationDelay: '0.1s' }}
    >
      <div className="flex items-center justify-between mb-5">
        <h2
          className="text-xl font-bold flex items-center gap-2"
          style={{ color: '#EAECEF' }}
        >
          <TrendingUp className="w-5 h-5" style={{ color: '#F0B90B' }} />
          {t('currentPositions', language)}
        </h2>
        {positions && positions.length > 0 && (
          <div
            className="text-xs px-3 py-1 rounded"
            style={{
              background: 'rgba(240, 185, 11, 0.1)',
              color: '#F0B90B',
              border: '1px solid rgba(240, 185, 11, 0.2)',
            }}
          >
            {positions.length} {t('active', language)}
          </div>
        )}
      </div>
      {positions && positions.length > 0 ? (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-left border-b border-gray-800">
              <tr>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('symbol', language)}
                </th>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('side', language)}
                </th>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('entryPrice', language)}
                </th>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('markPrice', language)}
                </th>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('quantity', language)}
                </th>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('positionValue', language)}
                </th>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('leverage', language)}
                </th>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('unrealizedPnL', language)}
                </th>
                <th className="pb-3 font-semibold text-gray-400">
                  {t('liqPrice', language)}
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedPositions.map((pos, i) => (
                <tr
                  key={i}
                  className="border-b border-gray-800 last:border-0"
                >
                  <td className="py-3 font-mono font-semibold">
                    {pos.symbol}
                  </td>
                  <td className="py-3">
                    <span
                      className="px-2 py-1 rounded text-xs font-bold"
                      style={
                        pos.side === 'long'
                          ? {
                              background: 'rgba(14, 203, 129, 0.1)',
                              color: '#0ECB81',
                            }
                          : {
                              background: 'rgba(246, 70, 93, 0.1)',
                              color: '#F6465D',
                            }
                      }
                    >
                      {t(pos.side === 'long' ? 'long' : 'short', language)}
                    </span>
                  </td>
                  <td
                    className="py-3 font-mono"
                    style={{ color: '#EAECEF' }}
                  >
                    {pos.entry_price.toFixed(4)}
                  </td>
                  <td
                    className="py-3 font-mono"
                    style={{ color: '#EAECEF' }}
                  >
                    {pos.mark_price.toFixed(4)}
                  </td>
                  <td
                    className="py-3 font-mono"
                    style={{ color: '#EAECEF' }}
                  >
                    {pos.quantity.toFixed(4)}
                  </td>
                  <td
                    className="py-3 font-mono font-bold"
                    style={{ color: '#EAECEF' }}
                  >
                    {(pos.quantity * pos.mark_price).toFixed(2)} USDT
                  </td>
                  <td
                    className="py-3 font-mono"
                    style={{ color: '#F0B90B' }}
                  >
                    {pos.leverage}x
                  </td>
                  <td className="py-3 font-mono">
                    <span
                      style={{
                        color:
                          pos.unrealized_pnl >= 0 ? '#0ECB81' : '#F6465D',
                        fontWeight: 'bold',
                      }}
                    >
                      {pos.unrealized_pnl >= 0 ? '+' : ''}
                      {pos.unrealized_pnl.toFixed(2)} (
                      {pos.unrealized_pnl_pct.toFixed(2)}%)
                    </span>
                  </td>
                  <td
                    className="py-3 font-mono"
                    style={{ color: '#848E9C' }}
                  >
                    {pos.liquidation_price.toFixed(4)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="text-center py-16" style={{ color: '#848E9C' }}>
          <div className="mb-4 opacity-50 flex justify-center">
            <PieChart className="w-16 h-16" />
          </div>
          <div className="text-lg font-semibold mb-2">
            {t('noPositions', language)}
          </div>
          <div className="text-sm">{t('noActivePositions', language)}</div>
        </div>
      )}
    </div>
  )
}
