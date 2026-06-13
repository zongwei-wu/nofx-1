import useSWR from 'swr'
import { useLanguage } from '../../contexts/LanguageContext'
import { api } from '../../lib/api'
import type { PerformanceAnalysis } from './types'
import {
  AILearningEmpty,
  AILearningError,
  AILearningLoading,
} from './sections/AILearningStates'
import { PerformanceOverview } from './sections/PerformanceOverview'
import { SymbolAnalysis } from './sections/SymbolAnalysis'

interface AILearningProps {
  traderId: string
}

export default function AILearning({ traderId }: AILearningProps) {
  const { language } = useLanguage()
  const { data: performance, error } = useSWR<PerformanceAnalysis>(
    traderId ? `performance-${traderId}` : 'performance',
    () => api.getPerformance(traderId),
    {
      refreshInterval: 30000,
      revalidateOnFocus: false,
      dedupingInterval: 20000,
    }
  )

  if (error) return <AILearningError language={language} />
  if (!performance) return <AILearningLoading language={language} />
  if (performance.total_trades === 0) {
    return <AILearningEmpty language={language} />
  }

  const symbolStats = performance.symbol_stats || {}
  const symbolStatsList = Object.values(symbolStats)
    .filter((stat) => stat != null)
    .sort((a, b) => (b.total_pn_l || 0) - (a.total_pn_l || 0))

  return (
    <div className="space-y-8">
      <PerformanceOverview performance={performance} language={language} />
      <SymbolAnalysis
        performance={performance}
        symbolStats={symbolStats}
        symbolStatsList={symbolStatsList}
        language={language}
      />
    </div>
  )
}
