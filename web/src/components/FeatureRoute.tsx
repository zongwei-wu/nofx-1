import { ReactNode, useEffect, useRef } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '../contexts/AuthContext'
import type { FeatureKey } from '../config/features'

interface FeatureRouteProps {
  feature: FeatureKey
  children: ReactNode
}

export function FeatureRoute({ feature, children }: FeatureRouteProps) {
  const { user, hasFeature, isLoading, refreshPermissions } = useAuth()
  const navigate = useNavigate()

  const pendingAuth =
    !user &&
    typeof localStorage !== 'undefined' &&
    !!localStorage.getItem('auth_token')

  const authSyncRef = useRef(false)
  useEffect(() => {
    if (!pendingAuth) {
      authSyncRef.current = false
      return
    }
    if (authSyncRef.current) return
    authSyncRef.current = true
    void refreshPermissions()
  }, [pendingAuth, refreshPermissions])

  useEffect(() => {
    if (!isLoading && !pendingAuth && user && !hasFeature(feature)) {
      toast.error('当前套餐无此功能权限，已跳转到竞赛页')
      navigate('/competition', { replace: true })
    }
  }, [user, isLoading, pendingAuth, feature, hasFeature, navigate])

  if (isLoading || pendingAuth) {
    return null
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  if (!hasFeature(feature)) {
    return null
  }

  return <>{children}</>
}
