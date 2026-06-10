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
  setApiKey,
  getApiKey,
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

// hermesPath returns the API path prefix based on auth method.
// API key auth uses /api/mcp/hermes (apiKeyAuthMiddleware),
// JWT auth uses /api/hermes (authMiddleware + requireFeature).
function hermesPath(suffix: string): string {
  const s = getSession()
  if (s?.authMethod === 'apikey') {
    return `/api/mcp/hermes${suffix}`
  }
  return `/api/hermes${suffix}`
}

// mcpPath returns API path for non-hermes MCP endpoints (exchanges, models)
function mcpPath(suffix: string): string {
  const s = getSession()
  if (s?.authMethod === 'apikey') {
    return `/api/mcp${suffix}`
  }
  return `/api${suffix}`
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
    name: 'hermes_use_api_key',
    description:
      '设置当前会话使用的 NOFX API Key（从 Web UI 获取：https://www.ttai.me/access-keys）。设置后所有交易/行情请求自动使用该 Key，无需每次登录。多个 Agent 共享同一个 MCP Server 时，每个 Agent 应设置自己的 Key。',
    inputSchema: {
      type: 'object',
      properties: {
        api_key: { type: 'string', description: 'NOFX API Key，格式 nfx_sk_xxx' },
      },
      required: ['api_key'],
    },
  },
  {
    name: 'hermes_get_api_key',
    description: '查看当前会话使用的 API Key 前缀（仅显示前8位，不暴露完整 Key）',
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
  {
    name: 'hermes_get_settings',
    description: '读取 Hermes Runner 设置、就绪状态与运行摘要',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'hermes_update_settings',
    description: '更新 Hermes 设置（交易所、AI 模型、扫描间隔、杠杆、交易币种等）',
    inputSchema: {
      type: 'object',
      properties: {
        exchange_id: { type: 'string' },
        ai_model_id: { type: 'string' },
        scan_interval_minutes: { type: 'number' },
        btc_eth_leverage: { type: 'number' },
        altcoin_leverage: { type: 'number' },
        trading_coins: { type: 'array', items: { type: 'string' } },
        initial_balance: { type: 'number' },
        system_prompt_template: { type: 'string' },
      },
    },
  },
  {
    name: 'hermes_enable_autonomous',
    description: '开启半自动模式（用户明确授权后 Runner 可免逐笔 confirmed 下单）',
    inputSchema: {
      type: 'object',
      properties: {
        enabled: { type: 'boolean' },
      },
      required: ['enabled'],
    },
  },
  {
    name: 'hermes_start_runner',
    description: '启动 Hermes 后台自主交易 Runner（需已配置交易所、AI 模型且 autonomous_enabled=true）',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'hermes_stop_runner',
    description: '停止 Hermes 后台 Runner',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'hermes_get_runner_status',
    description: '获取 Hermes Runner 运行状态',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'hermes_get_decisions',
    description: '获取 Hermes 最近决策日志',
    inputSchema: {
      type: 'object',
      properties: {
        limit: { type: 'number' },
      },
    },
  },
  {
    name: 'hermes_list_models',
    description: '列出用户已配置的 AI 模型（供选择 Hermes ai_model_id）',
    inputSchema: { type: 'object', properties: {} },
  },
  {
    name: 'hermes_require_exchange_setup',
    description: '门禁检查：交易所、模型、自主授权是否就绪；未配置则返回缺项与引导',
    inputSchema: { type: 'object', properties: {} },
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
            authMethod: 'jwt',
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
          authMethod: 'jwt',
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
          auth_method: s.authMethod,
        })
      }

      case 'hermes_use_api_key': {
        const key = String(a.api_key).trim()
        if (!key) {
          return textResult('请提供 NOFX API Key（在 https://www.ttai.me/access-keys 获取）')
        }
        setApiKey(key)
        const preview = key.length > 8 ? key.substring(0, 8) + '...' : key + '...'
        return jsonResult({
          status: 'ok',
          message: `API Key 已设置 (${preview})，后续所有请求将使用此 Key`,
        })
      }

      case 'hermes_get_api_key': {
        const key = getApiKey()
        if (!key) {
          return jsonResult({ api_key_set: false, message: '未设置 API Key' })
        }
        const preview = key.length > 8 ? key.substring(0, 8) + '...' : key + '...'
        return jsonResult({
          api_key_set: true,
          preview,
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
        >(mcpPath('/exchanges'))
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
            mcpPath('/exchanges/test'),
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
          mcpPath('/exchanges'),
          saveEncrypted
        )
        return jsonResult(result)
      }

      case 'hermes_get_klines': {
        const result = await nofxClient.get(hermesPath('/klines'), {
          exchange_id: String(a.exchange_id || 'binance'),
          symbol: String(a.symbol),
          interval: String(a.interval || '1h'),
          limit: String(a.limit || 100),
        })
        return jsonResult(result)
      }

      case 'hermes_get_balance': {
        const result = await nofxClient.get(hermesPath('/balance'), {
          exchange_id: String(a.exchange_id),
        })
        return jsonResult(result)
      }

      case 'hermes_get_positions': {
        const result = await nofxClient.get(hermesPath('/positions'), {
          exchange_id: String(a.exchange_id),
        })
        return jsonResult(result)
      }

      case 'hermes_get_market_price': {
        const result = await nofxClient.get(hermesPath('/market-price'), {
          exchange_id: String(a.exchange_id),
          symbol: String(a.symbol),
        })
        return jsonResult(result)
      }

      case 'hermes_trade': {
        const result = await nofxClient.post(hermesPath('/trade'), {
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
        const result = await nofxClient.post(hermesPath('/leverage'), a)
        return jsonResult(result)
      }

      case 'hermes_set_stop_loss': {
        const result = await nofxClient.post(hermesPath('/stop-loss'), a)
        return jsonResult(result)
      }

      case 'hermes_set_take_profit': {
        const result = await nofxClient.post(hermesPath('/take-profit'), a)
        return jsonResult(result)
      }

      case 'hermes_get_settings': {
        const result = await nofxClient.get(hermesPath('/settings'))
        return jsonResult(result)
      }

      case 'hermes_update_settings': {
        const result = await nofxClient.put(hermesPath('/settings'), a)
        return jsonResult(result)
      }

      case 'hermes_enable_autonomous': {
        const result = await nofxClient.put(hermesPath('/settings'), {
          autonomous_enabled: a.enabled === true,
        })
        return jsonResult(result)
      }

      case 'hermes_start_runner': {
        const result = await nofxClient.post(hermesPath('/start'), {})
        return jsonResult(result)
      }

      case 'hermes_stop_runner': {
        const result = await nofxClient.post(hermesPath('/stop'), {})
        return jsonResult(result)
      }

      case 'hermes_get_runner_status': {
        const result = await nofxClient.get(hermesPath('/status'))
        return jsonResult(result)
      }

      case 'hermes_get_decisions': {
        const result = await nofxClient.get(hermesPath('/decisions'))
        return jsonResult(result)
      }

      case 'hermes_list_models': {
        const result = await nofxClient.get(mcpPath('/models'))
        return jsonResult(result)
      }

      case 'hermes_require_exchange_setup': {
        const result = await nofxClient.get(hermesPath('/require-setup'))
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
