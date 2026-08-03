import { useEffect, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { wsClient } from '@/lib/ws'
import { claimKeys } from './useClaims'
import type { WSMessage } from '@/types'
import { toast } from '@/store/toast'

export function useWebSocketUpdates() {
  const qc = useQueryClient()

  const handleMessage = useCallback(
    (msg: WSMessage) => {
      // Backend sends { type, payload: { claim_id, ... } } — use payload.claim_id
      const claimId = msg.payload?.claim_id
      if (!claimId) return

      const shortId = claimId.slice(0, 8)

      switch (msg.type) {
        case 'claim.analyzed':
          // Invalidate specific claim and list caches so they refetch.
          qc.invalidateQueries({ queryKey: claimKeys.detail(claimId) })
          qc.invalidateQueries({ queryKey: claimKeys.all })
          qc.invalidateQueries({ queryKey: claimKeys.stats() })
          toast.info(
            `Claim ${shortId}… analyzed — score ${
              msg.payload?.fraud_score != null
                ? `${(msg.payload.fraud_score * 100).toFixed(0)}%`
                : 'computed'
            }`,
            'AI Analysis Complete'
          )
          break

        case 'claim.flagged':
          qc.invalidateQueries({ queryKey: claimKeys.detail(claimId) })
          qc.invalidateQueries({ queryKey: claimKeys.all })
          qc.invalidateQueries({ queryKey: claimKeys.stats() })
          toast.warning(
            `Claim ${shortId}… flagged for review${
              msg.payload?.fraud_score != null
                ? ` (score: ${(msg.payload.fraud_score * 100).toFixed(0)}%)`
                : ''
            }`,
            'Fraud Alert'
          )
          break

        case 'claim.approved':
          qc.invalidateQueries({ queryKey: claimKeys.detail(claimId) })
          qc.invalidateQueries({ queryKey: claimKeys.all })
          qc.invalidateQueries({ queryKey: claimKeys.stats() })
          toast.success(
            `Claim ${shortId}… has been approved`,
            'Claim Approved'
          )
          break

        case 'claim.rejected':
          qc.invalidateQueries({ queryKey: claimKeys.detail(claimId) })
          qc.invalidateQueries({ queryKey: claimKeys.all })
          qc.invalidateQueries({ queryKey: claimKeys.stats() })
          toast.error(
            `Claim ${shortId}… has been rejected`,
            'Claim Rejected'
          )
          break

        case 'ping':
          // Silent heartbeat — no toast
          break

        default:
          break
      }
    },
    [qc]
  )

  useEffect(() => {
    const unsub = wsClient.subscribe(handleMessage)
    return unsub
  }, [handleMessage])
}
