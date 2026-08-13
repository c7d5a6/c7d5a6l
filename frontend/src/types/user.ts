import { apiUrl } from '../lib/api'

export type UserRole = 'ADMIN' | 'USER'

export type UserTitleKind = 'fantasy' | 'tournament'

export type UserTitle = {
  id: number
  userId: number
  userAlias: string
  kind: UserTitleKind
  name: string
  fantasyLeagueId: number | null
  fantasyLeagueName?: string | null
  hasImage: boolean
  createdAt: string
}

export function titleImageSrc(id: number): string {
  return apiUrl(`/api/user-titles/${id}/image`)
}

export type AuthUser = {
  id: number
  alias: string
  telegramId: number | null
  telegramUsername: string | null
  firstName: string
  lastName: string | null
  photoUrl: string | null
  role: UserRole
  createdAt: string
  updatedAt: string
  lastLoginAt: string | null
  titles?: UserTitle[]
}

export type TelegramAuthPayload = {
  id: number
  first_name: string
  last_name?: string
  username?: string
  photo_url?: string
  auth_date: number
  hash: string
}
