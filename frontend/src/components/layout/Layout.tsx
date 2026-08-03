import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/auth'
import { useLogout } from '@/hooks/useAuth'
import {
  Shield, LayoutDashboard, FileText, Upload, Settings, LogOut,
  Menu, X, Sun, Moon, ChevronRight, Bell, BarChart2, Wifi, WifiOff,
} from 'lucide-react'
import { useState, useEffect } from 'react'
import clsx from 'clsx'
import { useToastStore } from '@/store/toast'
import { wsClient } from '@/lib/ws'

const nav = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/claims',    label: 'Claims',    icon: FileText },
  { to: '/analytics', label: 'Analytics', icon: BarChart2 },
  { to: '/upload',    label: 'New Claim', icon: Upload },
  { to: '/settings',  label: 'Settings',  icon: Settings },
]

// ── WebSocket status hook (polls every 2s) ─────────────────────────────────
function useWsConnected() {
  const [connected, setConnected] = useState(() => wsClient.isConnected)
  useEffect(() => {
    const id = setInterval(() => setConnected(wsClient.isConnected), 2000)
    return () => clearInterval(id)
  }, [])
  return connected
}

/** Map route paths to human-readable breadcrumb labels */
function useBreadcrumbs() {
  const { pathname } = useLocation()
  const parts = pathname.split('/').filter(Boolean)
  const crumbs: { label: string; to: string }[] = []
  let accumulated = ''
  for (const part of parts) {
    accumulated += `/${part}`
    const label =
      part === 'dashboard' ? 'Dashboard' :
      part === 'claims'    ? 'Claims' :
      part === 'upload'    ? 'New Claim' :
      part === 'settings'  ? 'Settings' :
      part.length === 36   ? `Claim ${part.slice(0, 8)}…` : part
    crumbs.push({ label, to: accumulated })
  }
  return crumbs
}

// ── Dark-mode hook ─────────────────────────────────────────────────────────
function useDarkMode() {
  const [dark, setDark] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false
    const saved = localStorage.getItem('goshield-theme')
    if (saved) return saved === 'dark'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  })

  useEffect(() => {
    const root = document.documentElement
    if (dark) {
      root.classList.add('dark')
      localStorage.setItem('goshield-theme', 'dark')
    } else {
      root.classList.remove('dark')
      localStorage.setItem('goshield-theme', 'light')
    }
  }, [dark])

  return [dark, () => setDark(d => !d)] as const
}

