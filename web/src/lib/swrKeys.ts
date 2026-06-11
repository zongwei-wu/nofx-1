/** SWR cache keys — include auth guard token in key factories where needed */

export const swrKeys = {
  traders: (authed: boolean) => (authed ? 'traders' : null),
  status: (authed: boolean, traderId?: string) =>
    authed ? (traderId ? `status-${traderId}` : 'status') : null,
  account: (authed: boolean, traderId?: string) =>
    authed ? (traderId ? `account-${traderId}` : 'account') : null,
  positions: (authed: boolean, traderId?: string) =>
    authed ? (traderId ? `positions-${traderId}` : 'positions') : null,
  decisions: (authed: boolean, traderId?: string) =>
    authed ? (traderId ? `decisions-${traderId}` : 'decisions') : null,
  statistics: (authed: boolean, traderId?: string) =>
    authed ? (traderId ? `statistics-${traderId}` : 'statistics') : null,
  competition: () => 'competition',
  equityHistory: (traderId?: string) =>
    traderId ? `equity-history-${traderId}` : 'equity-history',
  performance: (traderId?: string) =>
    traderId ? `performance-${traderId}` : 'performance',
} as const
