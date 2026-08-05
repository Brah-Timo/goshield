import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Bell, X, CheckCheck, Trash2, Info, AlertTriangle,
  CheckCircle, XCircle, ExternalLink,
} from 'lucide-react'
import clsx from 'clsx'
import { useNotifStore, type Notification, type NotifSeverity } from '@/store/notifications'
import { formatDistanceToNow } from 'date-fns'

// ── Severity styling ──────────────────────────────────────────────────────────
const SEVERITY_STYLES: Record<NotifSeverity, {
  icon: React.ElementType
  ring: string
  bg: string
  iconCls: string
}> = {
  info: {
    icon: Info,
    ring: 'ring-brand-200 dark:ring-brand-700',
    bg:   'bg-brand-50 dark:bg-brand-900/20',
    iconCls: 'text-brand-500',
  },
  warning: {
    icon: AlertTriangle,
    ring: 'ring-amber-200 dark:ring-amber-700',
    bg:   'bg-amber-50 dark:bg-amber-900/20',
    iconCls: 'text-amber-500',
  },
  error: {
    icon: XCircle,
    ring: 'ring-red-200 dark:ring-red-700',
    bg:   'bg-red-50 dark:bg-red-900/20',
    iconCls: 'text-red-500',
  },
  success: {
    icon: CheckCircle,
    ring: 'ring-green-200 dark:ring-green-700',
    bg:   'bg-green-50 dark:bg-green-900/20',
    iconCls: 'text-green-500',
  },
}

// ── Single notification row ───────────────────────────────────────────────────
function NotifRow({ n, onDismiss, onRead }: {
  n: Notification
  onDismiss: (id: string) => void
  onRead: (id: string) => void
}) {
  const s = SEVERITY_STYLES[n.severity]
  const Icon = s.icon

  return (
    <div
      className={clsx(
        'group relative flex items-start gap-3 px-4 py-3.5 border-b border-gray-100 dark:border-gray-700/60 last:border-0 transition-colors',
        !n.read && 'bg-brand-50/30 dark:bg-brand-900/10',
        'hover:bg-gray-50 dark:hover:bg-gray-800/40',
      )}
      onClick={() => !n.read && onRead(n.id)}
    >
      {/* Icon */}
      <div className={clsx('mt-0.5 rounded-full p-1.5 ring-1 shrink-0', s.bg, s.ring)}>
        <Icon className={clsx('h-3.5 w-3.5', s.iconCls)} />
      </div>

      {/* Body */}
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <p className={clsx('text-xs font-semibold', !n.read ? 'text-gray-900 dark:text-white' : 'text-gray-600 dark:text-gray-300')}>
            {n.title}
          </p>
          <span className="text-[10px] text-gray-400 shrink-0 mt-0.5">
            {formatDistanceToNow(n.ts, { addSuffix: true })}
          </span>
        </div>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 leading-relaxed">
          {n.body}
        </p>
        {n.claimId && (
          <Link
            to={`/claims/${n.claimId}`}
            onClick={e => e.stopPropagation()}
            className="mt-1 inline-flex items-center gap-1 text-[10px] font-medium text-brand-600 dark:text-brand-400 hover:underline"
          >
            <ExternalLink className="h-2.5 w-2.5" />
            View claim
          </Link>
        )}
      </div>

      {/* Unread dot */}
      {!n.read && (
        <span className="absolute right-3 top-3.5 h-2 w-2 rounded-full bg-brand-500" />
      )}

      {/* Dismiss X (appears on hover) */}
      <button
        onClick={e => { e.stopPropagation(); onDismiss(n.id) }}
        className="absolute right-2.5 top-2.5 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-0.5 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

// ── Main component ────────────────────────────────────────────────────────────
export default function NotificationCenter() {
  const [open, setOpen] = useState(false)
  const panelRef        = useRef<HTMLDivElement>(null)
  const btnRef          = useRef<HTMLButtonElement>(null)

  const { items, unread, markRead, markAllRead, dismiss, clearAll } = useNotifStore()

  // Close on outside click
  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (
        panelRef.current && !panelRef.current.contains(e.target as Node) &&
        btnRef.current   && !btnRef.current.contains(e.target as Node)
      ) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  // Close on Escape
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false) }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [])

  return (
    <div className="relative">
      {/* Bell button */}
      <button
        ref={btnRef}
        onClick={() => setOpen(v => !v)}
        title="Notifications"
        className={clsx(
          'relative p-2 rounded-lg transition-colors',
          open
            ? 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200'
            : 'text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 dark:text-gray-400',
        )}
      >
        <Bell className="h-4 w-4" />
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-white text-[9px] font-bold leading-none">
            {unread > 9 ? '9+' : unread}
          </span>
        )}
      </button>

      {/* Panel */}
      {open && (
        <div
          ref={panelRef}
          className="absolute right-0 top-full mt-2 z-50 w-80 sm:w-96 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-2xl overflow-hidden animate-slide-in"
        >
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100 dark:border-gray-700">
            <div className="flex items-center gap-2">
              <Bell className="h-4 w-4 text-gray-500 dark:text-gray-400" />
              <span className="text-sm font-semibold text-gray-800 dark:text-gray-200">Notifications</span>
              {unread > 0 && (
                <span className="rounded-full bg-brand-100 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 text-[10px] font-bold px-1.5 py-0.5">
                  {unread} new
                </span>
              )}
            </div>
            <div className="flex items-center gap-1">
              {unread > 0 && (
                <button
                  onClick={markAllRead}
                  title="Mark all as read"
                  className="p-1.5 rounded-lg text-gray-400 hover:text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-900/20 transition-colors"
                >
                  <CheckCheck className="h-3.5 w-3.5" />
                </button>
              )}
              {items.length > 0 && (
                <button
                  onClick={clearAll}
                  title="Clear all"
                  className="p-1.5 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              )}
              <button
                onClick={() => setOpen(false)}
                className="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>

          {/* List */}
          <div className="max-h-[420px] overflow-y-auto">
            {items.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 gap-3">
                <div className="h-12 w-12 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
                  <Bell className="h-6 w-6 text-gray-300 dark:text-gray-600" />
                </div>
                <p className="text-sm text-gray-400">No notifications yet</p>
                <p className="text-xs text-gray-300 dark:text-gray-600">
                  Real-time fraud alerts will appear here
                </p>
              </div>
            ) : (
              items.map(n => (
                <NotifRow
                  key={n.id}
                  n={n}
                  onDismiss={dismiss}
                  onRead={markRead}
                />
              ))
            )}
          </div>

          {/* Footer */}
          {items.length > 0 && (
            <div className="border-t border-gray-100 dark:border-gray-700 px-4 py-2.5 flex items-center justify-between">
              <span className="text-xs text-gray-400">{items.length} notification{items.length !== 1 ? 's' : ''}</span>
              <Link
                to="/analytics"
                onClick={() => setOpen(false)}
                className="text-xs font-medium text-brand-600 dark:text-brand-400 hover:underline"
              >
                View analytics →
              </Link>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
