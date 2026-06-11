import { httpClient } from '../httpClient'
import { API_BASE, getAuthHeaders, throwIfNotOk } from './client'

export interface ApiKeyRecord {
  id: string
  key_prefix: string
  name: string
  last_used: string | null
  expires_at: string | null
  created_at: string
}

export const apiKeysApi = {
  async list(): Promise<ApiKeyRecord[]> {
    const res = await httpClient.get(`${API_BASE}/api-keys`, getAuthHeaders())
    await throwIfNotOk(res, '获取 API Keys 失败')
    const data = await res.json()
    return Array.isArray(data) ? data : []
  },

  async create(name: string): Promise<{ api_key: string }> {
    const res = await httpClient.post(
      `${API_BASE}/api-keys`,
      { name: name || '' },
      getAuthHeaders()
    )
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error((body as { error?: string }).error || '创建失败')
    }
    return res.json()
  },

  async revoke(id: string): Promise<void> {
    const res = await httpClient.delete(
      `${API_BASE}/api-keys/${id}`,
      getAuthHeaders()
    )
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error((body as { error?: string }).error || '删除失败')
    }
  },
}
