import { useState, useRef } from 'react'
import { useAuthStore } from '@/store/auth'
import { useNotifStore } from '@/store/notifications'
import { format, formatDistanceToNow } from 'date-fns'
import {
  User, Mail, Building2, Shield, Calendar, Camera,
  Bell, CheckCircle, AlertTriangle, XCircle, Info,
  Edit2, Save, X, Clock, Activity, ChevronRight,
  Trash2,
} from 'lucide-react'
import { toast } from '@/store/toast'
import type { NotifSeverity } from '@/store/notifications'

// ── Severity icon ─────────────────────────────────────────────────────────────
const SEV_ICON: Record<NotifSeverity, React.ReactNode> = {
  info:    <Info        className="h-3.5 w-3.5 text-brand-500" />,
  warning: <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />,
  error:   <XCircle    className="h-3.5 w-3.5 text-red-500" />,
  success: <CheckCircle className="h-3.5 w-3.5 text-green-500" />,
}

const SEV_RING: Record<NotifSeverity, string> = {
  info:    'ring-brand-400/30 bg-brand-50 dark:bg-brand-900/20',
  warning: 'ring-amber-400/30 bg-amber-50 dark:bg-amber-900/20',
  error:   'ring-red-400/30 bg-red-50 dark:bg-red-900/20',
  success: 'ring-green-400/30 bg-green-50 dark:bg-green-900/20',
}

// ── Role badge ────────────────────────────────────────────────────────────────
const ROLE_BADGE: Record<string, string> = {
  ADMIN:   'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
  ANALYST: 'bg-brand-100  text-brand-700  dark:bg-brand-900/40  dark:text-brand-300',
  VIEWER:  'bg-gray-100   text-gray-600   dark:bg-gray-800       dark:text-gray-400',
}

// ── Avatar initials ───────────────────────────────────────────────────────────
function AvatarCircle({ firstName, lastName, src, size = 'lg' }: {
  firstName: string
  lastName:  string
  src?:      string
  size?:     'sm' | 'md' | 'lg' | 'xl'
}) {
  const dim = { sm: 'h-8 w-8 text-sm', md: 'h-10 w-10 text-base', lg: 'h-16 w-16 text-xl', xl: 'h-24 w-24 text-3xl' }[size]
  const initials = `${firstName?.[0] ?? ''}${lastName?.[0] ?? ''}`.toUpperCase()

  if (src) {
    return <img src={src} alt={initials} className={`${dim} rounded-full object-cover ring-2 ring-white dark:ring-gray-800`} />
  }
  return (
    <div className={`${dim} rounded-full bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center font-bold text-white ring-2 ring-white dark:ring-gray-800`}>
      {initials || <User className="h-1/2 w-1/2" />}
    </div>
  )
}

// ── Info row ──────────────────────────────────────────────────────────────────
function InfoRow({ icon: Icon, label, value }: {
  icon:  React.ComponentType<{ className?: string }>
  label: string
  value: React.ReactNode
}) {
  return (
    <div className="flex items-start gap-3 py-3 border-b border-gray-50 dark:border-gray-800 last:border-0">
      <div className="shrink-0 h-8 w-8 rounded-lg bg-gray-50 dark:bg-gray-800 flex items-center justify-center">
        <Icon className="h-4 w-4 text-gray-400" />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-gray-400 dark:text-gray-500">{label}</p>
        <p className="text-sm font-medium text-gray-800 dark:text-gray-200 truncate">{value}</p>
      </div>
    </div>
  )
}

