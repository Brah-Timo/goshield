import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useClaimList } from '@/hooks/useClaims'
import { useReactTable, getCoreRowModel, createColumnHelper, flexRender } from '@tanstack/react-table'
import type { Claim, ClaimFilter } from '@/types'
import { format } from 'date-fns'
import { ChevronLeft, ChevronRight, Search } from 'lucide-react'

const col = createColumnHelper<Claim>()
const columns = [
  col.accessor('id', { header: 'ID', cell: i => <span className="font-mono text-xs">{i.getValue().slice(0,8)}…</span> }),
  col.accessor('policyNumber', { header: 'Policy' }),
  col.accessor('claimType',    { header: 'Type' }),
  col.accessor('status', { header: 'Status', cell: i => {
    const s = i.getValue()
    return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
      s==='FLAGGED'?'bg-red-100 text-red-700': s==='APPROVED'?'bg-green-100 text-green-700':
      s==='REJECTED'?'bg-gray-200 text-gray-600':'bg-amber-100 text-amber-700'}`}>{s}</span>
  }}),
  col.accessor('amount', { header: 'Amount', cell: i => `$${i.getValue().toLocaleString()}` }),
  col.accessor('fraudScore', { header: 'Score', cell: i => i.getValue() != null ? (i.getValue()! * 100).toFixed(0)+'%' : '—' }),
  col.accessor('riskLevel', { header: 'Risk', cell: i => {
    const lvl = i.getValue()
    const color = lvl==='CRITICAL'?'text-purple-600':lvl==='HIGH'?'text-red-600':lvl==='MEDIUM'?'text-amber-600':'text-green-600'
    return lvl ? <span className={`font-medium ${color}`}>{lvl}</span> : '—'
  }}),
  col.accessor('createdAt', { header: 'Date', cell: i => format(new Date(i.getValue()), 'MMM d, yyyy') }),
  col.display({ id: 'actions', header: '', cell: i => (
    <Link to={`/claims/${i.row.original.id}`} className="text-brand-600 hover:text-brand-800 text-sm font-medium">View</Link>
  )}),
]

export default function ClaimsPage() {
  const [filter, setFilter] = useState<ClaimFilter>({ page: 1, pageSize: 20 })
  const { data, isLoading } = useClaimList(filter)

  const table = useReactTable({
    data: data?.claims ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    pageCount: data?.totalPages ?? 0,
  })

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Claims</h1>
        <Link to="/upload" className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
          + New Claim
        </Link>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input placeholder="Search policy…" className="pl-9 rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
            onChange={e => setFilter(f => ({ ...f, page: 1 }))} />
        </div>
        {(['PENDING','PROCESSING','FLAGGED','APPROVED','REJECTED'] as const).map(s => (
          <button key={s} onClick={() => setFilter(f => ({ ...f, status: f.status===s ? undefined : s, page:1 }))}
            className={`rounded-full px-3 py-1 text-xs font-medium border transition-colors ${
              filter.status===s ? 'bg-brand-600 text-white border-brand-600' : 'border-gray-300 hover:border-brand-400'}`}>
            {s}
          </button>
        ))}
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border overflow-x-auto">
        {isLoading ? (
          <div className="p-12 text-center text-gray-400">Loading claims…</div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800">
              {table.getHeaderGroups().map(hg => (
                <tr key={hg.id}>{hg.headers.map(h => (
                  <th key={h.id} className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {flexRender(h.column.columnDef.header, h.getContext())}
                  </th>
                ))}</tr>
              ))}
            </thead>
            <tbody className="divide-y">
              {table.getRowModel().rows.map(row => (
                <tr key={row.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                  {row.getVisibleCells().map(cell => (
                    <td key={cell.id} className="px-4 py-3">{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>
                  ))}
                </tr>
              ))}
              {table.getRowModel().rows.length === 0 && (
                <tr><td colSpan={9} className="px-4 py-8 text-center text-gray-400">No claims found.</td></tr>
              )}
            </tbody>
          </table>
        )}
        {/* Pagination */}
        <div className="flex items-center justify-between px-4 py-3 border-t text-sm text-gray-500">
          <span>Showing {((filter.page!-1)*filter.pageSize!)+1}–{Math.min(filter.page!*filter.pageSize!, data?.total??0)} of {data?.total??0}</span>
          <div className="flex gap-2">
            <button disabled={filter.page===1} onClick={() => setFilter(f=>({...f,page:f.page!-1}))}
              className="p-1 rounded disabled:opacity-30 hover:bg-gray-100"><ChevronLeft className="h-4 w-4"/></button>
            <button disabled={filter.page===data?.totalPages} onClick={() => setFilter(f=>({...f,page:f.page!+1}))}
              className="p-1 rounded disabled:opacity-30 hover:bg-gray-100"><ChevronRight className="h-4 w-4"/></button>
          </div>
        </div>
      </div>
    </div>
  )
}
