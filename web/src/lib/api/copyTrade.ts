import { httpClient } from '../httpClient'
import type { Position } from '../../types'
import type { LeaderboardTrader } from '../../components/copy-trade/copyTradeLeaderboardUtils'
import { API_BASE, getAuthHeaders, throwIfNotOk } from './client'

export interface CopyTradeSettings {
  ai_trader_id: string
  execution_exchange_id?: string
  ai_trader_name?: string
  ai_model_name?: string
  fallback_used?: boolean
}

export interface CopyConfig {
  id: number
  portfolio_id: string
  nickname: string
  enabled: boolean
  auto_follow: boolean
  max_copy_size: number
  size_multiplier: number
  copy_open_only: boolean
  last_order_time: number
}

export interface CopyRecord {
  id: number
  portfolio_id: string
  nickname: string
  order_id?: string
  symbol: string
  side: string
  position_side: string
  executed_qty: number
  avg_price: number
  close_price?: number
  total_pnl: number
  status: string
  error_message?: string
  lead_order_time: number
  copy_time?: string
  close_time?: string
}

export interface CopyConfigPayload {
  id?: number
  portfolio_id: string
  nickname: string
  enabled: boolean
  auto_follow: boolean
  max_copy_size: number
  size_multiplier: number
  copy_open_only: boolean
}

export interface CopyOrderPayload {
  portfolio_id: string
  nickname?: string
  symbol: string
  side: string
  position_side: string
  executed_qty: number
  avg_price?: number
  order_time?: number
  skip_ai?: boolean
  [key: string]: unknown
}

export interface LeaderboardResponse {
  code?: string
  data?: {
    highestPnlLeads?: LeaderboardTrader[]
    highestRoiLeads?: LeaderboardTrader[]
  }
  stale?: boolean
  warning?: string
  error?: string
}

function dedupeLeaderboard(
  highestPnlLeads: LeaderboardTrader[] = [],
  highestRoiLeads: LeaderboardTrader[] = []
): LeaderboardTrader[] {
  const all = [...highestPnlLeads, ...highestRoiLeads]
  const unique = new Map<string, LeaderboardTrader>()
  all.forEach((raw) => {
    if (!raw?.leadPortfolioId) return
    unique.set(raw.leadPortfolioId, {
      ...raw,
      leadPortfolioId: raw.leadPortfolioId,
      nickname: raw.nickname || '',
      pnl: Number(raw.pnl) || 0,
      roi: Number(raw.roi) || 0,
    })
  })
  return Array.from(unique.values())
}

export const copyTradeApi = {
  async getCopyTradeSettings(): Promise<CopyTradeSettings> {
    const res = await httpClient.get(
      `${API_BASE}/copy-trade/settings`,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取跟单设置失败')
    return res.json()
  },

  async updateCopyTradeSettings(
    aiTraderId: string,
    executionExchangeId?: string
  ): Promise<void> {
    const res = await httpClient.put(
      `${API_BASE}/copy-trade/settings`,
      {
        ai_trader_id: aiTraderId,
        execution_exchange_id: executionExchangeId ?? '',
      },
      getAuthHeaders()
    )
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error((data as { error?: string }).error || '保存跟单设置失败')
    }
  },

  async getConfigs(): Promise<CopyConfig[]> {
    const res = await httpClient.get(
      `${API_BASE}/copy-trade/configs`,
      getAuthHeaders()
    )
    if (!res.ok) return []
    const data = await res.json()
    return Array.isArray(data) ? data : []
  },

  async getRecords(): Promise<CopyRecord[]> {
    const res = await httpClient.get(
      `${API_BASE}/copy-trade/records`,
      getAuthHeaders()
    )
    if (!res.ok) return []
    const data = await res.json()
    return Array.isArray(data) ? data : []
  },

  async getLeaderboardRaw(): Promise<LeaderboardResponse> {
    const res = await httpClient.get(
      `${API_BASE}/copy-trading/leaderboard`,
      getAuthHeaders()
    )
    return res.json()
  },

  async getLeaderboard(): Promise<LeaderboardTrader[]> {
    const lbData = await copyTradeApi.getLeaderboardRaw()
    if (lbData.code === '000000' && lbData.data) {
      return dedupeLeaderboard(
        lbData.data.highestPnlLeads,
        lbData.data.highestRoiLeads
      )
    }
    return []
  },

  async getExchangePositions(): Promise<Position[]> {
    const res = await httpClient
      .get(`${API_BASE}/copy-trade/exchange-positions`, getAuthHeaders())
      .catch(() => null)
    if (!res?.ok) return []
    const data = await res.json()
    return Array.isArray(data) ? data : []
  },

  async upsertConfig(payload: CopyConfigPayload): Promise<void> {
    const res = await httpClient.post(
      `${API_BASE}/copy-trade/configs`,
      payload,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '保存跟单配置失败')
  },

  async sync(): Promise<{ copied?: number }> {
    const res = await httpClient.post(
      `${API_BASE}/copy-trade/sync`,
      undefined,
      getAuthHeaders()
    )
    return res.json()
  },

  async refreshPnl(): Promise<{ updated?: number; error?: string }> {
    const res = await httpClient.post(
      `${API_BASE}/copy-trade/refresh-pnl`,
      undefined,
      getAuthHeaders()
    )
    const data = await res.json()
    if (!res.ok) {
      throw new Error(data.error || '刷新盈亏失败')
    }
    return data
  },

  async copyOrder(payload: CopyOrderPayload): Promise<Response> {
    return httpClient.post(
      `${API_BASE}/copy-trade/copy-order`,
      payload,
      getAuthHeaders()
    )
  },

  async getLeaderboardOrders(
    portfolioId: string,
    pageSize = 10
  ): Promise<{
    code?: string
    data?: { list?: unknown[] }
    error?: string
  }> {
    const res = await httpClient.get(
      `${API_BASE}/copy-trading/orders?portfolio_id=${portfolioId}&page_size=${pageSize}`,
      getAuthHeaders()
    )
    return res.json()
  },

  async getMonitor(): Promise<unknown> {
    const res = await httpClient.get(
      `${API_BASE}/copy-trade/monitor`,
      getAuthHeaders()
    )
    await throwIfNotOk(res, '获取跟单监控数据失败')
    return res.json()
  },
}
