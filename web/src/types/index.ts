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

// === Clientes ===

export interface Customer {
  id: string
  name: string
  email: string
  phone?: string
  document: string
  document_type: string
  status: string
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateCustomerRequest {
  name: string
  email: string
  phone?: string
  document: string
  document_type: string
  metadata?: Record<string, unknown>
  address?: CreateAddressRequest
}

export interface UpdateCustomerRequest {
  name: string
  email: string
  document: string
  document_type: string
  phone?: string
  metadata?: Record<string, unknown>
}

// === Endereços ===

export type AddressType = 'billing' | 'shipping' | 'home'

export interface Address {
  id: string
  customer_id: string
  type: AddressType
  postal_code: string
  street: string
  number?: string
  complement?: string
  neighborhood?: string
  city: string
  state: string
  country: string
  is_main: boolean
  created_at: string
}

export interface CreateAddressRequest {
  type?: AddressType
  postal_code: string
  street: string
  number?: string
  complement?: string
  neighborhood?: string
  city: string
  state: string
  country?: string
  is_main?: boolean
}

export interface UpdateAddressRequest extends CreateAddressRequest { }

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

// === Paginação ===

export interface PagedResponse<T> {
  data: T[]
  limit: number
  offset: number
}

// === Erros da API ===

export interface ApiError {
  message: string
  status: number
  validationErrors?: Record<string, string>
  code?: string
}

export interface ApiValidationError {
  field: string
  rejected_value: string
  message: string
}

// Common error codes from backend
export enum ApiErrorCode {
  UNAUTHORIZED = 'unauthorized',
  FORBIDDEN = 'forbidden',
  NOT_FOUND = 'not_found',
  CONFLICT = 'conflict',
  INVALID_REQUEST = 'invalid_request',
  INTERNAL_ERROR = 'internal_error',
  MAINTENANCE = 'maintenance',
  RATE_LIMIT = 'rate_limit',
}

// Auth error codes
export enum AuthErrorCode {
  INVALID_CREDENTIALS = 'invalid_credentials',
  ACCOUNT_LOCKED = 'account_locked',
  ACCOUNT_DISABLED = 'account_disabled',
  SESSION_EXPIRED = 'session_expired',
  TOKEN_INVALID = 'token_invalid',
  TOKEN_EXPIRED = 'token_expired',
}
