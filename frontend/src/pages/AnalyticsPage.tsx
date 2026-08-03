import { useState, useMemo } from 'react'
import {
  AreaChart, Area, BarChart, Bar, PieChart, Pie, Cell, LineChart, Line,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from 'recharts'
import { useClaimList, useClaimStats } from '@/hooks/useClaims'
import { TrendingUp, TrendingDown, Minus, BarChart2, AlertTriangle, CheckCircle, Clock } from 'lucide-react'
import type { Claim, ClaimStatus, ClaimType, RiskLevel } from '@/types'

// ── Color palettes ────────────────────────────────────────────────────────────
const STATUS_COLORS: Record<ClaimStatus, string> = {
  PENDING:    '#94a3b8',
  PROCESSING: '#60a5fa',
  FLAGGED:    '#f97316',
  APPROVED:   '#22c55e',
  REJECTED:   '#ef4444',
  MORE_INFO:  '#a78bfa',
}

const RISK_COLORS: Record<RiskLevel, string> = {
  LOW:      '#22c55e',
  MEDIUM:   '#eab308',
  HIGH:     '#f97316',
  CRITICAL: '#ef4444',
}

const TYPE_COLORS = ['#3b82f6', '#8b5cf6', '#10b981', '#f59e0b', '#ec4899', '#6b7280']

const DAYS_OPTIONS = [7, 14, 30, 60, 90]

// ── Helpers ───────────────────────────────────────────────────────────────────
function formatCurrency(n: number) {
  if (n >= 1_000_000) return `$${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000)     return `$${(n / 1_000).toFixed(1)}K`
  return `$${n.toFixed(0)}`
}

function formatDate(iso: string, short = false) {
  const d = new Date(iso)
  if (short) return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: '2-digit' })
}

function buildTrend(claims: Claim[], days: number) {
  const now   = Date.now()
  const slots: Record<string, { date: string; total: number; flagged: number; approved: number; rejected: number; amount: number }> = {}

  for (let i = days - 1; i >= 0; i--) {
    const d    = new Date(now - i * 86_400_000)
    const key  = d.toISOString().slice(0, 10)
    const label = formatDate(d.toISOString(), true)
    slots[key] = { date: label, total: 0, flagged: 0, approved: 0, rejected: 0, amount: 0 }
  }

  for (const c of claims) {
    const key = c.createdAt.slice(0, 10)
    if (slots[key]) {
      slots[key].total++
      slots[key].amount += c.amount
      if (c.status === 'FLAGGED')   slots[key].flagged++
      if (c.status === 'APPROVED')  slots[key].approved++
      if (c.status === 'REJECTED')  slots[key].rejected++
    }
  }

  return Object.values(slots)
}

function buildStatusData(claims: Claim[]) {
  const counts: Record<string, number> = {}
  for (const c of claims) {
    counts[c.status] = (counts[c.status] ?? 0) + 1
  }
  return Object.entries(counts).map(([status, value]) => ({
    name: status,
    value,
    color: STATUS_COLORS[status as ClaimStatus] ?? '#94a3b8',
  }))
}

function buildRiskData(claims: Claim[]) {
  const counts: Record<string, number> = {}
  for (const c of claims) {
    if (c.riskLevel) counts[c.riskLevel] = (counts[c.riskLevel] ?? 0) + 1
  }
  return (['LOW', 'MEDIUM', 'HIGH', 'CRITICAL'] as RiskLevel[])
    .filter(r => counts[r])
    .map(r => ({ name: r, value: counts[r], color: RISK_COLORS[r] }))
}

function buildTypeData(claims: Claim[]) {
  const counts: Record<string, { count: number; amount: number }> = {}
  for (const c of claims) {
    if (!counts[c.claimType]) counts[c.claimType] = { count: 0, amount: 0 }
    counts[c.claimType].count++
    counts[c.claimType].amount += c.amount
  }
  return Object.entries(counts).map(([type, d]) => ({
    name: type,
    count: d.count,
    amount: d.amount,
  }))
}

function buildFraudScoreData(claims: Claim[]) {
  const buckets = [
    { range: '0-20',  min: 0,    max: 0.2,  count: 0 },
    { range: '20-40', min: 0.2,  max: 0.4,  count: 0 },
    { range: '40-60', min: 0.4,  max: 0.6,  count: 0 },
    { range: '60-80', min: 0.6,  max: 0.8,  count: 0 },
    { range: '80+',   min: 0.8,  max: 1.01, count: 0 },
  ]
  for (const c of claims) {
    if (c.fraudScore == null) continue
    const b = buckets.find(b => c.fraudScore! >= b.min && c.fraudScore! < b.max)
    if (b) b.count++
  }
  return buckets
}

// ── Custom tooltip ────────────────────────────────────────────────────────────
function ChartTooltip({ active, payload, label }: any) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-xl p-3 text-xs space-y-1">
      <p className="font-semibold text-gray-700 dark:text-gray-200 mb-1">{label}</p>
      {payload.map((p: any) => (
        <div key={p.name} className="flex items-center gap-2">
          <span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ background: p.color }} />
          <span className="text-gray-500 dark:text-gray-400">{p.name}:</span>
          <span className="font-medium text-gray-700 dark:text-gray-200">{p.value}</span>
        </div>
      ))}
    </div>
  )
}

// ── KPI card ──────────────────────────────────────────────────────────────────
function KpiCard({
  label, value, sub, icon: Icon, color, trend, trendLabel,
}: {
  label: string
  value: string | number
  sub?: string
  icon: React.ElementType
  color: string
  trend?: 'up' | 'down' | 'neutral'
  trendLabel?: string
}) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
      <div className="flex items-start justify-between">
        <div className={`rounded-lg p-2.5 ${color}`}>
          <Icon className="h-5 w-5 text-white" />
        </div>
        {trend && trendLabel && (
          <span className={`flex items-center gap-0.5 text-xs font-medium px-2 py-1 rounded-full ${
            trend === 'up'      ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' :
            trend === 'down'    ? 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400' :
                                  'bg-gray-100 dark:bg-gray-800 text-gray-500'
          }`}>
            {trend === 'up' && <TrendingUp className="h-3 w-3" />}
            {trend === 'down' && <TrendingDown className="h-3 w-3" />}
            {trend === 'neutral' && <Minus className="h-3 w-3" />}
            {trendLabel}
          </span>
        )}
      </div>
      <div className="mt-4">
        <p className="text-2xl font-bold text-gray-900 dark:text-white">{value}</p>
        <p className="text-sm font-medium text-gray-600 dark:text-gray-400 mt-0.5">{label}</p>
        {sub && <p className="text-xs text-gray-400 mt-1">{sub}</p>}
      </div>
    </div>
  )
}

// ── Section wrapper ───────────────────────────────────────────────────────────
function ChartCard({ title, sub, children }: { title: string; sub?: string; children: React.ReactNode }) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
      <div className="mb-4">
        <h3 className="font-semibold text-gray-800 dark:text-gray-200 text-sm">{title}</h3>
        {sub && <p className="text-xs text-gray-400 mt-0.5">{sub}</p>}
      </div>
      {children}
    </div>
  )
}

// ══════════════════════════════════════════════════════════════════════════════
export default function AnalyticsPage() {
  const [days, setDays] = useState(30)

  // Fetch a large page of claims for client-side analytics
  const { data, isLoading } = useClaimList({ pageSize: 200, page: 1 })
  const { data: stats }     = useClaimStats()
  const claims = data?.claims ?? []

  const trendData    = useMemo(() => buildTrend(claims, days),      [claims, days])
  const statusData   = useMemo(() => buildStatusData(claims),        [claims])
  const riskData     = useMemo(() => buildRiskData(claims),          [claims])
  const typeData     = useMemo(() => buildTypeData(claims),          [claims])
  const fraudBuckets = useMemo(() => buildFraudScoreData(claims),    [claims])

  // Trend KPIs
  const half = Math.floor(days / 2)
  const recentHalf = claims.filter(c => {
    const d = Date.now() - new Date(c.createdAt).getTime()
    return d <= half * 86_400_000
  })
  const olderHalf = claims.filter(c => {
    const d = Date.now() - new Date(c.createdAt).getTime()
    return d > half * 86_400_000 && d <= days * 86_400_000
  })

  const fraudRate = claims.length
    ? ((claims.filter(c => c.status === 'FLAGGED' || c.status === 'REJECTED').length / claims.length) * 100).toFixed(1)
    : '0'

  const avgScore = claims.length
    ? (claims.reduce((s, c) => s + (c.fraudScore ?? 0), 0) / claims.filter(c => c.fraudScore != null).length * 100).toFixed(1)
    : '0'

  const totalAmount = claims.reduce((s, c) => s + c.amount, 0)

  const volumeTrend: 'up' | 'down' | 'neutral' =
    recentHalf.length > olderHalf.length ? 'up' :
    recentHalf.length < olderHalf.length ? 'down' : 'neutral'

  if (isLoading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-8 w-48 bg-gray-200 dark:bg-gray-800 rounded" />
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-28 bg-gray-200 dark:bg-gray-800 rounded-xl" />
          ))}
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-64 bg-gray-200 dark:bg-gray-800 rounded-xl" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
            <BarChart2 className="h-6 w-6 text-brand-500" />
            Analytics
          </h1>
          <p className="text-sm text-gray-400 mt-0.5">Deep-dive into fraud patterns and claim trends</p>
        </div>
        {/* Time range selector */}
        <div className="flex items-center gap-1 bg-gray-100 dark:bg-gray-800 rounded-lg p-1 self-start sm:self-auto">
          {DAYS_OPTIONS.map(d => (
            <button
              key={d}
              onClick={() => setDays(d)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                days === d
                  ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                  : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'
              }`}
            >
              {d}d
            </button>
          ))}
        </div>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard
          label="Total Claims"
          value={claims.length}
          icon={BarChart2}
          color="bg-brand-500"
          trend={volumeTrend}
          trendLabel={`vs prev ${half}d`}
          sub={`${stats?.totalClaims ?? 0} all-time`}
        />
        <KpiCard
          label="Fraud Rate"
          value={`${fraudRate}%`}
          icon={AlertTriangle}
          color="bg-orange-500"
          trend={parseFloat(fraudRate) > 20 ? 'up' : 'down'}
          trendLabel={parseFloat(fraudRate) > 20 ? 'High' : 'Normal'}
          sub={`${claims.filter(c => c.status === 'FLAGGED').length} flagged`}
        />
        <KpiCard
          label="Avg Fraud Score"
          value={`${avgScore}%`}
          icon={Clock}
          color="bg-purple-500"
          trend="neutral"
          trendLabel="AI model"
          sub="of analyzed claims"
        />
        <KpiCard
          label="Total Value"
          value={formatCurrency(totalAmount)}
          icon={CheckCircle}
          color="bg-green-500"
          sub={`${claims.filter(c => c.status === 'APPROVED').length} approved`}
        />
      </div>

      {/* Charts grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">

        {/* Volume trend */}
        <ChartCard
          title={`Claim Volume — Last ${days} Days`}
          sub="Total, flagged, approved, rejected by day"
        >
          <ResponsiveContainer width="100%" height={220}>
            <AreaChart data={trendData} margin={{ top: 5, right: 5, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="gradTotal" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%"  stopColor="#3b82f6" stopOpacity={0.2} />
                  <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="gradFlagged" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%"  stopColor="#f97316" stopOpacity={0.2} />
                  <stop offset="95%" stopColor="#f97316" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" strokeOpacity={0.5} />
              <XAxis dataKey="date" tick={{ fontSize: 10 }} tickLine={false} axisLine={false}
                interval={Math.ceil(days / 7) - 1} />
              <YAxis tick={{ fontSize: 10 }} tickLine={false} axisLine={false} />
              <Tooltip content={<ChartTooltip />} />
              <Area type="monotone" dataKey="total"    name="Total"    stroke="#3b82f6" fill="url(#gradTotal)"   strokeWidth={2} dot={false} />
              <Area type="monotone" dataKey="flagged"  name="Flagged"  stroke="#f97316" fill="url(#gradFlagged)" strokeWidth={2} dot={false} />
              <Area type="monotone" dataKey="approved" name="Approved" stroke="#22c55e" fill="none"              strokeWidth={1.5} dot={false} strokeDasharray="4 2" />
            </AreaChart>
          </ResponsiveContainer>
        </ChartCard>

        {/* Claim status distribution */}
        <ChartCard title="Claim Status Distribution" sub="Current breakdown by status">
          <div className="flex items-center gap-4">
            <ResponsiveContainer width="55%" height={200}>
              <PieChart>
                <Pie
                  data={statusData}
                  cx="50%" cy="50%"
                  innerRadius={55} outerRadius={80}
                  paddingAngle={3}
                  dataKey="value"
                >
                  {statusData.map((entry, i) => (
                    <Cell key={i} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip formatter={(v: number) => [v, 'Claims']} />
              </PieChart>
            </ResponsiveContainer>
            <div className="flex-1 space-y-2">
              {statusData.map(d => (
                <div key={d.name} className="flex items-center justify-between">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ background: d.color }} />
                    <span className="text-xs text-gray-600 dark:text-gray-400 truncate">{d.name}</span>
                  </div>
                  <span className="text-xs font-semibold text-gray-800 dark:text-gray-200 ml-2">{d.value}</span>
                </div>
              ))}
            </div>
          </div>
        </ChartCard>

        {/* Claim type performance */}
        <ChartCard title="Claims by Type" sub="Volume and total value per claim type">
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={typeData} margin={{ top: 5, right: 5, left: -20, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" strokeOpacity={0.5} />
              <XAxis dataKey="name" tick={{ fontSize: 10 }} tickLine={false} axisLine={false} />
              <YAxis yAxisId="left"  tick={{ fontSize: 10 }} tickLine={false} axisLine={false} />
              <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 10 }} tickLine={false} axisLine={false}
                tickFormatter={v => `$${(v/1000).toFixed(0)}K`} />
              <Tooltip content={<ChartTooltip />} />
              <Bar yAxisId="left"  dataKey="count"  name="Count"  radius={[4, 4, 0, 0]}>
                {typeData.map((_, i) => <Cell key={i} fill={TYPE_COLORS[i % TYPE_COLORS.length]} />)}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </ChartCard>

        {/* Fraud score histogram */}
        <ChartCard title="Fraud Score Distribution" sub="Claim count by AI-assigned fraud probability bucket">
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={fraudBuckets} margin={{ top: 5, right: 5, left: -20, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" strokeOpacity={0.5} />
              <XAxis dataKey="range" tick={{ fontSize: 10 }} tickLine={false} axisLine={false} />
              <YAxis tick={{ fontSize: 10 }} tickLine={false} axisLine={false} />
              <Tooltip content={<ChartTooltip />} />
              <Bar dataKey="count" name="Claims" radius={[4, 4, 0, 0]}>
                {fraudBuckets.map((b, i) => (
                  <Cell key={i} fill={
                    i === 0 ? '#22c55e' :
                    i === 1 ? '#84cc16' :
                    i === 2 ? '#eab308' :
                    i === 3 ? '#f97316' : '#ef4444'
                  } />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </ChartCard>

        {/* Risk level breakdown */}
        <ChartCard title="Risk Level Distribution" sub="Claim count by AI-assessed risk level">
          <div className="flex items-center gap-4">
            <ResponsiveContainer width="55%" height={200}>
              <PieChart>
                <Pie
                  data={riskData}
                  cx="50%" cy="50%"
                  outerRadius={80}
                  paddingAngle={3}
                  dataKey="value"
                >
                  {riskData.map((entry, i) => (
                    <Cell key={i} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip formatter={(v: number) => [v, 'Claims']} />
              </PieChart>
            </ResponsiveContainer>
            <div className="flex-1 space-y-2">
              {riskData.map(d => (
                <div key={d.name} className="flex items-center justify-between">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ background: d.color }} />
                    <span className="text-xs text-gray-600 dark:text-gray-400 truncate">{d.name}</span>
                  </div>
                  <span className="text-xs font-semibold text-gray-800 dark:text-gray-200 ml-2">{d.value}</span>
                </div>
              ))}
              {riskData.length === 0 && (
                <p className="text-xs text-gray-400 text-center">No risk data available</p>
              )}
            </div>
          </div>
        </ChartCard>

        {/* Daily amount trend */}
        <ChartCard title="Daily Claim Value" sub={`Total monetary value submitted per day (last ${days} days)`}>
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={trendData} margin={{ top: 5, right: 5, left: -5, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" strokeOpacity={0.5} />
              <XAxis dataKey="date" tick={{ fontSize: 10 }} tickLine={false} axisLine={false}
                interval={Math.ceil(days / 7) - 1} />
              <YAxis tick={{ fontSize: 10 }} tickLine={false} axisLine={false}
                tickFormatter={v => `$${(v/1000).toFixed(0)}K`} />
              <Tooltip
                formatter={(v: number) => [formatCurrency(v), 'Value']}
                content={<ChartTooltip />}
              />
              <Line
                type="monotone" dataKey="amount" name="Value"
                stroke="#8b5cf6" strokeWidth={2.5} dot={false}
                activeDot={{ r: 4, fill: '#8b5cf6' }}
              />
            </LineChart>
          </ResponsiveContainer>
        </ChartCard>

      </div>
    </div>
  )
}
