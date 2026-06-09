import { getSession } from './session.js'
import type { EncryptedPayload } from './crypto.js'

function apiBase(): string {
  return (process.env.NOFX_API_URL || 'http://localhost:8080').replace(/\/$/, '')
}

export class NofxClient {
  private authHeader(): { Authorization: string } {
    const s = getSession()
    if (!s?.token) {
      throw new Error('未认证，请设置 NOFX_API_KEY 或 NOFX_JWT 环境变量')
    }
    if (s.authMethod === 'apikey') {
      return { Authorization: `ApiKey ${s.token}` }
    }
    return { Authorization: `Bearer ${s.token}` }
  }

  async get<T>(path: string, params?: Record<string, string>): Promise<T> {
    const url = new URL(`${apiBase()}${path}`)
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        if (v) url.searchParams.set(k, v)
      }
    }
    const res = await fetch(url.toString(), {
      headers: this.authHeader(),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error((body as { error?: string }).error || res.statusText)
    }
    return body as T
  }

  async post<T>(path: string, data?: unknown): Promise<T> {
    const res = await fetch(`${apiBase()}${path}`, {
      method: 'POST',
      headers: {
        ...this.authHeader(),
        'Content-Type': 'application/json',
      },
      body: data ? JSON.stringify(data) : undefined,
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error((body as { error?: string }).error || res.statusText)
    }
    return body as T
  }

  async put<T>(path: string, data?: unknown): Promise<T> {
    const res = await fetch(`${apiBase()}${path}`, {
      method: 'PUT',
      headers: {
        ...this.authHeader(),
        'Content-Type': 'application/json',
      },
      body: data ? JSON.stringify(data) : undefined,
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error((body as { error?: string }).error || res.statusText)
    }
    return body as T
  }

  async putEncrypted(path: string, payload: EncryptedPayload): Promise<unknown> {
    const res = await fetch(`${apiBase()}${path}`, {
      method: 'PUT',
      headers: {
        ...this.authHeader(),
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error((body as { error?: string }).error || res.statusText)
    }
    return body
  }

  async postEncrypted(path: string, payload: EncryptedPayload): Promise<unknown> {
    const res = await fetch(`${apiBase()}${path}`, {
      method: 'POST',
      headers: {
        ...this.authHeader(),
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error((body as { error?: string }).error || res.statusText)
    }
    return body
  }

  async loginPublic<T>(path: string, data: unknown): Promise<T> {
    const res = await fetch(`${apiBase()}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error((body as { error?: string }).error || res.statusText)
    }
    return body as T
  }

  async getPublic<T>(path: string): Promise<T> {
    const res = await fetch(`${apiBase()}${path}`)
    const body = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new Error((body as { error?: string }).error || res.statusText)
    }
    return body as T
  }
}

export const nofxClient = new NofxClient()