// ════════════════════════════════════════════════════════════════════════════
export default function ProfilePage() {
  const user       = useAuthStore(s => s.user)
  const notifs     = useNotifStore(s => s.items)
  const dismiss    = useNotifStore(s => s.dismiss)
  const clearAll   = useNotifStore(s => s.clearAll)
  const markRead   = useNotifStore(s => s.markRead)

  // Avatar state (local preview only — no backend upload in this scaffold)
  const [avatarSrc,  setAvatarSrc]  = useState<string | undefined>()
  const [editBio,    setEditBio]    = useState(false)
  const [bio,        setBio]        = useState('')
  const [draftBio,   setDraftBio]   = useState('')
  const fileRef = useRef<HTMLInputElement>(null)

  // Stats derived from notifications
  const totalEvents = notifs.length
  const flagged     = notifs.filter(n => n.severity === 'error' || n.severity === 'warning').length
  const successes   = notifs.filter(n => n.severity === 'success').length
  const unread      = notifs.filter(n => !n.read).length

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (file.size > 2 * 1024 * 1024) { toast.error('Image must be < 2 MB', 'Too large'); return }
    const reader = new FileReader()
    reader.onload = ev => setAvatarSrc(ev.target?.result as string)
    reader.readAsDataURL(file)
    toast.success('Avatar updated (preview only)', 'Avatar')
  }

  if (!user) return null

  return (
    <div className="max-w-4xl space-y-6 animate-fade-in">
      {/* ── Page header ─────────────────────────────────────────────────── */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">My Profile</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
          Manage your personal information and view your activity log
        </p>
      </div>

      <div className="grid lg:grid-cols-3 gap-6">
        {/* ── Left: identity card ────────────────────────────────────────── */}
        <div className="lg:col-span-1 space-y-4">
          {/* Avatar + name */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
            <div className="flex flex-col items-center text-center gap-3">
              {/* Avatar with upload overlay */}
              <div className="relative group">
                <AvatarCircle
                  firstName={user.firstName}
                  lastName={user.lastName}
                  src={avatarSrc}
                  size="xl"
                />
                <button
                  onClick={() => fileRef.current?.click()}
                  className="absolute inset-0 rounded-full bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center"
                  title="Change avatar"
                >
                  <Camera className="h-6 w-6 text-white" />
                </button>
                <input
                  ref={fileRef}
                  type="file"
                  accept="image/png,image/jpeg,image/webp"
                  className="hidden"
                  onChange={handleAvatarChange}
                />
              </div>

              <div>
                <h2 className="text-lg font-bold text-gray-900 dark:text-white">
                  {user.firstName} {user.lastName}
                </h2>
                <p className="text-sm text-gray-500 dark:text-gray-400">{user.email}</p>
                <span className={`mt-2 inline-flex items-center gap-1.5 rounded-full px-3 py-0.5 text-xs font-semibold ${ROLE_BADGE[user.role] ?? ROLE_BADGE.VIEWER}`}>
                  <Shield className="h-3 w-3" />
                  {user.role}
                </span>
              </div>

              {/* Bio */}
              <div className="w-full text-left">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-xs font-medium text-gray-400">Bio</p>
                  {!editBio && (
                    <button
                      onClick={() => { setDraftBio(bio); setEditBio(true) }}
                      className="text-xs text-brand-600 dark:text-brand-400 hover:underline flex items-center gap-0.5"
                    >
                      <Edit2 className="h-3 w-3" /> Edit
                    </button>
                  )}
                </div>
                {editBio ? (
                  <div className="space-y-2">
                    <textarea
                      autoFocus
                      value={draftBio}
                      onChange={e => setDraftBio(e.target.value)}
                      maxLength={200}
                      rows={3}
                      placeholder="A short bio about yourself…"
                      className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none"
                    />
                    <div className="flex gap-2">
                      <button
                        onClick={() => { setBio(draftBio); setEditBio(false); toast.success('Bio saved', 'Profile') }}
                        className="flex items-center gap-1 rounded-lg bg-brand-600 text-white px-3 py-1.5 text-xs font-medium hover:bg-brand-700"
                      >
                        <Save className="h-3 w-3" /> Save
                      </button>
                      <button
                        onClick={() => setEditBio(false)}
                        className="flex items-center gap-1 rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800"
                      >
                        <X className="h-3 w-3" /> Cancel
                      </button>
                    </div>
                  </div>
                ) : (
                  <p className="text-xs text-gray-500 dark:text-gray-400 leading-relaxed min-h-[2rem]">
                    {bio || <span className="italic text-gray-300 dark:text-gray-600">No bio set yet.</span>}
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* Account info */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">Account Details</h3>
            <div>
              <InfoRow icon={Mail}      label="Email"      value={user.email} />
              <InfoRow icon={Building2} label="Company ID" value={<span className="font-mono text-xs">{user.companyId}</span>} />
              <InfoRow icon={Shield}    label="Role"       value={user.role} />
              <InfoRow icon={Calendar}  label="Member since" value={format(new Date(user.createdAt), 'MMMM d, yyyy')} />
              <InfoRow
                icon={CheckCircle}
                label="Account status"
                value={
                  <span className={`inline-flex items-center gap-1 text-xs font-medium ${user.active ? 'text-green-600 dark:text-green-400' : 'text-red-500'}`}>
                    <span className={`h-1.5 w-1.5 rounded-full ${user.active ? 'bg-green-500' : 'bg-red-500'}`} />
                    {user.active ? 'Active' : 'Inactive'}
                  </span>
                }
              />
            </div>
          </div>

          {/* Activity stats */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
              <Activity className="h-4 w-4 text-gray-400" /> Session Stats
            </h3>
            <div className="grid grid-cols-2 gap-3">
              {[
                { label: 'Total Events', value: totalEvents, color: 'text-brand-600 dark:text-brand-400' },
                { label: 'Unread',       value: unread,      color: 'text-amber-600 dark:text-amber-400' },
                { label: 'Flagged',      value: flagged,     color: 'text-red-600 dark:text-red-400' },
                { label: 'Approved',     value: successes,   color: 'text-green-600 dark:text-green-400' },
              ].map(stat => (
                <div key={stat.label} className="rounded-lg bg-gray-50 dark:bg-gray-800 p-3 text-center">
                  <p className={`text-xl font-bold ${stat.color}`}>{stat.value}</p>
                  <p className="text-[10px] text-gray-400 mt-0.5">{stat.label}</p>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* ── Right: activity log ─────────────────────────────────────────── */}
        <div className="lg:col-span-2">
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 h-full flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100 dark:border-gray-700">
              <div className="flex items-center gap-2">
                <Bell className="h-4 w-4 text-gray-400" />
                <h3 className="text-sm font-semibold text-gray-900 dark:text-white">Activity Log</h3>
                {unread > 0 && (
                  <span className="rounded-full bg-brand-500 text-white text-[10px] font-bold px-1.5 py-0.5 min-w-[1.25rem] text-center">
                    {unread}
                  </span>
                )}
              </div>
              {notifs.length > 0 && (
                <button
                  onClick={clearAll}
                  className="flex items-center gap-1 text-xs text-red-400 hover:text-red-600 transition-colors"
                >
                  <Trash2 className="h-3 w-3" /> Clear all
                </button>
              )}
            </div>

            {/* Log entries */}
            <div className="flex-1 overflow-y-auto divide-y divide-gray-50 dark:divide-gray-800">
              {notifs.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-20 text-center">
                  <Clock className="h-10 w-10 text-gray-200 dark:text-gray-700 mb-3" />
                  <p className="text-sm font-medium text-gray-400">No activity yet</p>
                  <p className="text-xs text-gray-300 dark:text-gray-600 mt-1">
                    WebSocket events will appear here as they arrive.
                  </p>
                </div>
              ) : (
                notifs.map(n => (
                  <div
                    key={n.id}
                    onClick={() => markRead(n.id)}
                    className={`flex items-start gap-3 px-5 py-3.5 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors cursor-default ${!n.read ? 'bg-brand-50/30 dark:bg-brand-900/10' : ''}`}
                  >
                    {/* Severity icon ring */}
                    <div className={`shrink-0 h-8 w-8 rounded-full ring-1 flex items-center justify-center ${SEV_RING[n.severity]}`}>
                      {SEV_ICON[n.severity]}
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2">
                        <p className={`text-sm font-medium truncate ${!n.read ? 'text-gray-900 dark:text-white' : 'text-gray-700 dark:text-gray-300'}`}>
                          {n.title}
                        </p>
                        <span className="shrink-0 text-[10px] text-gray-400 whitespace-nowrap">
                          {formatDistanceToNow(new Date(n.ts), { addSuffix: true })}
                        </span>
                      </div>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 truncate">{n.body}</p>
                      {n.claimId && (
                        <a
                          href={`/claims/${n.claimId}`}
                          className="inline-flex items-center gap-1 text-[10px] text-brand-600 dark:text-brand-400 hover:underline mt-1"
                          onClick={e => e.stopPropagation()}
                        >
                          View claim <ChevronRight className="h-3 w-3" />
                        </a>
                      )}
                    </div>

                    {/* Unread dot */}
                    {!n.read && (
                      <div className="shrink-0 h-2 w-2 rounded-full bg-brand-500 mt-1.5" />
                    )}

                    {/* Dismiss */}
                    <button
                      onClick={e => { e.stopPropagation(); dismiss(n.id) }}
                      className="shrink-0 opacity-0 hover:opacity-100 group-hover:opacity-100 text-gray-300 dark:text-gray-600 hover:text-gray-500 dark:hover:text-gray-400 transition-opacity"
                      title="Dismiss"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))
              )}
            </div>

            {/* Footer */}
            {notifs.length > 0 && (
              <div className="px-5 py-3 border-t border-gray-100 dark:border-gray-700 text-xs text-gray-400">
                {notifs.length} event{notifs.length !== 1 ? 's' : ''} · Showing last session activity
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
