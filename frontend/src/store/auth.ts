import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '@/types'
import { setAccessToken, clearAccessToken } from '@/lib/api'

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  setUser: (user: User, accessToken: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,

      setUser(user, accessToken) {
        setAccessToken(accessToken)
        set({ user, isAuthenticated: true })
      },

      logout() {
        clearAccessToken()
        set({ user: null, isAuthenticated: false })
      },
    }),
    {
      name: 'goshield-auth',
      partialize: (state) => ({ user: state.user, isAuthenticated: state.isAuthenticated }),
    }
  )
)
