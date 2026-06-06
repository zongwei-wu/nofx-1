/**
 * HTTP Client with unified error handling and 401 interception
 *
 * Features:
 * - Unified fetch wrapper
 * - Automatic 401 token expiration handling
 * - Auth state cleanup on unauthorized
 * - Automatic redirect to login page
 * - Notification shown on login page after redirect
 */

export class HttpClient {
  // Singleton flag to prevent duplicate 401 handling
  private static isHandling401 = false

  /**
   * Reset 401 handling flag (call after successful login)
   */
  public reset401Flag(): void {
    HttpClient.isHandling401 = false
  }

  private extractBearerToken(headers?: Record<string, string>): string | undefined {
    const auth = headers?.Authorization ?? headers?.authorization
    if (!auth?.startsWith('Bearer ')) return undefined
    return auth.slice(7).trim() || undefined
  }

  /**
   * Response interceptor - handles common HTTP errors
   *
   * @param response - Fetch Response object
   * @param requestToken - Bearer token used for this request (ignore stale 401 after re-login)
   * @returns Response if successful
   * @throws Error with user-friendly message
   */
  private async handleResponse(
    response: Response,
    requestToken?: string
  ): Promise<Response> {
    // Handle 401 Unauthorized - Token expired or invalid
    if (response.status === 401) {
      const currentToken = localStorage.getItem('auth_token')
      // 忽略过期请求：用户已用新 token 重新登录，旧请求返回 401 不应清除新会话
      if (
        requestToken &&
        currentToken &&
        requestToken !== currentToken
      ) {
        throw new Error('登录已过期，请重新登录')
      }

      // Prevent duplicate 401 handling when multiple API calls fail simultaneously
      if (HttpClient.isHandling401) {
        throw new Error('登录已过期，请重新登录')
      }

      // Set flag to prevent race conditions
      HttpClient.isHandling401 = true

      // Clean up local storage
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_user')
      localStorage.removeItem('auth_features')

      // Notify global listeners (AuthContext will react to this)
      window.dispatchEvent(new Event('unauthorized'))

      // Only redirect if not already on login page
      if (!window.location.pathname.includes('/login')) {
        // Save current location for post-login redirect
        const returnUrl = window.location.pathname + window.location.search
        if (returnUrl !== '/login' && returnUrl !== '/') {
          sessionStorage.setItem('returnUrl', returnUrl)
        }

        // Mark that user came from 401 (login page will show notification)
        sessionStorage.setItem('from401', 'true')

        // Redirect immediately to login page
        window.location.href = '/login'

        // Return pending promise to prevent error from being caught by SWR/React
        // The notification will be shown on the login page
        return new Promise(() => {}) as Promise<Response>
      }

      throw new Error('登录已过期，请重新登录')
    }

    // Handle other common errors
    if (response.status === 403) {
      throw new Error('没有权限访问此资源')
    }

    if (response.status === 404) {
      throw new Error('请求的资源不存在')
    }

    if (response.status >= 500) {
      throw new Error('服务器错误，请稍后重试')
    }

    return response
  }

  /**
   * GET request
   */
  async get(url: string, headers?: Record<string, string>): Promise<Response> {
    const requestToken = this.extractBearerToken(headers)
    const response = await fetch(url, {
      method: 'GET',
      headers,
    })
    return this.handleResponse(response, requestToken)
  }

  /**
   * POST request
   */
  async post(
    url: string,
    body?: any,
    headers?: Record<string, string>
  ): Promise<Response> {
    const requestToken = this.extractBearerToken(headers)
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...headers,
      },
      body: body ? JSON.stringify(body) : undefined,
    })
    return this.handleResponse(response, requestToken)
  }

  /**
   * PUT request
   */
  async put(
    url: string,
    body?: any,
    headers?: Record<string, string>
  ): Promise<Response> {
    const requestToken = this.extractBearerToken(headers)
    const response = await fetch(url, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        ...headers,
      },
      body: body ? JSON.stringify(body) : undefined,
    })
    return this.handleResponse(response, requestToken)
  }

  /**
   * DELETE request
   */
  async delete(
    url: string,
    headers?: Record<string, string>
  ): Promise<Response> {
    const requestToken = this.extractBearerToken(headers)
    const response = await fetch(url, {
      method: 'DELETE',
      headers,
    })
    return this.handleResponse(response, requestToken)
  }

  /**
   * Generic request method for custom configurations
   */
  async request(url: string, options: RequestInit = {}): Promise<Response> {
    const hdrs = options.headers as Record<string, string> | undefined
    const requestToken = this.extractBearerToken(hdrs)
    const response = await fetch(url, options)
    return this.handleResponse(response, requestToken)
  }
}

// Export singleton instance
export const httpClient = new HttpClient()

// Export helper function to reset 401 flag
export const reset401Flag = () => httpClient.reset401Flag()
