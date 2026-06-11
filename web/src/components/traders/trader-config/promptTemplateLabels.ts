import { t, type Language } from '../../../i18n/translations'

const TEMPLATE_NAME_KEYS: Record<string, string> = {
  default: 'promptTemplateDefault',
  adaptive: 'promptTemplateAdaptive',
  adaptive_relaxed: 'promptTemplateAdaptiveRelaxed',
  Hansen: 'promptTemplateHansen',
  nof1: 'promptTemplateNof1',
  taro_long_prompts: 'promptTemplateTaroLong',
}

const TEMPLATE_TITLE_KEYS: Record<string, string> = {
  default: 'promptDescDefault',
  adaptive: 'promptDescAdaptive',
  adaptive_relaxed: 'promptDescAdaptiveRelaxed',
  Hansen: 'promptDescHansen',
  nof1: 'promptDescNof1',
  taro_long_prompts: 'promptDescTaroLong',
}

const TEMPLATE_CONTENT_KEYS: Record<string, string> = {
  default: 'promptDescDefaultContent',
  adaptive: 'promptDescAdaptiveContent',
  adaptive_relaxed: 'promptDescAdaptiveRelaxedContent',
  Hansen: 'promptDescHansenContent',
  nof1: 'promptDescNof1Content',
  taro_long_prompts: 'promptDescTaroLongContent',
}

export function getTemplateDisplayName(name: string, language: Language): string {
  const key = TEMPLATE_NAME_KEYS[name]
  return key ? t(key, language) : name.charAt(0).toUpperCase() + name.slice(1)
}

export function getTemplateTitle(
  templateName: string,
  language: Language
): string {
  const key = TEMPLATE_TITLE_KEYS[templateName]
  return key ? t(key, language) : t('promptDescDefault', language)
}

export function getTemplateDescription(
  templateName: string,
  language: Language
): string {
  const key = TEMPLATE_CONTENT_KEYS[templateName]
  return key ? t(key, language) : t('promptDescDefaultContent', language)
}
