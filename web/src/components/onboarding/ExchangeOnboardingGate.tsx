import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Landmark, X, Shield, KeyRound, PlugZap } from 'lucide-react'
import { toast } from 'sonner'
import { useAuth } from '../../contexts/AuthContext'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { FEATURES } from '../../config/features'
import { useTradersConfigStore } from '../../stores'
import { ExchangeConfigModal } from '../traders/ExchangeConfigModal'
import { getExchangeIcon } from '../ExchangeIcons'
import { getShortName } from '../traders/utils'
import {
  EXCHANGE_ONBOARDING_PROMPT_KEY,
  hasConfiguredExchange,
  isExchangeOnboardingDone,
  markExchangeOnboardingDone,
} from '../../lib/exchangeOnboarding'
import { saveExchangeConfig } from '../../lib/saveExchangeConfig'

const ONBOARDING_EXCHANGE_IDS = ['binance', 'okx', 'hyperliquid', 'aster'] as const

export function ExchangeOnboardingGate() {
  const { user, token, hasFeature } = useAuth()
  const { language } = useLanguage()
  const navigate = useNavigate()

  const {
    allExchanges,
    supportedExchanges,
    loadConfigs,
    setAllExchanges,
  } = useTradersConfigStore()

  const [ready, setReady] = useState(false)
  const [showWelcome, setShowWelcome] = useState(false)
  const [showExchangeModal, setShowExchangeModal] = useState(false)
  const [editingExchangeId, setEditingExchangeId] = useState<string | null>(null)

  const needsTradingSetup =
    hasFeature(FEATURES.ai_trader) ||
    hasFeature(FEATURES.copy_trade) ||
    hasFeature(FEATURES.gateway)

  useEffect(() => {
    if (!user?.id || !token || !needsTradingSetup) {
      setReady(false)
      setShowWelcome(false)
      return
    }

    let cancelled = false
    void loadConfigs(user, token).then(() => {
      if (!cancelled) setReady(true)
    })
    return () => {
      cancelled = true
    }
  }, [user, token, needsTradingSetup, loadConfigs])

  useEffect(() => {
    if (!ready || !user?.id) return

    if (hasConfiguredExchange(allExchanges)) {
      markExchangeOnboardingDone(user.id)
      setShowWelcome(false)
      return
    }

    const prompted =
      sessionStorage.getItem(EXCHANGE_ONBOARDING_PROMPT_KEY) === '1'
    const dismissed = isExchangeOnboardingDone(user.id)

    if (prompted || !dismissed) {
      setShowWelcome(true)
      sessionStorage.removeItem(EXCHANGE_ONBOARDING_PROMPT_KEY)
    }
  }, [ready, allExchanges, user?.id])

  const finishOnboarding = useCallback(() => {
    if (user?.id) markExchangeOnboardingDone(user.id)
    setShowWelcome(false)
    setShowExchangeModal(false)
    setEditingExchangeId(null)
  }, [user?.id])

  const handleSkip = () => {
    finishOnboarding()
  }

  const handleSelectExchange = (exchangeId: string) => {
    setEditingExchangeId(exchangeId)
    setShowExchangeModal(true)
  }

  const handleSaveExchange = async (
    exchangeId: string,
    apiKey: string,
    secretKey?: string,
    testnet?: boolean,
    hyperliquidWalletAddr?: string,
    asterUser?: string,
    asterSigner?: string,
    asterPrivateKey?: string,
    passphrase?: string
  ) => {
    try {
      const refreshed = await toast.promise(
        saveExchangeConfig(
          exchangeId,
          {
            apiKey,
            secretKey,
            testnet,
            hyperliquidWalletAddr,
            asterUser,
            asterSigner,
            asterPrivateKey,
            passphrase,
          },
          allExchanges,
          supportedExchanges
        ),
        {
          loading: t('savingExchangeConfig', language),
          success: t('exchangeConfigSaved', language),
          error: t('saveConfigFailed', language),
        }
      )
      setAllExchanges(refreshed)
      setShowExchangeModal(false)
      setEditingExchangeId(null)

      if (hasConfiguredExchange(refreshed)) {
        finishOnboarding()
        toast.success(t('exchangeOnboardingComplete', language))
      }
    } catch (error) {
      console.error('Failed to save exchange config:', error)
      toast.error(t('saveConfigFailed', language))
    }
  }

  const handleGoToTraders = () => {
    finishOnboarding()
    navigate('/traders')
  }

  if (!showWelcome && !showExchangeModal) return null

  const exchangeOptions = ONBOARDING_EXCHANGE_IDS.map((id) => {
    const meta =
      supportedExchanges.find((e) => e.id === id) ||
      allExchanges.find((e) => e.id === id)
    return {
      id,
      name: meta?.name || id,
      type: meta?.type || 'cex',
    }
  })

  return (
    <>
      {showWelcome && !showExchangeModal && (
        <div
          className="fixed inset-0 z-[100] flex items-center justify-center p-4"
          style={{ background: 'rgba(0,0,0,0.72)' }}
        >
          <div
            className="w-full max-w-lg rounded-2xl shadow-2xl overflow-hidden"
            style={{ background: '#1E2329', border: '1px solid #2B3139' }}
            role="dialog"
            aria-modal="true"
            aria-labelledby="exchange-onboarding-title"
          >
            <div
              className="flex items-start justify-between px-5 py-4"
              style={{ borderBottom: '1px solid #2B3139' }}
            >
              <div>
                <h2
                  id="exchange-onboarding-title"
                  className="text-lg font-bold"
                  style={{ color: '#EAECEF' }}
                >
                  {t('exchangeOnboardingTitle', language)}
                </h2>
                <p className="text-sm mt-1" style={{ color: '#848E9C' }}>
                  {t('exchangeOnboardingSubtitle', language)}
                </p>
              </div>
              <button
                type="button"
                onClick={handleSkip}
                className="p-1 rounded hover:bg-white/5"
                aria-label={t('cancel', language)}
              >
                <X className="w-5 h-5" style={{ color: '#848E9C' }} />
              </button>
            </div>

            <div className="px-5 py-4 space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-2 text-xs">
                <div
                  className="flex items-center gap-2 p-2 rounded-lg"
                  style={{ background: '#0B0E11' }}
                >
                  <Landmark className="w-4 h-4 shrink-0" style={{ color: '#F0B90B' }} />
                  <span style={{ color: '#EAECEF' }}>
                    {t('exchangeOnboardingStepSelect', language)}
                  </span>
                </div>
                <div
                  className="flex items-center gap-2 p-2 rounded-lg"
                  style={{ background: '#0B0E11' }}
                >
                  <KeyRound className="w-4 h-4 shrink-0" style={{ color: '#F0B90B' }} />
                  <span style={{ color: '#EAECEF' }}>
                    {t('exchangeOnboardingStepKeys', language)}
                  </span>
                </div>
                <div
                  className="flex items-center gap-2 p-2 rounded-lg"
                  style={{ background: '#0B0E11' }}
                >
                  <PlugZap className="w-4 h-4 shrink-0" style={{ color: '#F0B90B' }} />
                  <span style={{ color: '#EAECEF' }}>
                    {t('exchangeOnboardingStepTest', language)}
                  </span>
                </div>
              </div>

              <p className="text-xs flex items-start gap-2" style={{ color: '#848E9C' }}>
                <Shield className="w-4 h-4 shrink-0 mt-0.5" style={{ color: '#0ECB81' }} />
                {t('exchangeOnboardingSecurityNote', language)}
              </p>

              <div>
                <p
                  className="text-xs font-semibold mb-2 uppercase tracking-wide"
                  style={{ color: '#5E6673' }}
                >
                  {t('selectExchange', language)}
                </p>
                <div className="grid grid-cols-2 gap-2">
                  {exchangeOptions.map((exchange) => (
                    <button
                      key={exchange.id}
                      type="button"
                      onClick={() => handleSelectExchange(exchange.id)}
                      className="flex items-center gap-3 p-3 rounded-xl text-left transition-all hover:scale-[1.02]"
                      style={{
                        background: '#0B0E11',
                        border: '1px solid #2B3139',
                      }}
                    >
                      <div className="w-8 h-8 flex items-center justify-center shrink-0">
                        {getExchangeIcon(exchange.id, { width: 28, height: 28 })}
                      </div>
                      <div className="min-w-0">
                        <div
                          className="font-semibold text-sm truncate"
                          style={{ color: '#EAECEF' }}
                        >
                          {getShortName(exchange.name)}
                        </div>
                        <div className="text-[10px]" style={{ color: '#848E9C' }}>
                          {exchange.type.toUpperCase()}
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            </div>

            <div
              className="flex flex-col-reverse sm:flex-row sm:justify-between gap-2 px-5 py-4"
              style={{ borderTop: '1px solid #2B3139' }}
            >
              <button
                type="button"
                onClick={handleSkip}
                className="px-4 py-2 rounded-lg text-sm"
                style={{ color: '#848E9C' }}
              >
                {t('exchangeOnboardingSkip', language)}
              </button>
              <button
                type="button"
                onClick={handleGoToTraders}
                className="px-4 py-2 rounded-lg text-sm font-semibold"
                style={{ background: '#2B3139', color: '#EAECEF' }}
              >
                {t('exchangeOnboardingGoTraders', language)}
              </button>
            </div>
          </div>
        </div>
      )}

      {showExchangeModal && (
        <ExchangeConfigModal
          allExchanges={supportedExchanges}
          configuredExchanges={allExchanges}
          editingExchangeId={editingExchangeId}
          onSave={handleSaveExchange}
          onDelete={() => {}}
          onClose={() => {
            setShowExchangeModal(false)
            setEditingExchangeId(null)
            if (!showWelcome) finishOnboarding()
          }}
          language={language}
        />
      )}
    </>
  )
}
