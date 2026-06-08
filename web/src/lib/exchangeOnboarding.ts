import type { Exchange } from '../types'

/** 是否已有可用的交易所凭证 */
export function hasConfiguredExchange(exchanges: Exchange[]): boolean {
  return exchanges.some((e) => isExchangeConfigured(e))
}

export function isExchangeConfigured(exchange: Exchange): boolean {
  if (exchange.id === 'aster') {
    return Boolean(
      exchange.asterUser?.trim() && exchange.asterSigner?.trim()
    )
  }
  if (exchange.id === 'hyperliquid') {
    return Boolean(exchange.hyperliquidWalletAddr?.trim())
  }
  return Boolean(exchange.enabled && exchange.apiKey?.trim())
}

export function exchangeOnboardingDoneKey(userId: string): string {
  return `nofx_exchange_onboarding_done_${userId}`
}

export function isExchangeOnboardingDone(userId: string): boolean {
  return localStorage.getItem(exchangeOnboardingDoneKey(userId)) === '1'
}

export function markExchangeOnboardingDone(userId: string): void {
  localStorage.setItem(exchangeOnboardingDoneKey(userId), '1')
}

export const EXCHANGE_ONBOARDING_PROMPT_KEY = 'nofx_prompt_exchange_setup'
