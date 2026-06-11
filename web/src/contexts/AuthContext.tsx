import React, {
  createContext,
  useCallback,
  useContext,
  useState,
  useEffect,
} from 'react'
import { authApi, type AuthSessionPayload } from '../lib/api/auth'
import { getSystemConfig } from '../lib/config'
import { EXCHANGE_ONBOARDING_PROMPT_KEY } from '../lib/exchangeOnboarding'
import { reset401Flag } from '../lib/httpClient'

interface User {
  id: string
  email: string
  plan?: string
}

interface AuthContextType {
  user: User | null
  token: string | null
  features: string[]
  hasFeature: (feature: string) => boolean
  refreshPermissions: () => Promise<void>
  login: (
    email: string,
    password: string
  ) => Promise<{
    success: boolean
    message?: string
    userID?: string
    requiresOTP?: boolean
  }>
  loginAdmin: (password: string) => Promise<{
    success: boolean
    message?: string
  }>
  register: (
    email: string,
    password: string,
    betaCode?: string
  ) => Promise<{
    success: boolean
    message?: string
    userID?: string
    otpSecret?: string
    qrCodeURL?: string
  }>
  verifyOTP: (
    userID: string,
    otpCode: string
  ) => Promise<{ success: boolean; message?: string }>
  completeRegistration: (
    userID: string,
    otpCode: string
  ) => Promise<{ success: boolean; message?: string }>
  resetPassword: (
    email: string,
    newPassword: string,
    otpCode: string
  ) => Promise<{ success: boolean; message?: string }>
  logout: () => void
  isLoading: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

function clearStoredAuth() {
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_user')
  localStorage.removeItem('auth_features')
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [features, setFeatures] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(true)

  const navigateAfterLogin = (defaultPath: string) => {
    sessionStorage.removeItem('from401')
    reset401Flag()
    const returnUrl = sessionStorage.getItem('returnUrl')
    const target = returnUrl || defaultPath
    if (returnUrl) {
      sessionStorage.removeItem('returnUrl')
    }
    void import('../routes').then(({ router }) => {
      router.navigate(target, { replace: true })
    })
  }

  const applyAuthSession = async (data: AuthSessionPayload) => {
    const userInfo: User = {
      id: data.user_id,
      email: data.email,
      plan: data.plan,
    }
    const featureList = data.features ?? []
    setToken(data.token)
    setUser(userInfo)
    setFeatures(featureList)
    localStorage.setItem('auth_token', data.token)
    localStorage.setItem('auth_user', JSON.stringify(userInfo))
    localStorage.setItem('auth_features', JSON.stringify(featureList))
    sessionStorage.setItem(EXCHANGE_ONBOARDING_PROMPT_KEY, '1')
    await fetchMe(data.token)
  }

  const applyMePayload = useCallback(
    (data: {
      user_id?: string
      email?: string
      plan?: string
      features?: string[]
    }) => {
      if (Array.isArray(data.features)) {
        setFeatures((prev) => {
          if (
            prev.length === data.features!.length &&
            prev.every((f, i) => f === data.features![i])
          ) {
            return prev
          }
          localStorage.setItem(
            'auth_features',
            JSON.stringify(data.features)
          )
          return data.features!
        })
      }
      setUser((prev) => {
        const next: User = {
          id: data.user_id ?? prev?.id ?? '',
          email: data.email ?? prev?.email ?? '',
          plan: data.plan ?? prev?.plan,
        }
        if (
          prev &&
          prev.id === next.id &&
          prev.email === next.email &&
          prev.plan === next.plan
        ) {
          return prev
        }
        localStorage.setItem('auth_user', JSON.stringify(next))
        return next
      })
    },
    []
  )

  const fetchMe = useCallback(
    async (authToken: string): Promise<boolean> => {
      const result = await authApi.fetchMe(authToken)
      if (result.status === 401) return false
      if (result.data) applyMePayload(result.data)
      return true
    },
    [applyMePayload]
  )

  const restoreSession = async () => {
    const savedToken = localStorage.getItem('auth_token')
    const savedUser = localStorage.getItem('auth_user')
    if (!savedToken || !savedUser) return

    const result = await authApi.fetchMe(savedToken)
    if (result.status === 401) {
      clearStoredAuth()
      return
    }

    try {
      setToken(savedToken)
      const parsedUser = JSON.parse(savedUser) as User
      setUser(parsedUser)
      const savedFeatures = localStorage.getItem('auth_features')
      if (savedFeatures) {
        setFeatures(JSON.parse(savedFeatures) as string[])
      }
      if (result.data) {
        applyMePayload(result.data)
      }
    } catch {
      clearStoredAuth()
      setToken(null)
      setUser(null)
      setFeatures([])
    }
  }

  const refreshPermissions = useCallback(async () => {
    const savedToken = localStorage.getItem('auth_token')
    if (!savedToken) return
    await fetchMe(savedToken)
  }, [fetchMe])

  const hasFeature = (feature: string) => features.includes(feature)

  useEffect(() => {
    reset401Flag()

    getSystemConfig()
      .then(async () => {
        await restoreSession()
        setIsLoading(false)
      })
      .catch(async (err) => {
        console.error('Failed to fetch system config:', err)
        await restoreSession()
        setIsLoading(false)
      })
  }, [])

  useEffect(() => {
    const handleUnauthorized = () => {
      setUser(null)
      setToken(null)
      setFeatures([])
    }

    window.addEventListener('unauthorized', handleUnauthorized)
    return () => window.removeEventListener('unauthorized', handleUnauthorized)
  }, [])

  const login = async (email: string, password: string) => {
    try {
      const { ok, data } = await authApi.login(email, password)
      if (ok) {
        if ((data as { requires_otp?: boolean }).requires_otp) {
          return {
            success: true,
            userID: data.user_id,
            requiresOTP: true,
            message: data.message,
          }
        }
        await applyAuthSession(data as AuthSessionPayload)
        sessionStorage.removeItem('from401')
        reset401Flag()
        return { success: true, message: data.message }
      }
      return { success: false, message: data.error }
    } catch {
      return { success: false, message: '登录失败，请重试' }
    }
  }

  const loginAdmin = async (password: string) => {
    try {
      const { ok, data } = await authApi.loginAdmin(password)
      if (ok) {
        await applyAuthSession({
          token: data.token,
          user_id: data.user_id || 'admin',
          email: data.email || 'admin@localhost',
          plan: data.plan,
          features: data.features,
        })
        navigateAfterLogin('/dashboard')
        return { success: true }
      }
      return { success: false, message: data.error || '登录失败' }
    } catch {
      return { success: false, message: '登录失败，请重试' }
    }
  }

  const register = async (
    email: string,
    password: string,
    betaCode?: string
  ) => {
    try {
      const { ok, data } = await authApi.register(email, password, betaCode)
      if (ok) {
        return {
          success: true,
          userID: data.user_id,
          otpSecret: data.otp_secret,
          qrCodeURL: data.qr_code_url,
          message: data.message,
        }
      }
      return { success: false, message: data.error }
    } catch {
      return { success: false, message: '注册失败，请重试' }
    }
  }

  const verifyOTP = async (userID: string, otpCode: string) => {
    try {
      const { ok, data } = await authApi.verifyOTP(userID, otpCode)
      if (ok) {
        await applyAuthSession(data as AuthSessionPayload)
        navigateAfterLogin('/competition')
        return { success: true, message: data.message }
      }
      return { success: false, message: data.error }
    } catch {
      return { success: false, message: 'OTP验证失败，请重试' }
    }
  }

  const completeRegistration = async (userID: string, otpCode: string) => {
    try {
      const { ok, data } = await authApi.completeRegistration(userID, otpCode)
      if (ok) {
        await applyAuthSession(data as AuthSessionPayload)
        navigateAfterLogin('/competition')
        return { success: true, message: data.message }
      }
      return { success: false, message: data.error }
    } catch {
      return { success: false, message: '注册完成失败，请重试' }
    }
  }

  const resetPassword = async (
    email: string,
    newPassword: string,
    otpCode: string
  ) => {
    try {
      const { ok, data } = await authApi.resetPassword(
        email,
        newPassword,
        otpCode
      )
      if (ok) {
        return { success: true, message: data.message }
      }
      return { success: false, message: data.error }
    } catch {
      return { success: false, message: '密码重置失败，请重试' }
    }
  }

  const logout = () => {
    const savedToken = localStorage.getItem('auth_token')
    if (savedToken) {
      authApi.logout(savedToken)
    }
    setUser(null)
    setToken(null)
    setFeatures([])
    clearStoredAuth()
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        features,
        hasFeature,
        refreshPermissions,
        login,
        loginAdmin,
        register,
        verifyOTP,
        completeRegistration,
        resetPassword,
        logout,
        isLoading,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
