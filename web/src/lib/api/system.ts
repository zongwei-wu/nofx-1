import { httpClient } from '../httpClient'
import { API_BASE, throwIfNotOk } from './client'
import type { SystemConfig } from '../config'

export interface AppConfig extends SystemConfig {
  default_coins?: string[]
}

export const systemApi = {
  async getAppConfig(): Promise<AppConfig> {
    const res = await httpClient.get(`${API_BASE}/config`)
    await throwIfNotOk(res, '获取系统配置失败')
    return res.json()
  },

  async getPromptTemplates(): Promise<{ name: string }[]> {
    const res = await httpClient.get(`${API_BASE}/prompt-templates`)
    if (!res.ok) return [{ name: 'default' }, { name: 'aggressive' }]
    const data = await res.json()
    return data.templates ?? [{ name: 'default' }, { name: 'aggressive' }]
  },
}
