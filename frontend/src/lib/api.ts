import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import type { APIError } from '@/types'

const BASE_URL = import.meta.env.VITE_API_URL ?? ''

export const api = axios.create({
  baseURL: BASE_URL,
  timeout: 30_000,
  withCredentials: true,          // send refresh-token cookie automatically
  headers: { 'Content-Type': 'application/json' },
})

// ── Request interceptor — attach access token from memory ───────────────────
api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// ── Response interceptor — silent token refresh on 401 ──────────────────────
let refreshing = false
let refreshQueue: Array<(token: string) => void> = []

api.interceptors.response.use(
  (res) => res,
  async (err: AxiosError<APIError>) => {
    const original = err.config as InternalAxiosRequestConfig & { _retry?: boolean }
    if (err.response?.status === 401 && !original._retry) {
      if (refreshing) {
        return new Promise((resolve) => {
          refreshQueue.push((token) => {
            original.headers.Authorization = `Bearer ${token}`
            resolve(api(original))
          })
        })
      }
      original._retry = true
      refreshing = true
      try {
        const { data } = await axios.post<{ access_token: string }>(
          `${BASE_URL}/auth/v1/refresh`,
          {},
          { withCredentials: true }
        )
        setAccessToken(data.access_token)
        refreshQueue.forEach((cb) => cb(data.access_token))
        refreshQueue = []
        original.headers.Authorization = `Bearer ${data.access_token}`
        return api(original)
      } catch {
        clearAccessToken()
        window.location.href = '/login'
        return Promise.reject(err)
      } finally {
        refreshing = false
      }
    }
    return Promise.reject(err)
  }
)

// ── In-memory token store (never put access tokens in localStorage) ──────────
let _accessToken: string | null = null

export function setAccessToken(token: string) {
  _accessToken = token
}
export function getAccessToken(): string | null {
  return _accessToken
}
export function clearAccessToken() {
  _accessToken = null
}

// ── Typed helpers ────────────────────────────────────────────────────────────
export function extractError(err: unknown): string {
  if (axios.isAxiosError(err)) {
    return (err as AxiosError<APIError>).response?.data?.error ?? err.message
  }
  return String(err)
}
