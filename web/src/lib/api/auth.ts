import { httpClient } from '../httpClient'

export type AuthSessionPayload = {
  token: string
  user_id: string
  email: string
  plan?: string
  features?: string[]
  requires_otp?: boolean
  message?: string
}

export type MePayload = {
  user_id?: string
  email?: string
  plan?: string
  features?: string[]
}

async function postJson<T>(path: string, body: unknown): Promise<{ ok: boolean; status: number; data: T }> {
  const response = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const data = (await response.json()) as T
  return { ok: response.ok, status: response.status, data }
}

export const authApi = {
  async fetchMe(token: string): Promise<{ ok: boolean; status: number; data?: MePayload }> {
    const response = await fetch('/api/me', {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!response.ok) {
      return { ok: false, status: response.status }
    }
    const data = (await response.json()) as MePayload
    return { ok: true, status: response.status, data }
  },

  async login(email: string, password: string) {
    return postJson<AuthSessionPayload & { error?: string; user_id?: string }>(
      '/api/login',
      { email, password }
    )
  },

  async loginAdmin(password: string) {
    return postJson<AuthSessionPayload & { error?: string }>('/api/admin-login', {
      password,
    })
  },

  async register(email: string, password: string, betaCode?: string) {
    const body: { email: string; password: string; beta_code?: string } = {
      email,
      password,
    }
    if (betaCode) body.beta_code = betaCode
    return postJson<{
      user_id?: string
      otp_secret?: string
      qr_code_url?: string
      message?: string
      error?: string
    }>('/api/register', body)
  },

  async verifyOTP(userID: string, otpCode: string) {
    return postJson<AuthSessionPayload & { error?: string; message?: string }>(
      '/api/verify-otp',
      { user_id: userID, otp_code: otpCode }
    )
  },

  async completeRegistration(userID: string, otpCode: string) {
    return postJson<AuthSessionPayload & { error?: string; message?: string }>(
      '/api/complete-registration',
      { user_id: userID, otp_code: otpCode }
    )
  },

  async resetPassword(email: string, newPassword: string, otpCode: string) {
    return postJson<{ message?: string; error?: string }>('/api/reset-password', {
      email,
      new_password: newPassword,
      otp_code: otpCode,
    })
  },

  logout(token: string): void {
    httpClient
      .post('/api/logout', undefined, { Authorization: `Bearer ${token}` })
      .catch(() => {
        /* ignore network errors on logout */
      })
  },
}
