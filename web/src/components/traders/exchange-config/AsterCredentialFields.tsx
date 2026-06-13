import { HelpCircle } from 'lucide-react'
import type { Exchange } from '../../../types'
import { t, type Language } from '../../../i18n/translations'
import { Tooltip } from '../Tooltip'

interface AsterCredentialFieldsProps {
  selectedExchange: Exchange
  language: Language
  asterUser: string
  asterSigner: string
  asterPrivateKey: string
  onAsterUserChange: (v: string) => void
  onAsterSignerChange: (v: string) => void
  onAsterPrivateKeyChange: (v: string) => void
}

export function AsterCredentialFields({
  selectedExchange,
  language,
  asterUser,
  asterSigner,
  asterPrivateKey,
  onAsterUserChange,
  onAsterSignerChange,
  onAsterPrivateKeyChange,
}: AsterCredentialFieldsProps) {
  return (
    <>
                    <div>
                      <label
                        className="block text-sm font-semibold mb-2 flex items-center gap-2"
                        style={{ color: '#EAECEF' }}
                      >
                        {t('user', language)}
                        <Tooltip content={t('asterUserDesc', language)}>
                          <HelpCircle
                            className="w-4 h-4 cursor-help"
                            style={{ color: '#F0B90B' }}
                          />
                        </Tooltip>
                      </label>
                      <input
                        type="text"
                        value={asterUser}
                        onChange={(e) => onAsterUserChange(e.target.value)}
                        placeholder={t('enterUser', language)}
                        className="w-full px-3 py-2 rounded"
                        style={{
                          background: '#0B0E11',
                          border: '1px solid #2B3139',
                          color: '#EAECEF',
                        }}
                        required
                      />
                    </div>

                    <div>
                      <label
                        className="block text-sm font-semibold mb-2 flex items-center gap-2"
                        style={{ color: '#EAECEF' }}
                      >
                        {t('signer', language)}
                        <Tooltip content={t('asterSignerDesc', language)}>
                          <HelpCircle
                            className="w-4 h-4 cursor-help"
                            style={{ color: '#F0B90B' }}
                          />
                        </Tooltip>
                      </label>
                      <input
                        type="text"
                        value={asterSigner}
                        onChange={(e) => onAsterSignerChange(e.target.value)}
                        placeholder={t('enterSigner', language)}
                        className="w-full px-3 py-2 rounded"
                        style={{
                          background: '#0B0E11',
                          border: '1px solid #2B3139',
                          color: '#EAECEF',
                        }}
                        required
                      />
                    </div>

                    <div>
                      <label
                        className="block text-sm font-semibold mb-2 flex items-center gap-2"
                        style={{ color: '#EAECEF' }}
                      >
                        {t('privateKey', language)}
                        <Tooltip content={t('asterPrivateKeyDesc', language)}>
                          <HelpCircle
                            className="w-4 h-4 cursor-help"
                            style={{ color: '#F0B90B' }}
                          />
                        </Tooltip>
                      </label>
                      <input
                        type="password"
                        value={asterPrivateKey}
                        onChange={(e) => onAsterPrivateKeyChange(e.target.value)}
                        placeholder={t('enterPrivateKey', language)}
                        className="w-full px-3 py-2 rounded"
                        style={{
                          background: '#0B0E11',
                          border: '1px solid #2B3139',
                          color: '#EAECEF',
                        }}
                        required
                      />
                    </div>
    </>
  )
}
