import type { AccountInfo } from '../../../types'
import { t, type Language } from '../../../i18n/translations'
import { StatCard } from './StatCard'

interface AccountOverviewProps {
  account?: AccountInfo
  language: Language
}

export function AccountOverview({ account, language }: AccountOverviewProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
      <StatCard
        title={t('initialBalance', language)}
        value={`${account?.initial_balance?.toFixed(2) || '0.00'} USDT`}
      />
      <StatCard
        title={t('totalEquity', language)}
        value={`${account?.total_equity?.toFixed(2) || '0.00'} USDT`}
        change={account?.total_pnl_pct || 0}
        positive={(account?.total_pnl ?? 0) > 0}
      />
      <StatCard
        title={t('availableBalance', language)}
        value={`${account?.available_balance?.toFixed(2) || '0.00'} USDT`}
        subtitle={`${account?.available_balance && account?.total_equity ? ((account.available_balance / account.total_equity) * 100).toFixed(1) : '0.0'}% ${t('free', language)}`}
      />
      <StatCard
        title={t('totalPnL', language)}
        value={`${account?.total_pnl !== undefined && account.total_pnl >= 0 ? '+' : ''}${account?.total_pnl?.toFixed(2) || '0.00'} USDT`}
        change={account?.total_pnl_pct || 0}
        positive={(account?.total_pnl ?? 0) >= 0}
      />
      <StatCard
        title={t('positions', language)}
        value={`${account?.position_count || 0}`}
        subtitle={`${t('margin', language)}: ${account?.margin_used_pct?.toFixed(1) || '0.0'}%`}
      />
    </div>
  )
}
