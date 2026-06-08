export interface AuthSession {
  token: string
  userId: string
  email: string
}

let session: AuthSession | null = null
let pendingOtp: { userId: string; email: string } | null = null

export function getSession(): AuthSession | null {
  if (session) return session
  const envToken = process.env.NOFX_JWT?.trim()
  const envUserId = process.env.NOFX_USER_ID?.trim()
  if (envToken && envUserId) {
    return {
      token: envToken,
      userId: envUserId,
      email: process.env.NOFX_EMAIL?.trim() || '',
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
