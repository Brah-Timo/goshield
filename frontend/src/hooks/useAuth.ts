import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api, extractError } from '@/lib/api'
import { wsClient } from '@/lib/ws'
import { useAuthStore } from '@/store/auth'
import type { LoginRequest, RegisterRequest, UpdateUserRequest, User, AuthTokens } from '@/types'

export function useLogin() {
  const { setUser } = useAuthStore()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: async (data: LoginRequest) => {
      const res = await api.post<{ user: User; tokens: AuthTokens }>('/auth/v1/login', data)
      return res.data
    },
    onSuccess({ user, tokens }) {
      setUser(user, tokens.accessToken)
      // WebSocket uses companyId — connect after auth is stored
      if (user.companyId) {
        wsClient.connect(user.companyId)
      }
      navigate('/dashboard')
    },
  })
}

export function useRegister() {
  const { setUser } = useAuthStore()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: async (data: RegisterRequest) => {
      const res = await api.post<{ user: User; tokens: AuthTokens }>('/auth/v1/register', data)
      return res.data
    },
    onSuccess({ user, tokens }) {
      setUser(user, tokens.accessToken)
      navigate('/dashboard')
    },
  })
}

export function useLogout() {
  const { logout } = useAuthStore()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: () => api.post('/auth/v1/logout'),
    onSettled() {
      wsClient.disconnect()
      logout()
      navigate('/login')
    },
  })
}

export function useCurrentUser() {
  return useAuthStore((s) => s.user)
}

export function useUpdateUser() {
  const { setUser } = useAuthStore()

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: UpdateUserRequest }) => {
      const res = await api.patch<User>(`/auth/v1/users/${id}`, data)
      return res.data
    },
    onSuccess(updatedUser) {
      // Re-read current access token (stays unchanged) and update user in store
      const currentUser = useAuthStore.getState().user
      if (currentUser && updatedUser.id === currentUser.id) {
        // Access token is held separately — just update the user object
        const token = (api.defaults.headers.common['Authorization'] as string | undefined)
          ?.replace('Bearer ', '') ?? ''
        setUser(updatedUser, token)
      }
    },
  })
}

export { extractError }
