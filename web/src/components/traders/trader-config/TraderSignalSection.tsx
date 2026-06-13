import type { TraderConfigData } from './types'

interface TraderSignalSectionProps {
  formData: TraderConfigData
  onChange: (field: keyof TraderConfigData, value: unknown) => void
}

export function TraderSignalSection({
  formData,
  onChange,
}: TraderSignalSectionProps) {
  const handleInputChange = onChange
  return (
    <>
          {/* Signal Sources */}
          <div className="bg-[#0B0E11] border border-[#2B3139] rounded-lg p-5">
            <h3 className="text-lg font-semibold text-[#EAECEF] mb-5 flex items-center gap-2">
              📡 信号源配置
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  checked={formData.use_coin_pool}
                  onChange={(e) =>
                    handleInputChange('use_coin_pool', e.target.checked)
                  }
                  className="w-4 h-4"
                />
                <label className="text-sm text-[#EAECEF]">
                  使用 Coin Pool 信号
                </label>
              </div>
              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  checked={formData.use_oi_top}
                  onChange={(e) =>
                    handleInputChange('use_oi_top', e.target.checked)
                  }
                  className="w-4 h-4"
                />
                <label className="text-sm text-[#EAECEF]">
                  使用 OI Top 信号
                </label>
              </div>
            </div>
          </div>
    </>
  )
}
