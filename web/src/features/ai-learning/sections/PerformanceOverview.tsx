import {
  Brain,
  BarChart3,
  TrendingUp,
  TrendingDown,
  Sparkles,
  Coins,
} from 'lucide-react'
import { t, type Language } from '../../../i18n/translations'
import type { PerformanceAnalysis } from '../types'

interface PerformanceOverviewProps {
  performance: PerformanceAnalysis
  language: Language
}

export function PerformanceOverview({
  performance,
  language,
}: PerformanceOverviewProps) {
  return (
    <>
      {/* 标题区 - 优化设计 */}
      <div
        className="relative rounded-2xl p-6 overflow-hidden"
        style={{
          background:
            'linear-gradient(135deg, rgba(139, 92, 246, 0.15) 0%, rgba(99, 102, 241, 0.1) 50%, rgba(30, 35, 41, 0.8) 100%)',
          border: '1px solid rgba(139, 92, 246, 0.3)',
          boxShadow: '0 8px 32px rgba(139, 92, 246, 0.2)',
        }}
      >
        <div
          className="absolute top-0 right-0 w-96 h-96 rounded-full opacity-10"
          style={{
            background: 'radial-gradient(circle, #8B5CF6 0%, transparent 70%)',
            filter: 'blur(60px)',
          }}
        />
        <div className="relative flex items-center gap-4">
          <div
            className="w-16 h-16 rounded-2xl flex items-center justify-center"
            style={{
              background: 'linear-gradient(135deg, #8B5CF6 0%, #6366F1 100%)',
              boxShadow: '0 8px 24px rgba(139, 92, 246, 0.5)',
              border: '2px solid rgba(255, 255, 255, 0.1)',
            }}
          >
            <Brain className="w-8 h-8" style={{ color: '#FFF' }} />
          </div>
          <div>
            <h2
              className="text-3xl font-bold mb-1"
              style={{
                color: '#EAECEF',
                textShadow: '0 2px 8px rgba(139, 92, 246, 0.3)',
              }}
            >
              {t('aiLearning', language)}
            </h2>
            <p className="text-base" style={{ color: '#A78BFA' }}>
              {t('tradesAnalyzed', language, {
                count: performance.total_trades,
              })}
            </p>
          </div>
        </div>
      </div>

      {/* 核心指标卡片 - 4列网格 */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {/* 总交易数 */}
        <div
          className="rounded-2xl p-5 relative overflow-hidden group hover:scale-105 transition-transform"
          style={{
            background:
              'linear-gradient(135deg, rgba(99, 102, 241, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)',
            border: '1px solid rgba(99, 102, 241, 0.3)',
            boxShadow: '0 4px 16px rgba(99, 102, 241, 0.2)',
          }}
        >
          <div
            className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, #6366F1 0%, transparent 70%)',
              filter: 'blur(20px)',
            }}
          />
          <div className="relative">
            <div
              className="text-xs font-semibold mb-3 uppercase tracking-wider"
              style={{ color: '#A5B4FC' }}
            >
              {t('totalTrades', language)}
            </div>
            <div
              className="text-4xl font-bold mono mb-1"
              style={{ color: '#E0E7FF' }}
            >
              {performance.total_trades}
            </div>
            <div
              className="text-xs flex items-center gap-1"
              style={{ color: '#6366F1' }}
            >
              <BarChart3 className="w-3 h-3" /> Trades
            </div>
          </div>
        </div>

        {/* 胜率 */}
        <div
          className="rounded-2xl p-5 relative overflow-hidden group hover:scale-105 transition-transform"
          style={{
            background:
              (performance.win_rate || 0) >= 50
                ? 'linear-gradient(135deg, rgba(16, 185, 129, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)'
                : 'linear-gradient(135deg, rgba(248, 113, 113, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)',
            border: `1px solid ${(performance.win_rate || 0) >= 50 ? 'rgba(16, 185, 129, 0.4)' : 'rgba(248, 113, 113, 0.4)'}`,
            boxShadow: `0 4px 16px ${(performance.win_rate || 0) >= 50 ? 'rgba(16, 185, 129, 0.2)' : 'rgba(248, 113, 113, 0.2)'}`,
          }}
        >
          <div
            className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-20"
            style={{
              background: `radial-gradient(circle, ${(performance.win_rate || 0) >= 50 ? '#10B981' : '#F87171'} 0%, transparent 70%)`,
              filter: 'blur(20px)',
            }}
          />
          <div className="relative">
            <div
              className="text-xs font-semibold mb-3 uppercase tracking-wider"
              style={{
                color:
                  (performance.win_rate || 0) >= 50 ? '#6EE7B7' : '#FCA5A5',
              }}
            >
              {t('winRate', language)}
            </div>
            <div
              className="text-4xl font-bold mono mb-1"
              style={{
                color:
                  (performance.win_rate || 0) >= 50 ? '#10B981' : '#F87171',
              }}
            >
              {(performance.win_rate || 0).toFixed(1)}%
            </div>
            <div className="text-xs" style={{ color: '#94A3B8' }}>
              {performance.winning_trades || 0}W /{' '}
              {performance.losing_trades || 0}L
            </div>
          </div>
        </div>

        {/* 平均盈利 */}
        <div
          className="rounded-2xl p-5 relative overflow-hidden group hover:scale-105 transition-transform"
          style={{
            background:
              'linear-gradient(135deg, rgba(14, 203, 129, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)',
            border: '1px solid rgba(14, 203, 129, 0.3)',
            boxShadow: '0 4px 16px rgba(14, 203, 129, 0.2)',
          }}
        >
          <div
            className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, #0ECB81 0%, transparent 70%)',
              filter: 'blur(20px)',
            }}
          />
          <div className="relative">
            <div
              className="text-xs font-semibold mb-3 uppercase tracking-wider"
              style={{ color: '#6EE7B7' }}
            >
              {t('avgWin', language)}
            </div>
            <div
              className="text-4xl font-bold mono mb-1"
              style={{ color: '#10B981' }}
            >
              +{(performance.avg_win || 0).toFixed(2)}
            </div>
            <div
              className="text-xs flex items-center gap-1"
              style={{ color: '#6EE7B7' }}
            >
              <TrendingUp className="w-3 h-3" /> USDT Average
            </div>
          </div>
        </div>

        {/* 平均亏损 */}
        <div
          className="rounded-2xl p-5 relative overflow-hidden group hover:scale-105 transition-transform"
          style={{
            background:
              'linear-gradient(135deg, rgba(246, 70, 93, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)',
            border: '1px solid rgba(246, 70, 93, 0.3)',
            boxShadow: '0 4px 16px rgba(246, 70, 93, 0.2)',
          }}
        >
          <div
            className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, #F6465D 0%, transparent 70%)',
              filter: 'blur(20px)',
            }}
          />
          <div className="relative">
            <div
              className="text-xs font-semibold mb-3 uppercase tracking-wider"
              style={{ color: '#FCA5A5' }}
            >
              {t('avgLoss', language)}
            </div>
            <div
              className="text-4xl font-bold mono mb-1"
              style={{ color: '#F87171' }}
            >
              {(performance.avg_loss || 0).toFixed(2)}
            </div>
            <div
              className="text-xs flex items-center gap-1"
              style={{ color: '#FCA5A5' }}
            >
              <TrendingDown className="w-3 h-3" /> USDT Average
            </div>
          </div>
        </div>
      </div>

      {/* 关键指标：夏普比率 & 盈亏比 - 2列网格 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 夏普比率 */}
        <div
          className="rounded-2xl p-6 relative overflow-hidden"
          style={{
            background:
              'linear-gradient(135deg, rgba(139, 92, 246, 0.25) 0%, rgba(99, 102, 241, 0.15) 50%, rgba(30, 35, 41, 0.9) 100%)',
            border: '2px solid rgba(139, 92, 246, 0.5)',
            boxShadow: '0 12px 40px rgba(139, 92, 246, 0.3)',
          }}
        >
          <div
            className="absolute top-0 right-0 w-48 h-48 rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, #8B5CF6 0%, transparent 70%)',
              filter: 'blur(40px)',
            }}
          />
          <div className="relative">
            <div className="flex items-center gap-3 mb-4">
              <div
                className="w-12 h-12 rounded-xl flex items-center justify-center"
                style={{
                  background: 'rgba(139, 92, 246, 0.3)',
                  border: '1px solid rgba(139, 92, 246, 0.5)',
                }}
              >
                <Sparkles className="w-6 h-6" style={{ color: '#A78BFA' }} />
              </div>
              <div>
                <div className="text-lg font-bold" style={{ color: '#C4B5FD' }}>
                  夏普比率
                </div>
                <div className="text-xs" style={{ color: '#94A3B8' }}>
                  风险调整后收益 · AI自我进化指标
                </div>
              </div>
            </div>

            <div className="flex items-end justify-between mb-4">
              <div
                className="text-6xl font-bold mono"
                style={{
                  color:
                    (performance.sharpe_ratio || 0) >= 2
                      ? '#10B981'
                      : (performance.sharpe_ratio || 0) >= 1
                        ? '#22D3EE'
                        : (performance.sharpe_ratio || 0) >= 0
                          ? '#F0B90B'
                          : '#F87171',
                  textShadow: '0 4px 12px rgba(0, 0, 0, 0.3)',
                }}
              >
                {performance.sharpe_ratio
                  ? performance.sharpe_ratio.toFixed(2)
                  : 'N/A'}
              </div>

              {performance.sharpe_ratio !== undefined && (
                <div className="text-right mb-2">
                  <div
                    className="text-sm font-bold px-3 py-1 rounded-lg"
                    style={{
                      color:
                        (performance.sharpe_ratio || 0) >= 2
                          ? '#10B981'
                          : (performance.sharpe_ratio || 0) >= 1
                            ? '#22D3EE'
                            : (performance.sharpe_ratio || 0) >= 0
                              ? '#F0B90B'
                              : '#F87171',
                      background:
                        (performance.sharpe_ratio || 0) >= 2
                          ? 'rgba(16, 185, 129, 0.2)'
                          : (performance.sharpe_ratio || 0) >= 1
                            ? 'rgba(34, 211, 238, 0.2)'
                            : (performance.sharpe_ratio || 0) >= 0
                              ? 'rgba(240, 185, 11, 0.2)'
                              : 'rgba(248, 113, 113, 0.2)',
                    }}
                  >
                    {performance.sharpe_ratio >= 2
                      ? '🟢 卓越表现'
                      : performance.sharpe_ratio >= 1
                        ? '🟢 良好表现'
                        : performance.sharpe_ratio >= 0
                          ? '🟡 波动较大'
                          : '🔴 需要调整'}
                  </div>
                </div>
              )}
            </div>

            {performance.sharpe_ratio !== undefined && (
              <div
                className="rounded-xl p-4"
                style={{
                  background: 'rgba(0, 0, 0, 0.4)',
                  border: '1px solid rgba(139, 92, 246, 0.3)',
                }}
              >
                <div
                  className="text-sm leading-relaxed"
                  style={{ color: '#DDD6FE' }}
                >
                  {performance.sharpe_ratio >= 2 &&
                    '✨ AI策略非常有效！风险调整后收益优异，可适度扩大仓位但保持纪律。'}
                  {performance.sharpe_ratio >= 1 &&
                    performance.sharpe_ratio < 2 &&
                    '✅ 策略表现稳健，风险收益平衡良好，继续保持当前策略。'}
                  {performance.sharpe_ratio >= 0 &&
                    performance.sharpe_ratio < 1 &&
                    '⚠️ 收益为正但波动较大，AI正在优化策略，降低风险。'}
                  {performance.sharpe_ratio < 0 &&
                    '🚨 当前策略需要调整！AI已自动进入保守模式，减少仓位和交易频率。'}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 盈亏比 */}
        <div
          className="rounded-2xl p-6 relative overflow-hidden"
          style={{
            background:
              'linear-gradient(135deg, rgba(240, 185, 11, 0.25) 0%, rgba(252, 213, 53, 0.15) 50%, rgba(30, 35, 41, 0.9) 100%)',
            border: '2px solid rgba(240, 185, 11, 0.5)',
            boxShadow: '0 12px 40px rgba(240, 185, 11, 0.3)',
          }}
        >
          <div
            className="absolute top-0 right-0 w-48 h-48 rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, #F0B90B 0%, transparent 70%)',
              filter: 'blur(40px)',
            }}
          />
          <div className="relative">
            <div className="flex items-center gap-3 mb-4">
              <div
                className="w-12 h-12 rounded-xl flex items-center justify-center"
                style={{
                  background: 'rgba(240, 185, 11, 0.3)',
                  border: '1px solid rgba(240, 185, 11, 0.5)',
                }}
              >
                <Coins className="w-6 h-6" style={{ color: '#FCD34D' }} />
              </div>
              <div>
                <div className="text-lg font-bold" style={{ color: '#FCD34D' }}>
                  {t('profitFactor', language)}
                </div>
                <div className="text-xs" style={{ color: '#94A3B8' }}>
                  {t('avgWinDivLoss', language)}
                </div>
              </div>
            </div>

            <div className="flex items-end justify-between mb-4">
              <div
                className="text-6xl font-bold mono"
                style={{
                  color:
                    (performance.profit_factor || 0) >= 2.0
                      ? '#10B981'
                      : (performance.profit_factor || 0) >= 1.5
                        ? '#F0B90B'
                        : (performance.profit_factor || 0) >= 1.0
                          ? '#FB923C'
                          : '#F87171',
                  textShadow: '0 4px 12px rgba(0, 0, 0, 0.3)',
                }}
              >
                {(performance.profit_factor || 0) > 0
                  ? (performance.profit_factor || 0).toFixed(2)
                  : 'N/A'}
              </div>

              <div className="text-right mb-2">
                <div
                  className="text-sm font-bold px-3 py-1 rounded-lg"
                  style={{
                    color:
                      (performance.profit_factor || 0) >= 2.0
                        ? '#10B981'
                        : (performance.profit_factor || 0) >= 1.5
                          ? '#F0B90B'
                          : '#94A3B8',
                    background:
                      (performance.profit_factor || 0) >= 2.0
                        ? 'rgba(16, 185, 129, 0.2)'
                        : (performance.profit_factor || 0) >= 1.5
                          ? 'rgba(240, 185, 11, 0.2)'
                          : 'rgba(148, 163, 184, 0.2)',
                  }}
                >
                  {(performance.profit_factor || 0) >= 2.0 &&
                    t('excellent', language)}
                  {(performance.profit_factor || 0) >= 1.5 &&
                    (performance.profit_factor || 0) < 2.0 &&
                    t('good', language)}
                  {(performance.profit_factor || 0) >= 1.0 &&
                    (performance.profit_factor || 0) < 1.5 &&
                    t('fair', language)}
                  {(performance.profit_factor || 0) > 0 &&
                    (performance.profit_factor || 0) < 1.0 &&
                    t('poor', language)}
                </div>
              </div>
            </div>

            <div
              className="rounded-xl p-4"
              style={{
                background: 'rgba(0, 0, 0, 0.4)',
                border: '1px solid rgba(240, 185, 11, 0.3)',
              }}
            >
              <div
                className="text-sm leading-relaxed"
                style={{ color: '#FEF3C7' }}
              >
                {(performance.profit_factor || 0) >= 2.0 &&
                  '🔥 盈利能力出色！每亏1元能赚' +
                    (performance.profit_factor || 0).toFixed(1) +
                    '元，AI策略表现优异。'}
                {(performance.profit_factor || 0) >= 1.5 &&
                  (performance.profit_factor || 0) < 2.0 &&
                  '✓ 策略稳定盈利，盈亏比健康，继续保持纪律性交易。'}
                {(performance.profit_factor || 0) >= 1.0 &&
                  (performance.profit_factor || 0) < 1.5 &&
                  '⚠️ 策略略有盈利但需优化，AI正在调整仓位和止损策略。'}
                {(performance.profit_factor || 0) > 0 &&
                  (performance.profit_factor || 0) < 1.0 &&
                  '❌ 平均亏损大于盈利，需要调整策略或降低交易频率。'}
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
