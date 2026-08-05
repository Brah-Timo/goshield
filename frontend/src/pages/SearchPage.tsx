import { useState, useEffect, useRef, useCallback } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useClaimList } from '@/hooks/useClaims'
import { format, parseISO } from 'date-fns'
import {
  Search, X, Filter, SlidersHorizontal, ChevronDown,
  FileText, AlertTriangle, CheckCircle, Clock, XCircle,
  ArrowUpRight, Loader2, Brain,
} from 'lucide-react'
import type { ClaimFilter, ClaimStatus, ClaimType, RiskLevel } from '@/types'

// ── Status badge ─────────────────────────────────────────────────────────────
const STATUS_STYLES: Record<string, string> = {
  PENDING:    'bg-gray-100  text-gray-600  dark:bg-gray-800 dark:text-gray-300',
  PROCESSING: 'bg-blue-100  text-blue-700  dark:bg-blue-900/40 dark:text-blue-300',
  FLAGGED:    'bg-red-100   text-red-700   dark:bg-red-900/40 dark:text-red-300',
  APPROVED:   'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
  REJECTED:   'bg-gray-200  text-gray-700  dark:bg-gray-700 dark:text-gray-400',
  MORE_INFO:  'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
}

const STATUS_ICONS: Record<string, React.ReactNode> = {
  PENDING:    <Clock    className="h-3 w-3" />,
  PROCESSING: <Loader2  className="h-3 w-3 animate-spin" />,
  FLAGGED:    <AlertTriangle className="h-3 w-3" />,
  APPROVED:   <CheckCircle  className="h-3 w-3" />,
  REJECTED:   <XCircle  className="h-3 w-3" />,
  MORE_INFO:  <AlertTriangle className="h-3 w-3" />,
}

