import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Link } from 'react-router-dom'
import { Shield, Eye, EyeOff, CheckCircle, AlertCircle } from 'lucide-react'
import { useRegister, extractError } from '@/hooks/useAuth'
import type { RegisterRequest } from '@/types'

// ── Zod validation schema ─────────────────────────────────────────────────────
const schema = z
  .object({
    firstName:  z.string().min(1, 'First name required').max(50),
    lastName:   z.string().min(1, 'Last name required').max(50),
    email:      z.string().email('Valid email required'),
    password:   z.string()
      .min(8, 'Minimum 8 characters')
      .max(72, 'Maximum 72 characters')
      .regex(/[A-Z]/, 'Must contain an uppercase letter')
      .regex(/[0-9]/, 'Must contain a number'),
    confirmPassword: z.string(),
    companyId:  z.string().uuid('Must be a valid UUID (e.g. from your company admin)'),
    role: z.enum(['ANALYST', 'VIEWER', 'ADMIN']).optional(),
  })
  .refine(d => d.password === d.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword'],
  })

type FormData = z.infer<typeof schema>

// ── Password strength meter ───────────────────────────────────────────────────
function passwordStrength(pw: string): { score: number; label: string; color: string } {
  if (!pw) return { score: 0, label: '', color: '' }
  let score = 0
  if (pw.length >= 8)  score++
  if (pw.length >= 12) score++
  if (/[A-Z]/.test(pw)) score++
  if (/[0-9]/.test(pw)) score++
  if (/[^A-Za-z0-9]/.test(pw)) score++
  if (score <= 1) return { score, label: 'Weak',    color: 'bg-red-500' }
  if (score <= 3) return { score, label: 'Fair',    color: 'bg-amber-500' }
  if (score === 4) return { score, label: 'Good',   color: 'bg-brand-500' }
  return                { score, label: 'Strong', color: 'bg-green-500' }
}

// ── Inline field error ────────────────────────────────────────────────────────
function FieldError({ msg }: { msg?: string }) {
  if (!msg) return null
  return (
    <p className="mt-1 flex items-center gap-1 text-xs text-danger-500">
      <AlertCircle className="h-3 w-3 shrink-0" />
      {msg}
    </p>
  )
}

// ── InputWrapper ──────────────────────────────────────────────────────────────
function InputLabel({ label, required }: { label: string; required?: boolean }) {
  return (
    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
      {label}
      {required && <span className="ml-0.5 text-danger-500">*</span>}
    </label>
  )
}

