import { BarChart3, Brain } from 'lucide-react'
import { t, type Language } from '../../../i18n/translations'
import { stripLeadingIcons } from '../../../lib/text'

export function AILearningError({ language }: { language: Language }) {
  return (
    <div
      className="rounded p-6"
      style={{ background: '#1E2329', border: '1px solid #2B3139' }}
    >
      <div style={{ color: '#F6465D' }}>
        {stripLeadingIcons(t('loadingError', language))}
      </div>
    </div>
  )
}

export function AILearningLoading({ language }: { language: Language }) {
  return (
    <div
      className="rounded p-6"
      style={{ background: '#1E2329', border: '1px solid #2B3139' }}
    >
      <div className="flex items-center gap-2" style={{ color: '#848E9C' }}>
        <BarChart3 className="w-4 h-4" /> {t('loading', language)}
      </div>
    </div>
  )
}

export function AILearningEmpty({ language }: { language: Language }) {
  return (
    <div
      className="rounded p-6"
      style={{ background: '#1E2329', border: '1px solid #2B3139' }}
    >
      <div className="flex items-center gap-2 mb-2">
        <Brain className="w-5 h-5" style={{ color: '#8B5CF6' }} />
        <h2 className="text-lg font-bold" style={{ color: '#EAECEF' }}>
          {t('aiLearning', language)}
        </h2>
      </div>
      <div style={{ color: '#848E9C' }}>{t('noCompleteData', language)}</div>
    </div>
  )
}
