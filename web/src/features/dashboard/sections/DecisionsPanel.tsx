import { useState } from 'react'
import {
  AlertTriangle,
  Brain,
  Check,
  Inbox,
  Send,
  X,
  XCircle,
} from 'lucide-react'
import type { DecisionRecord } from '../../../types'
import { t, type Language } from '../../../i18n/translations'
import { stripLeadingIcons } from '../../../lib/text'

function copyTradeOutcomeLabel(
  actionTaken: string | undefined,
  language: Language
): string {
  if (!actionTaken) return ''
  const zh: Record<string, string> = {
    copied_open: '已执行跟单',
    open_failed: '跟单下单失败',
    ai_error: 'AI 分析失败',
    parse_failed: 'AI 解析失败',
    ai_rejected: 'AI 拒绝跟单',
    pending_confirm: '待确认跟单',
    pending_open: '待开仓',
  }
  const en: Record<string, string> = {
    copied_open: 'Copied',
    open_failed: 'Order failed',
    ai_error: 'AI error',
    parse_failed: 'Parse failed',
    ai_rejected: 'AI rejected',
    pending_confirm: 'Pending confirm',
    pending_open: 'Pending open',
  }
  const map = language === 'zh' ? zh : en
  return map[actionTaken] || actionTaken
}

function DecisionCard({
  decision,
  language,
}: {
  decision: DecisionRecord
  language: Language
}) {
  const [showInputPrompt, setShowInputPrompt] = useState(false)
  const [showCoT, setShowCoT] = useState(false)

  return (
    <div
      className="rounded p-5 transition-all duration-300 hover:translate-y-[-2px]"
      style={{
        border: '1px solid #2B3139',
        background: '#1E2329',
        boxShadow: '0 2px 8px rgba(0, 0, 0, 0.3)',
      }}
    >
      <div className="flex items-start justify-between mb-3">
        <div>
          {decision.source === 'copy_trade' ? (
            <div className="font-semibold" style={{ color: '#EAECEF' }}>
              {language === 'zh' ? '跟单风控' : 'Copy Trade Risk'}
              {decision.copy_trade_meta?.nickname && (
                <span style={{ color: '#F0B90B' }}>
                  {' '}
                  · {decision.copy_trade_meta.nickname}
                </span>
              )}
              {decision.copy_trade_meta?.lead_operation && (
                <span style={{ color: '#38BDF8' }}>
                  {' '}
                  · {language === 'zh' ? '跟随' : 'Follow'}{' '}
                  {decision.copy_trade_meta.lead_operation}
                </span>
              )}
              {decision.copy_trade_meta?.ai_trader_name && (
                <span style={{ color: '#848E9C' }}>
                  {' '}
                  · {decision.copy_trade_meta.ai_trader_name}
                </span>
              )}
              {decision.decisions?.[0]?.symbol && (
                <span
                  className="font-mono text-sm ml-1"
                  style={{ color: '#848E9C' }}
                >
                  {decision.decisions[0].symbol.replace('USDT', '')}
                </span>
              )}
            </div>
          ) : (
            <div className="font-semibold" style={{ color: '#EAECEF' }}>
              {t('cycle', language)} #{decision.cycle_number}
            </div>
          )}
          <div className="text-xs mt-0.5" style={{ color: '#848E9C' }}>
            {new Date(decision.timestamp).toLocaleString()}
            {decision.source === 'copy_trade' &&
              decision.copy_trade_meta?.action_taken && (
                <span className="ml-2" style={{ color: '#5E6673' }}>
                  {copyTradeOutcomeLabel(
                    decision.copy_trade_meta.action_taken,
                    language
                  )}
                </span>
              )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {decision.source === 'copy_trade' &&
            decision.copy_trade_meta?.lead_operation && (
              <span
                className="px-2 py-0.5 rounded text-[10px] font-bold"
                style={{
                  background: 'rgba(56, 189, 248, 0.15)',
                  color: '#38BDF8',
                }}
              >
                {language === 'zh' ? '跟随' : 'Follow'}{' '}
                {decision.copy_trade_meta.lead_operation}
              </span>
            )}
          {decision.source === 'copy_trade' && (
            <span
              className="px-2 py-0.5 rounded text-[10px] font-bold"
              style={{
                background: 'rgba(240, 185, 11, 0.15)',
                color: '#F0B90B',
              }}
            >
              {language === 'zh' ? '跟单' : 'Copy'}
            </span>
          )}
          <div
            className="px-3 py-1 rounded text-xs font-bold"
            style={
              decision.success
                ? { background: 'rgba(14, 203, 129, 0.1)', color: '#0ECB81' }
                : { background: 'rgba(246, 70, 93, 0.1)', color: '#F6465D' }
            }
          >
            {t(decision.success ? 'success' : 'failed', language)}
          </div>
        </div>
      </div>

      {decision.input_prompt && (
        <div className="mb-3">
          <button
            onClick={() => setShowInputPrompt(!showInputPrompt)}
            className="flex items-center gap-2 text-sm transition-colors"
            style={{ color: '#60a5fa' }}
          >
            <span className="font-semibold flex items-center gap-2">
              <Inbox className="w-4 h-4" /> {t('inputPrompt', language)}
            </span>
            <span className="text-xs">
              {showInputPrompt
                ? t('collapse', language)
                : t('expand', language)}
            </span>
          </button>
          {showInputPrompt && (
            <div
              className="mt-2 rounded p-4 text-sm font-mono whitespace-pre-wrap max-h-96 overflow-y-auto"
              style={{
                background: '#0B0E11',
                border: '1px solid #2B3139',
                color: '#EAECEF',
              }}
            >
              {decision.input_prompt}
            </div>
          )}
        </div>
      )}

      {decision.cot_trace && (
        <div className="mb-3">
          <button
            onClick={() => setShowCoT(!showCoT)}
            className="flex items-center gap-2 text-sm transition-colors"
            style={{ color: '#F0B90B' }}
          >
            <span className="font-semibold flex items-center gap-2">
              <Send className="w-4 h-4" />{' '}
              {stripLeadingIcons(t('aiThinking', language))}
            </span>
            <span className="text-xs">
              {showCoT ? t('collapse', language) : t('expand', language)}
            </span>
          </button>
          {showCoT && (
            <div
              className="mt-2 rounded p-4 text-sm font-mono whitespace-pre-wrap max-h-96 overflow-y-auto"
              style={{
                background: '#0B0E11',
                border: '1px solid #2B3139',
                color: '#EAECEF',
              }}
            >
              {decision.cot_trace}
            </div>
          )}
        </div>
      )}

      {decision.decisions && decision.decisions.length > 0 && (
        <div className="space-y-2 mb-3">
          {decision.decisions.map((action, j) => (
            <div
              key={j}
              className="flex items-center gap-2 text-sm rounded px-3 py-2"
              style={{ background: '#0B0E11' }}
            >
              <span
                className="font-mono font-bold"
                style={{ color: '#EAECEF' }}
              >
                {action.symbol}
              </span>
              <span
                className="px-2 py-0.5 rounded text-xs font-bold"
                style={
                  action.action.includes('open')
                    ? {
                        background: 'rgba(96, 165, 250, 0.1)',
                        color: '#60a5fa',
                      }
                    : {
                        background: 'rgba(240, 185, 11, 0.1)',
                        color: '#F0B90B',
                      }
                }
              >
                {action.action}
              </span>
              {action.leverage > 0 && (
                <span style={{ color: '#F0B90B' }}>{action.leverage}x</span>
              )}
              {action.price > 0 && (
                <span
                  className="font-mono text-xs"
                  style={{ color: '#848E9C' }}
                >
                  @{action.price.toFixed(4)}
                </span>
              )}
              <span style={{ color: action.success ? '#0ECB81' : '#F6465D' }}>
                {action.success ? (
                  <Check className="w-3 h-3 inline" />
                ) : (
                  <X className="w-3 h-3 inline" />
                )}
              </span>
              {action.error && (
                <span className="text-xs ml-2" style={{ color: '#F6465D' }}>
                  {action.error}
                </span>
              )}
            </div>
          ))}
        </div>
      )}

      {decision.account_state && (
        <div
          className="flex gap-4 text-xs mb-3 rounded px-3 py-2"
          style={{ background: '#0B0E11', color: '#848E9C' }}
        >
          <span>
            净值: {decision.account_state.total_balance.toFixed(2)} USDT
          </span>
          <span>
            可用: {decision.account_state.available_balance.toFixed(2)} USDT
          </span>
          <span>
            保证金率: {decision.account_state.margin_used_pct.toFixed(1)}%
          </span>
          <span>持仓: {decision.account_state.position_count}</span>
          <span
            style={{
              color:
                decision.candidate_coins &&
                decision.candidate_coins.length === 0
                  ? '#F6465D'
                  : '#848E9C',
            }}
          >
            {t('candidateCoins', language)}:{' '}
            {decision.candidate_coins?.length || 0}
          </span>
        </div>
      )}

      {decision.source !== 'copy_trade' &&
        decision.candidate_coins &&
        decision.candidate_coins.length === 0 && (
          <div
            className="text-sm rounded px-4 py-3 mb-3 flex items-start gap-3"
            style={{
              background: 'rgba(246, 70, 93, 0.1)',
              border: '1px solid rgba(246, 70, 93, 0.3)',
              color: '#F6465D',
            }}
          >
            <AlertTriangle size={16} className="flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <div className="font-semibold mb-1">
                {t('candidateCoinsZeroWarning', language)}
              </div>
              <div className="text-xs space-y-1" style={{ color: '#848E9C' }}>
                <div>{t('possibleReasons', language)}</div>
                <ul className="list-disc list-inside space-y-0.5 ml-2">
                  <li>{t('coinPoolApiNotConfigured', language)}</li>
                  <li>{t('apiConnectionTimeout', language)}</li>
                  <li>{t('noCustomCoinsAndApiFailed', language)}</li>
                </ul>
                <div className="mt-2">
                  <strong>{t('solutions', language)}</strong>
                </div>
                <ul className="list-disc list-inside space-y-0.5 ml-2">
                  <li>{t('setCustomCoinsInConfig', language)}</li>
                  <li>{t('orConfigureCorrectApiUrl', language)}</li>
                  <li>{t('orDisableCoinPoolOptions', language)}</li>
                </ul>
              </div>
            </div>
          </div>
        )}

      {decision.execution_log && decision.execution_log.length > 0 && (
        <div className="space-y-1">
          {decision.execution_log.map((log, k) => (
            <div
              key={k}
              className="text-xs font-mono"
              style={{
                color:
                  log.includes('✓') || log.includes('成功')
                    ? '#0ECB81'
                    : '#F6465D',
              }}
            >
              {log}
            </div>
          ))}
        </div>
      )}

      {decision.error_message && (
        <div
          className="text-sm rounded px-3 py-2 mt-3 flex items-center gap-2"
          style={{ color: '#F6465D', background: 'rgba(246, 70, 93, 0.1)' }}
        >
          <XCircle className="w-4 h-4" /> {decision.error_message}
        </div>
      )}
    </div>
  )
}

interface DecisionsPanelProps {
  decisions?: DecisionRecord[]
  decisionLimit: number
  onLimitChange: (limit: number) => void
  language: Language
}

export function DecisionsPanel({
  decisions,
  decisionLimit,
  onLimitChange,
  language,
}: DecisionsPanelProps) {
  return (
    <div
      className="binance-card p-6 animate-slide-in h-fit lg:sticky lg:top-24 lg:max-h-[calc(100vh-120px)]"
      style={{ animationDelay: '0.2s' }}
    >
      <div
        className="flex items-center justify-between mb-5 pb-4 border-b"
        style={{ borderColor: '#2B3139' }}
      >
        <div className="flex items-center gap-3">
          <div
            className="w-10 h-10 rounded-xl flex items-center justify-center"
            style={{
              background: 'linear-gradient(135deg, #6366F1 0%, #8B5CF6 100%)',
              boxShadow: '0 4px 14px rgba(99, 102, 241, 0.4)',
            }}
          >
            <Brain className="w-5 h-5" style={{ color: '#FFFFFF' }} />
          </div>
          <div>
            <h2 className="text-xl font-bold" style={{ color: '#EAECEF' }}>
              {t('recentDecisions', language)}
            </h2>
            {decisions && decisions.length > 0 && (
              <div className="text-xs" style={{ color: '#848E9C' }}>
                {t('lastCycles', language, { count: decisions.length })}
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs" style={{ color: '#848E9C' }}>
            {language === 'zh' ? '显示' : 'Show'}:
          </span>
          <select
            value={decisionLimit}
            onChange={(e) => onLimitChange(parseInt(e.target.value, 10))}
            className="rounded px-2 py-1 text-xs font-medium cursor-pointer transition-colors"
            style={{
              background: '#1E2329',
              border: '1px solid #2B3139',
              color: '#EAECEF',
            }}
          >
            <option value={5}>5</option>
            <option value={10}>10</option>
            <option value={20}>20</option>
            <option value={50}>50</option>
          </select>
          <span className="text-xs" style={{ color: '#848E9C' }}>
            {language === 'zh' ? '条' : ''}
          </span>
        </div>
      </div>

      <div
        className="space-y-4 overflow-y-auto pr-2"
        style={{ maxHeight: 'calc(100vh - 280px)' }}
      >
        {decisions && decisions.length > 0 ? (
          decisions.map((decision, i) => (
            <DecisionCard key={i} decision={decision} language={language} />
          ))
        ) : (
          <div className="py-16 text-center">
            <div className="mb-4 opacity-30 flex justify-center">
              <Brain className="w-16 h-16" />
            </div>
            <div
              className="text-lg font-semibold mb-2"
              style={{ color: '#EAECEF' }}
            >
              {t('noDecisionsYet', language)}
            </div>
            <div className="text-sm" style={{ color: '#848E9C' }}>
              {t('aiDecisionsWillAppear', language)}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
