import { useEffect, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { wsClient } from '@/lib/ws'
import { claimKeys } from './useClaims'
import type { WSMessage } from '@/types'

export function useWebSocketUpdates() {
  const qc = useQueryClient()

  const handleMessage = useCallback(
    (msg: WSMessage) => {
      if (!msg.claimId) return
      switch (msg.type) {
        case 'claim.analyzed':
        case 'claim.flagged':
        case 'claim.approved':
        case 'claim.rejected':
          // Invalidate specific claim and list caches so they refetch.
          qc.invalidateQueries({ queryKey: claimKeys.detail(msg.claimId) })
          qc.invalidateQueries({ queryKey: claimKeys.all })
          qc.invalidateQueries({ queryKey: claimKeys.stats() })
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
