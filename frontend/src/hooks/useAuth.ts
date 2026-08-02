import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api, extractError } from '@/lib/api'
import { wsClient } from '@/lib/ws'
import { useAuthStore } from '@/store/auth'
import type { LoginRequest, RegisterRequest, User, AuthTokens } from '@/types'

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
      wsClient.connect(user.companyId)
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

export { extractError }