// ── Toast component ────────────────────────────────────────────────────────
export function ToastContainer() {
  const { toasts, dismiss } = useToastStore()
  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm w-full pointer-events-none">
      {toasts.map(t => (
        <div
          key={t.id}
          className={clsx(
            'pointer-events-auto flex items-start gap-3 rounded-xl border px-4 py-3 shadow-lg text-sm animate-slide-in',
            t.type === 'success' && 'bg-green-50 border-green-200 text-green-800 dark:bg-green-900/50 dark:border-green-700 dark:text-green-200',
            t.type === 'error'   && 'bg-red-50 border-red-200 text-red-800 dark:bg-red-900/50 dark:border-red-700 dark:text-red-200',
            t.type === 'info'    && 'bg-brand-50 border-brand-200 text-brand-800 dark:bg-brand-900/50 dark:border-brand-700 dark:text-brand-200',
            t.type === 'warning' && 'bg-amber-50 border-amber-200 text-amber-800 dark:bg-amber-900/50 dark:border-amber-700 dark:text-amber-200',
          )}
        >
          <Bell className="h-4 w-4 mt-0.5 shrink-0" />
          <div className="flex-1 min-w-0">
            {t.title && <p className="font-semibold">{t.title}</p>}
            <p className="text-xs opacity-90">{t.message}</p>
          </div>
          <button onClick={() => dismiss(t.id)} className="shrink-0 opacity-60 hover:opacity-100">
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}
    </div>
  )
}

export default function Layout({ children }: { children: React.ReactNode }) {
  const user        = useAuthStore((s) => s.user)
  const logout      = useLogout()
  const location    = useLocation()
  const navigate    = useNavigate()
  const [open, setOpen]   = useState(false)
  const [dark, toggleDark] = useDarkMode()
  const crumbs      = useBreadcrumbs()
  const wsConnected = useWsConnected()

  // Close sidebar on route change
  useEffect(() => { setOpen(false) }, [location.pathname])

  const initials = `${user?.firstName?.[0] ?? ''}${user?.lastName?.[0] ?? ''}`

  return (
    <div className="flex h-full bg-gray-50 dark:bg-gray-950">
      {/* ── Sidebar ─────────────────────────────────────────────────────── */}
      <aside className={clsx(
        'fixed inset-y-0 left-0 z-40 flex w-64 flex-col bg-gray-900 text-white',
        'transition-transform duration-200 ease-in-out lg:static lg:translate-x-0',
        open ? 'translate-x-0' : '-translate-x-full'
      )}>
        {/* Logo */}
        <div className="flex h-16 items-center gap-3 px-6 border-b border-gray-700/60">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600">
            <Shield className="h-5 w-5 text-white" />
          </div>
          <span className="text-xl font-bold tracking-tight">GoShield</span>
        </div>

        {/* Nav links */}
        <nav className="flex-1 space-y-0.5 px-3 py-4">
          {nav.map(({ to, label, icon: Icon }) => {
            const active = location.pathname === to ||
              (to !== '/dashboard' && location.pathname.startsWith(to))
            return (
              <Link
                key={to} to={to}
                className={clsx(
                  'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                  active
                    ? 'bg-brand-600/20 text-brand-300 border border-brand-600/30'
                    : 'text-gray-400 hover:bg-gray-800 hover:text-white'
                )}
              >
                <Icon className={clsx('h-4 w-4', active && 'text-brand-400')} />
                {label}
                {active && <span className="ml-auto h-1.5 w-1.5 rounded-full bg-brand-400" />}
              </Link>
            )
          })}
        </nav>

        {/* User section */}
        <div className="border-t border-gray-700/60 p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-full bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center text-sm font-bold shrink-0">
              {initials}
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium truncate">{user?.firstName} {user?.lastName}</p>
              <p className="text-xs text-gray-400 truncate">{user?.email}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="flex-1 inline-flex items-center justify-center rounded-md bg-gray-800 px-2 py-1 text-xs text-gray-300 font-medium">
              {user?.role}
            </span>
            <button
              onClick={() => logout.mutate()}
              title="Logout"
              className="flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-gray-400 hover:bg-gray-800 hover:text-white transition-colors"
            >
              <LogOut className="h-3.5 w-3.5" />Logout
            </button>
          </div>
        </div>
      </aside>

      {/* Mobile overlay */}
      {open && (
        <div
          className="fixed inset-0 z-30 bg-black/60 backdrop-blur-sm lg:hidden"
          onClick={() => setOpen(false)}
        />
      )}

      {/* ── Main area ───────────────────────────────────────────────────── */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Top bar */}
        <header className="flex h-14 items-center justify-between gap-4 border-b bg-white dark:bg-gray-900 px-4 lg:px-6 shrink-0">
          <div className="flex items-center gap-3 min-w-0">
            {/* Mobile menu toggle */}
            <button
              onClick={() => setOpen(o => !o)}
              className="lg:hidden p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800"
            >
              {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
            {/* Breadcrumbs */}
            <nav className="flex items-center gap-1 text-sm min-w-0">
              <button
                onClick={() => navigate('/dashboard')}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 shrink-0"
              >
                <Shield className="h-4 w-4" />
              </button>
              {crumbs.map((crumb, i) => (
                <span key={crumb.to} className="flex items-center gap-1 min-w-0">
                  <ChevronRight className="h-3.5 w-3.5 text-gray-300 dark:text-gray-600 shrink-0" />
                  {i === crumbs.length - 1 ? (
                    <span className="font-medium text-gray-800 dark:text-gray-200 truncate">
                      {crumb.label}
                    </span>
                  ) : (
                    <Link
                      to={crumb.to}
                      className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 truncate"
                    >
                      {crumb.label}
                    </Link>
                  )}
                </span>
              ))}
            </nav>
          </div>

          {/* Right actions */}
          <div className="flex items-center gap-2 shrink-0">
            {/* WebSocket connection status */}
            <div
              title={wsConnected ? 'Real-time connection active' : 'Real-time connection offline'}
              className={clsx(
                'hidden sm:flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium transition-colors',
                wsConnected
                  ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
              )}
            >
              {wsConnected
                ? <><Wifi className="h-3 w-3" />Live</>
                : <><WifiOff className="h-3 w-3" />Offline</>
              }
            </div>
            {/* Dark mode toggle */}
            <button
              onClick={toggleDark}
              title={dark ? 'Switch to light mode' : 'Switch to dark mode'}
              className="p-2 rounded-lg text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 dark:text-gray-400 transition-colors"
            >
              {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </button>
            {/* Avatar chip */}
            <div className="hidden sm:flex items-center gap-2 rounded-full bg-gray-100 dark:bg-gray-800 pl-1.5 pr-3 py-1">
              <div className="h-6 w-6 rounded-full bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center text-xs font-bold text-white">
                {initials}
              </div>
              <span className="text-xs font-medium text-gray-700 dark:text-gray-300">
                {user?.firstName}
              </span>
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto p-5 lg:p-7">
          {children}
        </main>
      </div>

      {/* Toast notifications */}
      <ToastContainer />
    </div>
  )
}
