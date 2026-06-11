import React, { useState, useEffect } from 'react'
import type { Exchange } from '../../types'
import { t, type Language } from '../../i18n/translations'
import { api } from '../../lib/api'
import { getExchangeIcon } from '../ExchangeIcons'
import {
  TwoStageKeyModal,
  type TwoStageKeyModalResult,
} from '../TwoStageKeyModal'
import {
  WebCryptoEnvironmentCheck,
  type WebCryptoCheckStatus,
} from '../WebCryptoEnvironmentCheck'
import { BookOpen, Trash2, PlugZap } from 'lucide-react'
import { toast } from 'sonner'
import { getShortName } from './utils'
import {
  buildTestPayload,
  canTestConnection,
  type ExchangeFormValues,
} from './exchange-config/exchangeConfigUtils'
import { ExchangeCredentialFields } from './exchange-config/ExchangeCredentialFields'
import { BinanceSetupGuideModal } from './exchange-config/BinanceSetupGuideModal'

interface ExchangeConfigModalProps {
  allExchanges: Exchange[]
  configuredExchanges?: Exchange[]
  editingExchangeId: string | null
  onSave: (
    exchangeId: string,
    apiKey: string,
    secretKey?: string,
    testnet?: boolean,
    hyperliquidWalletAddr?: string,
    asterUser?: string,
    asterSigner?: string,
    asterPrivateKey?: string,
    passphrase?: string
  ) => Promise<void>
  onDelete: (exchangeId: string) => void
  onClose: () => void
  language: Language
}

