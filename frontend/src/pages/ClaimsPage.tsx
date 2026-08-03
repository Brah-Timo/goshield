import { useState, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { useClaimList, useDeleteClaim } from '@/hooks/useClaims'
import {
  useReactTable, getCoreRowModel, createColumnHelper,
  flexRender, SortingState, getSortedRowModel,
} from '@tanstack/react-table'
import type { Claim, ClaimFilter, ClaimStatus, RiskLevel } from '@/types'
import { format, parseISO } from 'date-fns'
import {
  ChevronLeft, ChevronRight, Search, Download, Filter,
  ArrowUpDown, ArrowUp, ArrowDown, X, Trash2, AlertTriangle,
  CheckSquare, Square,
} from 'lucide-react'
import { toast } from '@/store/toast'

// ── Status & risk badges ──────────────────────────────────────────────────────
const STATUS_CLS: Record<ClaimStatus, string> = {
  FLAGGED:    'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
  APPROVED:   'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
  REJECTED:   'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-300',
  PENDING:    'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
  PROCESSING: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
  MORE_INFO:  'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
}
const RISK_CLS: Record<RiskLevel, string> = {
  LOW:      'text-green-600 dark:text-green-400',
  MEDIUM:   'text-amber-600 dark:text-amber-400',
  HIGH:     'text-red-600 dark:text-red-400',
  CRITICAL: 'text-purple-600 dark:text-purple-400',
}

// ── CSV export helper ─────────────────────────────────────────────────────────
function exportCSV(claims: Claim[]) {
  const headers = ['ID', 'Policy', 'Type', 'Status', 'Amount', 'Fraud Score', 'Risk', 'Date']
  const rows = claims.map(c => [
    c.id,
    c.policyNumber,
    c.claimType,
    c.status,
    c.amount,
    c.fraudScore != null ? (c.fraudScore * 100).toFixed(1) + '%' : '',
    c.riskLevel ?? '',
    format(parseISO(c.createdAt), 'yyyy-MM-dd'),
  ])
  const csv  = [headers, ...rows].map(r => r.join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url  = URL.createObjectURL(blob)
  const a    = document.createElement('a')
  a.href     = url
  a.download = `goshield-claims-${format(new Date(), 'yyyy-MM-dd')}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

// ── Sort icon ─────────────────────────────────────────────────────────────────
function SortIcon({ state }: { state: false | 'asc' | 'desc' }) {
  if (state === 'asc')  return <ArrowUp   className="h-3 w-3 ml-1 text-brand-500" />
  if (state === 'desc') return <ArrowDown className="h-3 w-3 ml-1 text-brand-500" />
  return <ArrowUpDown className="h-3 w-3 ml-1 text-gray-300" />
}

// ── Delete confirm dialog ─────────────────────────────────────────────────────
function DeleteConfirmDialog({
  count,
  onConfirm,
  onCancel,
  isPending,
}: {
  count: number
  onConfirm: () => void
  onCancel: () => void
  isPending: boolean
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onCancel}
      />
      {/* Modal */}
      <div className="relative bg-white dark:bg-gray-900 rounded-2xl shadow-2xl p-6 max-w-sm w-full border border-gray-200 dark:border-gray-700 animate-slide-in">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-danger-100 dark:bg-danger-900/30">
            <AlertTriangle className="h-5 w-5 text-danger-600 dark:text-danger-400" />
          </div>
          <div>
            <h3 className="font-bold text-gray-900 dark:text-white text-sm">Delete {count} claim{count !== 1 ? 's' : ''}?</h3>
            <p className="text-xs text-gray-500 dark:text-gray-400">This action cannot be undone.</p>
          </div>
        </div>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-5">
          You are about to permanently delete{' '}
          <span className="font-semibold text-danger-600 dark:text-danger-400">{count} claim{count !== 1 ? 's' : ''}</span>.
          All associated documents and audit data will be removed.
        </p>
        <div className="flex gap-3">
          <button
            onClick={onConfirm}
            disabled={isPending}
            className="flex-1 rounded-lg bg-danger-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-danger-700 disabled:opacity-50 transition-colors"
          >
            {isPending ? 'Deleting…' : `Delete ${count > 1 ? 'all' : ''}`}
          </button>
          <button
            onClick={onCancel}
            disabled={isPending}
            className="flex-1 rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-2.5 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  )
}

// ── Column definitions ────────────────────────────────────────────────────────
const col = createColumnHelper<Claim>()

function buildColumns(
  selectedIds: Set<string>,
  onToggle: (id: string) => void,
  onToggleAll: (ids: string[]) => void,
  allIds: string[],
  onDelete: (id: string) => void,
) {
  return [
    col.display({
      id: 'select',
      header: () => {
        const allSelected = allIds.length > 0 && allIds.every(id => selectedIds.has(id))
        return (
          <button onClick={() => onToggleAll(allIds)} className="p-0.5">
            {allSelected
              ? <CheckSquare className="h-4 w-4 text-brand-600" />
              : <Square className="h-4 w-4 text-gray-400" />
            }
          </button>
        )
      },
      cell: i => (
        <button onClick={() => onToggle(i.row.original.id)} className="p-0.5">
          {selectedIds.has(i.row.original.id)
            ? <CheckSquare className="h-4 w-4 text-brand-600" />
            : <Square className="h-4 w-4 text-gray-300" />
          }
        </button>
      ),
      enableSorting: false,
    }),
    col.accessor('id', {
      header: 'Claim ID',
      cell: i => (
        <span className="font-mono text-xs text-gray-500 dark:text-gray-400">
          {i.getValue().slice(0, 8)}…
        </span>
      ),
      enableSorting: false,
    }),
    col.accessor('policyNumber', { header: 'Policy' }),
    col.accessor('claimType',    { header: 'Type' }),
    col.accessor('status', {
      header: 'Status',
      cell: i => {
        const s = i.getValue()
        return (
          <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_CLS[s] ?? 'bg-gray-100 text-gray-600'}`}>
            {s}
          </span>
        )
      },
    }),
    col.accessor('amount', {
      header: 'Amount',
      cell: i => <span className="font-medium">${i.getValue().toLocaleString()}</span>,
    }),
    col.accessor('fraudScore', {
      header: 'Fraud Score',
      cell: i => {
        const v = i.getValue()
        if (v == null) return <span className="text-gray-300 dark:text-gray-600">—</span>
        const pct   = (v * 100).toFixed(0)
        const color = v > 0.75 ? 'text-red-600' : v > 0.4 ? 'text-amber-600' : 'text-green-600'
        return (
          <div className="flex items-center gap-2">
            <div className="h-1.5 w-16 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${v > 0.75 ? 'bg-red-500' : v > 0.4 ? 'bg-amber-500' : 'bg-green-500'}`}
                style={{ width: `${pct}%` }}
              />
            </div>
            <span className={`text-xs font-medium ${color}`}>{pct}%</span>
          </div>
        )
      },
    }),
    col.accessor('riskLevel', {
      header: 'Risk',
      cell: i => {
        const lvl = i.getValue()
        return lvl
          ? <span className={`font-semibold text-xs ${RISK_CLS[lvl]}`}>{lvl}</span>
          : <span className="text-gray-300 dark:text-gray-600">—</span>
      },
    }),
    col.accessor('createdAt', {
      header: 'Date',
      cell: i => (
        <span className="text-gray-500 dark:text-gray-400 text-xs">
          {format(parseISO(i.getValue()), 'MMM d, yyyy')}
        </span>
      ),
    }),
    col.display({
      id: 'actions',
      header: '',
      cell: i => (
        <div className="flex items-center gap-3">
          <Link
            to={`/claims/${i.row.original.id}`}
            className="text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-300 text-sm font-medium whitespace-nowrap"
          >
            View →
          </Link>
          <button
            onClick={e => { e.stopPropagation(); onDelete(i.row.original.id) }}
            className="p-1 rounded-lg text-gray-400 hover:text-danger-500 hover:bg-danger-50 dark:hover:bg-danger-900/20 transition-colors"
            title="Delete claim"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ),
      enableSorting: false,
    }),
  ]
}

