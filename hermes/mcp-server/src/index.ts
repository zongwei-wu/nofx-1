#!/usr/bin/env node
import { Server } from '@modelcontextprotocol/sdk/server/index.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from '@modelcontextprotocol/sdk/types.js'
import { nofxClient } from './client.js'
import { encryptSensitiveData } from './crypto.js'
import { getExchangeCardSection } from './exchange-cards.js'
import {
  getPendingOtp,
  getSession,
  setPendingOtp,
  setSession,
} from './session.js'

const server = new Server(
  { name: 'hermes', version: '1.0.0' },
  { capabilities: { tools: {} } }
)

function apiBase(): string {
  return (process.env.NOFX_API_URL || 'http://localhost:8080').replace(/\/$/, '')
}

function textResult(text: string) {
  return { content: [{ type: 'text' as const, text }] }
}

function jsonResult(data: unknown) {
  return textResult(JSON.stringify(data, null, 2))
}

const tools = [
  {
    name: 'hermes_login',
    description: '使用邮箱密码登录 NOFX；若需 2FA 则返回 requires_otp，再调用 hermes_verify_otp',
    inputSchema: {
      type: 'object',
      properties: {
        email: { type: 'string' },
        password: { type: 'string' },
      },
      required: ['email', 'password'],
    },
  },
  {
    name: 'hermes_verify_otp',
    description: '完成 Google Authenticator 二次验证登录',
    inputSchema: {
      type: 'object',
      properties: {
        otp_code: { type: 'string' },
      },
      required: ['otp_code'],
    },
  },
  {
    name: 'hermes_auth_status',
    description: '检查当前 Hermes 登录状态',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'hermes_get_server_ip',
    description: '获取 NOFX 服务器公网 IP，用于交易所 API 白名单配置',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'hermes_get_exchange_card',
    description: '获取指定交易所的配置引导卡片（步骤说明）',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: {
          type: 'string',
          enum: ['binance', 'okx', 'hyperliquid', 'aster'],
        },
      },
      required: ['exchange_id'],
    },
  },
  {
    name: 'hermes_list_exchanges',
    description: '列出用户已配置的交易所（脱敏，不含 secret）',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'hermes_test_exchange',
    description: '测试交易所 API 连接（凭证经 RSA 加密传输）',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        api_key: { type: 'string' },
        secret_key: { type: 'string' },
        passphrase: { type: 'string' },
        testnet: { type: 'boolean' },
        hyperliquid_wallet_addr: { type: 'string' },
        aster_user: { type: 'string' },
        aster_signer: { type: 'string' },
        aster_private_key: { type: 'string' },
      },
      required: ['exchange_id'],
    },
  },
  {
    name: 'hermes_save_exchange_credentials',
    description: '保存交易所凭证到 NOFX（RSA 加密传输后入库）',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        enabled: { type: 'boolean' },
        api_key: { type: 'string' },
        secret_key: { type: 'string' },
        passphrase: { type: 'string' },
        testnet: { type: 'boolean' },
        hyperliquid_wallet_addr: { type: 'string' },
        aster_user: { type: 'string' },
        aster_signer: { type: 'string' },
        aster_private_key: { type: 'string' },
      },
      required: ['exchange_id'],
    },
  },
  {
    name: 'hermes_get_klines',
    description: '获取 K 线数据（binance/okx）',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        symbol: { type: 'string' },
        interval: { type: 'string' },
        limit: { type: 'number' },
      },
      required: ['symbol'],
    },
  },
  {
    name: 'hermes_get_balance',
    description: '获取交易所账户余额',
    inputSchema: {
      type: 'object',
      properties: { exchange_id: { type: 'string' } },
      required: ['exchange_id'],
    },
  },
  {
    name: 'hermes_get_positions',
    description: '获取交易所持仓',
    inputSchema: {
      type: 'object',
      properties: { exchange_id: { type: 'string' } },
      required: ['exchange_id'],
    },
  },
  {
    name: 'hermes_get_market_price',
    description: '获取指定币种市价',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        symbol: { type: 'string' },
      },
      required: ['exchange_id', 'symbol'],
    },
  },
  {
    name: 'hermes_trade',
    description:
      '执行交易：open_long/open_short/close_long/close_short/add_long/add_short/reduce_long/reduce_short；需 confirmed=true',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        action: { type: 'string' },
        symbol: { type: 'string' },
        quantity: { type: 'number' },
        leverage: { type: 'number' },
        confirmed: { type: 'boolean' },
      },
      required: ['exchange_id', 'action', 'symbol', 'confirmed'],
    },
  },
  {
    name: 'hermes_set_leverage',
    description: '设置杠杆倍数',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        symbol: { type: 'string' },
        leverage: { type: 'number' },
      },
      required: ['exchange_id', 'symbol', 'leverage'],
    },
  },
  {
    name: 'hermes_set_stop_loss',
    description: '设置止损单',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        symbol: { type: 'string' },
        position_side: { type: 'string', enum: ['LONG', 'SHORT'] },
        quantity: { type: 'number' },
        stop_price: { type: 'number' },
      },
      required: ['exchange_id', 'symbol', 'position_side', 'quantity', 'stop_price'],
    },
  },
  {
    name: 'hermes_set_take_profit',
    description: '设置止盈单',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        symbol: { type: 'string' },
        position_side: { type: 'string', enum: ['LONG', 'SHORT'] },
        quantity: { type: 'number' },
        stop_price: { type: 'number' },
      },
      required: ['exchange_id', 'symbol', 'position_side', 'quantity', 'stop_price'],
    },
  },
]

