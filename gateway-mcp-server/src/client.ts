function apiBase(): string {
  return (process.env.NOFX_API_URL || 'http://localhost:8080').replace(/\/$/, '')
}

let sessionKey: string | null = null

export function setApiKey(key: string) {
  sessionKey = key.trim()
}

export function getApiKey(): string {
  if (sessionKey) return sessionKey
  const envKey = process.env.NOFX_API_KEY?.trim()
  if (envKey) {
    sessionKey = envKey
    return envKey
  }
  throw new Error('未认证，请先调用 gateway_auth 提供 API Key，或设置 NOFX_API_KEY 环境变量')
}

const authHeaders = () => ({
  'Authorization': `ApiKey ${getApiKey()}`,
  'Content-Type': 'application/json',
})

export class GatewayClient {
  async get<T>(path: string, params?: Record<string, any>): Promise<T> {
    const url = new URL(`${apiBase()}${path}`)
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        if (v != null) url.searchParams.set(k, String(v))
      }
    }
    const res = await fetch(url.toString(), { headers: authHeaders() })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error((body as any)?.error || body?.detail || res.statusText)
    return body as T
  }

  async post<T>(path: string, data: unknown): Promise<T> {
    const res = await fetch(`${apiBase()}${path}`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify(data),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error((body as any)?.error || body?.detail || res.statusText)
    return body as T
  }
}

export const gatewayClient = new GatewayClient()
