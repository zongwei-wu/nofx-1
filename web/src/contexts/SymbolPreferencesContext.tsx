import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  type ReactNode,
} from 'react'
import useSWR from 'swr'
import { httpClient } from '../lib/httpClient'
import { sortSymbolsByPreference } from '../lib/sortSymbols'
import { useAuth } from './AuthContext'

export interface SymbolPreferencesState {
  symbols: string[]
  useCustomOrder: boolean
  catalog: string[]
}

type SymbolPreferencesContextValue = {
  preferences: SymbolPreferencesState | null
  notionalBySymbol: Record<string, number>
  loading: boolean
  sortSymbols: (symbols: string[]) => string[]
  saveOrder: (symbols: string[], useCustomOrder?: boolean) => Promise<void>
  resetToDefault: () => Promise<void>
  refresh: () => Promise<void>
}

const SymbolPreferencesContext = createContext<SymbolPreferencesContextValue | null>(
  null
)

async function fetchWithAuth<T>(url: string, token: string | null): Promise<T> {
  const res = await httpClient.get(url, {
    Authorization: token ? `Bearer ${token}` : '',
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error((err as { error?: string }).error || `请求失败: ${url}`)
  }
  return res.json()
}

export function SymbolPreferencesProvider({ children }: { children: ReactNode }) {
  const { token, user, isLoading: authLoading } = useAuth()
  const authReady = !authLoading && !!user && !!token

  const { data: prefs, mutate: mutatePrefs, isLoading: prefsLoading } = useSWR(
    authReady ? ['symbol-preferences', token] : null,
    () =>
      fetchWithAuth<{
        symbols: string[]
        use_custom_order: boolean
        catalog: string[]
      }>('/api/symbol-preferences', token),
    { revalidateOnFocus: false }
  )

  const { data: valuesData, mutate: mutateValues, isLoading: valuesLoading } = useSWR(
    authReady ? ['symbol-values', token] : null,
    () => fetchWithAuth<{ values: Record<string, number> }>('/api/symbol-values', token),
    { refreshInterval: 30000, revalidateOnFocus: false }
  )

  const preferences = useMemo<SymbolPreferencesState | null>(() => {
    if (!user || !prefs) return null
    return {
      symbols: prefs.symbols ?? [],
      useCustomOrder: !!prefs.use_custom_order,
      catalog: prefs.catalog ?? [],
    }
  }, [user, prefs])

  const notionalBySymbol = valuesData?.values ?? {}

  const sortSymbols = useCallback(
    (symbols: string[]) => {
      if (!preferences) {
        return sortSymbolsByPreference(symbols, { notionalBySymbol })
      }
      return sortSymbolsByPreference(symbols, {
        customOrder: preferences.symbols,
        useCustomOrder: preferences.useCustomOrder,
        notionalBySymbol,
      })
    },
    [preferences, notionalBySymbol]
  )

  const saveOrder = useCallback(
    async (symbols: string[], useCustomOrder = true) => {
      const res = await httpClient.put(
        '/api/symbol-preferences',
        { symbols, use_custom_order: useCustomOrder },
        { Authorization: token ? `Bearer ${token}` : '' }
      )
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error((err as { error?: string }).error || '保存失败')
      }
      await mutatePrefs()
    },
    [token, mutatePrefs]
  )

  const resetToDefault = useCallback(async () => {
    const res = await httpClient.post(
      '/api/symbol-preferences/reset',
      undefined,
      { Authorization: token ? `Bearer ${token}` : '' }
    )
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error((err as { error?: string }).error || '恢复默认失败')
    }
    await mutatePrefs()
  }, [token, mutatePrefs])

  const refresh = useCallback(async () => {
    await Promise.all([mutatePrefs(), mutateValues()])
  }, [mutatePrefs, mutateValues])

  const value = useMemo(
    () => ({
      preferences,
      notionalBySymbol,
      loading: !!user && (prefsLoading || valuesLoading),
      sortSymbols,
      saveOrder,
      resetToDefault,
      refresh,
    }),
    [
      preferences,
      notionalBySymbol,
      user,
      prefsLoading,
      valuesLoading,
      sortSymbols,
      saveOrder,
      resetToDefault,
      refresh,
    ]
  )

  return (
    <SymbolPreferencesContext.Provider value={value}>
      {children}
    </SymbolPreferencesContext.Provider>
  )
}

export function useSymbolPreferences(): SymbolPreferencesContextValue {
  const ctx = useContext(SymbolPreferencesContext)
  if (!ctx) {
    return {
      preferences: null,
      notionalBySymbol: {},
      loading: false,
      sortSymbols: (symbols) => sortSymbolsByPreference(symbols),
      saveOrder: async () => {},
      resetToDefault: async () => {},
      refresh: async () => {},
    }
  }
  return ctx
}
