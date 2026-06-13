import { en } from './locales/en'
import { zh } from './locales/zh'

export type Language = 'en' | 'zh'

export const translations = { en, zh }

export function t(
  key: string,
  lang: Language,
  params?: Record<string, string | number>
): string {
  const keys = key.split('.')
  let value: unknown = translations[lang]
  for (const k of keys) {
    value = (value as Record<string, unknown>)?.[k]
  }
  let text = typeof value === 'string' ? value : key
  if (params) {
    Object.entries(params).forEach(([param, value]) => {
      text = text.replace(`{${param}}`, String(value))
    })
  }
  return text
}
