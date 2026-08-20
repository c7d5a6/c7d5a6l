import { authFetch } from '../auth'
import type { AuthUser } from '../../types/user'
import { readApiError } from './http'

export async function fetchUsers(): Promise<AuthUser[]> {
  const res = await authFetch('/api/users')
  if (!res.ok) throw new Error(await readApiError(res, `users uplink failed (${res.status})`))
  const data = (await res.json()) as { users: AuthUser[] }
  return data.users ?? []
}
