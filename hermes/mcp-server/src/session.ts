export interface AuthSession {
  token: string
  userId: string
  email: string
  authMethod: 'jwt' | 'apikey'
}

let session: AuthSession | null = null
let pendingOtp: { userId: string; email: string } | null = null

// ★ 动态 API Key：会话级别，通过 hermes_use_api_key 设置，优先于环境变量
let dynamicApiKey: string | null = null

export function setApiKey(key: string): void {
  dynamicApiKey = key
  // 同步更新 session，让后续请求自动带上
  session = {
    token: key,
    userId: '',
    email: '',
    authMethod: 'apikey',
  }
}

export function getApiKey(): string | null {
  if (dynamicApiKey) return dynamicApiKey
  return process.env.NOFX_API_KEY?.trim() || null
}

export function getSession(): AuthSession | null {
  if (session) return session

  // ★ 动态 API Key 优先
  if (dynamicApiKey) {
    return {
      token: dynamicApiKey,
      userId: '',
      email: '',
      authMethod: 'apikey',
    }
  }

  // ★ 方式 1: API Key 认证（环境变量）
  const apiKey = process.env.NOFX_API_KEY?.trim()
  if (apiKey) {
    return {
      token: apiKey,
      userId: '',
      email: '',
      authMethod: 'apikey',
    }
  }

  // ★ 方式 2: JWT 认证（兼容旧方式）
  const envToken = process.env.NOFX_JWT?.trim()
  const envUserId = process.env.NOFX_USER_ID?.trim()
  if (envToken && envUserId) {
    return {
      token: envToken,
      userId: envUserId,
      email: process.env.NOFX_EMAIL?.trim() || '',
      authMethod: 'jwt',
    }
  }
  return null
}

export function setSession(s: AuthSession): void {
  session = s
  pendingOtp = null
}

export function clearSession(): void {
  session = null
  pendingOtp = null
}

export function setPendingOtp(userId: string, email: string): void {
  pendingOtp = { userId, email }
}

export function getPendingOtp(): { userId: string; email: string } | null {
  return pendingOtp
}