server.setRequestHandler(ListToolsRequestSchema, async () => ({ tools }))

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params
  const a = (args || {}) as Record<string, unknown>

  try {
    switch (name) {
      case 'hermes_login': {
        const res = await nofxClient.loginPublic<{
          token?: string
          user_id?: string
          email?: string
          requires_otp?: boolean
          message?: string
        }>('/api/login', {
          email: a.email,
          password: a.password,
        })
        if (res.requires_otp && res.user_id) {
          setPendingOtp(res.user_id, res.email || String(a.email))
          return jsonResult({
            status: 'requires_otp',
            user_id: res.user_id,
            message: res.message || '请输入 Google Authenticator 验证码，然后调用 hermes_verify_otp',
          })
        }
        if (res.token && res.user_id) {
          setSession({
            token: res.token,
            userId: res.user_id,
            email: res.email || String(a.email),
          })
          return jsonResult({ status: 'logged_in', user_id: res.user_id, email: res.email })
        }
        return jsonResult(res)
      }

      case 'hermes_verify_otp': {
        const pending = getPendingOtp()
        if (!pending) {
          return textResult('无待验证的登录会话，请先调用 hermes_login')
        }
        const res = await nofxClient.loginPublic<{
          token: string
          user_id: string
          email: string
        }>('/api/verify-otp', {
          user_id: pending.userId,
          otp_code: a.otp_code,
        })
        setSession({
          token: res.token,
          userId: res.user_id,
          email: res.email,
        })
        return jsonResult({ status: 'logged_in', user_id: res.user_id, email: res.email })
      }

      case 'hermes_auth_status': {
        const s = getSession()
        if (!s) {
          return jsonResult({ logged_in: false })
        }
        return jsonResult({
          logged_in: true,
          user_id: s.userId,
          email: s.email,
        })
      }

      case 'hermes_get_server_ip': {
        const ip = await nofxClient.get<{ public_ip: string; message: string }>(
          '/api/server-ip'
        )
        return jsonResult(ip)
      }

      case 'hermes_get_exchange_card': {
        const card = getExchangeCardSection(String(a.exchange_id))
        return textResult(card)
      }

      case 'hermes_list_exchanges': {
        const list = await nofxClient.get<
          Array<{ id: string; name: string; enabled: boolean; testnet: boolean }>
        >('/api/exchanges')
        const safe = list.map((ex) => ({
          id: ex.id,
          name: ex.name,
          enabled: ex.enabled,
          testnet: ex.testnet,
          configured: Boolean((ex as { apiKey?: string }).apiKey),
        }))
        return jsonResult(safe)
      }

      case 'hermes_test_exchange':
      case 'hermes_save_exchange_credentials': {
        const session = getSession()
        const exchangeId = String(a.exchange_id)
        const payload = {
          exchange_id: exchangeId,
          api_key: a.api_key || '',
          secret_key: a.secret_key || '',
          passphrase: a.passphrase || '',
          testnet: Boolean(a.testnet),
          hyperliquid_wallet_addr: a.hyperliquid_wallet_addr || '',
          aster_user: a.aster_user || '',
          aster_signer: a.aster_signer || '',
          aster_private_key: a.aster_private_key || '',
        }
        const encrypted = await encryptSensitiveData(
          apiBase(),
          JSON.stringify(payload),
          session?.userId
        )

        if (name === 'hermes_test_exchange') {
          const result = await nofxClient.postEncrypted(
            '/api/exchanges/test',
            encrypted
          )
          return jsonResult(result)
        }

        const savePayload = {
          exchanges: {
            [exchangeId]: {
              enabled: a.enabled !== false,
              api_key: payload.api_key,
              secret_key: payload.secret_key,
              passphrase: payload.passphrase,
              testnet: payload.testnet,
              hyperliquid_wallet_addr: payload.hyperliquid_wallet_addr,
              aster_user: payload.aster_user,
              aster_signer: payload.aster_signer,
              aster_private_key: payload.aster_private_key,
            },
          },
        }
        const saveEncrypted = await encryptSensitiveData(
          apiBase(),
          JSON.stringify(savePayload),
          session?.userId
        )
        const result = await nofxClient.putEncrypted(
          '/api/exchanges',
          saveEncrypted
        )
        return jsonResult(result)
      }

      case 'hermes_get_klines': {
        const result = await nofxClient.get('/api/hermes/klines', {
          exchange_id: String(a.exchange_id || 'binance'),
          symbol: String(a.symbol),
          interval: String(a.interval || '1h'),
          limit: String(a.limit || 100),
        })
        return jsonResult(result)
      }

      case 'hermes_get_balance': {
        const result = await nofxClient.get('/api/hermes/balance', {
          exchange_id: String(a.exchange_id),
        })
        return jsonResult(result)
      }

      case 'hermes_get_positions': {
        const result = await nofxClient.get('/api/hermes/positions', {
          exchange_id: String(a.exchange_id),
        })
        return jsonResult(result)
      }

      case 'hermes_get_market_price': {
        const result = await nofxClient.get('/api/hermes/market-price', {
          exchange_id: String(a.exchange_id),
          symbol: String(a.symbol),
        })
        return jsonResult(result)
      }

      case 'hermes_trade': {
        const result = await nofxClient.post('/api/hermes/trade', {
          exchange_id: a.exchange_id,
          action: a.action,
          symbol: a.symbol,
          quantity: a.quantity ?? 0,
          leverage: a.leverage ?? 0,
          confirmed: a.confirmed === true,
        })
        return jsonResult(result)
      }

      case 'hermes_set_leverage': {
        const result = await nofxClient.post('/api/hermes/leverage', a)
        return jsonResult(result)
      }

      case 'hermes_set_stop_loss': {
        const result = await nofxClient.post('/api/hermes/stop-loss', a)
        return jsonResult(result)
      }

      case 'hermes_set_take_profit': {
        const result = await nofxClient.post('/api/hermes/take-profit', a)
        return jsonResult(result)
      }

      default:
        return textResult(`未知工具: ${name}`)
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err)
    return textResult(`错误: ${msg}`)
  }
})

async function main() {
  const transport = new StdioServerTransport()
  await server.connect(transport)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
