import type { Exchange } from '../../../types'

export function maskSecret(secret: string): string {
  if (!secret || secret.length === 0) return ''
  if (secret.length <= 8) return '*'.repeat(secret.length)
  return (
    secret.slice(0, 4) +
    '*'.repeat(Math.max(secret.length - 8, 4)) +
    secret.slice(-4)
  )
}

export interface ExchangeFormValues {
  apiKey: string
  secretKey: string
  passphrase: string
  testnet: boolean
  hyperliquidWalletAddr: string
  asterUser: string
  asterSigner: string
  asterPrivateKey: string
}

export function canTestConnection(
  exchange: Exchange,
  values: ExchangeFormValues,
  hasSavedCredentials: boolean
): boolean {
  if (exchange.id === 'binance') {
    return Boolean(values.apiKey.trim()) && Boolean(values.secretKey.trim())
  }
  if (exchange.id === 'hyperliquid') {
    return (
      Boolean(values.apiKey.trim()) &&
      Boolean(values.hyperliquidWalletAddr.trim())
    )
  }
  if (exchange.id === 'aster') {
    const hasPrivateKey =
      Boolean(values.asterPrivateKey.trim()) || hasSavedCredentials
    return (
      Boolean(values.asterUser.trim()) &&
      Boolean(values.asterSigner.trim()) &&
      hasPrivateKey
    )
  }
  if (exchange.id === 'okx') {
    const hasPassphrase =
      Boolean(values.passphrase.trim()) || hasSavedCredentials
    return (
      Boolean(values.apiKey.trim()) &&
      Boolean(values.secretKey.trim()) &&
      hasPassphrase
    )
  }
  return Boolean(values.apiKey.trim()) && Boolean(values.secretKey.trim())
}

export function buildTestPayload(
  exchange: Exchange,
  values: ExchangeFormValues
) {
  return {
    exchange_id: exchange.id,
    api_key: values.apiKey.trim(),
    secret_key: values.secretKey.trim(),
    testnet: values.testnet,
    hyperliquid_wallet_addr: values.hyperliquidWalletAddr.trim(),
    aster_user: values.asterUser.trim(),
    aster_signer: values.asterSigner.trim(),
    aster_private_key: values.asterPrivateKey.trim(),
    passphrase: values.passphrase.trim(),
  }
}
