import { createSignal } from 'solid-js'
import { apiUrl } from './api'
import type { AuthUser, TelegramAuthPayload } from '../types/user'

const TOKEN_KEY = 'c7d5a6l.accessToken'
const USER_KEY = 'c7d5a6l.user'

const [user, setUser] = createSignal<AuthUser | null>(readStoredUser())
const [sessionReady, setSessionReady] = createSignal(false)

export function authUser() {
  return user()
}

export function isAdmin(): boolean {
  return authUser()?.role === 'ADMIN'
}

export function homePath(): string {
  return isAdmin() ? '/parser' : '/fantasy-league'
}

export function authReady() {
  return sessionReady()
}

export function getToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

export function getUser(): AuthUser | null {
  return user()
}

export function setSession(token: string, next: AuthUser) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify(next))
  setUser(next)
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  setUser(null)
}

export async function authFetch(input: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers)
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (!headers.has('Content-Type') && init.body && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }
  const res = await fetch(apiUrl(input), { ...init, headers })
  if (res.status === 401) clearSession()
  return res
}

export async function bootAuthSession(): Promise<void> {
  const token = getToken()
  if (!token) {
    setUser(null)
    setSessionReady(true)
    return
  }
  try {
    const res = await authFetch('/api/me')
    if (!res.ok) {
      clearSession()
      setSessionReady(true)
      return
    }
    const data = (await res.json()) as { user: AuthUser }
    setSession(token, data.user)
  } catch {
    clearSession()
  } finally {
    setSessionReady(true)
  }
}

export async function fetchAuthConfig(): Promise<{ botId?: string; botUsername: string }> {
  const res = await fetch(apiUrl('/api/auth/config'))
  if (!res.ok) {
    throw new Error(`auth config failed (${res.status})`)
  }
  const data = (await res.json()) as { botId?: string; botUsername?: string }
  if (!data.botUsername?.trim()) throw new Error('bot username missing')
  return { botId: data.botId, botUsername: data.botUsername.trim() }
}

export async function loginWithTelegram(payload: TelegramAuthPayload): Promise<AuthUser> {
  const res = await fetch(apiUrl('/api/auth/telegram'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    let msg = `login failed (${res.status})`
    try {
      const data = (await res.json()) as { error?: string }
      if (data.error) msg = data.error
    } catch {
      /* ignore */
    }
    throw new Error(msg)
  }
  const data = (await res.json()) as { token: string; user: AuthUser }
  setSession(data.token, data.user)
  return data.user
}

export async function logout(): Promise<void> {
  const token = getToken()
  if (token) {
    try {
      await authFetch('/api/auth/logout', { method: 'POST' })
    } catch {
      /* client still clears */
    }
  }
  clearSession()
}

export async function updateAlias(alias: string): Promise<AuthUser> {
  const res = await authFetch('/api/me', {
    method: 'PATCH',
    body: JSON.stringify({ alias }),
  })
  if (!res.ok) {
    let msg = `alias update failed (${res.status})`
    try {
      const data = (await res.json()) as { error?: string }
      if (data.error) msg = data.error
    } catch {
      /* ignore */
    }
    throw new Error(msg)
  }
  const data = (await res.json()) as { token: string; user: AuthUser }
  setSession(data.token, data.user)
  return data.user
}

function readStoredUser(): AuthUser | null {
  try {
    const raw = localStorage.getItem(USER_KEY)
    if (!raw) return null
    return JSON.parse(raw) as AuthUser
  } catch {
    return null
  }
}
