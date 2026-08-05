import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from '@/store/auth'
import { useWebSocketUpdates } from '@/hooks/useWebSocket'

import LoginPage       from '@/pages/LoginPage'
import RegisterPage    from '@/pages/RegisterPage'
import DashboardPage   from '@/pages/DashboardPage'
import ClaimsPage      from '@/pages/ClaimsPage'
import ClaimDetailPage from '@/pages/ClaimDetailPage'
import UploadPage      from '@/pages/UploadPage'
import SettingsPage    from '@/pages/SettingsPage'
import AnalyticsPage   from '@/pages/AnalyticsPage'
import SearchPage      from '@/pages/SearchPage'
import ProfilePage     from '@/pages/ProfilePage'
import NotFoundPage    from '@/pages/NotFoundPage'
import Layout          from '@/components/layout/Layout'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

function AuthenticatedApp() {
  useWebSocketUpdates()
  return (
    <Layout>
      <Routes>
        <Route path="/dashboard"  element={<DashboardPage />} />
        <Route path="/claims"     element={<ClaimsPage />} />
        <Route path="/claims/:id" element={<ClaimDetailPage />} />
        <Route path="/upload"     element={<UploadPage />} />
        <Route path="/analytics"  element={<AnalyticsPage />} />
        <Route path="/search"     element={<SearchPage />} />
        <Route path="/profile"    element={<ProfilePage />} />
        <Route path="/settings"   element={<SettingsPage />} />
        <Route path="/404"        element={<NotFoundPage />} />
        <Route path="/"           element={<Navigate to="/dashboard" replace />} />
        <Route path="*"           element={<Navigate to="/404" replace />} />
      </Routes>
    </Layout>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/login"    element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <AuthenticatedApp />
          </RequireAuth>
        }
      />
    </Routes>
  )
}
