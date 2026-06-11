import {
  BarChart3,
  TrendingDown,
  Trophy,
  ScrollText,
  Lightbulb,
} from 'lucide-react'
import { t, type Language } from '../../../i18n/translations'
import { stripLeadingIcons } from '../../../lib/text'
import type { PerformanceAnalysis, SymbolPerformance, TradeOutcome } from '../types'
import { formatDuration } from '../utils'

interface SymbolAnalysisProps {
  performance: PerformanceAnalysis
  symbolStats: Record<string, SymbolPerformance>
  symbolStatsList: SymbolPerformance[]
  language: Language
}

export function SymbolAnalysis({
  performance,
  symbolStats,
  symbolStatsList,
  language,
}: SymbolAnalysisProps) {
  return (
    <>
      {/* 最佳/最差币种 - 独立行 */}
      {(performance.best_symbol || performance.worst_symbol) && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {performance.best_symbol && (
            <div
              className="rounded-2xl p-6 backdrop-blur-sm"
              style={{
                background:
                  'linear-gradient(135deg, rgba(16, 185, 129, 0.15) 0%, rgba(14, 203, 129, 0.05) 100%)',
                border: '1px solid rgba(16, 185, 129, 0.3)',
                boxShadow: '0 4px 16px rgba(16, 185, 129, 0.1)',
              }}
            >
              <div className="flex items-center gap-2 mb-3">
                <Trophy className="w-6 h-6" style={{ color: '#10B981' }} />
                <span
                  className="text-sm font-semibold"
                  style={{ color: '#6EE7B7' }}
                >
                  {t('bestPerformer', language)}
                </span>
              </div>
              <div
                className="text-3xl font-bold mono mb-1"
                style={{ color: '#10B981' }}
              >
                {performance.best_symbol}
              </div>
              {symbolStats[performance.best_symbol] && (
                <div
                  className="text-lg font-semibold"
                  style={{ color: '#6EE7B7' }}
                >
                  {symbolStats[performance.best_symbol].total_pn_l > 0
                    ? '+'
                    : ''}
                  {symbolStats[performance.best_symbol].total_pn_l.toFixed(2)}{' '}
                  USDT {t('pnl', language)}
                </div>
              )}
            </div>
          )}

          {performance.worst_symbol && (
            <div
              className="rounded-2xl p-6 backdrop-blur-sm"
              style={{
                background:
                  'linear-gradient(135deg, rgba(248, 113, 113, 0.15) 0%, rgba(246, 70, 93, 0.05) 100%)',
                border: '1px solid rgba(248, 113, 113, 0.3)',
                boxShadow: '0 4px 16px rgba(248, 113, 113, 0.1)',
              }}
            >
              <div className="flex items-center gap-2 mb-3">
                <TrendingDown
                  className="w-6 h-6"
                  style={{ color: '#F87171' }}
                />
                <span
                  className="text-sm font-semibold"
                  style={{ color: '#FCA5A5' }}
                >
                  {t('worstPerformer', language)}
                </span>
              </div>
              <div
                className="text-3xl font-bold mono mb-1"
                style={{ color: '#F87171' }}
              >
                {performance.worst_symbol}
              </div>
              {symbolStats[performance.worst_symbol] && (
                <div
                  className="text-lg font-semibold"
                  style={{ color: '#FCA5A5' }}
                >
                  {symbolStats[performance.worst_symbol].total_pn_l > 0
                    ? '+'
                    : ''}
                  {symbolStats[performance.worst_symbol].total_pn_l.toFixed(2)}{' '}
                  USDT {t('pnl', language)}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* 币种表现 & 历史成交 - 左右分屏 2列布局 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 左侧：币种表现统计表格 */}
        {symbolStatsList.length > 0 && (
          <div
            className="rounded-2xl overflow-hidden"
            style={{
              background: 'rgba(30, 35, 41, 0.4)',
              border: '1px solid rgba(99, 102, 241, 0.2)',
              boxShadow: '0 4px 16px rgba(0, 0, 0, 0.2)',
              maxHeight: 'calc(100vh - 200px)',
            }}
          >
            <div
              className="p-5 border-b sticky top-0 z-10"
              style={{
                borderColor: 'rgba(99, 102, 241, 0.2)',
                background: 'rgba(30, 35, 41, 0.95)',
                backdropFilter: 'blur(10px)',
              }}
            >
              <h3
                className="font-bold flex items-center gap-2 text-lg"
                style={{ color: '#E0E7FF' }}
              >
                <BarChart3 className="w-5 h-5" />{' '}
                {stripLeadingIcons(t('symbolPerformance', language))}
              </h3>
            </div>
            <div
              className="overflow-y-auto"
              style={{ maxHeight: 'calc(100vh - 280px)' }}
            >
              <table className="w-full">
                <thead className="sticky top-0 z-10">
                  <tr
                    style={{
                      background: 'rgba(15, 23, 42, 0.95)',
                      backdropFilter: 'blur(10px)',
                    }}
                  >
                    <th
                      className="text-left px-4 py-3 text-xs font-semibold"
                      style={{ color: '#94A3B8' }}
                    >
                      Symbol
                    </th>
                    <th
                      className="text-right px-4 py-3 text-xs font-semibold"
                      style={{ color: '#94A3B8' }}
                    >
                      Trades
                    </th>
                    <th
                      className="text-right px-4 py-3 text-xs font-semibold"
                      style={{ color: '#94A3B8' }}
                    >
                      Win Rate
                    </th>
                    <th
                      className="text-right px-4 py-3 text-xs font-semibold"
                      style={{ color: '#94A3B8' }}
                    >
                      Total P&L (USDT)
                    </th>
                    <th
                      className="text-right px-4 py-3 text-xs font-semibold"
                      style={{ color: '#94A3B8' }}
                    >
                      Avg P&L (USDT)
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {symbolStatsList.map((stat, idx) => (
                    <tr
                      key={stat.symbol}
                      className="transition-colors hover:bg-white/5"
                      style={{
                        borderTop:
                          idx > 0
                            ? '1px solid rgba(99, 102, 241, 0.1)'
                            : 'none',
                      }}
                    >
                      <td className="px-4 py-3">
                        <span
                          className="font-bold mono text-sm"
                          style={{ color: '#E0E7FF' }}
                        >
                          {stat.symbol}
                        </span>
                      </td>
                      <td
                        className="px-4 py-3 text-right mono text-sm"
                        style={{ color: '#CBD5E1' }}
                      >
                        {stat.total_trades}
                      </td>
                      <td
                        className="px-4 py-3 text-right mono text-sm font-semibold"
                        style={{
                          color:
                            (stat.win_rate || 0) >= 50 ? '#10B981' : '#F87171',
                        }}
                      >
                        {(stat.win_rate || 0).toFixed(1)}%
                      </td>
                      <td
                        className="px-4 py-3 text-right mono text-sm font-bold"
                        style={{
                          color:
                            (stat.total_pn_l || 0) > 0 ? '#10B981' : '#F87171',
                        }}
                      >
                        {(stat.total_pn_l || 0) > 0 ? '+' : ''}
                        {(stat.total_pn_l || 0).toFixed(2)}
                      </td>
                      <td
                        className="px-4 py-3 text-right mono text-sm"
                        style={{
                          color:
                            (stat.avg_pn_l || 0) > 0 ? '#10B981' : '#F87171',
                        }}
                      >
                        {(stat.avg_pn_l || 0) > 0 ? '+' : ''}
                        {(stat.avg_pn_l || 0).toFixed(2)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* 右侧：历史成交记录 */}
        <div
          className="rounded-2xl overflow-hidden"
          style={{
            background: 'rgba(30, 35, 41, 0.4)',
            border: '1px solid rgba(240, 185, 11, 0.2)',
            maxHeight: 'calc(100vh - 200px)',
          }}
        >
          <div
            className="p-5 border-b sticky top-0 z-10"
            style={{
              background: 'rgba(240, 185, 11, 0.1)',
              borderColor: 'rgba(240, 185, 11, 0.3)',
              backdropFilter: 'blur(10px)',
            }}
          >
            <div className="flex items-center gap-2">
              <ScrollText className="w-6 h-6" style={{ color: '#FCD34D' }} />
              <div>
                <h3 className="font-bold text-lg" style={{ color: '#FCD34D' }}>
                  {t('tradeHistory', language)}
                </h3>
                <p className="text-xs" style={{ color: '#94A3B8' }}>
                  {performance?.recent_trades &&
                  performance.recent_trades.length > 0
                    ? t('completedTrades', language, {
                        count: performance.recent_trades.length,
                      })
                    : t('completedTradesWillAppear', language)}
                </p>
              </div>
            </div>
          </div>

          <div
            className="overflow-y-auto p-4 space-y-3"
            style={{ maxHeight: 'calc(100vh - 280px)' }}
          >
            {performance?.recent_trades &&
            performance.recent_trades.length > 0 ? (
              performance.recent_trades.map(
                (trade: TradeOutcome, idx: number) => {
                  const isProfitable = trade.pn_l >= 0
                  const isRecent = idx === 0

                  return (
                    <div
                      key={idx}
                      className="rounded-xl p-4 backdrop-blur-sm transition-all hover:scale-[1.02]"
                      style={{
                        background: isRecent
                          ? isProfitable
                            ? 'linear-gradient(135deg, rgba(16, 185, 129, 0.15) 0%, rgba(14, 203, 129, 0.05) 100%)'
                            : 'linear-gradient(135deg, rgba(248, 113, 113, 0.15) 0%, rgba(246, 70, 93, 0.05) 100%)'
                          : 'rgba(30, 35, 41, 0.4)',
                        border: isRecent
                          ? isProfitable
                            ? '1px solid rgba(16, 185, 129, 0.4)'
                            : '1px solid rgba(248, 113, 113, 0.4)'
                          : '1px solid rgba(71, 85, 105, 0.3)',
                        boxShadow: isRecent
                          ? '0 4px 16px rgba(139, 92, 246, 0.2)'
                          : '0 2px 8px rgba(0, 0, 0, 0.1)',
                      }}
                    >
                      <div className="flex items-center justify-between mb-3">
                        <div className="flex items-center gap-2">
                          <span
                            className="text-base font-bold mono"
                            style={{ color: '#E0E7FF' }}
                          >
                            {trade.symbol}
                          </span>
                          <span
                            className="text-xs px-2 py-1 rounded font-bold"
                            style={{
                              background:
                                trade.side === 'long'
                                  ? 'rgba(14, 203, 129, 0.2)'
                                  : 'rgba(246, 70, 93, 0.2)',
                              color:
                                trade.side === 'long' ? '#10B981' : '#F87171',
                            }}
                          >
                            {trade.side.toUpperCase()}
                          </span>
                          {isRecent && (
                            <span
                              className="text-xs px-2 py-0.5 rounded font-semibold"
                              style={{
                                background: 'rgba(240, 185, 11, 0.2)',
                                color: '#FCD34D',
                              }}
                            >
                              {t('latest', language)}
                            </span>
                          )}
                        </div>
                        <div
                          className="text-lg font-bold mono"
                          style={{
                            color: isProfitable ? '#10B981' : '#F87171',
                          }}
                        >
                          {isProfitable ? '+' : ''}
                          {trade.pn_l_pct.toFixed(2)}%
                        </div>
                      </div>

                      <div className="grid grid-cols-2 gap-2 mb-3 text-xs">
                        <div>
                          <div style={{ color: '#94A3B8' }}>
                            {t('entry', language)}
                          </div>
                          <div
                            className="font-mono font-semibold"
                            style={{ color: '#CBD5E1' }}
                          >
                            {trade.open_price.toFixed(4)}
                          </div>
                        </div>
                        <div className="text-right">
                          <div style={{ color: '#94A3B8' }}>
                            {t('exit', language)}
                          </div>
                          <div
                            className="font-mono font-semibold"
                            style={{ color: '#CBD5E1' }}
                          >
                            {trade.close_price.toFixed(4)}
                          </div>
                        </div>
                      </div>

                      {/* Position Details */}
                      <div className="grid grid-cols-2 gap-2 mb-3 text-xs">
                        <div>
                          <div style={{ color: '#94A3B8' }}>Quantity</div>
                          <div
                            className="font-mono font-semibold"
                            style={{ color: '#CBD5E1' }}
                          >
                            {trade.quantity ? trade.quantity.toFixed(4) : '-'}
                          </div>
                        </div>
                        <div className="text-right">
                          <div style={{ color: '#94A3B8' }}>Leverage</div>
                          <div
                            className="font-mono font-semibold"
                            style={{ color: '#FCD34D' }}
                          >
                            {trade.leverage ? `${trade.leverage}x` : '-'}
                          </div>
                        </div>
                        <div>
                          <div style={{ color: '#94A3B8' }}>Position Value</div>
                          <div
                            className="font-mono font-semibold"
                            style={{ color: '#CBD5E1' }}
                          >
                            {trade.position_value
                              ? `$${trade.position_value.toFixed(2)}`
                              : '-'}
                          </div>
                        </div>
                        <div className="text-right">
                          <div style={{ color: '#94A3B8' }}>Margin Used</div>
                          <div
                            className="font-mono font-semibold"
                            style={{ color: '#A78BFA' }}
                          >
                            {trade.margin_used
                              ? `$${trade.margin_used.toFixed(2)}`
                              : '-'}
                          </div>
                        </div>
                      </div>

                      <div
                        className="rounded-lg p-2 mb-2"
                        style={{
                          background: isProfitable
                            ? 'rgba(16, 185, 129, 0.1)'
                            : 'rgba(248, 113, 113, 0.1)',
                        }}
                      >
                        <div className="flex items-center justify-between text-xs">
                          <span style={{ color: '#94A3B8' }}>P&L</span>
                          <span
                            className="font-bold mono"
                            style={{
                              color: isProfitable ? '#10B981' : '#F87171',
                            }}
                          >
                            {isProfitable ? '+' : ''}
                            {trade.pn_l.toFixed(2)} USDT
                          </span>
                        </div>
                      </div>

                      <div
                        className="flex items-center justify-between text-xs"
                        style={{ color: '#94A3B8' }}
                      >
                        <span>⏱️ {formatDuration(trade.duration)}</span>
                        {trade.was_stop_loss && (
                          <span
                            className="px-2 py-0.5 rounded font-semibold"
                            style={{
                              background: 'rgba(248, 113, 113, 0.2)',
                              color: '#FCA5A5',
                            }}
                          >
                            {t('stopLoss', language)}
                          </span>
                        )}
                      </div>

                      <div
                        className="text-xs mt-2 pt-2 border-t"
                        style={{
                          color: '#64748B',
                          borderColor: 'rgba(71, 85, 105, 0.3)',
                        }}
                      >
                        {new Date(trade.close_time).toLocaleString('en-US', {
                          month: 'short',
                          day: '2-digit',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </div>
                    </div>
                  )
                }
              )
            ) : (
              <div className="p-6 text-center">
                <div className="mb-2 flex justify-center opacity-50">
                  <ScrollText
                    className="w-10 h-10"
                    style={{ color: '#94A3B8' }}
                  />
                </div>
                <div style={{ color: '#94A3B8' }}>
                  {t('noCompletedTrades', language)}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* AI学习说明 - 现代化设计 */}
      <div
        className="rounded-2xl p-6 backdrop-blur-sm"
        style={{
          background:
            'linear-gradient(135deg, rgba(240, 185, 11, 0.1) 0%, rgba(252, 213, 53, 0.05) 100%)',
          border: '1px solid rgba(240, 185, 11, 0.2)',
          boxShadow: '0 4px 16px rgba(240, 185, 11, 0.1)',
        }}
      >
        <div className="flex items-start gap-4">
          <div
            className="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0"
            style={{
              background: 'rgba(240, 185, 11, 0.2)',
              border: '1px solid rgba(240, 185, 11, 0.3)',
            }}
          >
            <Lightbulb className="w-5 h-5" style={{ color: '#FCD34D' }} />
          </div>
          <div>
            <h3
              className="font-bold mb-3 text-base"
              style={{ color: '#FCD34D' }}
            >
              {stripLeadingIcons(t('howAILearns', language))}
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 text-sm">
              <div className="flex items-start gap-2">
                <span style={{ color: '#F0B90B' }}>•</span>
                <span style={{ color: '#CBD5E1' }}>
                  {t('aiLearningPoint1', language)}
                </span>
              </div>
              <div className="flex items-start gap-2">
                <span style={{ color: '#F0B90B' }}>•</span>
                <span style={{ color: '#CBD5E1' }}>
                  {t('aiLearningPoint2', language)}
                </span>
              </div>
              <div className="flex items-start gap-2">
                <span style={{ color: '#F0B90B' }}>•</span>
                <span style={{ color: '#CBD5E1' }}>
                  {t('aiLearningPoint3', language)}
                </span>
              </div>
              <div className="flex items-start gap-2">
                <span style={{ color: '#F0B90B' }}>•</span>
                <span style={{ color: '#CBD5E1' }}>
                  {t('aiLearningPoint4', language)}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
