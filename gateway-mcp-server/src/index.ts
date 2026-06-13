#!/usr/bin/env node
import { Server } from '@modelcontextprotocol/sdk/server/index.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from '@modelcontextprotocol/sdk/types.js'
import { gatewayClient, setApiKey } from './client.js'

const server = new Server(
  { name: 'gateway', version: '1.0.0' },
  { capabilities: { tools: {} } }
)

function json(data: unknown) {
  return { content: [{ type: 'text' as const, text: JSON.stringify(data, null, 2) }] }
}

const tools = [
  {
    name: 'gateway_auth',
    description: '认证：提供 NOFX API Key 以建立会话。必须先调用此工具才能使用其他交易功能',
    inputSchema: {
      type: 'object',
      properties: {
        api_key: { type: 'string', description: '从 NOFX Web UI /access-keys 页面生成的 API Key，格式 nfx_sk_...' },
      },
      required: ['api_key'],
    },
  },
  {
    name: 'gateway_get_balance',
    description: '查询交易所账户余额和净值',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string', description: '交易所ID: binance/okx/hyperliquid/aster/gate' },
      },
      required: ['exchange_id'],
    },
  },
  {
    name: 'gateway_get_positions',
    description: '查询当前持仓列表',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string', description: '交易所ID' },
      },
      required: ['exchange_id'],
    },
  },
  {
    name: 'gateway_get_market_price',
    description: '查询指定交易对的当前市价',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string', description: '交易所ID' },
        symbol: { type: 'string', description: '交易对，如 ETHUSDT' },
      },
      required: ['exchange_id', 'symbol'],
    },
  },
  {
    name: 'gateway_get_klines',
    description: '获取K线数据（仅支持 binance/okx）',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string', description: '交易所ID (仅 binance/okx)' },
        symbol: { type: 'string', description: '交易对' },
        interval: { type: 'string', description: 'K线周期: 1m/5m/15m/1h/4h/1d' },
        limit: { type: 'integer', description: '数量，默认200，最大500' },
      },
      required: ['exchange_id', 'symbol'],
    },
  },
  {
    name: 'gateway_set_leverage',
    description: '设置交易对杠杆倍数',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string', description: '交易所ID' },
        symbol: { type: 'string', description: '交易对' },
        leverage: { type: 'integer', description: '杠杆倍数 1-125' },
      },
      required: ['exchange_id', 'symbol', 'leverage'],
    },
  },
  {
    name: 'gateway_place_order',
    description: '下单交易（开多/开空/平多/平空）。必须 confirmed: true',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string', description: '交易所ID' },
        action: { type: 'string', enum: ['open_long', 'open_short', 'add_long', 'add_short', 'close_long', 'close_short', 'reduce_long', 'reduce_short'], description: '交易动作' },
        symbol: { type: 'string', description: '交易对，如 ETHUSDT' },
        quantity: { type: 'number', description: '数量（合约张数）' },
        leverage: { type: 'integer', description: '杠杆倍数' },
        confirmed: { type: 'boolean', description: '必须为 true 才执行' },
      },
      required: ['exchange_id', 'action', 'symbol', 'quantity', 'confirmed'],
    },
  },
  {
    name: 'gateway_set_stop_loss',
    description: '设置持仓止损',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string', description: '交易所ID' },
        symbol: { type: 'string', description: '交易对' },
        position_side: { type: 'string', enum: ['LONG', 'SHORT'], description: '持仓方向' },
        quantity: { type: 'number', description: '数量' },
        stop_price: { type: 'number', description: '止损触发价' },
      },
      required: ['exchange_id', 'symbol', 'position_side', 'quantity', 'stop_price'],
    },
  },
  {
    name: 'gateway_set_take_profit',
    description: '设置持仓止盈',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string', description: '交易所ID' },
        symbol: { type: 'string', description: '交易对' },
        position_side: { type: 'string', enum: ['LONG', 'SHORT'], description: '持仓方向' },
        quantity: { type: 'number', description: '数量' },
        stop_price: { type: 'number', description: '止盈触发价' },
      },
      required: ['exchange_id', 'symbol', 'position_side', 'quantity', 'stop_price'],
    },
  },
  // 指标类
  {
    name: 'gateway_indicators_list',
    description: '列出所有可用技术指标及参数说明',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'gateway_indicators_compute',
    description: '对指定交易对计算一组技术指标',
    inputSchema: {
      type: 'object',
      properties: {
        symbol: { type: 'string', description: '交易对，如 BTCUSDT' },
        interval: { type: 'string', description: 'K线周期，默认 1h' },
        limit: { type: 'integer', description: 'K线数量，默认 200' },
        indicators: { type: 'array', items: { type: 'string' }, description: '指标列表，如 EMA20, RSI14, MACD' },
      },
      required: ['symbol'],
    },
  },
  {
    name: 'gateway_indicators_compare',
    description: '比较多个交易对的指标值',
    inputSchema: {
      type: 'object',
      properties: {
        symbols: { type: 'array', items: { type: 'string' }, description: '交易对列表' },
        interval: { type: 'string', description: 'K线周期' },
        indicators: { type: 'array', items: { type: 'string' }, description: '指标列表' },
      },
      required: ['symbols'],
    },
  },
  // 策略管理类
  {
    name: 'gateway_strategy_list',
    description: '列出我的所有策略',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'gateway_strategy_get',
    description: '获取策略详情',
    inputSchema: {
      type: 'object',
      properties: { id: { type: 'string', description: '策略 ID' } },
      required: ['id'],
    },
  },
  {
    name: 'gateway_strategy_create',
    description: '创建新策略',
    inputSchema: {
      type: 'object',
      properties: {
        name: { type: 'string' },
        exchange_id: { type: 'string' },
        symbol: { type: 'string' },
        timeframe: { type: 'string' },
        direction: { type: 'string', enum: ['long_only', 'short_only', 'both'] },
        entry_conditions: { type: 'array' },
        exit_conditions: { type: 'array' },
        exit_rules: { type: 'object' },
        risk: { type: 'object' },
      },
      required: ['name', 'exchange_id', 'symbol'],
    },
  },
  {
    name: 'gateway_strategy_update',
    description: '更新策略配置',
    inputSchema: {
      type: 'object',
      properties: {
        id: { type: 'string' },
        name: { type: 'string' },
        exchange_id: { type: 'string' },
        symbol: { type: 'string' },
        timeframe: { type: 'string' },
        direction: { type: 'string' },
        entry_conditions: { type: 'array' },
        exit_conditions: { type: 'array' },
        exit_rules: { type: 'object' },
        risk: { type: 'object' },
      },
      required: ['id'],
    },
  },
  {
    name: 'gateway_strategy_delete',
    description: '删除策略',
    inputSchema: {
      type: 'object',
      properties: { id: { type: 'string' } },
      required: ['id'],
    },
  },
  {
    name: 'gateway_strategy_validate',
    description: '在当前行情中验证策略信号',
    inputSchema: {
      type: 'object',
      properties: { id: { type: 'string' } },
      required: ['id'],
    },
  },
  {
    name: 'gateway_strategy_activate',
    description: '激活策略开始自动交易',
    inputSchema: {
      type: 'object',
      properties: { id: { type: 'string' } },
      required: ['id'],
    },
  },
  // 回测类
  {
    name: 'gateway_backtest_run',
    description: '对策略启动一次回测',
    inputSchema: {
      type: 'object',
      properties: {
        strategy_id: { type: 'string' },
        symbol: { type: 'string' },
        timeframe: { type: 'string' },
        initial_capital: { type: 'number' },
      },
      required: ['strategy_id'],
    },
  },
  {
    name: 'gateway_backtest_get',
    description: '获取回测结果',
    inputSchema: {
      type: 'object',
      properties: { id: { type: 'string' } },
      required: ['id'],
    },
  },
]

