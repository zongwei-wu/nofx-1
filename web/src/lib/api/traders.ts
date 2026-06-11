import type {
  TraderInfo,
  TraderConfigData,
  CreateTraderRequest,
} from '../../types'
import { httpClient } from '../httpClient'
import { API_BASE, getAuthHeaders, parseJson, throwIfNotOk } from './client'

export const tradersApi = {
  async getTraders(): Promise<TraderInfo[]> {
    const res = await httpClient.get(`${API_BASE}/my-traders`, getAuthHeaders())
    await throwIfNotOk(res, '获取trader列表失败')
    return parseJson(res)
  },

  async getPublicTraders(): Promise<TraderInfo[]> {
    const res = await httpClient.get(`${API_BASE}/traders`)
    await throwIfNotOk(res, '获取公开trader列表失败')
    return parseJson(res)
  },

  async createTrader(request: CreateTraderRequest): Promise<TraderInfo> {
    const res = await httpClient.post(`${API_BASE}/traders`, request, getAuthHeaders())
    await throwIfNotOk(res, '创建交易员失败')
    return parseJson(res)
  },

  async deleteTrader(traderId: string): Promise<void> {
    const res = await httpClient.delete(
      `${API_BASE}/traders/${traderId}`,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '删除交易员失败')
  },

  async startTrader(traderId: string): Promise<void> {
    const res = await httpClient.post(
      `${API_BASE}/traders/${traderId}/start`,
      undefined,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '启动交易员失败')
  },

  async stopTrader(traderId: string): Promise<void> {
    const res = await httpClient.post(
      `${API_BASE}/traders/${traderId}/stop`,
      undefined,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '停止交易员失败')
  },

  async updateTraderPrompt(traderId: string, customPrompt: string): Promise<void> {
    const res = await httpClient.put(
      `${API_BASE}/traders/${traderId}/prompt`,
      { custom_prompt: customPrompt },
      getAuthHeaders()
    )
    await throwIfNotOk(res, '更新自定义策略失败')
  },

  async getTraderConfig(traderId: string): Promise<TraderConfigData> {
    const res = await httpClient.get(
      `${API_BASE}/traders/${traderId}/config`,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取交易员配置失败')
    return parseJson(res)
  },

  async updateTrader(traderId: string, request: CreateTraderRequest): Promise<TraderInfo> {
    const res = await httpClient.put(
      `${API_BASE}/traders/${traderId}`,
      request,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '更新交易员失败')
    return parseJson(res)
  },

  async getPublicTraderConfig(traderId: string): Promise<unknown> {
    const res = await httpClient.get(`${API_BASE}/trader/${traderId}/config`)
    await throwIfNotOk(res, '获取公开交易员配置失败')
    return parseJson(res)
  },
}
