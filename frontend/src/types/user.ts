export type UserRole = 'ADMIN' | 'USER'

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