function StatusBadge({ status }: { status: string }) {
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLES[status] ?? STATUS_STYLES.PENDING}`}>
      {STATUS_ICONS[status]}
      {status}
    </span>
  )
}

// ── Risk dot ─────────────────────────────────────────────────────────────────
const RISK_COLORS: Record<string, string> = {
  LOW:      'bg-green-500',
  MEDIUM:   'bg-amber-500',
  HIGH:     'bg-red-500',
  CRITICAL: 'bg-purple-600',
}

function RiskDot({ level }: { level?: string }) {
  if (!level) return null
  return (
    <span className={`inline-block h-2 w-2 rounded-full ${RISK_COLORS[level] ?? 'bg-gray-400'}`} title={level + ' RISK'} />
  )
}

// ── Filter pill ───────────────────────────────────────────────────────────────
function Pill({ label, onRemove }: { label: string; onRemove: () => void }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-brand-100 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 px-2.5 py-1 text-xs font-medium">
      {label}
      <button onClick={onRemove} className="hover:text-brand-900 dark:hover:text-brand-100">
        <X className="h-3 w-3" />
      </button>
    </span>
  )
}

// ── Highlight matching text ───────────────────────────────────────────────────
function Highlight({ text, query }: { text: string; query: string }) {
  if (!query.trim()) return <>{text}</>
  const re = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
  const parts = text.split(re)
  return (
    <>
      {parts.map((p, i) =>
        re.test(p)
          ? <mark key={i} className="bg-yellow-200 dark:bg-yellow-700/60 text-yellow-900 dark:text-yellow-100 rounded px-0.5">{p}</mark>
          : <span key={i}>{p}</span>
      )}
    </>
  )
}

// ── Fraud score bar ───────────────────────────────────────────────────────────
function FraudBar({ score }: { score?: number }) {
  if (score == null) return <span className="text-xs text-gray-400">—</span>
  const pct = Math.round(score * 100)
  const color = pct > 75 ? 'bg-purple-500' : pct > 50 ? 'bg-red-500' : pct > 25 ? 'bg-amber-500' : 'bg-green-500'
  return (
    <div className="flex items-center gap-2 min-w-[80px]">
      <div className="flex-1 h-1.5 bg-gray-100 dark:bg-gray-800 rounded-full overflow-hidden">
        <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs font-mono text-gray-600 dark:text-gray-400 w-8 text-right">{pct}%</span>
    </div>
  )
}

// ════════════════════════════════════════════════════════════════════════════
export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const initialQ = searchParams.get('q') ?? ''

  const [query,       setQuery]       = useState(initialQ)
  const [debouncedQ,  setDebouncedQ]  = useState(initialQ)
  const [showFilters, setShowFilters] = useState(false)

  // Filter state
  const [status,    setStatus]    = useState<ClaimStatus | ''>((searchParams.get('status') as ClaimStatus) ?? '')
  const [claimType, setClaimType] = useState<ClaimType | ''>((searchParams.get('type') as ClaimType) ?? '')
  const [riskLevel, setRiskLevel] = useState<RiskLevel | ''>((searchParams.get('risk') as RiskLevel) ?? '')
  const [minAmount, setMinAmount] = useState(searchParams.get('min') ?? '')
  const [maxAmount, setMaxAmount] = useState(searchParams.get('max') ?? '')
  const [sortBy,    setSortBy]    = useState(searchParams.get('sort') ?? 'created_at')
  const [page,      setPage]      = useState(1)

  const inputRef = useRef<HTMLInputElement>(null)

  // Focus input on mount + Escape to clear
  useEffect(() => {
    inputRef.current?.focus()
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { setQuery(''); setDebouncedQ('') }
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        inputRef.current?.focus()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  // Debounce query
  useEffect(() => {
    const t = setTimeout(() => { setDebouncedQ(query); setPage(1) }, 350)
    return () => clearTimeout(t)
  }, [query])

  // Sync URL params
  useEffect(() => {
    const p: Record<string, string> = {}
    if (debouncedQ) p.q = debouncedQ
    if (status)     p.status = status
    if (claimType)  p.type = claimType
    if (riskLevel)  p.risk = riskLevel
    if (minAmount)  p.min = minAmount
    if (maxAmount)  p.max = maxAmount
    if (sortBy !== 'created_at') p.sort = sortBy
    setSearchParams(p, { replace: true })
  }, [debouncedQ, status, claimType, riskLevel, minAmount, maxAmount, sortBy, setSearchParams])

  // Build query filter
  const filter: ClaimFilter = {
    ...(status    && { status }),
    ...(claimType && { claimType }),
    ...(riskLevel && { riskLevel }),
    ...(minAmount && { minAmount: parseFloat(minAmount) }),
    ...(maxAmount && { maxAmount: parseFloat(maxAmount) }),
    sortBy,
    sortOrder: 'desc',
    page,
    pageSize: 25,
  }

  const { data, isLoading, isFetching } = useClaimList(filter)

  // Client-side text filter (policy number, description, type)
  const results = (data?.claims ?? []).filter(c => {
    if (!debouncedQ.trim()) return true
    const q = debouncedQ.toLowerCase()
    return (
      c.policyNumber?.toLowerCase().includes(q) ||
      c.description?.toLowerCase().includes(q)  ||
      c.claimType?.toLowerCase().includes(q)    ||
      c.id?.toLowerCase().includes(q)
    )
  })

  const hasFilters = !!(status || claimType || riskLevel || minAmount || maxAmount)

  const clearAll = useCallback(() => {
    setStatus(''); setClaimType(''); setRiskLevel('')
    setMinAmount(''); setMaxAmount(''); setPage(1)
  }, [])

  return (
    <div className="max-w-5xl space-y-5 animate-fade-in">
      {/* ── Page header ─────────────────────────────────────────────────── */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Search Claims</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
          Search by policy number, description, claim type, or ID · <kbd className="rounded bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 text-[10px] font-mono text-gray-500">Ctrl+K</kbd> to focus
        </p>
      </div>

      {/* ── Search bar ──────────────────────────────────────────────────── */}
      <div className="relative">
        <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={e => setQuery(e.target.value)}
          placeholder="Search policy number, description, claim ID…"
          className="w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 pl-10 pr-10 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 shadow-sm transition-shadow"
        />
        {query && (
          <button
            onClick={() => { setQuery(''); setDebouncedQ(''); inputRef.current?.focus() }}
            className="absolute right-3.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            <X className="h-4 w-4" />
          </button>
        )}
        {isFetching && (
          <Loader2 className="absolute right-10 top-1/2 -translate-y-1/2 h-4 w-4 text-brand-500 animate-spin" />
        )}
      </div>

      {/* ── Toolbar: filters toggle + active pills + sort ─────────────── */}
      <div className="flex flex-wrap items-center gap-2">
        <button
          onClick={() => setShowFilters(v => !v)}
          className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors ${
            showFilters
              ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-300'
              : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800'
          }`}
        >
          <SlidersHorizontal className="h-3.5 w-3.5" />
          Filters
          {hasFilters && (
            <span className="ml-0.5 rounded-full bg-brand-500 text-white text-[10px] font-bold w-4 h-4 flex items-center justify-center">
              {[status,claimType,riskLevel,minAmount,maxAmount].filter(Boolean).length}
            </span>
          )}
          <ChevronDown className={`h-3 w-3 transition-transform ${showFilters ? 'rotate-180' : ''}`} />
        </button>

        {/* Active filter pills */}
        {status    && <Pill label={`Status: ${status}`}       onRemove={() => setStatus('')} />}
        {claimType && <Pill label={`Type: ${claimType}`}      onRemove={() => setClaimType('')} />}
        {riskLevel && <Pill label={`Risk: ${riskLevel}`}      onRemove={() => setRiskLevel('')} />}
        {minAmount && <Pill label={`Min: $${minAmount}`}      onRemove={() => setMinAmount('')} />}
        {maxAmount && <Pill label={`Max: $${maxAmount}`}      onRemove={() => setMaxAmount('')} />}
        {hasFilters && (
          <button onClick={clearAll} className="text-xs text-red-500 hover:text-red-700 font-medium">
            Clear all
          </button>
        )}

        {/* Sort selector */}
        <div className="ml-auto flex items-center gap-1.5">
          <Filter className="h-3.5 w-3.5 text-gray-400" />
          <select
            value={sortBy}
            onChange={e => setSortBy(e.target.value)}
            className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-2.5 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-400 focus:outline-none focus:ring-1 focus:ring-brand-500"
          >
            <option value="created_at">Newest first</option>
            <option value="amount">Highest amount</option>
            <option value="fraud_score">Highest fraud score</option>
          </select>
        </div>
      </div>

      {/* ── Collapsible filter panel ──────────────────────────────────── */}
      {showFilters && (
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4 animate-slide-in">
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
            {/* Status */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Status</label>
              <select
                value={status}
                onChange={e => { setStatus(e.target.value as ClaimStatus | ''); setPage(1) }}
                className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
              >
                <option value="">All</option>
                {['PENDING','PROCESSING','FLAGGED','APPROVED','REJECTED','MORE_INFO'].map(s => (
                  <option key={s} value={s}>{s}</option>
                ))}
              </select>
            </div>
            {/* Claim type */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Type</label>
              <select
                value={claimType}
                onChange={e => { setClaimType(e.target.value as ClaimType | ''); setPage(1) }}
                className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
              >
                <option value="">All</option>
                {['HEALTH','CAR','PROPERTY','LIFE','TRAVEL','OTHER'].map(t => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
            </div>
            {/* Risk level */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Risk Level</label>
              <select
                value={riskLevel}
                onChange={e => { setRiskLevel(e.target.value as RiskLevel | ''); setPage(1) }}
                className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
              >
                <option value="">All</option>
                {['LOW','MEDIUM','HIGH','CRITICAL'].map(r => (
                  <option key={r} value={r}>{r}</option>
                ))}
              </select>
            </div>
            {/* Min amount */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Min Amount ($)</label>
              <input
                type="number"
                value={minAmount}
                onChange={e => { setMinAmount(e.target.value); setPage(1) }}
                placeholder="0"
                className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
              />
            </div>
            {/* Max amount */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Max Amount ($)</label>
              <input
                type="number"
                value={maxAmount}
                onChange={e => { setMaxAmount(e.target.value); setPage(1) }}
                placeholder="∞"
                className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
              />
            </div>
          </div>
        </div>
      )}

      {/* ── Results header ────────────────────────────────────────────── */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {isLoading ? (
            <span className="flex items-center gap-1.5"><Loader2 className="h-3.5 w-3.5 animate-spin" /> Searching…</span>
          ) : (
            <>
              <span className="font-semibold text-gray-800 dark:text-gray-200">{results.length}</span>
              {results.length !== (data?.total ?? 0) && (
                <> of <span className="font-semibold text-gray-800 dark:text-gray-200">{data?.total ?? 0}</span></>
              )}
              {' '}result{results.length !== 1 ? 's' : ''}
              {debouncedQ && <> for "<span className="font-medium text-brand-600 dark:text-brand-400">{debouncedQ}</span>"</>}
            </>
          )}
        </p>
        {(data?.totalPages ?? 0) > 1 && (
          <div className="flex items-center gap-1.5 text-xs text-gray-500">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className="rounded px-2 py-1 border border-gray-200 dark:border-gray-700 disabled:opacity-40 hover:bg-gray-50 dark:hover:bg-gray-800"
            >← Prev</button>
            <span>Page {page} / {data?.totalPages}</span>
            <button
              onClick={() => setPage(p => Math.min(data?.totalPages ?? 1, p + 1))}
              disabled={page === (data?.totalPages ?? 1)}
              className="rounded px-2 py-1 border border-gray-200 dark:border-gray-700 disabled:opacity-40 hover:bg-gray-50 dark:hover:bg-gray-800"
            >Next →</button>
          </div>
        )}
      </div>

      {/* ── Results list ─────────────────────────────────────────────── */}
      {isLoading ? (
        <div className="space-y-3">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="h-20 bg-gray-100 dark:bg-gray-800 rounded-xl animate-pulse" />
          ))}
        </div>
      ) : results.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <Search className="h-12 w-12 text-gray-200 dark:text-gray-700 mb-4" />
          <h3 className="text-lg font-semibold text-gray-700 dark:text-gray-300">No results found</h3>
          <p className="text-sm text-gray-400 mt-1 max-w-xs">
            {debouncedQ
              ? `No claims match "${debouncedQ}". Try different keywords or adjust filters.`
              : 'No claims match the selected filters. Try changing your filter criteria.'}
          </p>
          {(debouncedQ || hasFilters) && (
            <button
              onClick={() => { setQuery(''); setDebouncedQ(''); clearAll() }}
              className="mt-4 text-sm text-brand-600 dark:text-brand-400 hover:underline font-medium"
            >
              Clear search & filters
            </button>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {results.map(claim => (
            <Link
              key={claim.id}
              to={`/claims/${claim.id}`}
              className="group flex items-start gap-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4 hover:border-brand-400 dark:hover:border-brand-600 hover:shadow-sm transition-all animate-scale-in"
            >
              {/* Icon */}
              <div className="shrink-0 h-9 w-9 rounded-lg bg-brand-50 dark:bg-brand-900/30 flex items-center justify-center text-brand-600 dark:text-brand-400 mt-0.5">
                {claim.fraudScore != null && claim.fraudScore > 0.5
                  ? <Brain className="h-4 w-4" />
                  : <FileText className="h-4 w-4" />
                }
              </div>

              {/* Content */}
              <div className="flex-1 min-w-0">
                <div className="flex flex-wrap items-center gap-2 mb-0.5">
                  <span className="font-semibold text-sm text-gray-900 dark:text-white truncate">
                    <Highlight text={claim.policyNumber} query={debouncedQ} />
                  </span>
                  <span className="text-[10px] font-mono text-gray-400 dark:text-gray-500 truncate">
                    <Highlight text={claim.id.slice(0, 8) + '…'} query={debouncedQ} />
                  </span>
                  <StatusBadge status={claim.status} />
                  <RiskDot level={claim.riskLevel} />
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 truncate">
                  <span className="font-medium text-gray-700 dark:text-gray-300 mr-2">{claim.claimType}</span>
                  <Highlight text={claim.description ?? ''} query={debouncedQ} />
                </p>
              </div>

              {/* Right meta */}
              <div className="shrink-0 text-right space-y-1.5">
                <p className="text-sm font-bold text-gray-900 dark:text-white">
                  ${claim.amount.toLocaleString()}
                </p>
                <FraudBar score={claim.fraudScore} />
                <p className="text-[10px] text-gray-400">
                  {format(parseISO(claim.createdAt), 'MMM d, yyyy')}
                </p>
              </div>

              {/* Arrow */}
              <ArrowUpRight className="h-4 w-4 text-gray-300 dark:text-gray-600 group-hover:text-brand-500 transition-colors shrink-0 mt-1" />
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
