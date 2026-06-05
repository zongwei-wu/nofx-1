import { normalizeTradingSymbol } from './tradeEventChartUtils'

export type ChartInterval = '1h' | '4h'

/** TradingView 图表主体高度（不含版权脚注） */
export const DEFAULT_TV_CHART_HEIGHT = 520
export const TV_COPYRIGHT_HEIGHT = 28

const INTERVAL_MAP: Record<ChartInterval, string> = {
  '1h': '60',
  '4h': '240',
}

/** 币安 U 本位永续合约 symbol（TradingView 格式） */
export function toTradingViewSymbol(symbol?: string): string {
  const normalized = normalizeTradingSymbol(symbol)
  if (!normalized) return 'BINANCE:BTCUSDT.P'
  const base = normalized.replace(/USDT$/i, '')
  return `BINANCE:${base}USDT.P`
}

export function toTradingViewInterval(interval: ChartInterval): string {
  return INTERVAL_MAP[interval] ?? '60'
}
