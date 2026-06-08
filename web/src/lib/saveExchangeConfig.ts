import { api } from './api'
import type { Exchange } from '../types'

export interface SaveExchangeFields {
  apiKey: string
  secretKey?: string
  testnet?: boolean
  hyperliquidWalletAddr?: string
  asterUser?: string
  asterSigner?: string
  asterPrivateKey?: string
  passphrase?: string
}

export async function saveExchangeConfig(
  exchangeId: string,
  fields: SaveExchangeFields,
  allExchanges: Exchange[],
  supportedExchanges: Exchange[]
): Promise<Exchange[]> {
  const exchangeToUpdate = supportedExchanges.find((e) => e.id === exchangeId)
  if (!exchangeToUpdate) {
    throw new Error('EXCHANGE_NOT_EXIST')
  }

  const existingExchange = allExchanges.find((e) => e.id === exchangeId)
  let updatedExchanges: Exchange[]

  if (existingExchange) {
    updatedExchanges = allExchanges.map((e) =>
      e.id === exchangeId
        ? {
            ...e,
            apiKey: fields.apiKey,
            secretKey: fields.secretKey,
            testnet: fields.testnet,
            hyperliquidWalletAddr: fields.hyperliquidWalletAddr,
            asterUser: fields.asterUser,
            asterSigner: fields.asterSigner,
            asterPrivateKey: fields.asterPrivateKey,
            enabled: true,
          }
        : e
    )
  } else {
    updatedExchanges = [
      ...allExchanges,
      {
        ...exchangeToUpdate,
        apiKey: fields.apiKey,
        secretKey: fields.secretKey,
        testnet: fields.testnet,
        hyperliquidWalletAddr: fields.hyperliquidWalletAddr,
        asterUser: fields.asterUser,
        asterSigner: fields.asterSigner,
        asterPrivateKey: fields.asterPrivateKey,
        enabled: true,
      },
    ]
  }

  const request = {
    exchanges: Object.fromEntries(
      updatedExchanges.map((exchange) => [
        exchange.id,
        {
          enabled: exchange.enabled,
          api_key: exchange.apiKey || '',
          secret_key: exchange.secretKey || '',
          testnet: exchange.testnet || false,
          hyperliquid_wallet_addr: exchange.hyperliquidWalletAddr || '',
          aster_user: exchange.asterUser || '',
          aster_signer: exchange.asterSigner || '',
          aster_private_key: exchange.asterPrivateKey || '',
          passphrase:
            exchange.id === exchangeId ? fields.passphrase || '' : '',
        },
      ])
    ),
  }

  await api.updateExchangeConfigsEncrypted(request)
  return api.getExchangeConfigs()
}
