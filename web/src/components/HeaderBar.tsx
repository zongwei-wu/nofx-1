import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { Menu, X, ChevronDown } from 'lucide-react'
import { t, type Language } from '../i18n/translations'
import { Container } from './Container'
import { useSystemConfig } from '../hooks/useSystemConfig'
import { useAuth } from '../contexts/AuthContext'
import { HomeNavLinks } from './layout/HomeNavLinks'
import { LanguageToggle } from './layout/LanguageToggle'
import { AppNavMenu } from './layout/AppNavMenu'

interface HeaderBarProps {
  onLoginClick?: () => void
  isLoggedIn?: boolean
  isHomePage?: boolean
  currentPage?: string
  language?: Language
  onLanguageChange?: (lang: Language) => void
  user?: { email: string } | null
  onLogout?: () => void
  onPageChange?: (page: string) => void
}

export default function HeaderBar({
  isLoggedIn = false,
  isHomePage = false,
  currentPage,
  language = 'zh' as Language,
  onLanguageChange,
  user,
  onLogout,
}: HeaderBarProps) {
  const { hasFeature } = useAuth()

  const canShow = (feature: string | null) => {
    if (!feature) return true
    return hasFeature(feature)
  }

  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [userDropdownOpen, setUserDropdownOpen] = useState(false)
  const userDropdownRef = useRef<HTMLDivElement>(null)
  const { config: systemConfig } = useSystemConfig()
  const registrationEnabled = systemConfig?.registration_enabled !== false

  const closeMobileMenu = () => setMobileMenuOpen(false)

  const handleLanguageChange = (lang: Language) => {
    onLanguageChange?.(lang)
    closeMobileMenu()
  }

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        userDropdownRef.current &&
        !userDropdownRef.current.contains(event.target as Node)
      ) {
        setUserDropdownOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [])

  return (
    <nav className="fixed top-0 w-full z-50 header-bar">
      <Container className="flex items-center justify-between h-16">
        <Link
          to="/"
          className="flex items-center gap-3 hover:opacity-80 transition-opacity cursor-pointer"
        >
          <img src="/icons/nofx.svg" alt="NOFX Logo" className="w-8 h-8" />
          <span
            className="text-xl font-bold"
            style={{ color: 'var(--brand-yellow)' }}
          >
            NOFX
          </span>
          <span
            className="text-sm hidden sm:block"
            style={{ color: 'var(--text-secondary)' }}
          >
            Agentic Trading OS
          </span>
        </Link>

        <div className="hidden md:flex items-center justify-between flex-1 ml-8">
          <div className="flex items-center gap-4">
            <AppNavMenu
              language={language}
              currentPage={currentPage}
              isLoggedIn={isLoggedIn}
              canShow={canShow}
              variant="desktop"
            />
          </div>

          <div className="flex items-center gap-6">
            {isHomePage && (
              <div className="flex items-center gap-6">
                <HomeNavLinks
                  language={language}
                  linkClassName="text-sm transition-colors relative group"
                />
              </div>
            )}

            {isLoggedIn && user ? (
              <div className="flex items-center gap-3">
                <div className="relative" ref={userDropdownRef}>
                  <button
                    onClick={() => setUserDropdownOpen(!userDropdownOpen)}
                    className="flex items-center gap-2 px-3 py-2 rounded transition-colors"
                    style={{
                      background: 'var(--panel-bg)',
                      border: '1px solid var(--panel-border)',
                    }}
                    onMouseEnter={(e) =>
                      (e.currentTarget.style.background =
                        'rgba(255, 255, 255, 0.05)')
                    }
                    onMouseLeave={(e) =>
                      (e.currentTarget.style.background = 'var(--panel-bg)')
                    }
                  >
                    <div
                      className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                      style={{
                        background: 'var(--brand-yellow)',
                        color: 'var(--brand-black)',
                      }}
                    >
                      {user.email[0].toUpperCase()}
                    </div>
                    <span
                      className="text-sm"
                      style={{ color: 'var(--brand-light-gray)' }}
                    >
                      {user.email}
                    </span>
                    <ChevronDown
                      className="w-4 h-4"
                      style={{ color: 'var(--brand-light-gray)' }}
                    />
                  </button>

                  {userDropdownOpen && (
                    <div
                      className="absolute right-0 top-full mt-2 w-48 rounded-lg shadow-lg overflow-hidden z-50"
                      style={{
                        background: 'var(--brand-dark-gray)',
                        border: '1px solid var(--panel-border)',
                      }}
                    >
                      <div
                        className="px-3 py-2 border-b"
                        style={{ borderColor: 'var(--panel-border)' }}
                      >
                        <div
                          className="text-xs"
                          style={{ color: 'var(--text-secondary)' }}
                        >
                          {t('loggedInAs', language)}
                        </div>
                        <div
                          className="text-sm font-medium"
                          style={{ color: 'var(--brand-light-gray)' }}
                        >
                          {user.email}
                        </div>
                      </div>
                      {onLogout && (
                        <button
                          onClick={() => {
                            onLogout()
                            setUserDropdownOpen(false)
                          }}
                          className="w-full px-3 py-2 text-sm font-semibold transition-colors hover:opacity-80 text-center"
                          style={{
                            background: 'var(--binance-red-bg)',
                            color: 'var(--binance-red)',
                          }}
                        >
                          {t('exitLogin', language)}
                        </button>
                      )}
                    </div>
                  )}
                </div>
              </div>
            ) : (
              currentPage !== 'login' &&
              currentPage !== 'register' && (
                <div className="flex items-center gap-3">
                  <a
                    href="/login"
                    className="px-3 py-2 text-sm font-medium transition-colors rounded"
                    style={{ color: 'var(--brand-light-gray)' }}
                  >
                    {t('signIn', language)}
                  </a>
                  {registrationEnabled && (
                    <a
                      href="/register"
                      className="px-4 py-2 rounded font-semibold text-sm transition-colors hover:opacity-90"
                      style={{
                        background: 'var(--brand-yellow)',
                        color: 'var(--brand-black)',
                      }}
                    >
                      {t('signUp', language)}
                    </a>
                  )}
                </div>
              )
            )}

            {onLanguageChange && (
              <LanguageToggle
                language={language}
                onLanguageChange={onLanguageChange}
              />
            )}
          </div>
        </div>

        <motion.button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="md:hidden"
          style={{ color: 'var(--brand-light-gray)' }}
          whileTap={{ scale: 0.9 }}
        >
          {mobileMenuOpen ? (
            <X className="w-6 h-6" />
          ) : (
            <Menu className="w-6 h-6" />
          )}
        </motion.button>
      </Container>

      <motion.div
        initial={false}
        animate={
          mobileMenuOpen
            ? { height: 'auto', opacity: 1 }
            : { height: 0, opacity: 0 }
        }
        transition={{ duration: 0.3 }}
        className="md:hidden overflow-hidden"
        style={{
          background: 'var(--brand-dark-gray)',
          borderTop: '1px solid rgba(240, 185, 11, 0.1)',
        }}
      >
        <div className="px-4 py-4 space-y-3">
          <AppNavMenu
            language={language}
            currentPage={currentPage}
            isLoggedIn={isLoggedIn}
            canShow={canShow}
            variant="mobile"
            onNavigate={closeMobileMenu}
          />

          {isHomePage && (
            <HomeNavLinks
              language={language}
              linkClassName="block text-sm py-2"
              onClick={closeMobileMenu}
            />
          )}

          {onLanguageChange && (
            <div className="py-2">
              <div className="flex items-center gap-2 mb-2">
                <span
                  className="text-xs"
                  style={{ color: 'var(--brand-light-gray)' }}
                >
                  {t('language', language)}:
                </span>
              </div>
              <LanguageToggle
                language={language}
                onLanguageChange={handleLanguageChange}
                variant="inline"
              />
            </div>
          )}

          {isLoggedIn && user && (
            <div
              className="mt-4 pt-4"
              style={{ borderTop: '1px solid var(--panel-border)' }}
            >
              <div
                className="flex items-center gap-2 px-3 py-2 mb-2 rounded"
                style={{ background: 'var(--panel-bg)' }}
              >
                <div
                  className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  style={{
                    background: 'var(--brand-yellow)',
                    color: 'var(--brand-black)',
                  }}
                >
                  {user.email[0].toUpperCase()}
                </div>
                <div>
                  <div
                    className="text-xs"
                    style={{ color: 'var(--text-secondary)' }}
                  >
                    {t('loggedInAs', language)}
                  </div>
                  <div
                    className="text-sm"
                    style={{ color: 'var(--brand-light-gray)' }}
                  >
                    {user.email}
                  </div>
                </div>
              </div>
              {onLogout && (
                <button
                  onClick={() => {
                    onLogout()
                    closeMobileMenu()
                  }}
                  className="w-full px-4 py-2 rounded text-sm font-semibold transition-colors text-center"
                  style={{
                    background: 'var(--binance-red-bg)',
                    color: 'var(--binance-red)',
                  }}
                >
                  {t('exitLogin', language)}
                </button>
              )}
            </div>
          )}

          {!isLoggedIn &&
            currentPage !== 'login' &&
            currentPage !== 'register' && (
              <div className="space-y-2 mt-2">
                <a
                  href="/login"
                  className="block w-full px-4 py-2 rounded text-sm font-medium text-center transition-colors"
                  style={{
                    color: 'var(--brand-light-gray)',
                    border: '1px solid var(--brand-light-gray)',
                  }}
                  onClick={closeMobileMenu}
                >
                  {t('signIn', language)}
                </a>
                {registrationEnabled && (
                  <a
                    href="/register"
                    className="block w-full px-4 py-2 rounded font-semibold text-sm text-center transition-colors"
                    style={{
                      background: 'var(--brand-yellow)',
                      color: 'var(--brand-black)',
                    }}
                    onClick={closeMobileMenu}
                  >
                    {t('signUp', language)}
                  </a>
                )}
              </div>
            )}
        </div>
      </motion.div>
    </nav>
  )
}
