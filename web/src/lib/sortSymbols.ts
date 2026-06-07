import { normalizeTradingSymbol } from '../components/trade-events/tradeEventChartUtils'

export interface SortSymbolsOptions {
  customOrder?: string[]
  useCustomOrder?: boolean
  notionalBySymbol?: Record<string, number>
  /** 特别关注列表，始终排在最前（按此列表顺序） */
  starredOrder?: string[]
}

function normalizeList(symbols: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const sym of symbols) {
    const n = normalizeTradingSymbol(sym)
    if (!n || seen.has(n)) continue
    seen.add(n)
    out.push(n)
  }
  return out
}

function compareByNotional(
  a: string,
  b: string,
  notionalBySymbol: Record<string, number>
): number {
  const va = notionalBySymbol[a] ?? 0
  const vb = notionalBySymbol[b] ?? 0
  if (vb !== va) return vb - va
  return a.localeCompare(b)
}

function sortByNotional(symbols: string[], notionalBySymbol: Record<string, number>): string[] {
  return [...symbols].sort((a, b) => compareByNotional(a, b, notionalBySymbol))
}

function sortWithoutStarred(
  symbols: string[],
  options: SortSymbolsOptions
): string[] {
  if (symbols.length <= 1) return symbols

  const notional = options.notionalBySymbol ?? {}
  const customOrder = normalizeList(options.customOrder ?? [])

  if (options.useCustomOrder && customOrder.length > 0) {
    const index = new Map(customOrder.map((s, i) => [s, i]))
    const inCustom = symbols.filter((s) => index.has(s))
    const notInCustom = symbols.filter((s) => !index.has(s))
    inCustom.sort((a, b) => (index.get(a) ?? 0) - (index.get(b) ?? 0))
    return [...inCustom, ...sortByNotional(notInCustom, notional)]
  }

  return sortByNotional(symbols, notional)
}

function applyStarredFirst(
  symbols: string[],
  starredOrder: string[],
  options: SortSymbolsOptions
): string[] {
  const normalized = normalizeList(symbols)
  const starred = normalizeList(starredOrder)
  if (starred.length === 0) {
    return sortWithoutStarred(normalized, options)
  }

  const symbolSet = new Set(normalized)
  const starredFirst = starred.filter((s) => symbolSet.has(s))
  const starredSet = new Set(starredFirst)
  const rest = normalized.filter((s) => !starredSet.has(s))
  return [...starredFirst, ...sortWithoutStarred(rest, options)]
}

/**
 * 按用户偏好排序币种列表：特别关注优先，其次自定义顺序或持仓名义价值
 */
export function sortSymbolsByPreference(
  symbols: string[],
  options: SortSymbolsOptions = {}
): string[] {
  return applyStarredFirst(symbols, options.starredOrder ?? [], options)
}
