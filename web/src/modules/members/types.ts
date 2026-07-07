import type { Role } from '@/modules/auth/types'

// === Members ===

export interface User {
    id: string
    name: string
    email: string
    role: Role
    joined_at: string
}

export interface CreateUserRequest {
    name: string
    email: string
    password: string
    role: 'admin' | 'member'
}

export interface UpdateUserRequest {
    name: string
    role: 'admin' | 'member'
}
