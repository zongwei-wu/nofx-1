import { useState, useEffect, useMemo } from 'react'
import { useSymbolPreferences } from '../contexts/SymbolPreferencesContext'
import type { AIModel, Exchange, CreateTraderRequest } from '../types'
import { useLanguage } from '../contexts/LanguageContext'
import { api } from '../lib/api'
import { toast } from 'sonner'
import { Pencil, Plus, X as IconX } from 'lucide-react'
import type { TraderConfigData } from './traders/trader-config/types'
import { TraderBasicSection } from './traders/trader-config/TraderBasicSection'
import { TraderTradingSection } from './traders/trader-config/TraderTradingSection'
import { TraderSignalSection } from './traders/trader-config/TraderSignalSection'
import { TraderPromptSection } from './traders/trader-config/TraderPromptSection'

interface TraderConfigModalProps {
  isOpen: boolean
  onClose: () => void
  traderData?: TraderConfigData | null
  isEditMode?: boolean
  availableModels?: AIModel[]
  availableExchanges?: Exchange[]
  onSave?: (data: CreateTraderRequest) => Promise<void>
}

const DEFAULT_COINS = [
  'BTCUSDT',
  'ETHUSDT',
  'SOLUSDT',
  'BNBUSDT',
  'XRPUSDT',
  'DOGEUSDT',
  'ADAUSDT',
]

