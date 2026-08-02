import { useClaimList, useClaimStats } from '@/hooks/useClaims'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'
import { AlertTriangle, CheckCircle, Clock, TrendingUp, DollarSign, Shield } from 'lucide-react'
import { format } from 'date-fns'

const RISK_COLORS: Record<string, string> = {
  LOW: '#22c55e', MEDIUM: '#f59e0b', HIGH: '#ef4444', CRITICAL: '#7c3aed',
}

function StatCard({ title, value, sub, icon: Icon, color }: {
  title: string; value: string | number; sub?: string
  icon: React.ElementType; color: string
}) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border p-5 flex items-start gap-4">
      <div className={`rounded-lg p-2.5 ${color}`}><Icon className="h-5 w-5 text-white" /></div>
      <div>
        <p className="text-sm text-gray-500 dark:text-gray-400">{title}</p>
        <p className="text-2xl font-bold mt-0.5">{value}</p>
        {sub && <p className="text-xs text-gray-400 mt-0.5">{sub}</p>}
      </div>
    </div>
  )
}

export default function DashboardPage() {
  const stats = useClaimStats()
  const recent = useClaimList({ pageSize: 5, sortBy: 'created_at', sortOrder: 'desc' })

  const s = stats.data as Record<string, number> | undefined

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-gray-500 text-sm mt-1">Fraud detection overview — {format(new Date(), 'MMMM d, yyyy')}</p>
      </div>

      {/* KPI cards */}
      <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
        <StatCard title="Total Claims"   value={s?.totalClaims ?? '—'}   icon={Shield}       color="bg-brand-600" />
        <StatCard title="Pending"        value={s?.pendingClaims ?? '—'} icon={Clock}        color="bg-amber-500" />
        <StatCard title="Flagged"        value={s?.flaggedClaims ?? '—'} icon={AlertTriangle} color="bg-red-500" />
        <StatCard title="Approved"       value={s?.approvedClaims ?? '—'} icon={CheckCircle}  color="bg-green-500" />
        <StatCard title="Fraud Rate"     value={s ? `${(s.fraudRate * 100).toFixed(1)}%` : '—'} icon={TrendingUp} color="bg-purple-600" />
        <StatCard title="Total Exposure" value={s ? `$${(s.totalAmount / 1000).toFixed(0)}K` : '—'} icon={DollarSign} color="bg-slate-600" />
      </div>

      {/* Recent claims table */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border overflow-hidden">
        <div className="px-5 py-4 border-b"><h2 className="font-semibold">Recent Claims</h2></div>
        {recent.isLoading ? (
          <div className="p-8 text-center text-gray-400">Loading…</div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>{['ID','Type','Status','Amount','Risk','Date'].map(h => (
                <th key={h} className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{h}</th>
              ))}</tr>
            </thead>
            <tbody className="divide-y">
              {recent.data?.claims.map(c => (
                <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                  <td className="px-4 py-3 font-mono text-xs text-gray-500">{c.id.slice(0,8)}…</td>
                  <td className="px-4 py-3">{c.claimType}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                      c.status === 'FLAGGED' ? 'bg-red-100 text-red-700' :
                      c.status === 'APPROVED' ? 'bg-green-100 text-green-700' :
                      'bg-gray-100 text-gray-600'
                    }`}>{c.status}</span>
                  </td>
                  <td className="px-4 py-3">${c.amount.toLocaleString()}</td>
                  <td className="px-4 py-3">
                    {c.riskLevel && (
                      <span style={{ color: RISK_COLORS[c.riskLevel] }} className="font-medium">{c.riskLevel}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gray-500">{format(new Date(c.createdAt), 'MMM d')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