// ══════════════════════════════════════════════════════════════════════════════
export default function RegisterPage() {
  const register_mutation = useRegister()
  const [showPw,    setShowPw]    = useState(false)
  const [showCpw,   setShowCpw]   = useState(false)
  const [pwValue,   setPwValue]   = useState('')

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { role: 'ANALYST' },
  })

  const strength = passwordStrength(pwValue)

  const onSubmit = (d: FormData) => {
    const payload: RegisterRequest = {
      email:     d.email,
      password:  d.password,
      firstName: d.firstName,
      lastName:  d.lastName,
      companyId: d.companyId,
    }
    register_mutation.mutate(payload)
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-brand-950 via-gray-900 to-gray-950 px-4 py-10">
      <div className="w-full max-w-lg">
        {/* Logo */}
        <div className="flex justify-center mb-8">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-600 shadow-lg">
              <Shield className="h-6 w-6 text-white" />
            </div>
            <span className="text-3xl font-bold text-white tracking-tight">GoShield</span>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-900 rounded-2xl shadow-2xl p-8">
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Create account</h1>
            <p className="text-gray-500 dark:text-gray-400 text-sm mt-1">
              Join the GoShield Fraud Detection Platform
            </p>
          </div>

          {/* API error banner */}
          {register_mutation.isError && (
            <div className="mb-5 flex items-start gap-3 rounded-lg bg-danger-50 border border-danger-500/20 p-3 text-sm text-danger-700 dark:bg-danger-900/20 dark:border-danger-600/20 dark:text-danger-300">
              <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
              <span>{extractError(register_mutation.error)}</span>
            </div>
          )}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
            {/* Name row */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <InputLabel label="First name" required />
                <input
                  {...register('firstName')}
                  type="text"
                  autoComplete="given-name"
                  placeholder="Jane"
                  className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100 placeholder-gray-400"
                />
                <FieldError msg={errors.firstName?.message} />
              </div>
              <div>
                <InputLabel label="Last name" required />
                <input
                  {...register('lastName')}
                  type="text"
                  autoComplete="family-name"
                  placeholder="Doe"
                  className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100 placeholder-gray-400"
                />
                <FieldError msg={errors.lastName?.message} />
              </div>
            </div>

            {/* Email */}
            <div>
              <InputLabel label="Email address" required />
              <input
                {...register('email')}
                type="email"
                autoComplete="email"
                placeholder="jane@company.com"
                className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100 placeholder-gray-400"
              />
              <FieldError msg={errors.email?.message} />
            </div>

            {/* Password */}
            <div>
              <InputLabel label="Password" required />
              <div className="relative">
                <input
                  {...register('password', {
                    onChange: e => setPwValue(e.target.value),
                  })}
                  type={showPw ? 'text' : 'password'}
                  autoComplete="new-password"
                  placeholder="Min. 8 characters"
                  className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2.5 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100 placeholder-gray-400"
                />
                <button
                  type="button"
                  onClick={() => setShowPw(v => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                >
                  {showPw ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {/* Strength bar */}
              {pwValue && (
                <div className="mt-1.5 space-y-1">
                  <div className="flex gap-1">
                    {[1, 2, 3, 4, 5].map(i => (
                      <div
                        key={i}
                        className={`h-1 flex-1 rounded-full transition-all duration-300 ${
                          i <= strength.score ? strength.color : 'bg-gray-200 dark:bg-gray-700'
                        }`}
                      />
                    ))}
                  </div>
                  {strength.label && (
                    <p className="text-xs text-gray-400">
                      Password strength:{' '}
                      <span className={`font-medium ${
                        strength.score <= 1 ? 'text-red-500' :
                        strength.score <= 3 ? 'text-amber-500' :
                        strength.score === 4 ? 'text-brand-500' : 'text-green-500'
                      }`}>
                        {strength.label}
                      </span>
                    </p>
                  )}
                </div>
              )}
              <FieldError msg={errors.password?.message} />
            </div>

            {/* Confirm password */}
            <div>
              <InputLabel label="Confirm password" required />
              <div className="relative">
                <input
                  {...register('confirmPassword')}
                  type={showCpw ? 'text' : 'password'}
                  autoComplete="new-password"
                  placeholder="Re-enter password"
                  className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2.5 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100 placeholder-gray-400"
                />
                <button
                  type="button"
                  onClick={() => setShowCpw(v => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                >
                  {showCpw ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              <FieldError msg={errors.confirmPassword?.message} />
            </div>

            {/* Company ID */}
            <div>
              <InputLabel label="Company ID" required />
              <input
                {...register('companyId')}
                type="text"
                placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100 placeholder-gray-400 font-mono"
              />
              <p className="mt-1 text-xs text-gray-400">UUID provided by your company administrator</p>
              <FieldError msg={errors.companyId?.message} />
            </div>

            {/* Role */}
            <div>
              <InputLabel label="Role" />
              <select
                {...register('role')}
                className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 text-gray-900 dark:text-gray-100"
              >
                <option value="ANALYST">Analyst</option>
                <option value="VIEWER">Viewer (read-only)</option>
                <option value="ADMIN">Admin</option>
              </select>
              <FieldError msg={errors.role?.message} />
            </div>

            {/* Terms notice */}
            <div className="flex items-start gap-2 rounded-lg bg-brand-50 dark:bg-brand-900/20 border border-brand-100 dark:border-brand-800/30 p-3">
              <CheckCircle className="h-4 w-4 text-brand-500 mt-0.5 shrink-0" />
              <p className="text-xs text-brand-700 dark:text-brand-300">
                By registering, your account will be tied to the Company ID above and subject to your
                organization's fraud detection policies.
              </p>
            </div>

            {/* Submit */}
            <button
              type="submit"
              disabled={register_mutation.isPending}
              className="w-full rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 transition-colors shadow-sm"
            >
              {register_mutation.isPending ? (
                <span className="flex items-center justify-center gap-2">
                  <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                  </svg>
                  Creating account…
                </span>
              ) : (
                'Create account'
              )}
            </button>
          </form>

          {/* Sign in link */}
          <p className="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
            Already have an account?{' '}
            <Link
              to="/login"
              className="font-semibold text-brand-600 dark:text-brand-400 hover:underline"
            >
              Sign in
            </Link>
          </p>
        </div>

        <p className="mt-6 text-center text-xs text-gray-500">
          GoShield · Fraud Detection Platform
        </p>
      </div>
    </div>
  )
}
