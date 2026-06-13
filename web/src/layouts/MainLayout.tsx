import { ReactNode, useEffect, useRef } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import HeaderBar from '../components/HeaderBar'
import { Container } from '../components/Container'
import { ExchangeOnboardingGate } from '../components/onboarding/ExchangeOnboardingGate'
import { useLanguage } from '../contexts/LanguageContext'
import { useAuth } from '../contexts/AuthContext'
import { getCurrentPageFromPath } from '../config/navItems'
import { t } from '../i18n/translations'

interface MainLayoutProps {
  children?: ReactNode
}

export default function MainLayout({ children }: MainLayoutProps) {
  const { language, setLanguage } = useLanguage()
  const { user, logout, refreshPermissions } = useAuth()
  const location = useLocation()

  // 每个用户会话仅刷新一次权限，避免 /api/me 请求循环
  const permissionsSyncedRef = useRef<string | null>(null)
  useEffect(() => {
    if (!user?.id) return
    if (permissionsSyncedRef.current === user.id) return
    permissionsSyncedRef.current = user.id
    void refreshPermissions()
  }, [user?.id, refreshPermissions])

  const currentPage = getCurrentPageFromPath(location.pathname)

  return (
    <div
      className="min-h-screen"
      style={{ background: '#0B0E11', color: '#EAECEF' }}
    >
      <HeaderBar
        isLoggedIn={!!user}
        currentPage={currentPage}
        language={language}
        onLanguageChange={setLanguage}
        user={user}
        onLogout={logout}
        onPageChange={() => {
          // React Router handles navigation now
        }}
      />

      {/* Main Content */}
      <ExchangeOnboardingGate />

      <Container as="main" className="py-6 pt-24">
        {children || <Outlet />}
      </Container>

      {/* Footer */}
      <footer
        className="mt-16"
        style={{ borderTop: '1px solid #2B3139', background: '#181A20' }}
      >
        <Container
          className="py-6 text-center text-sm"
          style={{ color: '#5E6673' }}
        >
          <p>{t('footerTitle', language)}</p>
          <p className="mt-1">{t('footerWarning', language)}</p>
        </Container>
      </footer>
    </div>
  )
}
