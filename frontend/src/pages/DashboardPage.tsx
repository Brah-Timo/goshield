import { useClaimList, useClaimStats } from '@/hooks/useClaims'
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell, Legend, BarChart, Bar,
} from 'recharts'
import {
  AlertTriangle, CheckCircle, Clock, TrendingUp, TrendingDown,
  DollarSign, Shield, FileText, Activity,
} from 'lucide-react'
import { format, subDays, parseISO } from 'date-fns'
import { Link } from 'react-router-dom'
import type { DashboardStats, Claim, ClaimStatus, RiskLevel } from '@/types'

// ── Palette ────────────────────────────────────────────────────────────────
const RISK_COLORS: Record<RiskLevel, string>   = {
  LOW: '#22c55e', MEDIUM: '#f59e0b', HIGH: '#ef4444', CRITICAL: '#7c3aed',
}
const STATUS_COLORS: Record<ClaimStatus, string> = {
  PENDING:    '#f59e0b',
  PROCESSING: '#0ea5e9',
  FLAGGED:    '#ef4444',
  APPROVED:   '#22c55e',
  REJECTED:   '#6b7280',
  MORE_INFO:  '#8b5cf6',
}
const PIE_PALETTE = ['#0ea5e9','#22c55e','#f59e0b','#ef4444','#7c3aed','#64748b']

// ── KPI card ──────────────────────────────────────────────────────────────
function StatCard({
  title, value, sub, icon: Icon, color, trend, trendLabel,
}: {
  title: string
  value: string | number
  sub?: string
  icon: React.ElementType
  color: string
  trend?: 'up' | 'down' | 'neutral'
  trendLabel?: string
}) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5 flex items-start gap-4 hover:shadow-md transition-shadow">
      <div className={`rounded-xl p-2.5 ${color} shrink-0`}>
        <Icon className="h-5 w-5 text-white" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">{title}</p>
        <p className="text-2xl font-bold mt-0.5 text-gray-900 dark:text-white">{value}</p>
        {(sub || trendLabel) && (
          <div className="flex items-center gap-1.5 mt-0.5">
            {trend && trend !== 'neutral' && (
              trend === 'up'
                ? <TrendingUp className="h-3 w-3 text-green-500" />
                : <TrendingDown className="h-3 w-3 text-red-500" />
            )}
            <p className={`text-xs ${
              trend === 'up'   ? 'text-green-600 dark:text-green-400' :
              trend === 'down' ? 'text-red-600 dark:text-red-400' :
              'text-gray-400'
            }`}>{trendLabel ?? sub}</p>
          </div>
        )}
      </div>
    </div>
  )
}

