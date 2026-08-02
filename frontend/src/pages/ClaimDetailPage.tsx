import { useParams } from 'react-router-dom'
import { useClaim, useReviewClaim } from '@/hooks/useClaims'
import { format } from 'date-fns'
import { AlertTriangle, CheckCircle, XCircle, FileText, Brain } from 'lucide-react'
import type { RiskFactor } from '@/types'

function ShapBar({ factor }: { factor: RiskFactor }) {
  const pct = Math.min(Math.abs(factor.shapValue) * 100, 100)
  const positive = factor.direction === 'INCREASES_RISK'
  return (
    <div className="flex items-center gap-3 text-sm">
      <span className="w-40 truncate text-gray-600 dark:text-gray-400 text-xs">{factor.feature}</span>
      <div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden">
        <div className={`h-full rounded-full ${positive ? 'bg-red-500' : 'bg-green-500'}`} style={{ width: `${pct}%` }} />
      </div>
      <span className={`w-12 text-right text-xs font-medium ${positive ? 'text-red-600' : 'text-green-600'}`}>
        {factor.shapValue.toFixed(3)}
      </span>
    </div>
  )
}

export default function ClaimDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { data: claim, isLoading } = useClaim(id!)
  const review = useReviewClaim()

  if (isLoading) return <div className="p-8 text-center text-gray-400">Loading…</div>
  if (!claim) return <div className="p-8 text-center text-gray-400">Claim not found.</div>

  const riskColor =
    claim.riskLevel === 'CRITICAL' ? 'text-purple-600 bg-purple-50' :
    claim.riskLevel === 'HIGH'     ? 'text-red-600 bg-red-50' :
    claim.riskLevel === 'MEDIUM'   ? 'text-amber-600 bg-amber-50' :
                                     'text-green-600 bg-green-50'

  return (
    <div className="max-w-4xl space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">Claim {claim.id.slice(0,8)}…</h1>
          <p className="text-gray-500 text-sm mt-1">{claim.claimType} · {format(new Date(claim.createdAt), 'MMMM d, yyyy HH:mm')}</p>
        </div>
        <span className={`rounded-full px-3 py-1 text-sm font-semibold ${riskColor}`}>{claim.riskLevel ?? claim.status}</span>
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        {/* Claim details */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border p-5 space-y-3">
          <h2 className="font-semibold flex items-center gap-2"><FileText className="h-4 w-4"/>Claim Details</h2>
          {([
            ['Policy Number', claim.policyNumber],
            ['Amount', `$${claim.amount.toLocaleString()}`],
            ['Incident Date', format(new Date(claim.incidentDate), 'MMMM d, yyyy')],
            ['Status', claim.status],
          ] as [string,string][]).map(([k,v]) => (
            <div key={k} className="flex justify-between text-sm">
              <span className="text-gray-500">{k}</span>
              <span className="font-medium">{v}</span>
            </div>
          ))}
          <div className="pt-2 text-sm text-gray-600 dark:text-gray-300 border-t">
            <p className="font-medium mb-1">Description</p>
            <p>{claim.description}</p>
          </div>
          {claim.documentUrl && (
            <a href={claim.documentUrl} target="_blank" rel="noreferrer"
              className="inline-flex items-center gap-2 text-sm text-brand-600 hover:underline mt-2">
              <FileText className="h-4 w-4"/>View Document
            </a>
          )}
        </div>

        {/* AI Analysis card */}
        {claim.fraudScore !== undefined && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border p-5 space-y-4">
            <h2 className="font-semibold flex items-center gap-2"><Brain className="h-4 w-4"/>AI Analysis</h2>
            <div className="flex items-center gap-4">
              <div className="relative h-20 w-20">
                <svg className="h-20 w-20 -rotate-90" viewBox="0 0 36 36">
                  <circle cx="18" cy="18" r="15.9" fill="none" stroke="#e5e7eb" strokeWidth="3"/>
                  <circle cx="18" cy="18" r="15.9" fill="none"
                    stroke={claim.riskLevel==='CRITICAL'?'#7c3aed':claim.riskLevel==='HIGH'?'#ef4444':claim.riskLevel==='MEDIUM'?'#f59e0b':'#22c55e'}
                    strokeWidth="3" strokeDasharray={`${claim.fraudScore*100} 100`} strokeLinecap="round"/>
                </svg>
                <span className="absolute inset-0 flex items-center justify-center text-lg font-bold">
                  {(claim.fraudScore * 100).toFixed(0)}%
                </span>
              </div>
              <div>
                <p className="text-sm text-gray-500">Fraud Score</p>
                <p className={`font-bold text-lg ${riskColor.split(' ')[0]}`}>{claim.riskLevel}</p>
                <p className="text-xs text-gray-400 mt-1">Model v{claim.modelVersion} · {((claim.confidence??0)*100).toFixed(0)}% confidence</p>
              </div>
            </div>
            {claim.fraudReason && (
              <div className="rounded-lg bg-gray-50 dark:bg-gray-800 p-3 text-sm text-gray-600 dark:text-gray-300">
                {claim.fraudReason}
              </div>
            )}
            {claim.shapValues && claim.shapValues.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-medium text-gray-500 uppercase">Feature Importance (SHAP)</p>
                {claim.shapValues.map((f: RiskFactor) => <ShapBar key={f.feature} factor={f}/>)}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Review panel */}
      {['ANALYZED','FLAGGED'].includes(claim.status) && (
        <div className="bg-white dark:bg-gray-900 rounded-xl border p-5">
          <h2 className="font-semibold mb-4">Analyst Review</h2>
          <div className="flex gap-3">
            <button onClick={() => review.mutate({ id: claim.id, action: 'APPROVE' })}
              className="flex items-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white hover:bg-green-700 disabled:opacity-50"
              disabled={review.isPending}>
              <CheckCircle className="h-4 w-4"/>Approve
            </button>
            <button onClick={() => review.mutate({ id: claim.id, action: 'REJECT' })}
              className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-50"
              disabled={review.isPending}>
              <XCircle className="h-4 w-4"/>Reject
            </button>
            <button onClick={() => review.mutate({ id: claim.id, action: 'REQUEST_MORE_INFO' })}
              className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-semibold hover:bg-gray-50 disabled:opacity-50"
              disabled={review.isPending}>
              <AlertTriangle className="h-4 w-4"/>Request Info
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
