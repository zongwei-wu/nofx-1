import { httpClient } from '../httpClient'
import { API_BASE, getAuthHeaders, throwIfNotOk } from './client'

export const userApi = {
  async getUserSignalSource(): Promise<{ coin_pool_url: string; oi_top_url: string }> {
    const res = await httpClient.get(
      `${API_BASE}/user/signal-sources`,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取用户信号源配置失败')
    return res.json()
  },

  async saveUserSignalSource(coinPoolUrl: string, oiTopUrl: string): Promise<void> {
    const res = await httpClient.post(
      `${API_BASE}/user/signal-sources`,
      { coin_pool_url: coinPoolUrl, oi_top_url: oiTopUrl },
      getAuthHeaders()
    )
    await throwIfNotOk(res, '保存用户信号源配置失败')
  },

  async getServerIP(): Promise<{ public_ip: string; message: string }> {
    const res = await httpClient.get(`${API_BASE}/server-ip`, getAuthHeaders())
    await throwIfNotOk(res, '获取服务器IP失败')
    return res.json()
  },
}
