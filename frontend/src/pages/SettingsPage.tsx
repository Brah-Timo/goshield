import { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useAuthStore } from '@/store/auth'
import {
  Shield, User, Bell, Key, Moon, Sun, Copy,
  CheckCircle, Info, ExternalLink, Palette, Edit3, Save, X,
  Lock, Eye, EyeOff, AlertCircle,
} from 'lucide-react'
import { toast } from '@/store/toast'
import { useUpdateUser } from '@/hooks/useAuth'

const APP_VERSION = '3.0.0'

// ── Sub-components ────────────────────────────────────────────────────────────

function SectionCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
      <div className="px-5 py-3.5 border-b border-gray-100 dark:border-gray-800">
        <h2 className="font-semibold text-gray-800 dark:text-gray-200 text-sm">{title}</h2>
      </div>
      {children}
    </div>
  )
}

function ToggleRow({ label, sub, checked, onChange }: {
  label: string; sub?: string; checked: boolean; onChange: () => void
}) {
  return (
    <div className="flex items-center justify-between px-5 py-3.5 border-b border-gray-50 dark:border-gray-800 last:border-0">
      <div>
        <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{label}</p>
        {sub && <p className="text-xs text-gray-400 mt-0.5">{sub}</p>}
      </div>
      <button
        onClick={onChange}
        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-brand-500 focus:ring-offset-2 ${
          checked ? 'bg-brand-600' : 'bg-gray-200 dark:bg-gray-700'
        }`}
      >
        <span
          className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${
            checked ? 'translate-x-6' : 'translate-x-1'
          }`}
        />
      </button>
    </div>
  )
}

function StatPill({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className={`rounded-xl p-4 ${color}`}>
      <p className="text-2xl font-bold">{value}</p>
      <p className="text-xs mt-0.5 opacity-75">{label}</p>
    </div>
  )
}

// ── Dark-mode reader (same logic as Layout) ───────────────────────────────────
function getDark() {
  if (typeof window === 'undefined') return false
  const saved = localStorage.getItem('goshield-theme')
  return saved ? saved === 'dark' : window.matchMedia('(prefers-color-scheme: dark)').matches
}

// ── Profile edit form schema ──────────────────────────────────────────────────
const profileSchema = z.object({
  firstName: z.string().min(1, 'Required').max(50),
  lastName:  z.string().min(1, 'Required').max(50),
})
type ProfileForm = z.infer<typeof profileSchema>

// ── Password change schema ────────────────────────────────────────────────────
const passwordSchema = z.object({
  currentPassword: z.string().min(1, 'Current password required'),
  newPassword: z.string()
    .min(8, 'Minimum 8 characters')
    .regex(/[A-Z]/, 'Must contain an uppercase letter')
    .regex(/[0-9]/, 'Must contain a number'),
  confirmPassword: z.string(),
}).refine(d => d.newPassword === d.confirmPassword, {
  message: 'Passwords do not match',
  path: ['confirmPassword'],
})
type PasswordForm = z.infer<typeof passwordSchema>

