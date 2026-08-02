import { useState, useCallback } from 'react'
import { useDropzone } from 'react-dropzone'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useCreateClaim, useUploadDocument } from '@/hooks/useClaims'
import { useNavigate } from 'react-router-dom'
import { UploadCloud, File, X } from 'lucide-react'
import { extractError } from '@/hooks/useAuth'
import type { CreateClaimRequest } from '@/types'

const schema = z.object({
  policyNumber:  z.string().min(1,'Required'),
  claimType:     z.enum(['MEDICAL','AUTO','PROPERTY','LIFE','LIABILITY','OTHER']),
  amount:        z.number({ coerce: true }).positive('Must be positive'),
  description:   z.string().min(10,'At least 10 characters'),
  incidentDate:  z.string().min(1,'Required'),
})

export default function UploadPage() {
  const [file, setFile] = useState<File | null>(null)
  const create = useCreateClaim()
  const navigate = useNavigate()

  const { register, handleSubmit, formState: { errors } } = useForm<CreateClaimRequest>({
    resolver: zodResolver(schema),
  })

  const onDrop = useCallback((accepted: File[]) => {
    if (accepted[0]) setFile(accepted[0])
  }, [])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop, accept: { 'application/pdf': ['.pdf'], 'image/*': ['.jpg','.jpeg','.png'] }, maxFiles: 1,
  })

  const onSubmit = async (data: CreateClaimRequest) => {
    const claim = await create.mutateAsync(data)
    // If a document was attached, upload it immediately
    if (file) {
      const form = new FormData()
      form.append('document', file)
      await fetch(`/claims/v1/claims/${claim.id}/document`, {
        method: 'POST', body: form,
        headers: { Authorization: `Bearer ${localStorage.getItem('at') ?? ''}` }
      })
    }
    navigate(`/claims/${claim.id}`)
  }

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">New Claim</h1>
        <p className="text-gray-500 text-sm mt-1">Submit a new insurance claim for fraud analysis.</p>
      </div>

      {create.isError && (
        <div className="rounded-lg bg-danger-50 border border-danger-500/20 p-3 text-sm text-danger-600">
          {extractError(create.error)}
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)} className="bg-white dark:bg-gray-900 rounded-xl border p-6 space-y-5">
        <div className="grid sm:grid-cols-2 gap-5">
          <div>
            <label className="block text-sm font-medium mb-1.5">Policy Number</label>
            <input {...register('policyNumber')} className="w-full rounded-lg border px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"/>
            {errors.policyNumber && <p className="mt-1 text-xs text-danger-500">{errors.policyNumber.message}</p>}
          </div>
          <div>
            <label className="block text-sm font-medium mb-1.5">Claim Type</label>
            <select {...register('claimType')} className="w-full rounded-lg border px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500">
              {['MEDICAL','AUTO','PROPERTY','LIFE','LIABILITY','OTHER'].map(t => <option key={t}>{t}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1.5">Claim Amount ($)</label>
            <input {...register('amount')} type="number" step="0.01" className="w-full rounded-lg border px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"/>
            {errors.amount && <p className="mt-1 text-xs text-danger-500">{errors.amount.message}</p>}
          </div>
          <div>
            <label className="block text-sm font-medium mb-1.5">Incident Date</label>
            <input {...register('incidentDate')} type="date" className="w-full rounded-lg border px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"/>
            {errors.incidentDate && <p className="mt-1 text-xs text-danger-500">{errors.incidentDate.message}</p>}
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1.5">Description</label>
          <textarea {...register('description')} rows={4} className="w-full rounded-lg border px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"/>
          {errors.description && <p className="mt-1 text-xs text-danger-500">{errors.description.message}</p>}
        </div>

        {/* Dropzone */}
        <div>
          <label className="block text-sm font-medium mb-1.5">Supporting Document <span className="text-gray-400">(optional)</span></label>
          {file ? (
            <div className="flex items-center gap-3 rounded-lg border border-brand-300 bg-brand-50 px-4 py-3">
              <File className="h-5 w-5 text-brand-600"/>
              <span className="flex-1 text-sm truncate">{file.name}</span>
              <button type="button" onClick={() => setFile(null)}><X className="h-4 w-4 text-gray-400"/></button>
            </div>
          ) : (
            <div {...getRootProps()} className={`rounded-xl border-2 border-dashed p-8 text-center cursor-pointer transition-colors ${isDragActive ? 'border-brand-500 bg-brand-50' : 'border-gray-300 hover:border-brand-400'}`}>
              <input {...getInputProps()} />
              <UploadCloud className="mx-auto h-8 w-8 text-gray-400 mb-2"/>
              <p className="text-sm text-gray-600">{isDragActive ? 'Drop here…' : 'Drag & drop a PDF or image, or click to select'}</p>
            </div>
          )}
        </div>

        <button type="submit" disabled={create.isPending}
          className="w-full rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">
          {create.isPending ? 'Submitting…' : 'Submit Claim for Analysis'}
        </button>
      </form>
    </div>
  )
}
