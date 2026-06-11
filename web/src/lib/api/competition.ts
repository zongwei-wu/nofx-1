import type { CompetitionData } from '../../types'
import { httpClient } from '../httpClient'
import { API_BASE, throwIfNotOk } from './client'

export const competitionApi = {
  async getCompetition(): Promise<CompetitionData> {
    const res = await httpClient.get(`${API_BASE}/competition`)
    await throwIfNotOk(res, '获取竞赛数据失败')
    return res.json()
  },
}
