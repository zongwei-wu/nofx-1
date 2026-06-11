import { useNavigate } from 'react-router-dom'
import { LOGGED_IN_NAV_ITEMS, type NavItem } from '../../config/navItems'
import type { Language } from '../../i18n/translations'
import { AppNavItem, GuestNavLink } from './AppNavItem'

/** 未登录时展示的公开导航项 */
const GUEST_NAV_ITEMS = LOGGED_IN_NAV_ITEMS.filter(
  (item) => item.key === 'competition' || item.key === 'faq'
)

interface AppNavMenuProps {
  language: Language
  currentPage?: string
  isLoggedIn: boolean
  canShow: (feature: string | null) => boolean
  variant?: 'desktop' | 'mobile'
  onNavigate?: () => void
}

export function AppNavMenu({
  language,
  currentPage,
  isLoggedIn,
  canShow,
  variant = 'desktop',
  onNavigate,
}: AppNavMenuProps) {
  const navigate = useNavigate()

  const go = (path: string) => {
    navigate(path)
    onNavigate?.()
  }

  if (isLoggedIn) {
    return (
      <>
        {LOGGED_IN_NAV_ITEMS.filter((item) => canShow(item.feature)).map(
          (item) => (
            <AppNavItem
              key={item.key}
              item={item}
              language={language}
              isActive={currentPage === item.key}
              variant={variant}
              onClick={() => go(item.path)}
            />
          )
        )}
      </>
    )
  }

  return (
    <>
      {GUEST_NAV_ITEMS.map((item) => (
        <GuestNavLink
          key={item.key}
          item={item}
          language={language}
          isActive={currentPage === item.key}
          variant={variant}
          onNavigate={onNavigate}
        />
      ))}
    </>
  )
}

export function getVisibleNavItems(
  isLoggedIn: boolean,
  canShow: (feature: string | null) => boolean
): NavItem[] {
  if (!isLoggedIn) return GUEST_NAV_ITEMS
  return LOGGED_IN_NAV_ITEMS.filter((item) => canShow(item.feature))
}
