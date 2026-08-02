import { useAuthStore } from '@/store/auth'
import { Shield, User, Bell, Key } from 'lucide-react'

export default function SettingsPage() {
  const user = useAuthStore(s => s.user)
  return (
    <div className="max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>
      <div className="bg-white dark:bg-gray-900 rounded-xl border divide-y">
        <div className="p-5 flex items-center gap-4">
          <div className="h-12 w-12 rounded-full bg-brand-600 flex items-center justify-center text-white text-lg font-bold">
            {user?.firstName?.[0]}{user?.lastName?.[0]}
          </div>
          <div>
            <p className="font-semibold">{user?.firstName} {user?.lastName}</p>
            <p className="text-sm text-gray-500">{user?.email}</p>
            <span className="inline-flex rounded-full bg-brand-100 text-brand-700 px-2 py-0.5 text-xs font-medium mt-1">{user?.role}</span>
          </div>
        </div>
        {[
          { icon: User,  label: 'Profile', desc: 'Update your personal information' },
          { icon: Key,   label: 'Security', desc: 'Change password and 2FA settings' },
          { icon: Bell,  label: 'Notifications', desc: 'Email and Slack alert preferences' },
          { icon: Shield, label: 'API Keys', desc: 'Manage service API tokens' },
        ].map(({ icon: Icon, label, desc }) => (
          <button key={label} className="w-full flex items-center gap-4 p-5 hover:bg-gray-50 dark:hover:bg-gray-800/50 text-left transition-colors">
            <div className="rounded-lg bg-gray-100 dark:bg-gray-800 p-2.5"><Icon className="h-5 w-5 text-gray-600 dark:text-gray-300"/></div>
            <div><p className="font-medium">{label}</p><p className="text-sm text-gray-500">{desc}</p></div>
          </button>
        ))}
      </div>
    </div>
  )
}
