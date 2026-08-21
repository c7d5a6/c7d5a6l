import { authFetch } from '../auth'
import type { AuthUser, UserRole } from '../../types/user'
import { readApiError } from './http'

export type UserWritePayload = {
  alias: string
  firstName: string
  lastName: string | null
  photoUrl: string | null
  telegramUsername: string | null
  telegramId: number | null
  role: UserRole
}

export async function fetchUsers(): Promise<AuthUser[]> {
  const res = await authFetch('/api/users')
  if (!res.ok) throw new Error(await readApiError(res, `users uplink failed (${res.status})`))
  const data = (await res.json()) as { users: AuthUser[] }
  return data.users ?? []
}

export async function createUser(payload: UserWritePayload): Promise<AuthUser> {
  const res = await authFetch('/api/users', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.ok) throw new Error(await readApiError(res, `create user failed (${res.status})`))
  const data = (await res.json()) as { user: AuthUser }
  return data.user
}

export async function updateUser(id: number, payload: UserWritePayload): Promise<AuthUser> {
  const res = await authFetch(`/api/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
  if (!res.ok) throw new Error(await readApiError(res, `update user failed (${res.status})`))
  const data = (await res.json()) as { user: AuthUser }
  return data.user
}
