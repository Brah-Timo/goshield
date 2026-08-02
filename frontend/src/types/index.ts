// ── Auth ─────────────────────────────────────────────────────────────────────
export type UserRole = 'ADMIN' | 'ANALYST' | 'VIEWER'

export interface User {
  id: string
  email: string
  firstName: string
  lastName: string
  role: UserRole
  companyId: string
  active: boolean
  createdAt: string
  updatedAt: string
}

export interface AuthTokens {
  accessToken: string
  expiresAt: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  firstName: string
  lastName: string
  companyId: string
}

// ── Claims ────────────────────────────────────────────────────────────────────
export type ClaimStatus =
  | 'PENDING'
  | 'PROCESSING'
  | 'ANALYZED'
  | 'FLAGGED'
  | 'APPROVED'
  | 'REJECTED'
  | 'MORE_INFO'

export type ClaimType =
  | 'MEDICAL'
  | 'AUTO'
  | 'PROPERTY'
  | 'LIFE'
  | 'LIABILITY'
  | 'OTHER'

export type RiskLevel = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'

export interface RiskFactor {
  feature: string
  shapValue: number
  featureValue: string
  direction: 'INCREASES_RISK' | 'DECREASES_RISK'
}

export interface Claim {
  id: string
  companyId: string
  userId: string
  policyNumber: string
  claimType: ClaimType
  status: ClaimStatus
  amount: number
  description: string
  incidentDate: string
  documentUrl?: string
  fraudScore?: number
  riskLevel?: RiskLevel
  fraudReason?: string
  riskFactors?: RiskFactor[]
  shapValues?: RiskFactor[]
  modelVersion?: string
  confidence?: number
  reviewedBy?: string
  reviewedAt?: string
  reviewNotes?: string
  createdAt: string
  updatedAt: string
}

export interface CreateClaimRequest {
  policyNumber: string
  claimType: ClaimType
  amount: number
  description: string
  incidentDate: string
}

export interface ClaimListResponse {
  claims: Claim[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

export interface ClaimFilter {
  status?: ClaimStatus
  claimType?: ClaimType
  riskLevel?: RiskLevel
  minAmount?: number
  maxAmount?: number
  page?: number
  pageSize?: number
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
}

// ── Dashboard ─────────────────────────────────────────────────────────────────
export interface DashboardStats {
  totalClaims: number
  pendingClaims: number
  flaggedClaims: number
  approvedClaims: number
  rejectedClaims: number
  totalAmount: number
  avgFraudScore: number
  fraudRate: number
}

export interface DailyStat {
  date: string
  total: number
  flagged: number
  approved: number
  rejected: number
  totalAmount: number
}

// ── Notifications ─────────────────────────────────────────────────────────────
export interface WSMessage {
  type: 'claim.analyzed' | 'claim.flagged' | 'claim.approved' | 'claim.rejected' | 'ping'
  claimId?: string
  fraudScore?: number
  riskLevel?: RiskLevel
  status?: ClaimStatus
  timestamp: string
}

// ── API ───────────────────────────────────────────────────────────────────────
export interface APIError {
  error: string
  code?: string
  details?: Record<string, string>
}

export interface PaginationMeta {
  page: number
  pageSize: number
  total: number
  totalPages: number
}
