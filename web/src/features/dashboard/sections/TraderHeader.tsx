import { Bot } from 'lucide-react'
import type { SystemStatus, TraderInfo } from '../../../types'
import { t, type Language } from '../../../i18n/translations'
import { getModelDisplayName } from '../utils'

const highlightColor = '#60a5fa'

interface TraderHeaderProps {
  selectedTrader: TraderInfo
  traders?: TraderInfo[]
  selectedTraderId?: string
  status?: SystemStatus
  language: Language
  onTraderSelect: (traderId: string) => void
}

export function TraderHeader({
  selectedTrader,
  traders,
  selectedTraderId,
  status,
  language,
  onTraderSelect,
}: TraderHeaderProps) {
  return (
    <div
      className="mb-6 rounded p-6 animate-scale-in"
      style={{
        background:
          'linear-gradient(135deg, rgba(240, 185, 11, 0.15) 0%, rgba(252, 213, 53, 0.05) 100%)',
        border: '1px solid rgba(240, 185, 11, 0.2)',
        boxShadow: '0 0 30px rgba(240, 185, 11, 0.15)',
      }}
    >
      <div className="flex items-start justify-between mb-3">
        <h2
          className="text-2xl font-bold flex items-center gap-2"
          style={{ color: '#EAECEF' }}
        >
          <span
            className="w-10 h-10 rounded-full flex items-center justify-center"
            style={{
              background: 'linear-gradient(135deg, #F0B90B 0%, #FCD535 100%)',
            }}
          >
            <Bot className="w-5 h-5" style={{ color: '#0B0E11' }} />
          </span>
          {selectedTrader.trader_name}
        </h2>

        {traders && traders.length > 0 && (
          <div className="flex items-center gap-2">
            <span className="text-sm" style={{ color: '#848E9C' }}>
              {t('switchTrader', language)}:
            </span>
            <select
              value={selectedTraderId}
              onChange={(e) => onTraderSelect(e.target.value)}
              className="rounded px-3 py-2 text-sm font-medium cursor-pointer transition-colors"
              style={{
                background: '#1E2329',
                border: '1px solid #2B3139',
                color: '#EAECEF',
              }}
            >
              {traders.map((trader) => (
                <option key={trader.trader_id} value={trader.trader_id}>
                  {trader.trader_name}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>
      <div
        className="flex items-center gap-4 text-sm"
        style={{ color: '#848E9C' }}
      >
        <span>
          AI Model:{' '}
          <span
            className="font-semibold"
            style={{
              color: selectedTrader.ai_model.includes('qwen')
                ? '#c084fc'
                : highlightColor,
            }}
          >
            {getModelDisplayName(
              selectedTrader.ai_model.split('_').pop() ||
                selectedTrader.ai_model
            )}
          </span>
        </span>
        <span>•</span>
        <span>
          Prompt:{' '}
          <span className="font-semibold" style={{ color: highlightColor }}>
            {selectedTrader.system_prompt_template || '-'}
          </span>
        </span>
        {status && (
          <>
            <span>•</span>
            <span>Cycles: {status.call_count}</span>
            <span>•</span>
            <span>Runtime: {status.runtime_minutes} min</span>
          </>
        )}
      </div>
    </div>
  )
}
