// === Autenticação ===

export type Role = 'owner' | 'admin' | 'member'
export type AuthType = 'user' | 'system'

export interface LoginRequest {
    email?: string
    password?: string
    api_key?: string
}

export interface RefreshRequest {
    refresh_token: string
}

export interface TokenPair {
    access_token: string
    access_token_expires_at: string
    refresh_token: string
    refresh_token_expires_at: string
}

/** Returned when the user belongs to multiple organizations. */
export interface OrgRef {
    organization_id: string
    name: string
    role: Role
}

/** Returned by POST /v1/auth/login when the user has multiple organizations. */
export interface MultiOrgLoginResponse {
    session_token: string
    organizations: OrgRef[]
}

/** Discriminated union of possible login responses. */
export type LoginResponse = TokenPair | MultiOrgLoginResponse

export function isMultiOrgLoginResponse(r: LoginResponse): r is MultiOrgLoginResponse {
    return 'session_token' in r
}

/**
 * Type-safe discriminated union for login responses.
 *
 * @example
 * const response = await login(email, password)
 * if (isMultiOrgLoginResponse(response)) {
 *   // TypeScript narrows to MultiOrgLoginResponse
 *   const orgs = response.organizations
 * } else {
 *   // TypeScript narrows to TokenPair
 *   const token = response.access_token
 * }
 */
export type AuthResponse<T extends 'single' | 'multi'> =
    T extends 'single' ? TokenPair : MultiOrgLoginResponse

export interface SelectOrganizationRequest {
    session_token: string
    organization_id: string
}

export interface MeResponse {
    organization_id: string
    user_id?: string
    role?: Role
    permissions?: string[]
    issued_at: string | number
    expires_at: string | number
}
