import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  Statistics,
} from '../../types'
import { httpClient } from '../httpClient'
import { API_BASE, getAuthHeaders, throwIfNotOk } from './client'

function traderQuery(path: string, traderId?: string) {
  return traderId ? `${path}?trader_id=${traderId}` : path
}

export const dashboardApi = {
  async getStatus(traderId?: string): Promise<SystemStatus> {
    const res = await httpClient.get(
      traderQuery(`${API_BASE}/status`, traderId),
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取系统状态失败')
    return res.json()
  },

  async getAccount(traderId?: string): Promise<AccountInfo> {
    const res = await httpClient.request(traderQuery(`${API_BASE}/account`, traderId), {
      cache: 'no-store',
      headers: {
        ...getAuthHeaders(),
        'Cache-Control': 'no-cache',
      },
    })
    await throwIfNotOk(res, '获取账户信息失败')
    return res.json()
  },

  async getPositions(traderId?: string): Promise<Position[]> {
    const res = await httpClient.get(
      traderQuery(`${API_BASE}/positions`, traderId),
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取持仓列表失败')
    return res.json()
  },

  async getDecisions(traderId?: string): Promise<DecisionRecord[]> {
    const res = await httpClient.get(
      traderQuery(`${API_BASE}/decisions`, traderId),
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取决策日志失败')
    return res.json()
  },

  async getLatestDecisions(
    traderId?: string,
    limit = 5
  ): Promise<DecisionRecord[]> {
    const params = new URLSearchParams()
    if (traderId) params.append('trader_id', traderId)
    params.append('limit', limit.toString())
    const res = await httpClient.get(
      `${API_BASE}/decisions/latest?${params}`,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取最新决策失败')
    return res.json()
  },

  async getStatistics(traderId?: string): Promise<Statistics> {
    const res = await httpClient.get(
      traderQuery(`${API_BASE}/statistics`, traderId),
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取统计信息失败')
    return res.json()
  },

  async getEquityHistory(traderId?: string): Promise<unknown[]> {
    const res = await httpClient.get(
      traderQuery(`${API_BASE}/equity-history`, traderId),
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取历史数据失败')
    return res.json()
  },

  async getEquityHistoryBatch(traderIds: string[]): Promise<unknown> {
    const res = await httpClient.post(`${API_BASE}/equity-history-batch`, {
      trader_ids: traderIds,
    })
    await throwIfNotOk(res, '获取批量历史数据失败')
    return res.json()
  },

  async getTopTraders(): Promise<unknown[]> {
    const res = await httpClient.get(`${API_BASE}/top-traders`)
    await throwIfNotOk(res, '获取前5名交易员失败')
    return res.json()
  },

  async getPerformance(traderId?: string): Promise<unknown> {
    const res = await httpClient.get(
      traderQuery(`${API_BASE}/performance`, traderId),
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取AI学习数据失败')
    return res.json()
  },
}
