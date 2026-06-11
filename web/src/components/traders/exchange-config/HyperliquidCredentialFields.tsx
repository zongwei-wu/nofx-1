import { HelpCircle } from 'lucide-react'
import type { Exchange } from '../../../types'
import { t, type Language } from '../../../i18n/translations'
import { Tooltip } from '../Tooltip'
import { maskSecret } from './exchangeConfigUtils'
interface HyperliquidCredentialFieldsProps {
  selectedExchange: Exchange
  language: Language
  apiKey: string
  hyperliquidWalletAddr: string
  onHyperliquidWalletAddrChange: (v: string) => void
  onSecureInputHyperliquid: () => void
  onClearApiKey: () => void
}

export function HyperliquidCredentialFields({
  selectedExchange,
  language,
  apiKey,
  hyperliquidWalletAddr,
  onHyperliquidWalletAddrChange,
  onSecureInputHyperliquid,
  onClearApiKey,
}: HyperliquidCredentialFieldsProps) {
  return (
    <>
                    {/* 安全提示 banner */}
                    <div
                      className="p-3 rounded mb-4"
                      style={{
                        background: 'rgba(240, 185, 11, 0.1)',
                        border: '1px solid rgba(240, 185, 11, 0.3)',
                      }}
                    >
                      <div className="flex items-start gap-2">
                        <span style={{ color: '#F0B90B', fontSize: '16px' }}>
                          🔐
                        </span>
                        <div className="flex-1">
                          <div
                            className="text-sm font-semibold mb-1"
                            style={{ color: '#F0B90B' }}
                          >
                            {t('hyperliquidAgentWalletTitle', language)}
                          </div>
                          <div
                            className="text-xs"
                            style={{ color: '#848E9C', lineHeight: '1.5' }}
                          >
                            {t('hyperliquidAgentWalletDesc', language)}
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Agent Private Key 字段 */}
                    <div>
                      <label
                        className="block text-sm font-semibold mb-2"
                        style={{ color: '#EAECEF' }}
                      >
                        {t('hyperliquidAgentPrivateKey', language)}
                      </label>
                      <div className="flex flex-col gap-2">
                        <div className="flex gap-2">
                          <input
                            type="text"
                            value={maskSecret(apiKey)}
                            readOnly
                            placeholder={t(
                              'enterHyperliquidAgentPrivateKey',
                              language
                            )}
                            className="w-full px-3 py-2 rounded"
                            style={{
                              background: '#0B0E11',
                              border: '1px solid #2B3139',
                              color: '#EAECEF',
                            }}
                          />
                          <button
                            type="button"
                            onClick={onSecureInputHyperliquid}
                            className="px-3 py-2 rounded text-xs font-semibold transition-all hover:scale-105"
                            style={{
                              background: '#F0B90B',
                              color: '#000',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {apiKey
                              ? t('secureInputReenter', language)
                              : t('secureInputButton', language)}
                          </button>
                          {apiKey && (
                            <button
                              type="button"
                              onClick={onClearApiKey}
                              className="px-3 py-2 rounded text-xs font-semibold transition-all hover:scale-105"
                              style={{
                                background: '#1B1F2B',
                                color: '#848E9C',
                                whiteSpace: 'nowrap',
                              }}
                            >
                              {t('secureInputClear', language)}
                            </button>
                          )}
                        </div>
                        {apiKey && (
                          <div className="text-xs" style={{ color: '#848E9C' }}>
                            {t('secureInputHint', language)}
                          </div>
                        )}
                      </div>
                      <div
                        className="text-xs mt-1"
                        style={{ color: '#848E9C' }}
                      >
                        {t('hyperliquidAgentPrivateKeyDesc', language)}
                      </div>
                    </div>

                    {/* Main Wallet Address 字段 */}
                    <div>
                      <label
                        className="block text-sm font-semibold mb-2"
                        style={{ color: '#EAECEF' }}
                      >
                        {t('hyperliquidMainWalletAddress', language)}
                      </label>
                      <input
                        type="text"
                        value={hyperliquidWalletAddr}
                        onChange={(e) =>
                          onHyperliquidWalletAddrChange(e.target.value)
                        }
                        placeholder={t(
                          'enterHyperliquidMainWalletAddress',
                          language
                        )}
                        className="w-full px-3 py-2 rounded"
                        style={{
                          background: '#0B0E11',
                          border: '1px solid #2B3139',
                          color: '#EAECEF',
                        }}
                        required
                      />
                      <div
                        className="text-xs mt-1"
                        style={{ color: '#848E9C' }}
                      >
                        {t('hyperliquidMainWalletAddressDesc', language)}
                      </div>
                    </div>
    </>
  )
}
