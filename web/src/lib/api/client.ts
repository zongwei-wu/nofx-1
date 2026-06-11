export const API_BASE = '/api'

export function getAuthHeaders(): Record<string, string> {
  const token = localStorage.getItem('auth_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  return headers
}

export async function parseJson<T>(res: Response): Promise<T> {
  return res.json() as Promise<T>
}

export async function throwIfNotOk(res: Response, message: string): Promise<void> {
  if (!res.ok) throw new Error(message)
}
