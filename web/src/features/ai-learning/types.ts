export interface TradeOutcome {
  symbol: string
  side: string
  quantity: number
  leverage: number
  open_price: number
  close_price: number
  position_value: number
  margin_used: number
  pn_l: number
  pn_l_pct: number
  duration: string
  open_time: string
  close_time: string
  was_stop_loss: boolean
}

export interface SymbolPerformance {
  symbol: string
  total_trades: number
  winning_trades: number
  losing_trades: number
  win_rate: number
  total_pn_l: number
  avg_pn_l: number
}

export interface PerformanceAnalysis {
  total_trades: number
  winning_trades: number
  losing_trades: number
  win_rate: number
  avg_win: number
  avg_loss: number
  profit_factor: number
  sharpe_ratio: number
  recent_trades: TradeOutcome[]
  symbol_stats: { [key: string]: SymbolPerformance }
  best_symbol: string
  worst_symbol: string
}
