import { useEffect, useState } from 'react'
import { RouterProvider } from 'react-router-dom'
import { LanguageProvider } from './contexts/LanguageContext'
import { AuthProvider } from './contexts/AuthContext'
import { ConfirmDialogProvider } from './components/ConfirmDialog'
import { router } from './routes'
import { useSystemConfig } from './hooks/useSystemConfig'
import { useAuth } from './contexts/AuthContext'
import { useLanguage } from './contexts/LanguageContext'
import { t } from './i18n/translations'

function LoadingScreen() {
  const { language } = useLanguage()

  return (
    <div
      className="min-h-screen flex items-center justify-center"
      style={{ background: '#0B0E11' }}
    >
      <div className="text-center">
        <img
          src="/icons/nofx.svg"
          alt="NoFx Logo"
          className="w-16 h-16 mx-auto mb-4 animate-pulse"
        />
        <p style={{ color: '#EAECEF' }}>{t('loading', language)}</p>
      </div>
    </div>
  )
}

function AppContent() {
  const { isLoading } = useAuth()
  const { loading: configLoading } = useSystemConfig()
  const [bootTimeout, setBootTimeout] = useState(false)

  useEffect(() => {
    const t = window.setTimeout(() => setBootTimeout(true), 10000)
    return () => window.clearTimeout(t)
  }, [])

  if ((isLoading || configLoading) && !bootTimeout) {
    return <LoadingScreen />
  }

  return <RouterProvider router={router} />
}

export default function App() {
  return (
    <LanguageProvider>
      <AuthProvider>
        <ConfirmDialogProvider>
          <AppContent />
        </ConfirmDialogProvider>
      </AuthProvider>
    </LanguageProvider>
  )
}
