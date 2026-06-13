import { tradersApi } from './traders'
import { modelsApi } from './models'
import { exchangesApi } from './exchanges'
import { dashboardApi } from './dashboard'
import { competitionApi } from './competition'
import { userApi } from './user'
import { copyTradeApi } from './copyTrade'
import { systemApi } from './system'
import { apiKeysApi } from './apiKeys'
import { strategiesApi } from './strategies'

/** Unified API facade — domain modules composed for backward compatibility */
export const api = {
  ...tradersApi,
  ...modelsApi,
  ...exchangesApi,
  ...dashboardApi,
  ...competitionApi,
  ...userApi,
  ...copyTradeApi,
  ...systemApi,
  ...apiKeysApi,
  ...strategiesApi,
}

export { authApi } from './auth'
export { getAuthHeaders, API_BASE } from './client'
export type { StrategyRecord, StrategySignal, BacktestRecord } from './strategies'
