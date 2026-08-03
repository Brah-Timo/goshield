import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Claim, ClaimListResponse, CreateClaimRequest, ClaimFilter } from '@/types'

export const claimKeys = {
  all: ['claims'] as const,
  list: (filter: ClaimFilter) => ['claims', 'list', filter] as const,
  detail: (id: string) => ['claims', id] as const,
  stats: () => ['claims', 'stats'] as const,
}

export function useClaimList(filter: ClaimFilter = {}) {
  return useQuery({
    queryKey: claimKeys.list(filter),
    queryFn: async () => {
      const params = new URLSearchParams()
      Object.entries(filter).forEach(([k, v]) => {
        if (v !== undefined && v !== null && v !== '') params.set(k, String(v))
      })
      const { data } = await api.get<ClaimListResponse>(`/claims/v1/claims?${params}`)
      return data
    },
  })
}

export function useClaim(id: string) {
  return useQuery({
    queryKey: claimKeys.detail(id),
    queryFn: async () => {
      const { data } = await api.get<Claim>(`/claims/v1/claims/${id}`)
      return data
    },
    enabled: !!id,
  })
}

export function useCreateClaim() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: CreateClaimRequest) => {
      const { data } = await api.post<Claim>('/claims/v1/claims', body)
      return data
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: claimKeys.all }),
  })
}

export function useUploadDocument(claimId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (file: File) => {
      const form = new FormData()
      form.append('document', file)
      const { data } = await api.post<Claim>(
        `/claims/v1/claims/${claimId}/document`,
        form,
        { headers: { 'Content-Type': 'multipart/form-data' } }
      )
      return data
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: claimKeys.detail(claimId) })
      qc.invalidateQueries({ queryKey: claimKeys.all })
    },
  })
}

export function useReviewClaim() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      action,
      notes,
    }: {
      id: string
      action: 'APPROVE' | 'REJECT' | 'REQUEST_MORE_INFO'
      notes?: string
    }) => {
      const { data } = await api.post<Claim>(`/claims/v1/claims/${id}/review`, { action, notes })
      return data
    },
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: claimKeys.detail(id) })
      qc.invalidateQueries({ queryKey: claimKeys.all })
    },
  })
}

export function useClaimStats() {
  return useQuery({
    queryKey: claimKeys.stats(),
    queryFn: async () => {
      const { data } = await api.get('/claims/v1/stats')
      return data
    },
  })
}

export function useDeleteClaim() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/claims/v1/claims/${id}`)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: claimKeys.all })
    },
  })
}

export function useDailyStats(days = 30) {
  return useQuery({
    queryKey: ['claims', 'daily-stats', days],
    queryFn: async () => {
      const { data } = await api.get<import('@/types').DailyStat[]>(
        `/claims/v1/stats/daily?days=${days}`
      )
      return data
    },
  })
}
