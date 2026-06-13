import type {
  Exchange,
  UpdateExchangeConfigRequest,
  TestExchangeConnectionRequest,
  TestExchangeConnectionResponse,
} from '../../types'
import { CryptoService } from '../crypto'
import { httpClient } from '../httpClient'
import { API_BASE, getAuthHeaders, throwIfNotOk } from './client'

async function encryptPayload(request: UpdateExchangeConfigRequest | TestExchangeConnectionRequest) {
  const publicKey = await CryptoService.fetchPublicKey()
  await CryptoService.initialize(publicKey)
  const userId = localStorage.getItem('user_id') || ''
  const sessionId = sessionStorage.getItem('session_id') || ''
  return CryptoService.encryptSensitiveData(
    JSON.stringify(request),
    userId,
    sessionId
  )
}

export const exchangesApi = {
  async getExchangeConfigs(): Promise<Exchange[]> {
    const res = await httpClient.get(`${API_BASE}/exchanges`, getAuthHeaders())
    await throwIfNotOk(res, '获取交易所配置失败')
    return res.json()
  },

  async getSupportedExchanges(): Promise<Exchange[]> {
    const res = await httpClient.get(`${API_BASE}/supported-exchanges`)
    await throwIfNotOk(res, '获取支持的交易所失败')
    return res.json()
  },

  async updateExchangeConfigs(request: UpdateExchangeConfigRequest): Promise<void> {
    const res = await httpClient.put(
      `${API_BASE}/exchanges`,
      request,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '更新交易所配置失败')
  },

  async updateExchangeConfigsEncrypted(
    request: UpdateExchangeConfigRequest
  ): Promise<void> {
    const encryptedPayload = await encryptPayload(request)
    const res = await httpClient.put(
      `${API_BASE}/exchanges`,
      encryptedPayload,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '更新交易所配置失败')
  },

  async testExchangeConnectionEncrypted(
    request: TestExchangeConnectionRequest
  ): Promise<TestExchangeConnectionResponse> {
    const encryptedPayload = await encryptPayload(request)
    const res = await httpClient.post(
      `${API_BASE}/exchanges/test`,
      encryptedPayload,
      getAuthHeaders()
    )
    const data = (await res.json()) as TestExchangeConnectionResponse & {
      error?: string
    }
    if (!res.ok) {
      throw new Error(data.error || '测试交易所连接失败')
    }
    return data
  },
}
