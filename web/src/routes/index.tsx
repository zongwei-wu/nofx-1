import { createBrowserRouter, Navigate } from 'react-router-dom'
import MainLayout from '../layouts/MainLayout'
import AuthLayout from '../layouts/AuthLayout'
import { LandingPage } from '../pages/LandingPage'
import { FAQPage } from '../pages/FAQPage'
import { LoginPage } from '../components/LoginPage'
import { RegisterPage } from '../components/RegisterPage'
import { ResetPasswordPage } from '../components/ResetPasswordPage'
import { CompetitionPage } from '../components/CompetitionPage'
import { AITradersPage } from '../pages/AITradersPage'
import TraderDashboard from '../pages/TraderDashboard'
import { CopyTradingPage } from '../pages/CopyTradingPage'
import { CopyTradeDashboard } from '../pages/CopyTradeDashboard'
import { SymbolManagementPage } from '../pages/SymbolManagementPage'
import { ApiKeysPage } from '../pages/ApiKeysPage'
import { FeatureRoute } from '../components/FeatureRoute'
import { FEATURES } from '../config/features'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <LandingPage />,
  },
  // Auth routes - using AuthLayout
  {
    element: <AuthLayout />,
    children: [
      {
        path: '/login',
        element: <LoginPage />,
      },
      {
        path: '/register',
        element: <RegisterPage />,
      },
      {
        path: '/reset-password',
        element: <ResetPasswordPage />,
      },
    ],
  },
  // Main app routes - using MainLayout with nested routes
  {
    element: <MainLayout />,
    children: [
      {
        path: '/faq',
        element: <FAQPage />,
      },
      {
        path: '/competition',
        element: <CompetitionPage />,
      },
      {
        path: '/traders',
        element: (
          <FeatureRoute feature={FEATURES.ai_trader}>
            <AITradersPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/dashboard',
        element: (
          <FeatureRoute feature={FEATURES.ai_trader}>
            <TraderDashboard />
          </FeatureRoute>
        ),
      },
      {
        path: '/copy-trading',
        element: (
          <FeatureRoute feature={FEATURES.leaderboard}>
            <CopyTradingPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/copy-trade',
        element: (
          <FeatureRoute feature={FEATURES.copy_trade}>
            <CopyTradeDashboard />
          </FeatureRoute>
        ),
      },
      {
        path: '/symbols',
        element: (
          <FeatureRoute feature={FEATURES.symbols}>
            <SymbolManagementPage />
          </FeatureRoute>
        ),
      },
      {
        path: '/access-keys',
        element: <ApiKeysPage />,
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/" replace />,
  },
])
