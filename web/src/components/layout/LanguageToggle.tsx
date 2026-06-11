import { useRef, useEffect, useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { t, type Language } from '../../i18n/translations'

interface LanguageToggleProps {
  language: Language
  onLanguageChange: (lang: Language) => void
  variant?: 'dropdown' | 'inline'
}

export function LanguageToggle({
  language,
  onLanguageChange,
  variant = 'dropdown',
}: LanguageToggleProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  if (variant === 'inline') {
    return (
      <div
        className="flex gap-1 rounded p-1"
        style={{ background: 'var(--panel-bg)' }}
      >
        {(['zh', 'en'] as Language[]).map((lang) => (
          <button
            key={lang}
            type="button"
            onClick={() => onLanguageChange(lang)}
            className="px-3 py-1.5 rounded text-xs font-semibold transition-all"
            style={
              language === lang
                ? { background: 'var(--brand-yellow)', color: 'var(--brand-black)' }
                : { background: 'transparent', color: 'var(--text-secondary)' }
            }
          >
            {lang === 'zh' ? '中文' : 'EN'}
          </button>
        ))}
      </div>
    )
  }

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1 px-3 py-2 rounded-lg text-sm transition-colors"
        style={{
          background: 'var(--panel-bg)',
          color: 'var(--brand-light-gray)',
          border: '1px solid var(--panel-border)',
        }}
      >
        {language === 'zh' ? '🇨🇳' : '🇺🇸'}
        <ChevronDown className="w-4 h-4" />
      </button>
      {open && (
        <div
          className="absolute right-0 mt-2 py-1 rounded-lg shadow-lg min-w-[120px] z-50"
          style={{
            background: 'var(--panel-bg)',
            border: '1px solid var(--panel-border)',
          }}
        >
          {(['zh', 'en'] as Language[]).map((lang) => (
            <button
              key={lang}
              type="button"
              onClick={() => {
                onLanguageChange(lang)
                setOpen(false)
              }}
              className="w-full text-left px-4 py-2 text-sm hover:bg-white/5"
              style={{
                color:
                  language === lang
                    ? 'var(--brand-yellow)'
                    : 'var(--brand-light-gray)',
              }}
            >
              {lang === 'zh' ? t('language', 'zh') : 'English'}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
