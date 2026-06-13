import { API_BASE, getAuthHeaders, parseJson, throwIfNotOk } from './client'

export interface StrategyRecord {
  id: string
  user_id: string
  name: string
  config: string
  status: string
  created_at: string
  updated_at: string
}

export interface StrategySignal {
  id: string
  strategy_id: string
  symbol: string
  signal: string
  price: number
  indicators: string
  order_result: string
  created_at: string
}

export interface BacktestRecord {
  id: string
  strategy_id: string
  symbol: string
  timeframe: string
  status: string
  error?: string
  initial_capital: number
  result?: Record<string, unknown>
  created_at: string
  completed_at?: string
}

export const strategiesApi = {
  async getStrategies(): Promise<StrategyRecord[]> {
    const res = await fetch(`${API_BASE}/strategies`, { headers: getAuthHeaders() })
    await throwIfNotOk(res, '获取策略列表失败')
    const data = await parseJson<{ strategies: StrategyRecord[] }>(res)
    return data.strategies ?? []
  },

  async getStrategy(id: string): Promise<StrategyRecord> {
    const res = await fetch(`${API_BASE}/strategies/${id}`, { headers: getAuthHeaders() })
    await throwIfNotOk(res, '获取策略详情失败')
    const data = await parseJson<{ strategy: StrategyRecord }>(res)
    return data.strategy
  },

  async createStrategy(body: Record<string, unknown>): Promise<StrategyRecord> {
    const res = await fetch(`${API_BASE}/strategies`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(body),
    })
    await throwIfNotOk(res, '创建策略失败')
    const data = await parseJson<{ strategy: StrategyRecord }>(res)
    return data.strategy
  },

  async updateStrategy(id: string, body: Record<string, unknown>): Promise<StrategyRecord> {
    const res = await fetch(`${API_BASE}/strategies/${id}`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(body),
    })
    await throwIfNotOk(res, '更新策略失败')
    const data = await parseJson<{ strategy: StrategyRecord }>(res)
    return data.strategy
  },

  async deleteStrategy(id: string): Promise<void> {
    const res = await fetch(`${API_BASE}/strategies/${id}`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    })
    await throwIfNotOk(res, '删除策略失败')
  },

  async activateStrategy(id: string): Promise<void> {
    const res = await fetch(`${API_BASE}/strategies/${id}/activate`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: '{}',
    })
    await throwIfNotOk(res, '激活策略失败')
  },

  async pauseStrategy(id: string): Promise<void> {
    const res = await fetch(`${API_BASE}/strategies/${id}/pause`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: '{}',
    })
    await throwIfNotOk(res, '暂停策略失败')
  },

  async validateStrategy(id: string): Promise<Record<string, unknown>> {
    const res = await fetch(`${API_BASE}/strategies/${id}/validate`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: '{}',
    })
    await throwIfNotOk(res, '验证策略失败')
    return parseJson(res)
  },

  async getStrategySignals(id: string): Promise<StrategySignal[]> {
    const res = await fetch(`${API_BASE}/strategies/${id}/signals`, { headers: getAuthHeaders() })
    await throwIfNotOk(res, '获取信号历史失败')
    const data = await parseJson<{ signals: StrategySignal[] }>(res)
    return data.signals ?? []
  },

  async startBacktest(strategyId: string, body: Record<string, unknown> = {}): Promise<{ backtest_id: string }> {
    const res = await fetch(`${API_BASE}/strategies/${strategyId}/backtest`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(body),
    })
    await throwIfNotOk(res, '启动回测失败')
    return parseJson(res)
  },

  async getBacktest(id: string): Promise<BacktestRecord> {
    const res = await fetch(`${API_BASE}/backtests/${id}`, { headers: getAuthHeaders() })
    await throwIfNotOk(res, '获取回测结果失败')
    return parseJson(res)
  },

  async getBacktests(): Promise<BacktestRecord[]> {
    const res = await fetch(`${API_BASE}/backtests`, { headers: getAuthHeaders() })
    await throwIfNotOk(res, '获取回测列表失败')
    const data = await parseJson<{ backtests: BacktestRecord[] }>(res)
    return data.backtests ?? []
  },
}
