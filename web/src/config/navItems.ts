import { FEATURES, type FeatureKey } from './features'

export type NavItem = {
  key: string
  path: string
  feature: FeatureKey | null
  labelKey?: string
  label?: string
}

export type AppPageKey = NavItem['key']

/** 已登录用户主导航（faq 无 feature 限制） */
export const LOGGED_IN_NAV_ITEMS: NavItem[] = [
  {
    key: 'competition',
    path: '/competition',
    feature: FEATURES.competition,
    labelKey: 'realtimeNav',
  },
  {
    key: 'traders',
    path: '/traders',
    feature: FEATURES.ai_trader,
    labelKey: 'configNav',
  },
  {
    key: 'trader',
    path: '/dashboard',
    feature: FEATURES.ai_trader,
    labelKey: 'dashboardNav',
  },
  {
    key: 'copy-trading',
    path: '/copy-trading',
    feature: FEATURES.leaderboard,
    label: '排行榜',
  },
  {
    key: 'copy-trade',
    path: '/copy-trade',
    feature: FEATURES.copy_trade,
    label: '跟单管理',
  },
  {
    key: 'symbols',
    path: '/symbols',
    feature: FEATURES.symbols,
    label: '币种管理',
  },
  {
    key: 'strategies',
    path: '/strategies',
    feature: FEATURES.strategy,
    label: '策略',
  },
  { key: 'faq', path: '/faq', feature: null, labelKey: 'faqNav' },
  {
    key: 'access-keys',
    path: '/access-keys',
    feature: null,
    label: 'API Keys',
  },
]

const PATH_TO_PAGE: Record<string, AppPageKey> = Object.fromEntries(
  LOGGED_IN_NAV_ITEMS.map((item) => [item.path, item.key])
)

/** 根据当前路径解析导航高亮 key */
export function getCurrentPageFromPath(pathname: string): AppPageKey {
  if (pathname.startsWith('/strategies')) return 'strategies'
  return PATH_TO_PAGE[pathname] ?? 'competition'
}

/** 落地页锚点导航 */
export const HOME_NAV_ITEMS = [
  { key: 'features', hash: '#features', labelKey: 'features' as const },
  { key: 'how-it-works', hash: '#how-it-works', labelKey: 'howItWorks' as const },
]
