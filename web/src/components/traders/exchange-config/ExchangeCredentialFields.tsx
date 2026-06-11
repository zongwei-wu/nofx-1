import type { Exchange } from '../../../types'
import type { Language } from '../../../i18n/translations'
import { CexCredentialFields } from './CexCredentialFields'
import { AsterCredentialFields } from './AsterCredentialFields'
import { HyperliquidCredentialFields } from './HyperliquidCredentialFields'

interface ExchangeCredentialFieldsProps {
  selectedExchange: Exchange
  language: Language
  apiKey: string
  secretKey: string
  passphrase: string
  asterUser: string
  asterSigner: string
  asterPrivateKey: string
  hyperliquidWalletAddr: string
  showBinanceGuide: boolean
  serverIP: { public_ip: string; message: string } | null
  loadingIP: boolean
  copiedIP: boolean
  onApiKeyChange: (v: string) => void
  onSecretKeyChange: (v: string) => void
  onPassphraseChange: (v: string) => void
  onAsterUserChange: (v: string) => void
  onAsterSignerChange: (v: string) => void
  onAsterPrivateKeyChange: (v: string) => void
  onHyperliquidWalletAddrChange: (v: string) => void
  onToggleBinanceGuide: () => void
  onCopyIP: (ip: string) => void
  onSecureInputHyperliquid: () => void
  onClearApiKey: () => void
}

export function ExchangeCredentialFields({
  selectedExchange,
  language,
  apiKey,
  secretKey,
  passphrase,
  asterUser,
  asterSigner,
  asterPrivateKey,
  hyperliquidWalletAddr,
  showBinanceGuide,
  serverIP,
  loadingIP,
  copiedIP,
  onApiKeyChange,
  onSecretKeyChange,
  onPassphraseChange,
  onAsterUserChange,
  onAsterSignerChange,
  onAsterPrivateKeyChange,
  onHyperliquidWalletAddrChange,
  onToggleBinanceGuide,
  onCopyIP,
  onSecureInputHyperliquid,
  onClearApiKey,
}: ExchangeCredentialFieldsProps) {
  const isCex =
    (selectedExchange.id === 'binance' ||
      selectedExchange.id === 'okx' ||
      selectedExchange.type === 'cex') &&
    selectedExchange.id !== 'hyperliquid' &&
    selectedExchange.id !== 'aster'

  return (
    <>
      {isCex && (
        <CexCredentialFields
          selectedExchange={selectedExchange}
          language={language}
          apiKey={apiKey}
          secretKey={secretKey}
          passphrase={passphrase}
          showBinanceGuide={showBinanceGuide}
          serverIP={serverIP}
          loadingIP={loadingIP}
          copiedIP={copiedIP}
          onApiKeyChange={onApiKeyChange}
          onSecretKeyChange={onSecretKeyChange}
          onPassphraseChange={onPassphraseChange}
          onToggleBinanceGuide={onToggleBinanceGuide}
          onCopyIP={onCopyIP}
        />
      )}

      {selectedExchange.id === 'aster' && (
        <AsterCredentialFields
          selectedExchange={selectedExchange}
          language={language}
          asterUser={asterUser}
          asterSigner={asterSigner}
          asterPrivateKey={asterPrivateKey}
          onAsterUserChange={onAsterUserChange}
          onAsterSignerChange={onAsterSignerChange}
          onAsterPrivateKeyChange={onAsterPrivateKeyChange}
        />
      )}

      {selectedExchange.id === 'hyperliquid' && (
        <HyperliquidCredentialFields
          selectedExchange={selectedExchange}
          language={language}
          apiKey={apiKey}
          hyperliquidWalletAddr={hyperliquidWalletAddr}
          onHyperliquidWalletAddrChange={onHyperliquidWalletAddrChange}
          onSecureInputHyperliquid={onSecureInputHyperliquid}
          onClearApiKey={onClearApiKey}
        />
      )}
    </>
  )
}
