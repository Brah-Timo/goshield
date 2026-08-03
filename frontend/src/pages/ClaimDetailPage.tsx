import { useParams, Link } from 'react-router-dom'
import { useClaim, useReviewClaim } from '@/hooks/useClaims'
import { format, parseISO } from 'date-fns'
import {
  AlertTriangle, CheckCircle, XCircle, FileText, Brain,
  Clock, ChevronRight, ArrowLeft, Info, Copy, ExternalLink,
} from 'lucide-react'
import { useState } from 'react'
import type { RiskFactor, ClaimStatus } from '@/types'
import { toast } from '@/store/toast'

// ── SHAP bar ──────────────────────────────────────────────────────────────
function ShapBar({ factor }: { factor: RiskFactor }) {
  const pct      = Math.min(Math.abs(factor.shapValue) * 100, 100)
  const positive = factor.direction === 'INCREASES_RISK'
  return (
    <div className="flex items-center gap-3 text-sm group">
      <div className="w-40 shrink-0">
        <p className="text-xs text-gray-600 dark:text-gray-400 truncate" title={factor.feature}>
          {factor.feature}
        </p>
        <p className="text-[10px] text-gray-400 dark:text-gray-500 truncate">
          val: {factor.featureValue}
        </p>
      </div>
      <div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-2.5 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${positive ? 'bg-red-500' : 'bg-green-500'}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className={`w-14 text-right text-xs font-mono font-medium shrink-0 ${positive ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'}`}>
        {positive ? '+' : ''}{factor.shapValue.toFixed(3)}
      </span>
      <span className={`shrink-0 text-[10px] font-medium px-1.5 py-0.5 rounded ${
        positive
          ? 'bg-red-100 text-red-600 dark:bg-red-900/40 dark:text-red-300'
          : 'bg-green-100 text-green-600 dark:bg-green-900/40 dark:text-green-300'
      }`}>
        {positive ? '↑ risk' : '↓ risk'}
      </span>
    </div>
  )
}

