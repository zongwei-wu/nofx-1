export const FEATURES = {
  competition: 'competition',
  ai_trader: 'ai_trader',
  hermes: 'hermes',
  leaderboard: 'leaderboard',
  copy_trade: 'copy_trade',
  symbols: 'symbols',
} as const

export type FeatureKey = (typeof FEATURES)[keyof typeof FEATURES]
