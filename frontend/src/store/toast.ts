import { create } from 'zustand'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface Toast {
  id: string
  type: ToastType
  title?: string
  message: string
}

interface ToastStore {
  toasts: Toast[]
  push: (t: Omit<Toast, 'id'>) => void
  dismiss: (id: string) => void
}

let _counter = 0

export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],

  push(t) {
    const id = `toast-${++_counter}`
    set(s => ({ toasts: [...s.toasts.slice(-4), { ...t, id }] }))
    // Auto-dismiss after 5 s
    setTimeout(() => {
      set(s => ({ toasts: s.toasts.filter(x => x.id !== id) }))
    }, 5000)
  },

  dismiss(id) {
    set(s => ({ toasts: s.toasts.filter(x => x.id !== id) }))
  },
}))

/** Convenience helpers */
export const toast = {
  success: (message: string, title?: string) =>
    useToastStore.getState().push({ type: 'success', message, title }),
  error: (message: string, title?: string) =>
    useToastStore.getState().push({ type: 'error', message, title }),
  info: (message: string, title?: string) =>
    useToastStore.getState().push({ type: 'info', message, title }),
  warning: (message: string, title?: string) =>
    useToastStore.getState().push({ type: 'warning', message, title }),
}