// ── Claim timeline ─────────────────────────────────────────────────────────
function ClaimTimeline({ status, createdAt, reviewedAt }: {
  status: ClaimStatus
  createdAt: string
  reviewedAt?: string
}) {
  const steps: { label: ClaimStatus | 'CREATED'; date?: string; done: boolean }[] = [
    { label: 'CREATED',    date: createdAt, done: true },
    { label: 'PENDING',    done: ['PROCESSING','FLAGGED','APPROVED','REJECTED','MORE_INFO'].includes(status) },
    { label: 'PROCESSING', done: ['FLAGGED','APPROVED','REJECTED'].includes(status) },
    { label: 'FLAGGED',    done: ['APPROVED','REJECTED','MORE_INFO'].includes(status) || status === 'FLAGGED' },
    { label: 'APPROVED',   date: reviewedAt, done: status === 'APPROVED' },
  ]

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
      <h2 className="font-semibold flex items-center gap-2 mb-4 text-gray-900 dark:text-white">
        <Clock className="h-4 w-4 text-gray-400" />
        Status Timeline
      </h2>
      <div className="relative">
        {/* Connecting line */}
        <div className="absolute left-3.5 top-3 bottom-3 w-px bg-gray-200 dark:bg-gray-700" />
        <div className="space-y-4">
          {steps.map((step, i) => (
            <div key={i} className="flex items-start gap-3 relative">
              <div className={`h-7 w-7 rounded-full flex items-center justify-center shrink-0 z-10 border-2 transition-colors ${
                step.done
                  ? step.label === 'APPROVED' ? 'bg-green-500 border-green-500 text-white'
                    : 'bg-brand-600 border-brand-600 text-white'
                  : status === step.label
                    ? 'bg-amber-500 border-amber-500 text-white animate-pulse-slow'
                    : 'bg-white dark:bg-gray-900 border-gray-300 dark:border-gray-600'
              }`}>
                {step.done ? (
                  <CheckCircle className="h-3.5 w-3.5" />
                ) : status === step.label ? (
                  <Clock className="h-3.5 w-3.5" />
                ) : (
                  <span className="h-2 w-2 rounded-full bg-gray-300 dark:bg-gray-600" />
                )}
              </div>
              <div className="pt-0.5 min-w-0">
                <p className={`text-sm font-medium ${
                  step.done || status === step.label
                    ? 'text-gray-800 dark:text-gray-200'
                    : 'text-gray-400 dark:text-gray-500'
                }`}>
                  {step.label}
                </p>
                {step.date && (
                  <p className="text-xs text-gray-400 dark:text-gray-500 mt-0.5">
                    {format(parseISO(step.date), 'MMM d, yyyy HH:mm')}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

// ── Detail row ─────────────────────────────────────────────────────────────
function DetailRow({ label, value, mono }: { label: string; value: React.ReactNode; mono?: boolean }) {
  return (
    <div className="flex justify-between items-start gap-4 py-2.5 border-b border-gray-50 dark:border-gray-800 last:border-0">
      <span className="text-sm text-gray-500 dark:text-gray-400 shrink-0">{label}</span>
      <span className={`text-sm font-medium text-right ${mono ? 'font-mono text-xs text-gray-400 break-all' : 'text-gray-800 dark:text-gray-200'}`}>
        {value}
      </span>
    </div>
  )
}

// ── Risk color helper ─────────────────────────────────────────────────────
function riskStyle(level?: string) {
  return level === 'CRITICAL' ? { bg: 'bg-purple-50 dark:bg-purple-900/20', text: 'text-purple-700 dark:text-purple-300', border: 'border-purple-200 dark:border-purple-700' }
       : level === 'HIGH'     ? { bg: 'bg-red-50 dark:bg-red-900/20',       text: 'text-red-700 dark:text-red-300',       border: 'border-red-200 dark:border-red-700' }
       : level === 'MEDIUM'   ? { bg: 'bg-amber-50 dark:bg-amber-900/20',   text: 'text-amber-700 dark:text-amber-300',   border: 'border-amber-200 dark:border-amber-700' }
       :                        { bg: 'bg-green-50 dark:bg-green-900/20',   text: 'text-green-700 dark:text-green-300',   border: 'border-green-200 dark:border-green-700' }
}

// ══════════════════════════════════════════════════════════════════════════
export default function ClaimDetailPage() {
  const { id }              = useParams<{ id: string }>()
  const { data: claim, isLoading } = useClaim(id!)
  const review              = useReviewClaim()
  const [notes, setNotes]   = useState('')
  const [copied, setCopied] = useState(false)

  if (isLoading) {
    return (
      <div className="max-w-4xl space-y-4 animate-pulse">
        <div className="h-8 w-64 bg-gray-200 dark:bg-gray-700 rounded" />
        <div className="grid md:grid-cols-2 gap-4">
          <div className="h-64 bg-gray-100 dark:bg-gray-800 rounded-xl" />
          <div className="h-64 bg-gray-100 dark:bg-gray-800 rounded-xl" />
        </div>
      </div>
    )
  }
  if (!claim) {
    return (
      <div className="max-w-xl mx-auto mt-16 text-center">
        <AlertTriangle className="h-12 w-12 text-gray-300 mx-auto mb-4" />
        <h2 className="text-lg font-semibold text-gray-700 dark:text-gray-300">Claim not found</h2>
        <p className="text-sm text-gray-400 mt-1">This claim may have been deleted or you don't have access.</p>
        <Link to="/claims" className="mt-4 inline-flex items-center gap-2 text-sm text-brand-600 hover:underline">
          <ArrowLeft className="h-4 w-4" /> Back to Claims
        </Link>
      </div>
    )
  }

  const rs     = riskStyle(claim.riskLevel)
  const score  = claim.fraudScore != null ? claim.fraudScore * 100 : null
  const gauge  = score != null ? score : 0
  const gaugeColor =
    gauge > 75 ? '#7c3aed' : gauge > 50 ? '#ef4444' : gauge > 25 ? '#f59e0b' : '#22c55e'

  const copyId = () => {
    navigator.clipboard.writeText(claim.id)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const handleReview = (action: 'APPROVE' | 'REJECT' | 'REQUEST_MORE_INFO') => {
    review.mutate(
      { id: claim.id, action, notes: notes.trim() || undefined },
      {
        onSuccess: () => {
          toast.success(
            action === 'APPROVE'   ? 'Claim approved successfully' :
            action === 'REJECT'    ? 'Claim rejected' :
            'More information requested',
            'Review submitted'
          )
          setNotes('')
        },
        onError: (e) => toast.error(String(e), 'Review failed'),
      }
    )
  }

  return (
    <div className="max-w-4xl space-y-5 animate-fade-in">
      {/* Back + header */}
      <div>
        <Link to="/claims" className="inline-flex items-center gap-1 text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 mb-3 transition-colors">
          <ArrowLeft className="h-3.5 w-3.5" /> Back to Claims
        </Link>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                Claim {claim.id.slice(0, 8)}…
              </h1>
              <button onClick={copyId} title="Copy full ID" className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                {copied ? <CheckCircle className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
              </button>
            </div>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5 flex items-center gap-2">
              {claim.claimType}
              <ChevronRight className="h-3.5 w-3.5 text-gray-300" />
              {format(parseISO(claim.createdAt), 'MMMM d, yyyy HH:mm')}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {claim.riskLevel && (
              <span className={`rounded-full px-4 py-1.5 text-sm font-semibold border ${rs.bg} ${rs.text} ${rs.border}`}>
                {claim.riskLevel} RISK
              </span>
            )}
            <span className="rounded-full px-3 py-1.5 text-xs font-semibold bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300">
              {claim.status}
            </span>
          </div>
        </div>
      </div>

      {/* Main grid */}
      <div className="grid md:grid-cols-2 gap-5">
        {/* ── Claim Details ─────────────────────────────────────────────── */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
          <h2 className="font-semibold flex items-center gap-2 text-gray-900 dark:text-white mb-1">
            <FileText className="h-4 w-4 text-gray-400" />
            Claim Details
          </h2>
          <div className="mt-2">
            <DetailRow label="Claim ID"      value={claim.id} mono />
            <DetailRow label="Policy Number" value={claim.policyNumber} />
            <DetailRow label="Claim Type"    value={claim.claimType} />
            <DetailRow label="Amount"        value={<span className="font-bold text-gray-900 dark:text-white">${claim.amount.toLocaleString()}</span>} />
            <DetailRow label="Incident Date" value={format(parseISO(claim.incidentDate), 'MMMM d, yyyy')} />
            <DetailRow label="Status"        value={
              <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                claim.status === 'FLAGGED' ? 'bg-red-100 text-red-700' :
                claim.status === 'APPROVED' ? 'bg-green-100 text-green-700' :
                'bg-gray-100 text-gray-600'
              }`}>{claim.status}</span>
            } />
            {claim.reviewedAt && (
              <DetailRow label="Reviewed At" value={format(parseISO(claim.reviewedAt), 'MMM d, yyyy HH:mm')} />
            )}
            {claim.reviewedBy && (
              <DetailRow label="Reviewer" value={claim.reviewedBy} mono />
            )}
          </div>

          {/* Description */}
          <div className="mt-3 rounded-lg bg-gray-50 dark:bg-gray-800 p-3">
            <p className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Description</p>
            <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">{claim.description}</p>
          </div>

          {/* Analyst notes */}
          {claim.reviewNotes && (
            <div className="mt-3 rounded-lg bg-brand-50 dark:bg-brand-900/20 border border-brand-200 dark:border-brand-700 p-3">
              <p className="text-xs font-medium text-brand-600 dark:text-brand-400 mb-1 flex items-center gap-1">
                <Info className="h-3 w-3" />
                Analyst Notes
              </p>
              <p className="text-sm text-gray-700 dark:text-gray-300">{claim.reviewNotes}</p>
            </div>
          )}

          {/* Document link */}
          {claim.documentUrl && (
            <a
              href={claim.documentUrl}
              target="_blank"
              rel="noreferrer"
              className="mt-4 flex items-center gap-2 text-sm text-brand-600 dark:text-brand-400 hover:underline"
            >
              <ExternalLink className="h-4 w-4" />
              View Supporting Document
            </a>
          )}
        </div>

        {/* ── AI Analysis ───────────────────────────────────────────────── */}
        {score != null ? (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5 space-y-4">
            <h2 className="font-semibold flex items-center gap-2 text-gray-900 dark:text-white">
              <Brain className="h-4 w-4 text-gray-400" />
              AI Fraud Analysis
            </h2>

            {/* Gauge */}
            <div className="flex items-center gap-6">
              <div className="relative h-24 w-24 shrink-0">
                <svg className="h-24 w-24 -rotate-90" viewBox="0 0 36 36">
                  <circle cx="18" cy="18" r="15.9" fill="none" stroke="#e5e7eb" strokeWidth="3.5" className="dark:[stroke:#374151]" />
                  <circle
                    cx="18" cy="18" r="15.9" fill="none"
                    stroke={gaugeColor}
                    strokeWidth="3.5"
                    strokeDasharray={`${gauge} 100`}
                    strokeLinecap="round"
                    style={{ transition: 'stroke-dasharray 0.6s ease' }}
                  />
                </svg>
                <div className="absolute inset-0 flex flex-col items-center justify-center">
                  <span className="text-xl font-bold text-gray-900 dark:text-white" style={{ color: gaugeColor }}>
                    {gauge.toFixed(0)}%
                  </span>
                  <span className="text-[10px] text-gray-400">fraud</span>
                </div>
              </div>
              <div className="space-y-1.5 min-w-0">
                <div>
                  <p className="text-xs text-gray-400 dark:text-gray-500">Risk Level</p>
                  <p className={`font-bold text-lg ${rs.text}`}>{claim.riskLevel ?? '—'}</p>
                </div>
                {claim.confidence != null && (
                  <div>
                    <p className="text-xs text-gray-400 dark:text-gray-500">Model Confidence</p>
                    <div className="flex items-center gap-2">
                      <div className="h-1.5 flex-1 bg-gray-100 dark:bg-gray-800 rounded-full overflow-hidden">
                        <div className="h-full bg-brand-500 rounded-full" style={{ width: `${(claim.confidence * 100).toFixed(0)}%` }} />
                      </div>
                      <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
                        {(claim.confidence * 100).toFixed(0)}%
                      </span>
                    </div>
                  </div>
                )}
                {claim.modelVersion && (
                  <p className="text-[10px] text-gray-400">
                    Model v{claim.modelVersion}
                  </p>
                )}
              </div>
            </div>

            {/* Fraud reason */}
            {claim.fraudReason && (
              <div className={`rounded-lg border p-3 text-sm ${rs.bg} ${rs.border}`}>
                <p className="font-medium text-xs text-gray-500 dark:text-gray-400 mb-1 flex items-center gap-1">
                  <AlertTriangle className="h-3 w-3" /> Fraud Reason
                </p>
                <p className={`${rs.text} leading-relaxed`}>{claim.fraudReason}</p>
              </div>
            )}

            {/* SHAP values */}
            {claim.shapValues && claim.shapValues.length > 0 && (
              <div className="space-y-2.5">
                <p className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Feature Importance (SHAP)
                </p>
                <div className="space-y-2">
                  {claim.shapValues.slice(0, 8).map((f: RiskFactor) => (
                    <ShapBar key={f.feature} factor={f} />
                  ))}
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5 flex flex-col items-center justify-center gap-3 text-center">
            <Brain className="h-10 w-10 text-gray-200 dark:text-gray-700" />
            <p className="text-sm text-gray-400">AI analysis pending…</p>
            <p className="text-xs text-gray-300 dark:text-gray-600">
              The fraud model will process this claim shortly after submission.
            </p>
          </div>
        )}
      </div>

      {/* Timeline + Review row */}
      <div className="grid md:grid-cols-2 gap-5">
        {/* Timeline */}
        <ClaimTimeline
          status={claim.status}
          createdAt={claim.createdAt}
          reviewedAt={claim.reviewedAt}
        />

        {/* ── Analyst Review Panel ────────────────────────────────────────── */}
        {['FLAGGED', 'MORE_INFO'].includes(claim.status) ? (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
            <h2 className="font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
              <CheckCircle className="h-4 w-4 text-gray-400" />
              Analyst Review
            </h2>

            {/* Notes textarea */}
            <div className="mb-4">
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">
                Review Notes <span className="text-gray-400">(optional)</span>
              </label>
              <textarea
                value={notes}
                onChange={e => setNotes(e.target.value)}
                rows={3}
                maxLength={2000}
                placeholder="Add observations, reasons, or additional context…"
                className="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none"
              />
              <p className="text-[10px] text-gray-400 mt-1 text-right">{notes.length}/2000</p>
            </div>

            {/* Action buttons */}
            <div className="flex flex-wrap gap-2">
              <button
                onClick={() => handleReview('APPROVE')}
                disabled={review.isPending}
                className="flex items-center gap-2 rounded-lg bg-green-600 hover:bg-green-700 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50 transition-colors"
              >
                <CheckCircle className="h-4 w-4" />
                Approve
              </button>
              <button
                onClick={() => handleReview('REJECT')}
                disabled={review.isPending}
                className="flex items-center gap-2 rounded-lg bg-red-600 hover:bg-red-700 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50 transition-colors"
              >
                <XCircle className="h-4 w-4" />
                Reject
              </button>
              <button
                onClick={() => handleReview('REQUEST_MORE_INFO')}
                disabled={review.isPending}
                className="flex items-center gap-2 rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-800 px-4 py-2.5 text-sm font-semibold text-gray-700 dark:text-gray-300 disabled:opacity-50 transition-colors"
              >
                <AlertTriangle className="h-4 w-4" />
                Request Info
              </button>
            </div>

            {review.isPending && (
              <p className="text-xs text-gray-400 mt-3 flex items-center gap-1.5">
                <Clock className="h-3.5 w-3.5 animate-spin" /> Submitting review…
              </p>
            )}
          </div>
        ) : (
          <div className="bg-gray-50 dark:bg-gray-900/50 rounded-xl border border-dashed border-gray-200 dark:border-gray-700 p-5 flex flex-col items-center justify-center gap-2 text-center">
            <Info className="h-8 w-8 text-gray-200 dark:text-gray-700" />
            <p className="text-sm text-gray-400">
              {['APPROVED', 'REJECTED'].includes(claim.status)
                ? `This claim has been ${claim.status.toLowerCase()}.`
                : 'Review panel will appear when this claim is flagged.'}
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
