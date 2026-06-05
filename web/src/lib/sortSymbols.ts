import { normalizeTradingSymbol } from '../components/trade-events/tradeEventChartUtils'

export interface SortSymbolsOptions {
  customOrder?: string[]
  useCustomOrder?: boolean
  notionalBySymbol?: Record<string, number>
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

/**
 * 按用户偏好或持仓名义价值排序币种列表
 */
export function sortSymbolsByPreference(
  symbols: string[],
  options: SortSymbolsOptions = {}
): string[] {
  const normalized = normalizeList(symbols)
  if (normalized.length <= 1) return normalized

  const notional = options.notionalBySymbol ?? {}
  const customOrder = normalizeList(options.customOrder ?? [])

  if (options.useCustomOrder && customOrder.length > 0) {
    const index = new Map(customOrder.map((s, i) => [s, i]))
    const inCustom = normalized.filter((s) => index.has(s))
    const notInCustom = normalized.filter((s) => !index.has(s))
    inCustom.sort((a, b) => (index.get(a) ?? 0) - (index.get(b) ?? 0))
    return [...inCustom, ...sortByNotional(notInCustom, notional)]
  }

  return sortByNotional(normalized, notional)
}