// ══════════════════════════════════════════════════════════════════════════════
export default function ClaimsPage() {
  const [filter,        setFilter]        = useState<ClaimFilter>({ page: 1, pageSize: 20 })
  const [sorting,       setSorting]       = useState<SortingState>([])
  const [showFilters,   setShowFilters]   = useState(false)
  const [search,        setSearch]        = useState('')
  const [selectedIds,   setSelectedIds]   = useState<Set<string>>(new Set())
  const [deleteTarget,  setDeleteTarget]  = useState<string[] | null>(null) // null = dialog closed

  const { data, isLoading } = useClaimList(filter)
  const deleteMutation       = useDeleteClaim()

  // Client-side search
  const allClaims = data?.claims ?? []
  const displayed = search.trim()
    ? allClaims.filter(c =>
        c.policyNumber.toLowerCase().includes(search.toLowerCase()) ||
        c.id.toLowerCase().includes(search.toLowerCase())
      )
    : allClaims

  // Selection helpers
  const toggleSelect = useCallback((id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }, [])

  const toggleAll = useCallback((ids: string[]) => {
    setSelectedIds(prev => {
      const allSelected = ids.every(id => prev.has(id))
      if (allSelected) {
        const next = new Set(prev)
        ids.forEach(id => next.delete(id))
        return next
      }
      const next = new Set(prev)
      ids.forEach(id => next.add(id))
      return next
    })
  }, [])

  const allIds = displayed.map(c => c.id)

  // Delete handler
  const openDeleteDialog = (id: string) => setDeleteTarget([id])
  const openBulkDelete   = () => {
    if (selectedIds.size === 0) return
    setDeleteTarget([...selectedIds])
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await Promise.all(deleteTarget.map(id => deleteMutation.mutateAsync(id)))
      toast.success(
        `${deleteTarget.length} claim${deleteTarget.length > 1 ? 's' : ''} deleted`,
        'Deleted'
      )
      setSelectedIds(prev => {
        const next = new Set(prev)
        deleteTarget.forEach(id => next.delete(id))
        return next
      })
    } catch {
      toast.error('Failed to delete one or more claims', 'Error')
    } finally {
      setDeleteTarget(null)
    }
  }

  const columns = buildColumns(selectedIds, toggleSelect, toggleAll, allIds, openDeleteDialog)

  const table = useReactTable({
    data: displayed,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    manualPagination: true,
    pageCount: data?.totalPages ?? 0,
  })

  const setF = useCallback(<K extends keyof ClaimFilter>(k: K, v: ClaimFilter[K]) => {
    setFilter(f => ({ ...f, [k]: v, page: 1 }))
  }, [])

  const clearFilters = () => {
    setFilter({ page: 1, pageSize: 20 })
    setSearch('')
  }

  const activeFilterCount = [
    filter.status, filter.riskLevel, filter.minAmount,
    filter.maxAmount, filter.sortBy !== undefined,
  ].filter(Boolean).length

  return (
    <div className="space-y-4 animate-fade-in">
      {/* Delete confirm dialog */}
      {deleteTarget && (
        <DeleteConfirmDialog
          count={deleteTarget.length}
          onConfirm={confirmDelete}
          onCancel={() => setDeleteTarget(null)}
          isPending={deleteMutation.isPending}
        />
      )}

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Claims</h1>
          <p className="text-xs text-gray-400 mt-0.5">
            {data?.total ?? 0} total claims
            {selectedIds.size > 0 && (
              <span className="ml-2 text-brand-600 dark:text-brand-400 font-medium">
                · {selectedIds.size} selected
              </span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Bulk delete */}
          {selectedIds.size > 0 && (
            <button
              onClick={openBulkDelete}
              className="flex items-center gap-2 rounded-lg border border-danger-300 dark:border-danger-700 text-danger-600 dark:text-danger-400 px-3 py-2 text-sm font-medium hover:bg-danger-50 dark:hover:bg-danger-900/20 transition-colors"
            >
              <Trash2 className="h-4 w-4" />
              Delete ({selectedIds.size})
            </button>
          )}
          <button
            onClick={() => exportCSV(displayed)}
            disabled={displayed.length === 0}
            title="Export to CSV"
            className="flex items-center gap-2 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm font-medium hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-40 transition-colors"
          >
            <Download className="h-4 w-4" />
            <span className="hidden sm:inline">Export CSV</span>
          </button>
          <Link
            to="/upload"
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 transition-colors"
          >
            + New Claim
          </Link>
        </div>
      </div>

      {/* Search + filter bar */}
      <div className="flex flex-wrap items-center gap-3">
        {/* Search */}
        <div className="relative flex-1 min-w-[200px] max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
          <input
            placeholder="Search by policy or ID…"
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
          {search && (
            <button onClick={() => setSearch('')} className="absolute right-3 top-1/2 -translate-y-1/2">
              <X className="h-3.5 w-3.5 text-gray-400" />
            </button>
          )}
        </div>

        {/* Status quick filters */}
        <div className="flex flex-wrap gap-1.5">
          {(['PENDING', 'PROCESSING', 'FLAGGED', 'APPROVED', 'REJECTED'] as ClaimStatus[]).map(s => (
            <button
              key={s}
              onClick={() => setF('status', filter.status === s ? undefined : s)}
              className={`rounded-full px-3 py-1 text-xs font-medium border transition-colors ${
                filter.status === s
                  ? 'bg-brand-600 text-white border-brand-600'
                  : 'border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300 hover:border-brand-400'
              }`}
            >
              {s}
            </button>
          ))}
        </div>

        {/* Advanced filters toggle */}
        <button
          onClick={() => setShowFilters(f => !f)}
          className={`flex items-center gap-1.5 rounded-lg border px-3 py-2 text-xs font-medium transition-colors ${
            showFilters || activeFilterCount > 0
              ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/20 text-brand-700 dark:text-brand-300'
              : 'border-gray-300 dark:border-gray-600 hover:border-brand-400'
          }`}
        >
          <Filter className="h-3.5 w-3.5" />
          Filters
          {activeFilterCount > 0 && (
            <span className="ml-1 rounded-full bg-brand-600 text-white text-[10px] h-4 w-4 flex items-center justify-center">
              {activeFilterCount}
            </span>
          )}
        </button>

        {activeFilterCount > 0 && (
          <button
            onClick={clearFilters}
            className="text-xs text-gray-400 hover:text-red-500 transition-colors flex items-center gap-1"
          >
            <X className="h-3 w-3" /> Clear filters
          </button>
        )}
      </div>

      {/* Advanced filter panel */}
      {showFilters && (
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-4 grid sm:grid-cols-2 lg:grid-cols-4 gap-4 animate-slide-in">
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">Risk Level</label>
            <select
              value={filter.riskLevel ?? ''}
              onChange={e => setF('riskLevel', e.target.value as RiskLevel || undefined)}
              className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
            >
              <option value="">All levels</option>
              {(['LOW', 'MEDIUM', 'HIGH', 'CRITICAL'] as RiskLevel[]).map(r => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">Min Amount ($)</label>
            <input
              type="number" min="0" placeholder="0"
              value={filter.minAmount ?? ''}
              onChange={e => setF('minAmount', e.target.value ? +e.target.value : undefined)}
              className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">Max Amount ($)</label>
            <input
              type="number" min="0" placeholder="Unlimited"
              value={filter.maxAmount ?? ''}
              onChange={e => setF('maxAmount', e.target.value ? +e.target.value : undefined)}
              className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">Sort By</label>
            <select
              value={`${filter.sortBy ?? 'created_at'}:${filter.sortOrder ?? 'desc'}`}
              onChange={e => {
                const [sortBy, sortOrder] = e.target.value.split(':') as [string, 'asc' | 'desc']
                setFilter(f => ({ ...f, sortBy, sortOrder, page: 1 }))
              }}
              className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
            >
              <option value="created_at:desc">Newest first</option>
              <option value="created_at:asc">Oldest first</option>
              <option value="amount:desc">Highest amount</option>
              <option value="amount:asc">Lowest amount</option>
              <option value="fraud_score:desc">Highest fraud score</option>
            </select>
          </div>
        </div>
      )}

      {/* Table */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 overflow-x-auto">
        {isLoading ? (
          <div className="divide-y divide-gray-100 dark:divide-gray-700">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="px-4 py-3.5 flex items-center gap-4 animate-pulse">
                <div className="h-4 w-4 bg-gray-200 dark:bg-gray-700 rounded" />
                <div className="h-3.5 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
                <div className="h-3.5 w-24 bg-gray-100 dark:bg-gray-800 rounded" />
                <div className="h-3.5 w-16 bg-gray-100 dark:bg-gray-800 rounded" />
                <div className="h-5 w-16 bg-gray-100 dark:bg-gray-800 rounded-full ml-2" />
                <div className="h-3.5 w-12 bg-gray-100 dark:bg-gray-800 rounded ml-auto" />
              </div>
            ))}
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800/60 border-b border-gray-100 dark:border-gray-700">
              {table.getHeaderGroups().map(hg => (
                <tr key={hg.id}>
                  {hg.headers.map(h => (
                    <th
                      key={h.id}
                      onClick={h.column.getToggleSortingHandler()}
                      className={`px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider select-none ${
                        h.column.getCanSort() ? 'cursor-pointer hover:text-gray-700 dark:hover:text-gray-200' : ''
                      }`}
                    >
                      <span className="flex items-center">
                        {flexRender(h.column.columnDef.header, h.getContext())}
                        {h.column.getCanSort() && (
                          <SortIcon state={h.column.getIsSorted()} />
                        )}
                      </span>
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700/50">
              {table.getRowModel().rows.map(row => (
                <tr
                  key={row.id}
                  className={`hover:bg-gray-50 dark:hover:bg-gray-800/40 transition-colors ${
                    selectedIds.has(row.original.id)
                      ? 'bg-brand-50/50 dark:bg-brand-900/10'
                      : ''
                  }`}
                >
                  {row.getVisibleCells().map(cell => (
                    <td key={cell.id} className="px-4 py-3.5">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
              {table.getRowModel().rows.length === 0 && (
                <tr>
                  <td colSpan={12} className="px-4 py-12 text-center text-gray-400 text-sm">
                    {search ? `No claims matching "${search}"` : 'No claims found.'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}

        {/* Pagination */}
        <div className="flex items-center justify-between px-4 py-3 border-t border-gray-100 dark:border-gray-700 text-sm text-gray-500 dark:text-gray-400">
          <span className="text-xs">
            {data && data.total > 0
              ? `${(filter.page! - 1) * filter.pageSize! + 1}–${Math.min(filter.page! * filter.pageSize!, data.total)} of ${data.total} claims`
              : '0 claims'
            }
          </span>
          <div className="flex items-center gap-3">
            <select
              value={filter.pageSize}
              onChange={e => setFilter(f => ({ ...f, pageSize: +e.target.value, page: 1 }))}
              className="text-xs rounded border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-2 py-1 focus:outline-none"
            >
              {[10, 20, 50].map(n => <option key={n} value={n}>{n} / page</option>)}
            </select>
            <div className="flex gap-1">
              <button
                disabled={filter.page === 1}
                onClick={() => setFilter(f => ({ ...f, page: f.page! - 1 }))}
                className="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-30 transition-colors"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              <span className="px-2 py-1 text-xs font-medium">
                {filter.page} / {data?.totalPages ?? 1}
              </span>
              <button
                disabled={filter.page === data?.totalPages}
                onClick={() => setFilter(f => ({ ...f, page: f.page! + 1 }))}
                className="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-30 transition-colors"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
