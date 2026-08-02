import { Link, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/auth'
import { useLogout } from '@/hooks/useAuth'
import { Shield, LayoutDashboard, FileText, Upload, Settings, LogOut, Menu, X } from 'lucide-react'
import { useState } from 'react'
import clsx from 'clsx'

const nav = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/claims',    label: 'Claims',    icon: FileText },
  { to: '/upload',    label: 'Upload',    icon: Upload },
  { to: '/settings',  label: 'Settings',  icon: Settings },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((s) => s.user)
  const logout = useLogout()
  const [open, setOpen] = useState(false)

  return (
    <div className="flex h-full">
      {/* Sidebar */}
      <aside className={clsx(
        'fixed inset-y-0 left-0 z-40 flex w-64 flex-col bg-gray-900 text-white transition-transform lg:static lg:translate-x-0',
        open ? 'translate-x-0' : '-translate-x-full'
      )}>
        <div className="flex h-16 items-center gap-3 px-6 border-b border-gray-700">
          <Shield className="h-7 w-7 text-brand-400" />
          <span className="text-xl font-bold tracking-tight">GoShield</span>
        </div>
        <nav className="flex-1 space-y-1 px-3 py-4">
          {nav.map(({ to, label, icon: Icon }) => (
            <Link key={to} to={to} onClick={() => setOpen(false)}
              className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-gray-300 hover:bg-gray-800 hover:text-white transition-colors">
              <Icon className="h-5 w-5" />{label}
            </Link>
          ))}
        </nav>
        <div className="border-t border-gray-700 p-4">
          <div className="flex items-center gap-3 mb-3">
            <div className="h-8 w-8 rounded-full bg-brand-600 flex items-center justify-center text-sm font-bold">
              {user?.firstName?.[0]}{user?.lastName?.[0]}
            </div>
            <div className="min-w-0">
              <p className="text-sm font-medium truncate">{user?.firstName} {user?.lastName}</p>
              <p className="text-xs text-gray-400 truncate">{user?.role}</p>
            </div>
          </div>
          <button onClick={() => logout.mutate()}
            className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-gray-400 hover:bg-gray-800 hover:text-white transition-colors">
            <LogOut className="h-4 w-4" />Logout
          </button>
        </div>
      </aside>

      {/* Mobile overlay */}
      {open && <div className="fixed inset-0 z-30 bg-black/50 lg:hidden" onClick={() => setOpen(false)} />}

      <div className="flex-1 flex flex-col min-w-0 lg:pl-0">
        {/* Mobile header */}
        <header className="flex h-16 items-center gap-4 border-b bg-white dark:bg-gray-900 px-4 lg:hidden">
          <button onClick={() => setOpen(true)}><Menu className="h-6 w-6" /></button>
          <Shield className="h-6 w-6 text-brand-600" />
          <span className="font-bold">GoShield</span>
        </header>
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </div>
    </div>
  )
}
