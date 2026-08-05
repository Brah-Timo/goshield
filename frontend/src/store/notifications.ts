import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { WSMessage } from '@/types'

export type NotifSeverity = 'info' | 'warning' | 'error' | 'success'

export interface Notification {
  id: string
  severity: NotifSeverity
  title: string
  body: string
  claimId?: string
  read: boolean
  ts: number // Unix ms
}

let _seq = 0

interface NotifState {
  items: Notification[]
  unread: number
  add: (n: Omit<Notification, 'id' | 'read' | 'ts'>) => void
  markRead: (id: string) => void
  markAllRead: () => void
  dismiss: (id: string) => void
  clearAll: () => void
  fromWSMessage: (msg: WSMessage) => void
}

export const useNotifStore = create<NotifState>()(
  persist(
    (set, get) => ({
      items: [],
      unread: 0,

      add(payload) {
        const n: Notification = {
          ...payload,
          id: `notif-${Date.now()}-${++_seq}`,
          read: false,
          ts: Date.now(),
        }
        set(s => ({
          items: [n, ...s.items].slice(0, 50), // cap at 50
          unread: s.unread + 1,
        }))
      },

      markRead(id) {
        set(s => {
          const items = s.items.map(n => n.id === id && !n.read ? { ...n, read: true } : n)
          return { items, unread: items.filter(n => !n.read).length }
        })
      },

      markAllRead() {
        set(s => ({
          items: s.items.map(n => ({ ...n, read: true })),
          unread: 0,
        }))
      },

      dismiss(id) {
        set(s => {
          const items = s.items.filter(n => n.id !== id)
          return { items, unread: items.filter(n => !n.read).length }
        })
      },

      clearAll() {
        set({ items: [], unread: 0 })
      },

      fromWSMessage(msg) {
        const claimId   = msg.payload?.claim_id
        const shortId   = claimId ? claimId.slice(0, 8) + '…' : ''
        const scorePct  = msg.payload?.fraud_score != null
          ? `${(msg.payload.fraud_score * 100).toFixed(0)}%`
          : null

        switch (msg.type) {
          case 'claim.analyzed':
            get().add({
              severity: 'info',
              title: 'AI Analysis Complete',
              body: `Claim ${shortId} analyzed${scorePct ? ` — score ${scorePct}` : ''}.`,
              claimId,
            })
            break
          case 'claim.flagged':
            get().add({
              severity: 'warning',
              title: 'Fraud Alert',
              body: `Claim ${shortId} flagged for review${scorePct ? ` (score: ${scorePct})` : ''}.`,
              claimId,
            })
            break
          case 'claim.approved':
            get().add({
              severity: 'success',
              title: 'Claim Approved',
              body: `Claim ${shortId} has been approved.`,
              claimId,
            })
            break
          case 'claim.rejected':
            get().add({
              severity: 'error',
              title: 'Claim Rejected',
              body: `Claim ${shortId} has been rejected.`,
              claimId,
            })
            break
          case 'ping':
          default:
            break
        }
      },
    }),
    {
      name: 'goshield-notifications',
      partialize: s => ({ items: s.items.slice(0, 20), unread: s.unread }),
    }
  )
)
