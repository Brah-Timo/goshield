import { useState, useCallback } from 'react'
import { useDropzone } from 'react-dropzone'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useCreateClaim, useUploadDocument } from '@/hooks/useClaims'
import { useNavigate } from 'react-router-dom'
import {
  UploadCloud, FileText, X, CheckCircle, ChevronRight,
  ChevronLeft, AlertCircle, DollarSign, Calendar, Tag,
  Shield, Loader2,
} from 'lucide-react'
import { extractError } from '@/hooks/useAuth'
import type { CreateClaimRequest, ClaimType } from '@/types'
import { toast } from '@/store/toast'
import clsx from 'clsx'

// ── Validation schema ─────────────────────────────────────────────────────────
const schema = z.object({
  policyNumber:  z.string().min(3, 'Minimum 3 characters').max(50),
  claimType:     z.enum(['HEALTH', 'CAR', 'PROPERTY', 'LIFE', 'TRAVEL', 'OTHER']),
  amount:        z.number({ coerce: true, invalid_type_error: 'Enter a valid number' }).positive('Must be positive').max(10_000_000, 'Max $10M'),
  description:   z.string().min(10, 'At least 10 characters').max(2000),
  incidentDate:  z.string().min(1, 'Required'),
})
type FormData = z.infer<typeof schema>

// ── Step definitions ──────────────────────────────────────────────────────────
const STEPS = [
  { id: 1, label: 'Details',  icon: FileText },
  { id: 2, label: 'Document', icon: UploadCloud },
  { id: 3, label: 'Review',   icon: CheckCircle },
]

// ── Claim type metadata ───────────────────────────────────────────────────────
const CLAIM_TYPES: { value: ClaimType; label: string; emoji: string; desc: string }[] = [
  { value: 'HEALTH',   label: 'Health',   emoji: '🏥', desc: 'Medical and hospitalization claims' },
  { value: 'CAR',      label: 'Car',      emoji: '🚗', desc: 'Vehicle accident or theft' },
  { value: 'PROPERTY', label: 'Property', emoji: '🏠', desc: 'Home and property damage' },
  { value: 'LIFE',     label: 'Life',     emoji: '💛', desc: 'Life insurance claims' },
  { value: 'TRAVEL',   label: 'Travel',   emoji: '✈️', desc: 'Trip cancellation and delays' },
  { value: 'OTHER',    label: 'Other',    emoji: '📋', desc: 'Other insurance types' },
]

// ── Stepper header ────────────────────────────────────────────────────────────
function Stepper({ current }: { current: number }) {
  return (
    <div className="flex items-center justify-center gap-0 mb-8">
      {STEPS.map((step, i) => {
        const done    = current > step.id
        const active  = current === step.id
        const Icon    = step.icon
        return (
          <div key={step.id} className="flex items-center">
            <div className="flex flex-col items-center gap-1">
              <div className={clsx(
                'h-9 w-9 rounded-full flex items-center justify-center text-sm font-bold border-2 transition-all duration-300',
                done   && 'bg-brand-600 border-brand-600 text-white',
                active && 'bg-white dark:bg-gray-900 border-brand-600 text-brand-600',
                !done && !active && 'bg-white dark:bg-gray-900 border-gray-300 dark:border-gray-600 text-gray-400',
              )}>
                {done ? <CheckCircle className="h-4.5 w-4.5" /> : <Icon className="h-4 w-4" />}
              </div>
              <span className={clsx(
                'text-xs font-medium whitespace-nowrap',
                active ? 'text-brand-600 dark:text-brand-400' : done ? 'text-gray-600 dark:text-gray-300' : 'text-gray-400',
              )}>
                {step.label}
              </span>
            </div>
            {i < STEPS.length - 1 && (
              <div className={clsx(
                'h-0.5 w-16 sm:w-24 mx-2 mb-5 transition-colors duration-300',
                current > step.id ? 'bg-brand-600' : 'bg-gray-200 dark:bg-gray-700',
              )} />
            )}
          </div>
        )
      })}
    </div>
  )
}

// ── Review row ────────────────────────────────────────────────────────────────
function ReviewRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between py-2.5 border-b border-gray-100 dark:border-gray-800 last:border-0 gap-4">
      <span className="text-sm text-gray-500 dark:text-gray-400 shrink-0">{label}</span>
      <span className="text-sm font-medium text-gray-800 dark:text-gray-200 text-right break-words">{value}</span>
    </div>
  )
}