// ── Status badge ──────────────────────────────────────────────────────────
function StatusBadge({ status }: { status: ClaimStatus }) {
  const map: Record<ClaimStatus, string> = {
    FLAGGED:    'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
    APPROVED:   'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
    REJECTED:   'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-300',
    PENDING:    'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
    PROCESSING: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
    MORE_INFO:  'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
  }
  return (
    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${map[status] ?? 'bg-gray-100 text-gray-600'}`}>
      {status}
    </span>
  )
}

// ── Build area-chart data from claims list ────────────────────────────────
function buildTrendData(claims: Claim[]) {
  const buckets: Record<string, { date: string; total: number; flagged: number; approved: number }> = {}
  for (let i = 13; i >= 0; i--) {
    const d = format(subDays(new Date(), i), 'MMM d')
    buckets[d] = { date: d, total: 0, flagged: 0, approved: 0 }
  }
  for (const c of claims) {
    const d = format(parseISO(c.createdAt), 'MMM d')
    if (buckets[d]) {
      buckets[d].total++
      if (c.status === 'FLAGGED') buckets[d].flagged++
      if (c.status === 'APPROVED') buckets[d].approved++
    }
  }
  return Object.values(buckets)
}

// ── Build pie data from claims ─────────────────────────────────────────────
function buildTypeData(claims: Claim[]) {
  const counts: Record<string, number> = {}
  for (const c of claims) counts[c.claimType] = (counts[c.claimType] ?? 0) + 1
  return Object.entries(counts).map(([name, value]) => ({ name, value }))
}

function buildRiskData(claims: Claim[]) {
  const counts: Record<string, number> = { LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0 }
  for (const c of claims) if (c.riskLevel) counts[c.riskLevel]++
  return Object.entries(counts)
    .filter(([, v]) => v > 0)
    .map(([name, value]) => ({ name, value, fill: RISK_COLORS[name as RiskLevel] }))
}

// ── Custom tooltip ─────────────────────────────────────────────────────────
function ChartTooltip({ active, payload, label }: {
  active?: boolean; payload?: { name: string; value: number; color: string }[]; label?: string
}) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg border bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 p-3 shadow-lg text-xs">
      <p className="font-semibold mb-1.5 text-gray-700 dark:text-gray-300">{label}</p>
      {payload.map(p => (
        <div key={p.name} className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: p.color }} />
          <span className="text-gray-500 dark:text-gray-400">{p.name}:</span>
          <span className="font-medium text-gray-800 dark:text-gray-200">{p.value}</span>
        </div>
      ))}
    </div>
  )
}

// ══════════════════════════════════════════════════════════════════════════
export default function DashboardPage() {
  const statsQuery  = useClaimStats()
  const recentQuery = useClaimList({ pageSize: 100, sortBy: 'created_at', sortOrder: 'desc' })
  const top5Query   = useClaimList({ pageSize: 5,   sortBy: 'created_at', sortOrder: 'desc' })

  const s      = statsQuery.data as DashboardStats | undefined
  const claims = recentQuery.data?.claims ?? []
  const recent = top5Query.data?.claims  ?? []

  const trendData = buildTrendData(claims)
  const typeData  = buildTypeData(claims)
  const riskData  = buildRiskData(claims)

  // Compute simple trend: compare last 7d vs prior 7d
  const last7  = claims.filter(c => parseISO(c.createdAt) >= subDays(new Date(), 7)).length
  const prior7 = claims.filter(c => {
    const d = parseISO(c.createdAt)
    return d >= subDays(new Date(), 14) && d < subDays(new Date(), 7)
  }).length
  const volumeTrend = prior7 === 0 ? 'neutral' : last7 >= prior7 ? 'up' : 'down'
  const volumeDiff  = prior7 === 0 ? '—' : `${last7 >= prior7 ? '+' : ''}${last7 - prior7} vs prev week`

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
          <p className="text-gray-500 dark:text-gray-400 text-sm mt-0.5">
            Fraud detection overview — {format(new Date(), 'MMMM d, yyyy')}
          </p>
        </div>
        <Link
          to="/upload"
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 transition-colors"
        >
          <FileText className="h-4 w-4" />
          New Claim
        </Link>
      </div>

      {/* KPI cards */}
      <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
        <StatCard
          title="Total Claims"
          value={s?.totalClaims ?? '—'}
          icon={Shield}
          color="bg-brand-600"
          trend={volumeTrend}
          trendLabel={volumeDiff}
        />
        <StatCard
          title="Pending"
          value={s?.pendingClaims ?? '—'}
          sub="awaiting analysis"
          icon={Clock}
          color="bg-amber-500"
        />
        <StatCard
          title="Flagged"
          value={s?.flaggedClaims ?? '—'}
          sub="require review"
          icon={AlertTriangle}
          color="bg-red-500"
          trend={s && s.flaggedClaims > 0 ? 'down' : 'neutral'}
          trendLabel={s ? `${((s.flaggedClaims / Math.max(s.totalClaims, 1)) * 100).toFixed(1)}% of total` : undefined}
        />
        <StatCard
          title="Approved"
          value={s?.approvedClaims ?? '—'}
          icon={CheckCircle}
          color="bg-green-500"
          trend="up"
          trendLabel={s ? `${((s.approvedClaims / Math.max(s.totalClaims, 1)) * 100).toFixed(1)}% approval` : undefined}
        />
        <StatCard
          title="Fraud Rate"
          value={s ? `${(s.fraudRate * 100).toFixed(1)}%` : '—'}
          icon={TrendingUp}
          color="bg-purple-600"
          trend={s && s.fraudRate < 0.05 ? 'up' : 'down'}
          trendLabel={s && s.fraudRate < 0.05 ? 'within target' : 'above threshold'}
        />
        <StatCard
          title="Total Exposure"
          value={s ? `$${(s.totalAmount / 1_000).toFixed(0)}K` : '—'}
          icon={DollarSign}
          color="bg-slate-600"
          trendLabel={s ? `avg $${(s.totalAmount / Math.max(s.totalClaims, 1)).toFixed(0)}/claim` : undefined}
        />
      </div>

      {/* Charts row */}
      <div className="grid lg:grid-cols-3 gap-6">
        {/* Area chart — 14-day trend */}
        <div className="lg:col-span-2 bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="font-semibold text-gray-900 dark:text-white">Claims Volume (14 days)</h2>
              <p className="text-xs text-gray-400 mt-0.5">Daily submitted, flagged, and approved claims</p>
            </div>
            <Activity className="h-4 w-4 text-gray-400" />
          </div>
          <ResponsiveContainer width="100%" height={220}>
            <AreaChart data={trendData} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
              <defs>
                <linearGradient id="gradTotal" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#0ea5e9" stopOpacity={0.25}/>
                  <stop offset="95%" stopColor="#0ea5e9" stopOpacity={0}/>
                </linearGradient>
                <linearGradient id="gradFlagged" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#ef4444" stopOpacity={0.25}/>
                  <stop offset="95%" stopColor="#ef4444" stopOpacity={0}/>
                </linearGradient>
                <linearGradient id="gradApproved" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#22c55e" stopOpacity={0.25}/>
                  <stop offset="95%" stopColor="#22c55e" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" className="dark:[stroke:#374151]" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} tickLine={false} axisLine={false} />
              <YAxis tick={{ fontSize: 11 }} tickLine={false} axisLine={false} />
              <Tooltip content={<ChartTooltip />} />
              <Legend wrapperStyle={{ fontSize: 12, paddingTop: 8 }} />
              <Area type="monotone" dataKey="total"    name="Total"    stroke="#0ea5e9" fill="url(#gradTotal)"    strokeWidth={2} dot={false} />
              <Area type="monotone" dataKey="flagged"  name="Flagged"  stroke="#ef4444" fill="url(#gradFlagged)"  strokeWidth={2} dot={false} />
              <Area type="monotone" dataKey="approved" name="Approved" stroke="#22c55e" fill="url(#gradApproved)" strokeWidth={2} dot={false} />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Pie + bar col */}
        <div className="space-y-4">
          {/* Claim type distribution */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
            <h2 className="font-semibold text-gray-900 dark:text-white text-sm mb-3">By Claim Type</h2>
            {typeData.length > 0 ? (
              <ResponsiveContainer width="100%" height={140}>
                <PieChart>
                  <Pie data={typeData} cx="50%" cy="50%" outerRadius={55} dataKey="value" nameKey="name" label={({ name, percent }) => `${name} ${(percent*100).toFixed(0)}%`} labelLine={false} style={{ fontSize: 10 }}>
                    {typeData.map((_, i) => <Cell key={i} fill={PIE_PALETTE[i % PIE_PALETTE.length]} />)}
                  </Pie>
                  <Tooltip formatter={(v: number, name: string) => [v, name]} />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-32 flex items-center justify-center text-sm text-gray-400">No data yet</div>
            )}
          </div>

          {/* Risk level bar chart */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
            <h2 className="font-semibold text-gray-900 dark:text-white text-sm mb-3">Risk Distribution</h2>
            {riskData.length > 0 ? (
              <ResponsiveContainer width="100%" height={110}>
                <BarChart data={riskData} margin={{ top: 0, right: 0, bottom: 0, left: -25 }}>
                  <XAxis dataKey="name" tick={{ fontSize: 10 }} tickLine={false} axisLine={false} />
                  <YAxis tick={{ fontSize: 10 }} tickLine={false} axisLine={false} />
                  <Tooltip formatter={(v: number) => [v, 'Claims']} />
                  <Bar dataKey="value" radius={[4, 4, 0, 0]}>
                    {riskData.map((d, i) => <Cell key={i} fill={d.fill} />)}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-24 flex items-center justify-center text-sm text-gray-400">No risk data</div>
            )}
          </div>
        </div>
      </div>

      {/* Recent claims table */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between">
          <h2 className="font-semibold text-gray-900 dark:text-white">Recent Claims</h2>
          <Link to="/claims" className="text-xs text-brand-600 dark:text-brand-400 hover:underline font-medium">
            View all →
          </Link>
        </div>
        {top5Query.isLoading ? (
          <div className="divide-y">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="px-4 py-3 flex gap-4 animate-pulse">
                <div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
                <div className="h-4 w-16 bg-gray-100 dark:bg-gray-800 rounded" />
                <div className="h-4 w-16 bg-gray-100 dark:bg-gray-800 rounded" />
                <div className="h-4 w-12 bg-gray-100 dark:bg-gray-800 rounded ml-auto" />
              </div>
            ))}
          </div>
        ) : recent.length === 0 ? (
          <div className="py-12 text-center text-gray-400 text-sm">No claims yet. Submit your first claim.</div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800/60">
              <tr>
                {['ID', 'Type', 'Status', 'Amount', 'Risk', 'Date', ''].map(h => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700/50">
              {recent.map(c => (
                <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/40 transition-colors">
                  <td className="px-4 py-3 font-mono text-xs text-gray-400">{c.id.slice(0, 8)}…</td>
                  <td className="px-4 py-3 text-gray-700 dark:text-gray-300">{c.claimType}</td>
                  <td className="px-4 py-3"><StatusBadge status={c.status} /></td>
                  <td className="px-4 py-3 font-medium text-gray-800 dark:text-gray-200">
                    ${c.amount.toLocaleString()}
                  </td>
                  <td className="px-4 py-3">
                    {c.riskLevel ? (
                      <span className="font-semibold text-xs" style={{ color: RISK_COLORS[c.riskLevel] }}>
                        {c.riskLevel}
                      </span>
                    ) : <span className="text-gray-300">—</span>}
                  </td>
                  <td className="px-4 py-3 text-gray-400 text-xs">
                    {format(parseISO(c.createdAt), 'MMM d, HH:mm')}
                  </td>
                  <td className="px-4 py-3">
                    <Link
                      to={`/claims/${c.id}`}
                      className="text-brand-600 dark:text-brand-400 hover:underline text-xs font-medium"
                    >
                      View →
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