// ══════════════════════════════════════════════════════════════════════════════
export default function SettingsPage() {
  const user         = useAuthStore(s => s.user)
  const updateUser   = useUpdateUser()

  const [dark, setDarkState]       = useState(getDark)
  const [notifEmail, setNotifEmail] = useState(true)
  const [notifSlack, setNotifSlack] = useState(false)
  const [notifWS,    setNotifWS]   = useState(true)
  const [copied,     setCopied]    = useState(false)
  const [editingProfile, setEditingProfile] = useState(false)
  const [showPwPanel, setShowPwPanel] = useState(false)
  const [showCurPw,  setShowCurPw]  = useState(false)
  const [showNewPw,  setShowNewPw]  = useState(false)
  const [showConPw,  setShowConPw]  = useState(false)

  // Sync with document when toggled from this page
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

  const initials = `${user?.firstName?.[0] ?? ''}${user?.lastName?.[0] ?? ''}`

  // ── Profile form ─────────────────────────────────────────────────────────
  const profileForm = useForm<ProfileForm>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      firstName: user?.firstName ?? '',
      lastName:  user?.lastName  ?? '',
    },
  })

  const onSaveProfile = (d: ProfileForm) => {
    if (!user?.id) return
    updateUser.mutate(
      { id: user.id, data: { firstName: d.firstName, lastName: d.lastName } },
      {
        onSuccess: () => {
          toast.success('Profile updated successfully', 'Saved')
          setEditingProfile(false)
        },
        onError: (err: unknown) => {
          const msg = err instanceof Error ? err.message : 'Update failed'
          toast.error(msg, 'Error')
        },
      }
    )
  }

  // ── Password form ─────────────────────────────────────────────────────────
  const pwForm = useForm<PasswordForm>({
    resolver: zodResolver(passwordSchema),
  })

  const onChangePassword = (_d: PasswordForm) => {
    // Backend does not yet expose a change-password endpoint.
    // When it does, call: api.post('/auth/v1/users/change-password', { currentPassword, newPassword })
    toast.info('Password change endpoint coming soon', 'Not implemented')
    setShowPwPanel(false)
    pwForm.reset()
  }

  const copyId = () => {
    if (!user?.id) return
    navigator.clipboard.writeText(user.id)
    setCopied(true)
    toast.success('User ID copied to clipboard')
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className="max-w-2xl space-y-6 animate-fade-in">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Settings</h1>
        <p className="text-sm text-gray-400 mt-0.5">Manage your account and application preferences</p>
      </div>

      {/* ── Profile card ───────────────────────────────────────────────────── */}
      <SectionCard title="Profile">
        {editingProfile ? (
          <form onSubmit={profileForm.handleSubmit(onSaveProfile)} className="p-5 space-y-4">
            <div className="flex items-center gap-4 mb-2">
              <div className="h-14 w-14 rounded-full bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center text-white text-xl font-bold shrink-0">
                {initials}
              </div>
              <div>
                <p className="text-sm font-semibold text-gray-900 dark:text-white">Edit Profile</p>
                <p className="text-xs text-gray-400">Update your display name</p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">First name</label>
                <input
                  {...profileForm.register('firstName')}
                  className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
                />
                {profileForm.formState.errors.firstName && (
                  <p className="mt-0.5 text-xs text-danger-500">{profileForm.formState.errors.firstName.message}</p>
                )}
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Last name</label>
                <input
                  {...profileForm.register('lastName')}
                  className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
                />
                {profileForm.formState.errors.lastName && (
                  <p className="mt-0.5 text-xs text-danger-500">{profileForm.formState.errors.lastName.message}</p>
                )}
              </div>
            </div>
            {/* Read-only fields */}
            <div>
              <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Email</label>
              <input
                value={user?.email ?? ''}
                disabled
                className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-3 py-2 text-sm text-gray-400 cursor-not-allowed"
              />
              <p className="mt-0.5 text-xs text-gray-400">Email cannot be changed</p>
            </div>
            <div className="flex items-center gap-2 pt-1">
              <button
                type="submit"
                disabled={updateUser.isPending}
                className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-700 disabled:opacity-50 transition-colors"
              >
                <Save className="h-3.5 w-3.5" />
                {updateUser.isPending ? 'Saving…' : 'Save changes'}
              </button>
              <button
                type="button"
                onClick={() => { setEditingProfile(false); profileForm.reset() }}
                className="flex items-center gap-1.5 rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-2 text-xs font-medium text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
              >
                <X className="h-3.5 w-3.5" />
                Cancel
              </button>
            </div>
          </form>
        ) : (
          <>
            <div className="p-5 flex items-center gap-5">
              <div className="h-16 w-16 rounded-full bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center text-white text-2xl font-bold shrink-0 shadow-lg">
                {initials}
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-lg font-semibold text-gray-900 dark:text-white">
                  {user?.firstName} {user?.lastName}
                </p>
                <p className="text-sm text-gray-500 dark:text-gray-400">{user?.email}</p>
                <div className="flex items-center gap-2 mt-1.5">
                  <span className="inline-flex rounded-full bg-brand-100 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 px-2.5 py-0.5 text-xs font-semibold">
                    {user?.role}
                  </span>
                  {user?.active && (
                    <span className="inline-flex items-center gap-1 rounded-full bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300 px-2.5 py-0.5 text-xs font-medium">
                      <span className="h-1.5 w-1.5 rounded-full bg-green-500" />
                      Active
                    </span>
                  )}
                </div>
              </div>
              <button
                onClick={() => setEditingProfile(true)}
                className="flex items-center gap-1.5 rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors shrink-0"
              >
                <Edit3 className="h-3.5 w-3.5" />
                Edit
              </button>
            </div>
            <div className="border-t border-gray-50 dark:border-gray-800 px-5 py-3 flex items-center justify-between text-xs text-gray-400">
              <div className="flex items-center gap-1.5 font-mono truncate">
                <span>ID:</span>
                <span className="text-gray-500 dark:text-gray-300">{user?.id?.slice(0, 20)}…</span>
              </div>
              <button onClick={copyId} className="flex items-center gap-1 hover:text-gray-600 dark:hover:text-gray-200 transition-colors">
                {copied ? <CheckCircle className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
                Copy
              </button>
            </div>
          </>
        )}
      </SectionCard>

      {/* ── Account stats ──────────────────────────────────────────────────── */}
      <SectionCard title="Account Activity">
        <div className="grid grid-cols-3 gap-3 p-5">
          <StatPill
            label="Account role"
            value={user?.role ?? '—'}
            color="bg-brand-50 dark:bg-brand-900/20 text-brand-700 dark:text-brand-300"
          />
          <StatPill
            label="Company ID"
            value={user?.companyId?.slice(0, 6) ?? '—'}
            color="bg-gray-50 dark:bg-gray-800 text-gray-700 dark:text-gray-300"
          />
          <StatPill
            label="Member since"
            value={user?.createdAt ? new Date(user.createdAt).getFullYear().toString() : '—'}
            color="bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300"
          />
        </div>
      </SectionCard>

      {/* ── Appearance ─────────────────────────────────────────────────────── */}
      <SectionCard title="Appearance">
        <ToggleRow
          label="Dark Mode"
          sub="Switch between light and dark interface theme"
          checked={dark}
          onChange={() => setDarkState(d => !d)}
        />
        <div className="px-5 py-3.5 flex items-center gap-3">
          <Palette className="h-5 w-5 text-gray-400" />
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-800 dark:text-gray-200">Accent Color</p>
            <p className="text-xs text-gray-400">Brand blue (default)</p>
          </div>
          <span className="h-5 w-5 rounded-full bg-brand-600 border-2 border-white dark:border-gray-900 shadow-sm" />
        </div>
      </SectionCard>

      {/* ── Notifications ─────────────────────────────────────────────────── */}
      <SectionCard title="Notifications">
        <ToggleRow
          label="Email Alerts"
          sub="Receive fraud alerts and status changes by email"
          checked={notifEmail}
          onChange={() => { setNotifEmail(v => !v); toast.info('Email preference saved') }}
        />
        <ToggleRow
          label="Slack Notifications"
          sub="Send alerts to your connected Slack workspace"
          checked={notifSlack}
          onChange={() => { setNotifSlack(v => !v); toast.info('Slack preference saved') }}
        />
        <ToggleRow
          label="In-App (WebSocket)"
          sub="Real-time toast notifications for claim updates"
          checked={notifWS}
          onChange={() => { setNotifWS(v => !v); toast.info('In-app notification preference saved') }}
        />
      </SectionCard>

      {/* ── Security & Access ─────────────────────────────────────────────── */}
      <SectionCard title="Security & Access">
        {/* Change password panel */}
        <div className="border-b border-gray-50 dark:border-gray-800 last:border-0">
          <button
            onClick={() => setShowPwPanel(v => !v)}
            className="w-full flex items-center gap-4 px-5 py-4 hover:bg-gray-50 dark:hover:bg-gray-800/50 text-left transition-colors"
          >
            <div className="rounded-lg bg-gray-100 dark:bg-gray-800 p-2.5 shrink-0">
              <Lock className="h-4 w-4 text-gray-600 dark:text-gray-300" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-800 dark:text-gray-200">Change Password</p>
              <p className="text-xs text-gray-400 mt-0.5">Update your account password</p>
            </div>
            {showPwPanel
              ? <X className="h-4 w-4 text-gray-400 shrink-0" />
              : <ExternalLink className="h-4 w-4 text-gray-300 dark:text-gray-600 shrink-0" />
            }
          </button>

          {showPwPanel && (
            <form
              onSubmit={pwForm.handleSubmit(onChangePassword)}
              className="px-5 pb-5 space-y-3 border-t border-gray-50 dark:border-gray-800"
            >
              <div className="pt-3">
                {/* Current password */}
                <div className="space-y-3">
                  <div>
                    <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Current password</label>
                    <div className="relative">
                      <input
                        {...pwForm.register('currentPassword')}
                        type={showCurPw ? 'text' : 'password'}
                        autoComplete="current-password"
                        className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 pr-9 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
                      />
                      <button type="button" onClick={() => setShowCurPw(v => !v)} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400">
                        {showCurPw ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                      </button>
                    </div>
                    {pwForm.formState.errors.currentPassword && (
                      <p className="mt-0.5 text-xs text-danger-500">{pwForm.formState.errors.currentPassword.message}</p>
                    )}
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">New password</label>
                    <div className="relative">
                      <input
                        {...pwForm.register('newPassword')}
                        type={showNewPw ? 'text' : 'password'}
                        autoComplete="new-password"
                        className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 pr-9 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
                      />
                      <button type="button" onClick={() => setShowNewPw(v => !v)} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400">
                        {showNewPw ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                      </button>
                    </div>
                    {pwForm.formState.errors.newPassword && (
                      <p className="mt-0.5 text-xs text-danger-500">{pwForm.formState.errors.newPassword.message}</p>
                    )}
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Confirm new password</label>
                    <div className="relative">
                      <input
                        {...pwForm.register('confirmPassword')}
                        type={showConPw ? 'text' : 'password'}
                        autoComplete="new-password"
                        className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 pr-9 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
                      />
                      <button type="button" onClick={() => setShowConPw(v => !v)} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400">
                        {showConPw ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                      </button>
                    </div>
                    {pwForm.formState.errors.confirmPassword && (
                      <p className="mt-0.5 text-xs text-danger-500">{pwForm.formState.errors.confirmPassword.message}</p>
                    )}
                  </div>
                </div>
                <div className="flex gap-2 pt-3">
                  <button
                    type="submit"
                    className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-brand-700 transition-colors"
                  >
                    <Key className="h-3.5 w-3.5" />
                    Update password
                  </button>
                  <button
                    type="button"
                    onClick={() => { setShowPwPanel(false); pwForm.reset() }}
                    className="rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </form>
          )}
        </div>

        {/* Other security items */}
        {[
          { icon: Shield, label: 'Two-Factor Auth',  desc: 'Enable 2FA for additional security',    action: () => toast.info('2FA setup coming soon') },
          { icon: User,   label: 'Active Sessions',  desc: 'View and revoke active login sessions', action: () => toast.info('Session management coming soon') },
        ].map(({ icon: Icon, label, desc, action }) => (
          <button
            key={label}
            onClick={action}
            className="w-full flex items-center gap-4 px-5 py-4 hover:bg-gray-50 dark:hover:bg-gray-800/50 text-left transition-colors border-b border-gray-50 dark:border-gray-800 last:border-0"
          >
            <div className="rounded-lg bg-gray-100 dark:bg-gray-800 p-2.5 shrink-0">
              <Icon className="h-4 w-4 text-gray-600 dark:text-gray-300" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{label}</p>
              <p className="text-xs text-gray-400 mt-0.5">{desc}</p>
            </div>
            <ExternalLink className="h-4 w-4 text-gray-300 dark:text-gray-600 shrink-0" />
          </button>
        ))}
      </SectionCard>

      {/* ── Danger zone ───────────────────────────────────────────────────── */}
      <SectionCard title="Danger Zone">
        <div className="p-5">
          <div className="flex items-start gap-3 rounded-lg bg-danger-50 dark:bg-danger-900/20 border border-danger-200 dark:border-danger-800/30 p-4">
            <AlertCircle className="h-5 w-5 text-danger-500 shrink-0 mt-0.5" />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold text-danger-700 dark:text-danger-300">Delete account</p>
              <p className="text-xs text-danger-600 dark:text-danger-400 mt-0.5">
                Permanently delete your account and all associated data. This action cannot be undone.
              </p>
              <button
                onClick={() => toast.warning('Account deletion requires admin approval', 'Action required')}
                className="mt-3 rounded-lg bg-danger-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-danger-700 transition-colors"
              >
                Request account deletion
              </button>
            </div>
          </div>
        </div>
      </SectionCard>

      {/* ── App info ──────────────────────────────────────────────────────── */}
      <div className="text-center text-xs text-gray-400 space-y-1 pb-4">
        <div className="flex items-center justify-center gap-2">
          <Shield className="h-4 w-4 text-brand-400" />
          <span className="font-semibold text-gray-600 dark:text-gray-400">GoShield</span>
        </div>
        <p>Version {APP_VERSION} · Fraud Detection Platform</p>
        <p className="flex items-center justify-center gap-1">
          <Info className="h-3 w-3" />
          Built with Go, React, OpenTelemetry
        </p>
      </div>
    </div>
  )
}