// ══════════════════════════════════════════════════════════════════════════════
export default function UploadPage() {
  const [step,     setStep]     = useState(1)
  const [file,     setFile]     = useState<File | null>(null)
  const [claimId,  setClaimId]  = useState<string | null>(null)

  const create   = useCreateClaim()
  const navigate = useNavigate()

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isValid },
    trigger,
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: { claimType: 'HEALTH' },
  })

  const watchedValues = watch()
  const selectedType  = watch('claimType')

  // ── Dropzone ──────────────────────────────────────────────────────────────
  const onDrop = useCallback((accepted: File[]) => {
    if (accepted[0]) setFile(accepted[0])
  }, [])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'application/pdf': ['.pdf'],
      'image/*': ['.jpg', '.jpeg', '.png', '.webp'],
    },
    maxFiles: 1,
    maxSize: 20 * 1024 * 1024, // 20 MB
  })

  // ── Step navigation ───────────────────────────────────────────────────────
  const goNext = async () => {
    if (step === 1) {
      const ok = await trigger(['policyNumber', 'claimType', 'amount', 'incidentDate', 'description'])
      if (!ok) return
    }
    setStep(s => s + 1)
  }

  const goBack = () => setStep(s => s - 1)

  // ── Final submit ──────────────────────────────────────────────────────────
  const onSubmit = async (data: FormData) => {
    try {
      const claim = await create.mutateAsync(data as CreateClaimRequest)
      setClaimId(claim.id)

      if (file) {
        const form = new FormData()
        form.append('document', file)
        try {
          const { getAccessToken } = await import('@/lib/api')
          await fetch(`/claims/v1/claims/${claim.id}/document`, {
            method: 'POST',
            body: form,
            headers: { Authorization: `Bearer ${getAccessToken() ?? ''}` },
          })
        } catch {
          // non-fatal — claim was already created
          toast.warning('Claim submitted but document upload failed', 'Upload warning')
        }
      }

      toast.success('Claim submitted for AI analysis', 'Success')
      navigate(`/claims/${claim.id}`)
    } catch (err) {
      toast.error(extractError(err), 'Submission failed')
    }
  }

  // ── Format file size ──────────────────────────────────────────────────────
  const fmtSize = (b: number) =>
    b < 1024 ? `${b} B` : b < 1_048_576 ? `${(b / 1024).toFixed(1)} KB` : `${(b / 1_048_576).toFixed(1)} MB`

  return (
    <div className="max-w-2xl mx-auto animate-fade-in">
      {/* Page header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <Shield className="h-6 w-6 text-brand-500" />
          New Claim
        </h1>
        <p className="text-sm text-gray-400 mt-0.5">
          Submit a new insurance claim for AI-powered fraud analysis
        </p>
      </div>

      {/* Stepper */}
      <Stepper current={step} />

      <form onSubmit={handleSubmit(onSubmit)}>
        {/* ── Step 1: Details ───────────────────────────────────────────────── */}
        {step === 1 && (
          <div className="bg-white dark:bg-gray-900 rounded-2xl border border-gray-200 dark:border-gray-700 p-6 space-y-5 animate-fade-in">
            <div>
              <h2 className="font-semibold text-gray-900 dark:text-white mb-0.5">Claim Details</h2>
              <p className="text-xs text-gray-400">Fill in all required fields to proceed</p>
            </div>

            {/* Claim type cards */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Claim Type <span className="text-red-500">*</span>
              </label>
              <div className="grid grid-cols-3 sm:grid-cols-6 gap-2">
                {CLAIM_TYPES.map(ct => (
                  <button
                    key={ct.value}
                    type="button"
                    onClick={() => setValue('claimType', ct.value, { shouldValidate: true })}
                    title={ct.desc}
                    className={clsx(
                      'flex flex-col items-center gap-1 p-2.5 rounded-xl border-2 text-xs font-medium transition-all',
                      selectedType === ct.value
                        ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/20 text-brand-700 dark:text-brand-300 shadow-sm'
                        : 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:border-brand-300 hover:bg-brand-50/50 dark:hover:bg-brand-900/10',
                    )}
                  >
                    <span className="text-xl leading-none">{ct.emoji}</span>
                    <span>{ct.label}</span>
                  </button>
                ))}
              </div>
            </div>

            <div className="grid sm:grid-cols-2 gap-5">
              {/* Policy number */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                  Policy Number <span className="text-red-500">*</span>
                </label>
                <div className="relative">
                  <Tag className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
                  <input
                    {...register('policyNumber')}
                    placeholder="POL-2024-XXXXX"
                    className="w-full pl-9 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
                  />
                </div>
                {errors.policyNumber && (
                  <p className="mt-1 text-xs text-danger-500 flex items-center gap-1">
                    <AlertCircle className="h-3 w-3" />{errors.policyNumber.message}
                  </p>
                )}
              </div>

              {/* Incident date */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                  Incident Date <span className="text-red-500">*</span>
                </label>
                <div className="relative">
                  <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
                  <input
                    {...register('incidentDate')}
                    type="date"
                    max={new Date().toISOString().split('T')[0]}
                    className="w-full pl-9 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
                  />
                </div>
                {errors.incidentDate && (
                  <p className="mt-1 text-xs text-danger-500 flex items-center gap-1">
                    <AlertCircle className="h-3 w-3" />{errors.incidentDate.message}
                  </p>
                )}
              </div>
            </div>

            {/* Amount */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                Claim Amount <span className="text-red-500">*</span>
              </label>
              <div className="relative">
                <DollarSign className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
                <input
                  {...register('amount')}
                  type="number"
                  step="0.01"
                  min="1"
                  placeholder="0.00"
                  className="w-full pl-9 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
                />
              </div>
              {errors.amount && (
                <p className="mt-1 text-xs text-danger-500 flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" />{errors.amount.message}
                </p>
              )}
            </div>

            {/* Description */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                Description <span className="text-red-500">*</span>
              </label>
              <textarea
                {...register('description')}
                rows={4}
                maxLength={2000}
                placeholder="Describe the incident in detail — what happened, when, where, and how it relates to your policy…"
                className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100 resize-none"
              />
              <div className="flex items-center justify-between mt-1">
                {errors.description
                  ? <p className="text-xs text-danger-500 flex items-center gap-1"><AlertCircle className="h-3 w-3" />{errors.description.message}</p>
                  : <span />
                }
                <span className="text-xs text-gray-400">{watchedValues.description?.length ?? 0}/2000</span>
              </div>
            </div>
          </div>
        )}

        {/* ── Step 2: Document ─────────────────────────────────────────────── */}
        {step === 2 && (
          <div className="bg-white dark:bg-gray-900 rounded-2xl border border-gray-200 dark:border-gray-700 p-6 space-y-5 animate-fade-in">
            <div>
              <h2 className="font-semibold text-gray-900 dark:text-white mb-0.5">Supporting Document</h2>
              <p className="text-xs text-gray-400">Upload a PDF or image to strengthen your claim (optional)</p>
            </div>

            {file ? (
              <div className="flex items-center gap-4 rounded-xl border-2 border-brand-300 dark:border-brand-600 bg-brand-50 dark:bg-brand-900/20 px-5 py-4">
                <div className="rounded-lg bg-brand-100 dark:bg-brand-900/40 p-3 shrink-0">
                  <FileText className="h-6 w-6 text-brand-600 dark:text-brand-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-800 dark:text-gray-200 truncate">{file.name}</p>
                  <p className="text-xs text-gray-400 mt-0.5">{fmtSize(file.size)} · {file.type || 'Unknown type'}</p>
                </div>
                <button
                  type="button"
                  onClick={() => setFile(null)}
                  className="p-1.5 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            ) : (
              <div
                {...getRootProps()}
                className={clsx(
                  'rounded-2xl border-2 border-dashed p-10 text-center cursor-pointer transition-all duration-200',
                  isDragActive
                    ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/20 scale-[1.01]'
                    : 'border-gray-300 dark:border-gray-600 hover:border-brand-400 hover:bg-brand-50/30 dark:hover:bg-brand-900/10',
                )}
              >
                <input {...getInputProps()} />
                <UploadCloud className={clsx('mx-auto h-12 w-12 mb-3 transition-colors', isDragActive ? 'text-brand-500' : 'text-gray-300 dark:text-gray-600')} />
                {isDragActive ? (
                  <p className="text-sm font-semibold text-brand-600 dark:text-brand-400">Drop it here!</p>
                ) : (
                  <>
                    <p className="text-sm font-medium text-gray-600 dark:text-gray-300">
                      Drag & drop a file, or{' '}
                      <span className="text-brand-600 dark:text-brand-400">browse</span>
                    </p>
                    <p className="text-xs text-gray-400 mt-1.5">PDF, JPG, PNG, WEBP · Max 20 MB</p>
                  </>
                )}
              </div>
            )}

            <div className="rounded-xl bg-gray-50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 p-4 text-xs text-gray-500 dark:text-gray-400 space-y-1">
              <p className="font-semibold text-gray-700 dark:text-gray-300 mb-1.5">Why upload a document?</p>
              <p>• Increases AI model accuracy significantly</p>
              <p>• Accepted evidence: police reports, hospital invoices, photos</p>
              <p>• Files are encrypted at rest and access-controlled per company</p>
            </div>
          </div>
        )}

        {/* ── Step 3: Review ───────────────────────────────────────────────── */}
        {step === 3 && (
          <div className="space-y-4 animate-fade-in">
            <div className="bg-white dark:bg-gray-900 rounded-2xl border border-gray-200 dark:border-gray-700 p-6">
              <h2 className="font-semibold text-gray-900 dark:text-white mb-4">Review & Submit</h2>

              <div className="space-y-0">
                <ReviewRow label="Policy Number"  value={watchedValues.policyNumber ?? '—'} />
                <ReviewRow label="Claim Type"     value={CLAIM_TYPES.find(c => c.value === watchedValues.claimType)?.label ?? watchedValues.claimType ?? '—'} />
                <ReviewRow label="Amount"         value={watchedValues.amount ? `$${Number(watchedValues.amount).toLocaleString()}` : '—'} />
                <ReviewRow label="Incident Date"  value={watchedValues.incidentDate ?? '—'} />
                <ReviewRow label="Description"    value={watchedValues.description?.slice(0, 120) + (watchedValues.description?.length > 120 ? '…' : '') ?? '—'} />
                <ReviewRow label="Document"       value={file ? `${file.name} (${fmtSize(file.size)})` : 'None attached'} />
              </div>
            </div>

            {/* API error */}
            {create.isError && (
              <div className="flex items-start gap-3 rounded-xl bg-danger-50 dark:bg-danger-900/20 border border-danger-500/20 p-4 text-sm text-danger-700 dark:text-danger-300">
                <AlertCircle className="h-5 w-5 mt-0.5 shrink-0" />
                <span>{extractError(create.error)}</span>
              </div>
            )}

            {/* Info note */}
            <div className="flex items-start gap-3 rounded-xl bg-brand-50 dark:bg-brand-900/20 border border-brand-200 dark:border-brand-700/30 p-4 text-xs text-brand-700 dark:text-brand-300">
              <Shield className="h-4 w-4 mt-0.5 shrink-0" />
              <span>
                After submission, our AI model will analyze this claim within seconds.
                You will receive a real-time notification when analysis is complete.
              </span>
            </div>
          </div>
        )}

        {/* ── Navigation buttons ────────────────────────────────────────────── */}
        <div className="flex items-center justify-between mt-6 gap-3">
          <button
            type="button"
            onClick={goBack}
            disabled={step === 1}
            className="flex items-center gap-2 rounded-lg border border-gray-300 dark:border-gray-600 px-5 py-2.5 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            <ChevronLeft className="h-4 w-4" />
            Back
          </button>

          <div className="flex items-center gap-1">
            {STEPS.map(s => (
              <span
                key={s.id}
                className={clsx(
                  'h-1.5 rounded-full transition-all duration-300',
                  step === s.id ? 'w-6 bg-brand-600' : step > s.id ? 'w-3 bg-brand-300' : 'w-3 bg-gray-200 dark:bg-gray-700',
                )}
              />
            ))}
          </div>

          {step < 3 ? (
            <button
              type="button"
              onClick={goNext}
              className="flex items-center gap-2 rounded-lg bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 transition-colors"
            >
              Continue
              <ChevronRight className="h-4 w-4" />
            </button>
          ) : (
            <button
              type="submit"
              disabled={create.isPending}
              className="flex items-center gap-2 rounded-lg bg-brand-600 px-6 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-60 transition-colors"
            >
              {create.isPending ? (
                <><Loader2 className="h-4 w-4 animate-spin" />Submitting…</>
              ) : (
                <><CheckCircle className="h-4 w-4" />Submit Claim</>
              )}
            </button>
          )}
        </div>
      </form>
    </div>
  )
}
