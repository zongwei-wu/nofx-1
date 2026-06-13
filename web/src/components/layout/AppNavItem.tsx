import { cn } from '../../lib/cn'
import type { NavItem } from '../../config/navItems'
import { t, type Language } from '../../i18n/translations'

export function getNavLabel(item: NavItem, language: Language): string {
  if (item.labelKey) return t(item.labelKey, language)
  return item.label ?? item.key
}

interface AppNavItemProps {
  item: NavItem
  language: Language
  isActive: boolean
  onClick: () => void
  variant?: 'desktop' | 'mobile'
}

export function AppNavItem({
  item,
  language,
  isActive,
  onClick,
  variant = 'desktop',
}: AppNavItemProps) {
  const isMobile = variant === 'mobile'

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'text-sm font-bold transition-all duration-300 relative focus:outline-2 focus:outline-yellow-500 rounded-lg',
        isMobile ? 'block w-full text-left px-4 py-3' : 'px-4 py-2',
        isActive
          ? 'text-[var(--brand-yellow)]'
          : 'text-[var(--brand-light-gray)] hover:text-[var(--brand-yellow)]'
      )}
    >
      {isActive && (
        <span
          className="absolute inset-0 rounded-lg -z-10"
          style={{ background: 'rgba(240, 185, 11, 0.15)' }}
        />
      )}
      {getNavLabel(item, language)}
    </button>
  )
}

interface GuestNavLinkProps {
  item: NavItem
  language: Language
  isActive: boolean
  variant?: 'desktop' | 'mobile'
  onNavigate?: () => void
}

export function GuestNavLink({
  item,
  language,
  isActive,
  variant = 'desktop',
  onNavigate,
}: GuestNavLinkProps) {
  const isMobile = variant === 'mobile'

  return (
    <a
      href={item.path}
      onClick={() => onNavigate?.()}
      className={cn(
        'text-sm font-bold transition-all duration-300 relative focus:outline-2 focus:outline-yellow-500 rounded-lg',
        isMobile ? 'block px-4 py-3' : 'px-4 py-2',
        isActive
          ? 'text-[var(--brand-yellow)]'
          : 'text-[var(--brand-light-gray)] hover:text-[var(--brand-yellow)]'
      )}
    >
      {isActive && (
        <span
          className="absolute inset-0 rounded-lg -z-10"
          style={{ background: 'rgba(240, 185, 11, 0.15)' }}
        />
      )}
      {getNavLabel(item, language)}
    </a>
  )
}
