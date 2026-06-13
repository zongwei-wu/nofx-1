import type { AIModel, UpdateModelConfigRequest } from '../../types'
import { CryptoService } from '../crypto'
import { httpClient } from '../httpClient'
import { API_BASE, getAuthHeaders, throwIfNotOk } from './client'

async function encryptPayload(request: UpdateModelConfigRequest) {
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

export const modelsApi = {
  async getModelConfigs(): Promise<AIModel[]> {
    const res = await httpClient.get(`${API_BASE}/models`, getAuthHeaders())
    await throwIfNotOk(res, '获取模型配置失败')
    return res.json()
  },

  async getSupportedModels(): Promise<AIModel[]> {
    const res = await httpClient.get(`${API_BASE}/supported-models`)
    await throwIfNotOk(res, '获取支持的模型失败')
    return res.json()
  },

  async updateModelConfigs(request: UpdateModelConfigRequest): Promise<void> {
    const encryptedPayload = await encryptPayload(request)
    const res = await httpClient.put(
      `${API_BASE}/models`,
      encryptedPayload,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '更新模型配置失败')
  },
}
