import { t, type Language } from '../../../i18n/translations'
import type { TraderConfigData } from './types'

interface TraderTradingSectionProps {
  formData: TraderConfigData
  isEditMode: boolean
  isFetchingBalance: boolean
  balanceFetchError: string
  originalScanInterval: number | null
  selectedCoins: string[]
  showCoinSelector: boolean
  displayCoins: string[]
  language: Language
  onChange: (field: keyof TraderConfigData, value: unknown) => void
  onFetchBalance: () => void
  onCoinToggle: (coin: string) => void
  onShowCoinSelector: (show: boolean) => void
}

export function TraderTradingSection({
  formData,
  isEditMode,
  isFetchingBalance,
  balanceFetchError,
  originalScanInterval,
  selectedCoins,
  showCoinSelector,
  displayCoins,
  language,
  onChange,
  onFetchBalance,
  onCoinToggle,
  onShowCoinSelector,
}: TraderTradingSectionProps) {
  const handleInputChange = onChange
  return (
    <>
          {/* Trading Configuration */}
          <div className="bg-[#0B0E11] border border-[#2B3139] rounded-lg p-5">
            <h3 className="text-lg font-semibold text-[#EAECEF] mb-5 flex items-center gap-2">
              ⚖️ 交易配置
            </h3>
            <div className="space-y-4">
              {/* 第一行：保证金模式和初始余额 */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm text-[#EAECEF] block mb-2">
                    保证金模式
                  </label>
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => handleInputChange('is_cross_margin', true)}
                      className={`flex-1 px-3 py-2 rounded text-sm ${
                        formData.is_cross_margin
                          ? 'bg-[#F0B90B] text-black'
                          : 'bg-[#0B0E11] text-[#848E9C] border border-[#2B3139]'
                      }`}
                    >
                      全仓
                    </button>
                    <button
                      type="button"
                      onClick={() =>
                        handleInputChange('is_cross_margin', false)
                      }
                      className={`flex-1 px-3 py-2 rounded text-sm ${
                        !formData.is_cross_margin
                          ? 'bg-[#F0B90B] text-black'
                          : 'bg-[#0B0E11] text-[#848E9C] border border-[#2B3139]'
                      }`}
                    >
                      逐仓
                    </button>
                  </div>
                </div>
                {isEditMode && (
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <label className="text-sm text-[#EAECEF]">
                        初始余额 ($)
                      </label>
                      <button
                        type="button"
                        onClick={onFetchBalance}
                        disabled={isFetchingBalance}
                        className="px-3 py-1 text-xs bg-[#F0B90B] text-black rounded hover:bg-[#E1A706] transition-colors disabled:bg-[#848E9C] disabled:cursor-not-allowed"
                      >
                        {isFetchingBalance ? '获取中...' : '获取当前余额'}
                      </button>
                    </div>
                    <input
                      type="number"
                      value={formData.initial_balance || 0}
                      onChange={(e) =>
                        handleInputChange(
                          'initial_balance',
                          Number(e.target.value)
                        )
                      }
                      onBlur={(e) => {
                        // Force minimum value on blur
                        const value = Number(e.target.value)
                        if (value < 100) {
                          handleInputChange('initial_balance', 100)
                        }
                      }}
                      className="w-full px-3 py-2 bg-[#0B0E11] border border-[#2B3139] rounded text-[#EAECEF] focus:border-[#F0B90B] focus:outline-none"
                      min="100"
                      step="0.01"
                    />
                    <p className="text-xs text-[#848E9C] mt-1">
                      用于手动更新初始余额基准（例如充值/提现后）
                    </p>
                    {balanceFetchError && (
                      <p className="text-xs text-red-500 mt-1">
                        {balanceFetchError}
                      </p>
                    )}
                  </div>
                )}
                {!isEditMode && (
                  <div>
                    <label className="text-sm text-[#EAECEF] mb-2 block">
                      初始余额
                    </label>
                    <div className="w-full px-3 py-2 bg-[#1E2329] border border-[#2B3139] rounded text-[#848E9C] flex items-center gap-2">
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        className="w-4 h-4 text-[#F0B90B]"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <circle cx="12" cy="12" r="10" />
                        <line x1="12" x2="12" y1="8" y2="12" />
                        <line x1="12" x2="12.01" y1="16" y2="16" />
                      </svg>
                      <span className="text-sm">
                        系统将自动获取您的账户净值作为初始余额
                      </span>
                    </div>
                  </div>
                )}
              </div>

              {/* 第二行：AI 扫描决策间隔 */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm text-[#EAECEF] block mb-2">
                    {t('aiScanInterval', language)}
                  </label>
                  <input
                    type="number"
                    value={formData.scan_interval_minutes}
                    onChange={(e) => {
                      const parsedValue = Number(e.target.value)
                      const safeValue = Number.isFinite(parsedValue)
                        ? Math.max(3, parsedValue)
                        : 3
                      handleInputChange('scan_interval_minutes', safeValue)
                    }}
                    className="w-full px-3 py-2 bg-[#0B0E11] border border-[#2B3139] rounded text-[#EAECEF] focus:border-[#F0B90B] focus:outline-none"
                    min="3"
                    max="60"
                    step="1"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    {t('scanIntervalRecommend', language)}
                  </p>
                  {isEditMode &&
                    originalScanInterval !== null &&
                    formData.scan_interval_minutes !== originalScanInterval && (
                      <p className="text-xs text-[#F0B90B] mt-1">
                        {t('scanIntervalRestartHint', language)}
                      </p>
                    )}
                </div>
                <div></div>
              </div>

              {/* 第三行：杠杆设置 */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm text-[#EAECEF] block mb-2">
                    BTC/ETH 杠杆
                  </label>
                  <input
                    type="number"
                    value={formData.btc_eth_leverage}
                    onChange={(e) =>
                      handleInputChange(
                        'btc_eth_leverage',
                        Number(e.target.value)
                      )
                    }
                    className="w-full px-3 py-2 bg-[#0B0E11] border border-[#2B3139] rounded text-[#EAECEF] focus:border-[#F0B90B] focus:outline-none"
                    min="1"
                    max="125"
                  />
                </div>
                <div>
                  <label className="text-sm text-[#EAECEF] block mb-2">
                    山寨币杠杆
                  </label>
                  <input
                    type="number"
                    value={formData.altcoin_leverage}
                    onChange={(e) =>
                      handleInputChange(
                        'altcoin_leverage',
                        Number(e.target.value)
                      )
                    }
                    className="w-full px-3 py-2 bg-[#0B0E11] border border-[#2B3139] rounded text-[#EAECEF] focus:border-[#F0B90B] focus:outline-none"
                    min="1"
                    max="75"
                  />
                </div>
              </div>

              {/* 第三行：交易币种 */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-sm text-[#EAECEF]">
                    交易币种 (用逗号分隔，留空使用默认)
                  </label>
                  <button
                    type="button"
                    onClick={() => onShowCoinSelector(!showCoinSelector)}
                    className="px-3 py-1 text-xs bg-[#F0B90B] text-black rounded hover:bg-[#E1A706] transition-colors"
                  >
                    {showCoinSelector ? '收起选择' : '快速选择'}
                  </button>
                </div>
                <input
                  type="text"
                  value={formData.trading_symbols}
                  onChange={(e) =>
                    handleInputChange('trading_symbols', e.target.value)
                  }
                  className="w-full px-3 py-2 bg-[#0B0E11] border border-[#2B3139] rounded text-[#EAECEF] focus:border-[#F0B90B] focus:outline-none"
                  placeholder="例如: BTCUSDT,ETHUSDT,ADAUSDT"
                />

                {/* 币种选择器 */}
                {showCoinSelector && (
                  <div className="mt-3 p-3 bg-[#0B0E11] border border-[#2B3139] rounded">
                    <div className="text-xs text-[#848E9C] mb-2">
                      点击选择币种：
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {displayCoins.map((coin) => (
                        <button
                          key={coin}
                          type="button"
                          onClick={() => onCoinToggle(coin)}
                          className={`px-2 py-1 text-xs rounded transition-colors ${
                            selectedCoins.includes(coin)
                              ? 'bg-[#F0B90B] text-black'
                              : 'bg-[#1E2329] text-[#848E9C] border border-[#2B3139] hover:border-[#F0B90B]'
                          }`}
                        >
                          {coin.replace('USDT', '')}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
    </>
  )
}