server.setRequestHandler(ListToolsRequestSchema, async () => ({ tools }))

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name } = request.params
  const a = (request.params.arguments || {}) as Record<string, any>

  try {
    switch (name) {
      case 'gateway_auth': {
        setApiKey(a.api_key)
        return json({ status: 'authenticated', message: 'API Key 已设置，可以开始交易' })
      }

      case 'gateway_get_balance':
        return json(await gatewayClient.get('/api/gateway/balance', { exchange_id: a.exchange_id }))

      case 'gateway_get_positions':
        return json(await gatewayClient.get('/api/gateway/positions', { exchange_id: a.exchange_id }))

      case 'gateway_get_market_price':
        return json(await gatewayClient.get('/api/gateway/market-price', {
          exchange_id: a.exchange_id,
          symbol: a.symbol,
        }))

      case 'gateway_get_klines':
        return json(await gatewayClient.get('/api/gateway/klines', {
          exchange_id: a.exchange_id,
          symbol: a.symbol,
          interval: a.interval || '1h',
          limit: String(a.limit || 200),
        }))

      case 'gateway_set_leverage':
        return json(await gatewayClient.post('/api/gateway/leverage', {
          exchange_id: a.exchange_id,
          symbol: a.symbol,
          leverage: a.leverage,
        }))

      case 'gateway_place_order': {
        if (!a.confirmed) {
          return json({ error: '交易需确认，请设置 confirmed: true 后重试' })
        }
        return json(await gatewayClient.post('/api/gateway/trade', {
          exchange_id: a.exchange_id,
          action: a.action,
          symbol: a.symbol,
          quantity: a.quantity,
          leverage: a.leverage || 0,
          confirmed: true,
        }))
      }

      case 'gateway_set_stop_loss':
        return json(await gatewayClient.post('/api/gateway/stop-loss', {
          exchange_id: a.exchange_id,
          symbol: a.symbol,
          position_side: a.position_side,
          quantity: a.quantity,
          stop_price: a.stop_price,
        }))

      case 'gateway_set_take_profit':
        return json(await gatewayClient.post('/api/gateway/take-profit', {
          exchange_id: a.exchange_id,
          symbol: a.symbol,
          position_side: a.position_side,
          quantity: a.quantity,
          stop_price: a.stop_price,
        }))

      // 指标
      case 'gateway_indicators_list':
        return json(await gatewayClient.get('/api/gateway/indicators/list'))
      case 'gateway_indicators_compute':
        return json(await gatewayClient.post('/api/gateway/indicators/compute', {
          symbol: a.symbol,
          interval: a.interval || '1h',
          limit: a.limit || 200,
          indicators: a.indicators || ['EMA20', 'RSI14', 'MACD'],
        }))
      case 'gateway_indicators_compare':
        return json(await gatewayClient.post('/api/gateway/indicators/compare', {
          symbols: a.symbols,
          interval: a.interval || '1h',
          indicators: a.indicators || ['RSI14'],
        }))

      // 策略
      case 'gateway_strategy_list':
        return json(await gatewayClient.get('/api/gateway/strategies'))
      case 'gateway_strategy_get':
        return json(await gatewayClient.get(`/api/gateway/strategies/${a.id}`))
      case 'gateway_strategy_create':
        return json(await gatewayClient.post('/api/gateway/strategies', a))
      case 'gateway_strategy_update': {
        const { id, ...body } = a
        return json(await gatewayClient.put(`/api/gateway/strategies/${id}`, body))
      }
      case 'gateway_strategy_delete':
        return json(await gatewayClient.delete(`/api/gateway/strategies/${a.id}`))
      case 'gateway_strategy_validate':
        return json(await gatewayClient.post(`/api/gateway/strategies/${a.id}/validate`, {}))
      case 'gateway_strategy_activate':
        return json(await gatewayClient.post(`/api/gateway/strategies/${a.id}/activate`, {}))

      // 回测
      case 'gateway_backtest_run':
        return json(await gatewayClient.post(`/api/gateway/strategies/${a.strategy_id}/backtest`, {
          symbol: a.symbol,
          timeframe: a.timeframe,
          initial_capital: a.initial_capital,
        }))
      case 'gateway_backtest_get':
        return json(await gatewayClient.get(`/api/gateway/backtests/${a.id}`))

      default:
        return json({ error: `未知工具: ${name}` })
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err)
    return json({ error: msg })
  }
})

async function main() {
  const transport = new StdioServerTransport()
  await server.connect(transport)
}

main().catch(console.error)
