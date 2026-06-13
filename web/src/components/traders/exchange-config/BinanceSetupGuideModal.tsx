import { BookOpen } from 'lucide-react'
import { t, type Language } from '../../../i18n/translations'

interface BinanceSetupGuideModalProps {
  open: boolean
  language: Language
  onClose: () => void
}

export function BinanceSetupGuideModal({
  open,
  language,
  onClose,
}: BinanceSetupGuideModalProps) {
  if (!open) return null

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        className="bg-gray-800 rounded-lg p-6 w-full max-w-4xl relative"
        style={{ background: '#1E2329' }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3
            className="text-xl font-bold flex items-center gap-2"
            style={{ color: '#EAECEF' }}
          >
            <BookOpen className="w-6 h-6" style={{ color: '#F0B90B' }} />
            {t('binanceSetupGuide', language)}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded text-sm font-semibold transition-all hover:scale-105"
            style={{ background: '#2B3139', color: '#848E9C' }}
          >
            {t('closeGuide', language)}
          </button>
        </div>
        <div className="overflow-y-auto max-h-[80vh]">
          <img
            src="/images/guide.png"
            alt={t('binanceSetupGuide', language)}
            className="w-full h-auto rounded"
          />
        </div>
      </div>
    </div>
  )
}
