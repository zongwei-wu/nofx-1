import { HOME_NAV_ITEMS } from '../../config/navItems'
import { t, type Language } from '../../i18n/translations'

interface HomeNavLinksProps {
  language: Language
  className?: string
  linkClassName?: string
  onClick?: () => void
}

export function HomeNavLinks({
  language,
  className = '',
  linkClassName = '',
  onClick,
}: HomeNavLinksProps) {
  return (
    <>
      {HOME_NAV_ITEMS.map((item) => (
        <a
          key={item.key}
          href={item.hash}
          className={linkClassName}
          onClick={onClick}
          style={{ color: 'var(--brand-light-gray)' }}
        >
          {t(item.labelKey, language)}
        </a>
      ))}
    </>
  )
}
