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
        action: { type: 'string', enum: ['open_long', 'open_short', 'close_long', 'close_short'], description: '交易动作' },
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
          leverage: a.leverage || 10,
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