export function ExchangeConfigModal({
  allExchanges,
  configuredExchanges,
  editingExchangeId,
  onSave,
  onDelete,
  onClose,
  language,
}: ExchangeConfigModalProps) {
  const [selectedExchangeId, setSelectedExchangeId] = useState(
    editingExchangeId || ''
  )
  const [apiKey, setApiKey] = useState('')
  const [secretKey, setSecretKey] = useState('')
  const [passphrase, setPassphrase] = useState('')
  const [testnet, setTestnet] = useState(false)
  const [showGuide, setShowGuide] = useState(false)
  const [serverIP, setServerIP] = useState<{
    public_ip: string
    message: string
  } | null>(null)
  const [loadingIP, setLoadingIP] = useState(false)
  const [copiedIP, setCopiedIP] = useState(false)
  const [webCryptoStatus, setWebCryptoStatus] =
    useState<WebCryptoCheckStatus>('idle')
  const [testingConnection, setTestingConnection] = useState(false)
  const [testResult, setTestResult] = useState<{
    ok: boolean
    message: string
  } | null>(null)
  const [showBinanceGuide, setShowBinanceGuide] = useState(false)
  const [asterUser, setAsterUser] = useState('')
  const [asterSigner, setAsterSigner] = useState('')
  const [asterPrivateKey, setAsterPrivateKey] = useState('')
  const [hyperliquidWalletAddr, setHyperliquidWalletAddr] = useState('')
  const [secureInputTarget, setSecureInputTarget] = useState<
    null | 'hyperliquid' | 'aster'
  >(null)

  const configuredExchange = configuredExchanges?.find(
    (e) => e.id === selectedExchangeId
  )

  const selectedExchange = editingExchangeId
    ? (configuredExchange ??
      allExchanges?.find((e) => e.id === selectedExchangeId))
    : allExchanges?.find((e) => e.id === selectedExchangeId)

  const formValues: ExchangeFormValues = {
    apiKey,
    secretKey,
    passphrase,
    testnet,
    hyperliquidWalletAddr,
    asterUser,
    asterSigner,
    asterPrivateKey,
  }

  const hasSavedCredentials =
    Boolean(editingExchangeId) && Boolean(configuredExchange)

  useEffect(() => {
    if (editingExchangeId && configuredExchange) {
      setApiKey(configuredExchange.apiKey || '')
      setSecretKey(configuredExchange.secretKey || '')
      setPassphrase('')
      setTestnet(Boolean(configuredExchange.testnet))
      setAsterUser(configuredExchange.asterUser || '')
      setAsterSigner(configuredExchange.asterSigner || '')
      setAsterPrivateKey('')
      setHyperliquidWalletAddr(configuredExchange.hyperliquidWalletAddr || '')
    } else if (!editingExchangeId) {
      setTestnet(false)
    }
  }, [editingExchangeId, configuredExchange])

  useEffect(() => {
    setTestResult(null)
  }, [
    selectedExchangeId,
    apiKey,
    secretKey,
    testnet,
    hyperliquidWalletAddr,
    asterUser,
    asterSigner,
    asterPrivateKey,
  ])

  useEffect(() => {
    if (selectedExchangeId === 'binance' && !serverIP) {
      setLoadingIP(true)
      api
        .getServerIP()
        .then((data) => setServerIP(data))
        .catch((err) => console.error('Failed to load server IP:', err))
        .finally(() => setLoadingIP(false))
    }
  }, [selectedExchangeId, serverIP])

  const handleCopyIP = async (ip: string) => {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(ip)
      } else {
        const textArea = document.createElement('textarea')
        textArea.value = ip
        textArea.style.position = 'fixed'
        textArea.style.left = '-999999px'
        document.body.appendChild(textArea)
        textArea.focus()
        textArea.select()
        const successful = document.execCommand('copy')
        document.body.removeChild(textArea)
        if (!successful) throw new Error('复制命令执行失败')
      }
      setCopiedIP(true)
      setTimeout(() => setCopiedIP(false), 2000)
      toast.success(t('ipCopied', language))
    } catch (err) {
      console.error('复制失败:', err)
      toast.error(
        t('copyIPFailed', language) || `复制失败: ${ip}\n请手动复制此IP地址`
      )
    }
  }

  const secureInputContextLabel =
    secureInputTarget === 'aster'
      ? t('asterExchangeName', language)
      : secureInputTarget === 'hyperliquid'
        ? t('hyperliquidExchangeName', language)
        : undefined

  const handleSecureInputComplete = ({
    value,
    obfuscationLog,
  }: TwoStageKeyModalResult) => {
    const trimmed = value.trim()
    if (secureInputTarget === 'hyperliquid') setApiKey(trimmed)
    if (secureInputTarget === 'aster') setAsterPrivateKey(trimmed)
    console.log('Secure input obfuscation log:', obfuscationLog)
    setSecureInputTarget(null)
  }

  const handleTestConnection = async () => {
    if (!selectedExchange) return

    if (!canTestConnection(selectedExchange, formValues, hasSavedCredentials)) {
      const msg =
        selectedExchange.id === 'binance'
          ? t('testConnectionKeyPairRequired', language)
          : t('testConnectionFillRequired', language)
      setTestResult({ ok: false, message: msg })
      toast.error(msg)
      return
    }

    const payload = buildTestPayload(selectedExchange, formValues)
    setTestingConnection(true)
    setTestResult(null)

    try {
      const result = await api.testExchangeConnectionEncrypted(payload)
      if (result.success) {
        const equity = result.total_equity ?? 0
        const msg = t('testConnectionSuccess', language, {
          equity: equity.toLocaleString(undefined, {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }),
        })
        setTestResult({ ok: true, message: msg })
        toast.success(msg)
      } else {
        const msg = result.error || t('testConnectionFailed', language)
        setTestResult({ ok: false, message: msg })
        toast.error(msg)
      }
    } catch (error) {
      const msg =
        error instanceof Error
          ? error.message
          : t('testConnectionFailed', language)
      setTestResult({ ok: false, message: msg })
      toast.error(msg)
    } finally {
      setTestingConnection(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedExchangeId || !selectedExchange) return

    if (selectedExchange.id === 'binance') {
      if (!apiKey.trim() || !secretKey.trim()) return
      await onSave(selectedExchangeId, apiKey.trim(), secretKey.trim(), testnet)
    } else if (selectedExchange.id === 'hyperliquid') {
      if (!apiKey.trim() || !hyperliquidWalletAddr.trim()) return
      await onSave(
        selectedExchangeId,
        apiKey.trim(),
        '',
        testnet,
        hyperliquidWalletAddr.trim()
      )
    } else if (selectedExchange.id === 'aster') {
      if (!asterUser.trim() || !asterSigner.trim() || !asterPrivateKey.trim())
        return
      await onSave(
        selectedExchangeId,
        '',
        '',
        testnet,
        undefined,
        asterUser.trim(),
        asterSigner.trim(),
        asterPrivateKey.trim()
      )
    } else if (selectedExchange.id === 'okx') {
      const hasPassphrase =
        Boolean(passphrase.trim()) || hasSavedCredentials
      if (!apiKey.trim() || !secretKey.trim() || !hasPassphrase) {
        toast.error(t('enterPassphrase', language))
        return
      }
      await onSave(
        selectedExchangeId,
        apiKey.trim(),
        secretKey.trim(),
        testnet,
        undefined,
        undefined,
        undefined,
        undefined,
        passphrase.trim()
      )
    } else {
      if (!apiKey.trim() || !secretKey.trim()) return
      await onSave(selectedExchangeId, apiKey.trim(), secretKey.trim(), testnet)
    }
  }

  const canTest = selectedExchange
    ? canTestConnection(selectedExchange, formValues, hasSavedCredentials)
    : false

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4 overflow-y-auto">
      <div
        className="bg-gray-800 rounded-lg w-full max-w-lg relative my-8"
        style={{
          background: '#1E2329',
          maxHeight: 'calc(100vh - 4rem)',
        }}
      >
        <div
          className="flex items-center justify-between p-6 pb-4 sticky top-0 z-10"
          style={{ background: '#1E2329' }}
        >
          <h3 className="text-xl font-bold" style={{ color: '#EAECEF' }}>
            {editingExchangeId
              ? t('editExchange', language)
              : t('addExchange', language)}
          </h3>
          <div className="flex items-center gap-2">
            {selectedExchange?.id === 'binance' && (
              <button
                type="button"
                onClick={() => setShowGuide(true)}
                className="px-3 py-2 rounded text-sm font-semibold transition-all hover:scale-105 flex items-center gap-2"
                style={{
                  background: 'rgba(240, 185, 11, 0.1)',
                  color: '#F0B90B',
                }}
              >
                <BookOpen className="w-4 h-4" />
                {t('viewGuide', language)}
              </button>
            )}
            {editingExchangeId && (
              <button
                type="button"
                onClick={() => onDelete(editingExchangeId)}
                className="p-2 rounded hover:bg-red-100 transition-colors"
                style={{
                  background: 'rgba(246, 70, 93, 0.1)',
                  color: '#F6465D',
                }}
                title={t('delete', language)}
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>

        <form onSubmit={handleSubmit} className="px-6 pb-6">
          <div
            className="space-y-4 overflow-y-auto"
            style={{ maxHeight: 'calc(100vh - 16rem)' }}
          >
            {!editingExchangeId && (
              <div className="space-y-3">
                <div className="space-y-2">
                  <div
                    className="text-xs font-semibold uppercase tracking-wide"
                    style={{ color: '#F0B90B' }}
                  >
                    {t('environmentSteps.checkTitle', language)}
                  </div>
                  <WebCryptoEnvironmentCheck
                    language={language}
                    variant="card"
                    onStatusChange={setWebCryptoStatus}
                  />
                </div>
                <div className="space-y-2">
                  <div
                    className="text-xs font-semibold uppercase tracking-wide"
                    style={{ color: '#F0B90B' }}
                  >
                    {t('environmentSteps.selectTitle', language)}
                  </div>
                  <select
                    value={selectedExchangeId}
                    onChange={(e) => setSelectedExchangeId(e.target.value)}
                    className="w-full px-3 py-2 rounded"
                    style={{
                      background: '#0B0E11',
                      border: '1px solid #2B3139',
                      color: '#EAECEF',
                    }}
                    aria-label={t('selectExchange', language)}
                    disabled={webCryptoStatus !== 'secure'}
                    required
                  >
                    <option value="">
                      {t('pleaseSelectExchange', language)}
                    </option>
                    {(allExchanges || []).map((exchange) => (
                      <option key={exchange.id} value={exchange.id}>
                        {getShortName(exchange.name)} (
                        {exchange.type.toUpperCase()})
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            )}

            {selectedExchange && (
              <div
                className="p-4 rounded"
                style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
              >
                <div className="flex items-center gap-3 mb-3">
                  <div className="w-8 h-8 flex items-center justify-center">
                    {getExchangeIcon(selectedExchange.id, {
                      width: 32,
                      height: 32,
                    })}
                  </div>
                  <div>
                    <div className="font-semibold" style={{ color: '#EAECEF' }}>
                      {getShortName(selectedExchange.name)}
                    </div>
                    <div className="text-xs" style={{ color: '#848E9C' }}>
                      {selectedExchange.type.toUpperCase()} •{' '}
                      {selectedExchange.id}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {selectedExchange && (
              <div
                className="p-4 rounded"
                style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
              >
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <div
                      className="text-sm font-semibold"
                      style={{ color: '#EAECEF' }}
                    >
                      {t('useTestnet', language)}
                    </div>
                    <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                      {t('testnetDescription', language)}
                    </div>
                  </div>
                  <label className="inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={testnet}
                      onChange={(e) => setTestnet(e.target.checked)}
                      className="sr-only"
                    />
                    <span
                      className="w-11 h-6 rounded-full relative transition-all"
                      style={{ background: testnet ? '#0ECB81' : '#2B3139' }}
                    >
                      <span
                        className="absolute top-0.5 w-5 h-5 rounded-full transition-all"
                        style={{
                          left: testnet ? '22px' : '2px',
                          background: '#EAECEF',
                        }}
                      />
                    </span>
                  </label>
                </div>
              </div>
            )}

            {selectedExchange && (
              <ExchangeCredentialFields
                selectedExchange={selectedExchange}
                language={language}
                apiKey={apiKey}
                secretKey={secretKey}
                passphrase={passphrase}
                asterUser={asterUser}
                asterSigner={asterSigner}
                asterPrivateKey={asterPrivateKey}
                hyperliquidWalletAddr={hyperliquidWalletAddr}
                showBinanceGuide={showBinanceGuide}
                serverIP={serverIP}
                loadingIP={loadingIP}
                copiedIP={copiedIP}
                onApiKeyChange={setApiKey}
                onSecretKeyChange={setSecretKey}
                onPassphraseChange={setPassphrase}
                onAsterUserChange={setAsterUser}
                onAsterSignerChange={setAsterSigner}
                onAsterPrivateKeyChange={setAsterPrivateKey}
                onHyperliquidWalletAddrChange={setHyperliquidWalletAddr}
                onToggleBinanceGuide={() =>
                  setShowBinanceGuide(!showBinanceGuide)
                }
                onCopyIP={handleCopyIP}
                onSecureInputHyperliquid={() =>
                  setSecureInputTarget('hyperliquid')
                }
                onClearApiKey={() => setApiKey('')}
              />
            )}
          </div>

          {testResult && (
            <div
              className="mt-4 px-3 py-2 rounded text-xs"
              style={{
                background: testResult.ok
                  ? 'rgba(14, 203, 129, 0.1)'
                  : 'rgba(246, 70, 93, 0.1)',
                border: `1px solid ${testResult.ok ? 'rgba(14, 203, 129, 0.3)' : 'rgba(246, 70, 93, 0.3)'}`,
                color: testResult.ok ? '#0ECB81' : '#F6465D',
              }}
            >
              {testResult.message}
            </div>
          )}

          <div
            className="flex flex-col gap-3 mt-6 pt-4 sticky bottom-0"
            style={{ background: '#1E2329' }}
          >
            <button
              type="button"
              onClick={handleTestConnection}
              disabled={testingConnection || !selectedExchange || !canTest}
              className="w-full px-4 py-2 rounded text-sm font-semibold disabled:opacity-50 flex items-center justify-center gap-2"
              style={{ background: '#2B3139', color: '#EAECEF' }}
            >
              <PlugZap className="w-4 h-4" />
              {testingConnection
                ? t('testingConnection', language)
                : t('testConnection', language)}
            </button>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={onClose}
                className="flex-1 px-4 py-2 rounded text-sm font-semibold"
                style={{ background: '#2B3139', color: '#848E9C' }}
              >
                {t('cancel', language)}
              </button>
              <button
                type="submit"
                disabled={
                  !selectedExchange ||
                  (selectedExchange.id === 'binance' &&
                    (!apiKey.trim() || !secretKey.trim())) ||
                  (selectedExchange.id === 'okx' &&
                    (!apiKey.trim() ||
                      !secretKey.trim() ||
                      (!passphrase.trim() && !hasSavedCredentials))) ||
                  (selectedExchange.id === 'hyperliquid' &&
                    (!apiKey.trim() || !hyperliquidWalletAddr.trim())) ||
                  (selectedExchange.id === 'aster' &&
                    (!asterUser.trim() ||
                      !asterSigner.trim() ||
                      !asterPrivateKey.trim())) ||
                  (selectedExchange.type === 'cex' &&
                    selectedExchange.id !== 'hyperliquid' &&
                    selectedExchange.id !== 'aster' &&
                    selectedExchange.id !== 'binance' &&
                    selectedExchange.id !== 'okx' &&
                    (!apiKey.trim() || !secretKey.trim()))
                }
                className="flex-1 px-4 py-2 rounded text-sm font-semibold disabled:opacity-50"
                style={{ background: '#F0B90B', color: '#000' }}
              >
                {t('saveConfig', language)}
              </button>
            </div>
          </div>
        </form>
      </div>

      <BinanceSetupGuideModal
        open={showGuide}
        language={language}
        onClose={() => setShowGuide(false)}
      />

      <TwoStageKeyModal
        isOpen={secureInputTarget !== null}
        language={language}
        contextLabel={secureInputContextLabel}
        expectedLength={64}
        onCancel={() => setSecureInputTarget(null)}
        onComplete={handleSecureInputComplete}
      />
    </div>
  )
}
