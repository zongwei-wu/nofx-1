export interface TraderConfigData {
  trader_id?: string
  trader_name: string
  ai_model: string
  exchange_id: string
  btc_eth_leverage: number
  altcoin_leverage: number
  trading_symbols: string
  custom_prompt: string
  override_base_prompt: boolean
  system_prompt_template: string
  is_cross_margin: boolean
  use_coin_pool: boolean
  use_oi_top: boolean
  initial_balance?: number
  scan_interval_minutes: number
}
