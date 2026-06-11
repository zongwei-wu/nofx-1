function apiBase(): string {
  return (process.env.NOFX_API_URL || 'http://localhost:8080').replace(/\/$/, '')
}

function apiKey(): string {
  const key = process.env.NOFX_API_KEY?.trim()
  if (!key) throw new Error('未设置 NOFX_API_KEY 环境变量')
  return key
}

const authHeaders = () => ({
  'Authorization': `ApiKey ${apiKey()}`,
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
