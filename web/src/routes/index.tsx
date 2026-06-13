import { lazy, Suspense, type ReactNode } from 'react'
import { createBrowserRouter, Navigate } from 'react-router-dom'
import MainLayout from '../layouts/MainLayout'
import AuthLayout from '../layouts/AuthLayout'
import { LandingPage } from '../pages/LandingPage'
import { FAQPage } from '../pages/FAQPage'
import { FeatureRoute } from '../components/FeatureRoute'
import { FEATURES } from '../config/features'
import { t } from '../i18n/translations'

const LoginPage = lazy(() =>
  import('../components/LoginPage').then((m) => ({ default: m.LoginPage }))
)
const RegisterPage = lazy(() =>
  import('../components/RegisterPage').then((m) => ({ default: m.RegisterPage }))
)
const ResetPasswordPage = lazy(() =>
  import('../components/ResetPasswordPage').then((m) => ({
    default: m.ResetPasswordPage,
  }))
)
const CompetitionPage = lazy(() =>
  import('../components/CompetitionPage').then((m) => ({
    default: m.CompetitionPage,
  }))
)
const AITradersPage = lazy(() =>
  import('../pages/AITradersPage').then((m) => ({ default: m.AITradersPage }))
)
const TraderDashboard = lazy(() => import('../pages/TraderDashboard'))
const CopyTradingPage = lazy(() =>
  import('../pages/CopyTradingPage').then((m) => ({
    default: m.CopyTradingPage,
  }))
)
const CopyTradeDashboard = lazy(() =>
  import('../pages/CopyTradeDashboard').then((m) => ({
    default: m.CopyTradeDashboard,
  }))
)
const SymbolManagementPage = lazy(() =>
  import('../pages/SymbolManagementPage').then((m) => ({
    default: m.SymbolManagementPage,
  }))
)
const ApiKeysPage = lazy(() => import('../pages/ApiKeysPage'))
const StrategiesPage = lazy(() =>
  import('../pages/StrategiesPage').then((m) => ({ default: m.StrategiesPage }))
)
const StrategyDetailPage = lazy(() =>
  import('../pages/StrategyDetailPage').then((m) => ({ default: m.StrategyDetailPage }))
)
const BacktestPage = lazy(() =>
  import('../pages/BacktestPage').then((m) => ({ default: m.BacktestPage }))
)

function RouteFallback() {
  const lang =
    (localStorage.getItem('language') as 'en' | 'zh' | null) ?? 'en'
  return (
    <div
      className="min-h-[40vh] flex items-center justify-center"
      style={{ color: 'var(--text-secondary)' }}
    >
      {t('loading', lang)}
    </div>
  )
}

function withSuspense(node: ReactNode) {
  return <Suspense fallback={<RouteFallback />}>{node}</Suspense>
}

export const router = createBrowserRouter([
  {
    path: '/',
    element: <LandingPage />,
  },
  {
    element: <AuthLayout />,
    children: [
      {
        path: '/login',
        element: withSuspense(<LoginPage />),
      },
      {
        path: '/register',
        element: withSuspense(<RegisterPage />),
      },
      {
        path: '/reset-password',
        element: withSuspense(<ResetPasswordPage />),
      },
    ],
  },
  {
    element: <MainLayout />,
    children: [
      {
        path: '/faq',
        element: <FAQPage />,
      },
      {
        path: '/competition',
        element: withSuspense(<CompetitionPage />),
      },
      {
        path: '/traders',
        element: withSuspense(
          <FeatureRoute feature={FEATURES.ai_trader}>
            <AITradersPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/dashboard',
        element: withSuspense(
          <FeatureRoute feature={FEATURES.ai_trader}>
            <TraderDashboard />
          </FeatureRoute>
        ),
      },
      {
        path: '/copy-trading',
        element: withSuspense(
          <FeatureRoute feature={FEATURES.leaderboard}>
            <CopyTradingPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/copy-trade',
        element: withSuspense(
          <FeatureRoute feature={FEATURES.copy_trade}>
            <CopyTradeDashboard />
          </FeatureRoute>
        ),
      },
      {
        path: '/symbols',
        element: withSuspense(
          <FeatureRoute feature={FEATURES.symbols}>
            <SymbolManagementPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/strategies/:id/backtest/:backtestId',
        element: withSuspense(
          <FeatureRoute feature={FEATURES.strategy}>
            <BacktestPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/strategies/:id',
        element: withSuspense(
          <FeatureRoute feature={FEATURES.strategy}>
            <StrategyDetailPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/strategies',
        element: withSuspense(
          <FeatureRoute feature={FEATURES.strategy}>
            <StrategiesPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/access-keys',
        element: withSuspense(<ApiKeysPage />),
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/" replace />,
  },
])
