import { t, type Language } from '../../../i18n/translations'
import type { TraderConfigData } from './types'
import {
  getTemplateDescription,
  getTemplateDisplayName,
  getTemplateTitle,
} from './promptTemplateLabels'

interface TraderPromptSectionProps {
  formData: TraderConfigData
  promptTemplates: { name: string }[]
  language: Language
  onChange: (field: keyof TraderConfigData, value: unknown) => void
}

export function TraderPromptSection({
  formData,
  promptTemplates,
  language,
  onChange,
}: TraderPromptSectionProps) {
  const handleInputChange = onChange
  return (
    <>
          {/* Trading Prompt */}
          <div className="bg-[#0B0E11] border border-[#2B3139] rounded-lg p-5">
            <h3 className="text-lg font-semibold text-[#EAECEF] mb-5 flex items-center gap-2">
              💬 交易策略提示词
            </h3>
            <div className="space-y-4">
              {/* 系统提示词模板选择 */}
              <div>
                <label className="text-sm text-[#EAECEF] block mb-2">
                  {t('systemPromptTemplate', language)}
                </label>
                <select
                  value={formData.system_prompt_template}
                  onChange={(e) =>
                    handleInputChange('system_prompt_template', e.target.value)
                  }
                  className="w-full px-3 py-2 bg-[#0B0E11] border border-[#2B3139] rounded text-[#EAECEF] focus:border-[#F0B90B] focus:outline-none"
                >
                  {promptTemplates.map((template) => (
                    <option key={template.name} value={template.name}>
                      {getTemplateDisplayName(template.name, language)}
                    </option>
                  ))}
                </select>

                {/* 動態描述區域 */}
                <div
                  className="mt-2 p-3 rounded"
                  style={{
                    background: 'rgba(240, 185, 11, 0.05)',
                    border: '1px solid rgba(240, 185, 11, 0.15)',
                  }}
                >
                  <div
                    className="text-xs font-semibold mb-1"
                    style={{ color: '#F0B90B' }}
                  >
                    {getTemplateTitle(formData.system_prompt_template, language)}
                  </div>
                  <div className="text-xs" style={{ color: '#848E9C' }}>
                    {getTemplateDescription(
                      formData.system_prompt_template,
                      language
                    )}
                  </div>
                </div>
                <p className="text-xs text-[#848E9C] mt-1">
                  选择预设的交易策略模板（包含交易哲学、风控原则等）
                </p>
              </div>

              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  checked={formData.override_base_prompt}
                  onChange={(e) =>
                    handleInputChange('override_base_prompt', e.target.checked)
                  }
                  className="w-4 h-4"
                />
                <label className="text-sm text-[#EAECEF]">覆盖默认提示词</label>
                <span className="text-xs text-[#F0B90B] inline-flex items-center gap-1">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="w-3.5 h-3.5"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z" />
                    <line x1="12" x2="12" y1="9" y2="13" />
                    <line x1="12" x2="12.01" y1="17" y2="17" />
                  </svg>{' '}
                  启用后将完全替换默认策略
                </span>
              </div>
              <div>
                <label className="text-sm text-[#EAECEF] block mb-2">
                  {formData.override_base_prompt
                    ? '自定义提示词'
                    : '附加提示词'}
                </label>
                <textarea
                  value={formData.custom_prompt}
                  onChange={(e) =>
                    handleInputChange('custom_prompt', e.target.value)
                  }
                  className="w-full px-3 py-2 bg-[#0B0E11] border border-[#2B3139] rounded text-[#EAECEF] focus:border-[#F0B90B] focus:outline-none h-24 resize-none"
                  placeholder={
                    formData.override_base_prompt
                      ? '输入完整的交易策略提示词...'
                      : '输入额外的交易策略提示...'
                  }
                />
              </div>
            </div>
          </div>
    </>
  )
}
