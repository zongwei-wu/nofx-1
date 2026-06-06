import { ReactNode, useEffect } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '../contexts/AuthContext'
import type { FeatureKey } from '../config/features'

interface FeatureRouteProps {
  feature: FeatureKey
  children: ReactNode
}

export function FeatureRoute({ feature, children }: FeatureRouteProps) {
  const { user, hasFeature, isLoading } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (!isLoading && user && !hasFeature(feature)) {
      toast.error('无此功能使用权限')
      navigate('/competition', { replace: true })
    }
  }, [user, isLoading, feature, hasFeature, navigate])

  if (isLoading) {
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