export function TraderConfigModal({
  isOpen,
  onClose,
  traderData,
  isEditMode = false,
  availableModels = [],
  availableExchanges = [],
  onSave,
}: TraderConfigModalProps) {
  const { language } = useLanguage()
  const { sortSymbols } = useSymbolPreferences()
  const [formData, setFormData] = useState<TraderConfigData>({
    trader_name: '',
    ai_model: '',
    exchange_id: '',
    btc_eth_leverage: 5,
    altcoin_leverage: 3,
    trading_symbols: '',
    custom_prompt: '',
    override_base_prompt: false,
    system_prompt_template: 'default',
    is_cross_margin: true,
    use_coin_pool: false,
    use_oi_top: false,
    scan_interval_minutes: 3,
  })
  const [isSaving, setIsSaving] = useState(false)
  const [originalScanInterval, setOriginalScanInterval] = useState<number | null>(
    null
  )
  const [availableCoins, setAvailableCoins] = useState<string[]>([])
  const [selectedCoins, setSelectedCoins] = useState<string[]>([])
  const [showCoinSelector, setShowCoinSelector] = useState(false)
  const [promptTemplates, setPromptTemplates] = useState<{ name: string }[]>([])
  const [isFetchingBalance, setIsFetchingBalance] = useState(false)
  const [balanceFetchError, setBalanceFetchError] = useState('')

  useEffect(() => {
    if (traderData) {
      setFormData(traderData)
      setOriginalScanInterval(traderData.scan_interval_minutes ?? 3)
      if (traderData.trading_symbols) {
        const coins = traderData.trading_symbols
          .split(',')
          .map((s) => s.trim())
          .filter((s) => s)
        setSelectedCoins(coins)
      }
    } else if (!isEditMode) {
      setOriginalScanInterval(null)
      setFormData({
        trader_name: '',
        ai_model: availableModels[0]?.id || '',
        exchange_id: availableExchanges[0]?.id || '',
        btc_eth_leverage: 5,
        altcoin_leverage: 3,
        trading_symbols: '',
        custom_prompt: '',
        override_base_prompt: false,
        system_prompt_template: 'default',
        is_cross_margin: true,
        use_coin_pool: false,
        use_oi_top: false,
        initial_balance: 1000,
        scan_interval_minutes: 3,
      })
    }
    if (traderData && traderData.system_prompt_template === undefined) {
      setFormData((prev) => ({
        ...prev,
        system_prompt_template: 'default',
      }))
    }
  }, [traderData, isEditMode, availableModels, availableExchanges])

  useEffect(() => {
    api
      .getAppConfig()
      .then((config) => {
        if (config.default_coins) setAvailableCoins(config.default_coins)
      })
      .catch(() => setAvailableCoins(DEFAULT_COINS))
  }, [])

  useEffect(() => {
    api
      .getPromptTemplates()
      .then(setPromptTemplates)
      .catch(() =>
        setPromptTemplates([{ name: 'default' }, { name: 'aggressive' }])
      )
  }, [])

  const displayCoins = useMemo(
    () => sortSymbols(availableCoins),
    [availableCoins, sortSymbols]
  )

  if (!isOpen) return null

  const handleInputChange = (field: keyof TraderConfigData, value: unknown) => {
    setFormData((prev) => ({ ...prev, [field]: value }))

    if (field === 'trading_symbols') {
      const coins = String(value)
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s)
      setSelectedCoins(coins)
    }
  }

  const handleCoinToggle = (coin: string) => {
    setSelectedCoins((prev) => {
      const newCoins = prev.includes(coin)
        ? prev.filter((c) => c !== coin)
        : [...prev, coin]
      const symbolsString = newCoins.join(',')
      setFormData((current) => ({ ...current, trading_symbols: symbolsString }))
      return newCoins
    })
  }

  const handleFetchCurrentBalance = async () => {
    if (!isEditMode || !traderData?.trader_id) {
      setBalanceFetchError('只有在编辑模式下才能获取当前余额')
      return
    }

    setIsFetchingBalance(true)
    setBalanceFetchError('')

    try {
      const data = await api.getAccount(traderData.trader_id)
      const currentBalance = data.total_equity || data.balance || 0
      setFormData((prev) => ({ ...prev, initial_balance: currentBalance }))
      toast.success('已获取当前余额')
    } catch (error) {
      console.error('获取余额失败:', error)
      setBalanceFetchError('获取余额失败，请检查网络连接')
      toast.error('获取余额失败，请检查网络连接')
    } finally {
      setIsFetchingBalance(false)
    }
  }

  const handleSave = async () => {
    if (!onSave) return

    setIsSaving(true)
    try {
      const saveData: CreateTraderRequest = {
        name: formData.trader_name,
        ai_model_id: formData.ai_model,
        exchange_id: formData.exchange_id,
        btc_eth_leverage: formData.btc_eth_leverage,
        altcoin_leverage: formData.altcoin_leverage,
        trading_symbols: formData.trading_symbols,
        custom_prompt: formData.custom_prompt,
        override_base_prompt: formData.override_base_prompt,
        system_prompt_template: formData.system_prompt_template,
        is_cross_margin: formData.is_cross_margin,
        use_coin_pool: formData.use_coin_pool,
        use_oi_top: formData.use_oi_top,
        scan_interval_minutes: formData.scan_interval_minutes,
      }

      if (isEditMode && formData.initial_balance !== undefined) {
        saveData.initial_balance = formData.initial_balance
      }

      await toast.promise(onSave(saveData), {
        loading: '正在保存…',
        success: '保存成功',
        error: '保存失败',
      })
      onClose()
    } catch (error) {
      console.error('保存失败:', error)
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm p-4 overflow-y-auto">
      <div
        className="bg-[#1E2329] border border-[#2B3139] rounded-xl shadow-2xl max-w-3xl w-full my-8"
        style={{ maxHeight: 'calc(100vh - 4rem)' }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-6 border-b border-[#2B3139] bg-gradient-to-r from-[#1E2329] to-[#252B35] sticky top-0 z-10 rounded-t-xl">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-[#F0B90B] to-[#E1A706] flex items-center justify-center text-black">
              {isEditMode ? (
                <Pencil className="w-5 h-5" />
              ) : (
                <Plus className="w-5 h-5" />
              )}
            </div>
            <div>
              <h2 className="text-xl font-bold text-[#EAECEF]">
                {isEditMode ? '修改交易员' : '创建交易员'}
              </h2>
              <p className="text-sm text-[#848E9C] mt-1">
                {isEditMode ? '修改交易员配置参数' : '配置新的AI交易员'}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg text-[#848E9C] hover:text-[#EAECEF] hover:bg-[#2B3139] transition-colors flex items-center justify-center"
          >
            <IconX className="w-4 h-4" />
          </button>
        </div>

        <div
          className="p-6 space-y-8 overflow-y-auto"
          style={{ maxHeight: 'calc(100vh - 16rem)' }}
        >
          <TraderBasicSection
            formData={formData}
            availableModels={availableModels}
            availableExchanges={availableExchanges}
            onChange={handleInputChange}
          />
          <TraderTradingSection
            formData={formData}
            isEditMode={isEditMode}
            isFetchingBalance={isFetchingBalance}
            balanceFetchError={balanceFetchError}
            originalScanInterval={originalScanInterval}
            selectedCoins={selectedCoins}
            showCoinSelector={showCoinSelector}
            displayCoins={displayCoins}
            language={language}
            onChange={handleInputChange}
            onFetchBalance={handleFetchCurrentBalance}
            onCoinToggle={handleCoinToggle}
            onShowCoinSelector={setShowCoinSelector}
          />
          <TraderSignalSection
            formData={formData}
            onChange={handleInputChange}
          />
          <TraderPromptSection
            formData={formData}
            promptTemplates={promptTemplates}
            language={language}
            onChange={handleInputChange}
          />
        </div>

        <div className="flex justify-end gap-3 p-6 border-t border-[#2B3139] bg-gradient-to-r from-[#1E2329] to-[#252B35] sticky bottom-0 z-10 rounded-b-xl">
          <button
            onClick={onClose}
            className="px-6 py-3 bg-[#2B3139] text-[#EAECEF] rounded-lg hover:bg-[#404750] transition-all duration-200 border border-[#404750]"
          >
            取消
          </button>
          {onSave && (
            <button
              onClick={handleSave}
              disabled={
                isSaving ||
                !formData.trader_name ||
                !formData.ai_model ||
                !formData.exchange_id
              }
              className="px-8 py-3 bg-gradient-to-r from-[#F0B90B] to-[#E1A706] text-black rounded-lg hover:from-[#E1A706] hover:to-[#D4951E] transition-all duration-200 disabled:bg-[#848E9C] disabled:cursor-not-allowed font-medium shadow-lg"
            >
              {isSaving ? '保存中...' : isEditMode ? '保存修改' : '创建交易员'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
