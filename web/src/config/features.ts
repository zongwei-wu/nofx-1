export const FEATURES = {
  competition: 'competition',
  ai_trader: 'ai_trader',
  gateway: 'gateway',
  hermes: 'gateway', // 兼容旧 key
  leaderboard: 'leaderboard',
  copy_trade: 'copy_trade',
  symbols: 'symbols',
  strategy: 'strategy',
} as const

export type FeatureKey = (typeof FEATURES)[keyof typeof FEATURES]
