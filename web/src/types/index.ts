// === Autenticação ===

export type Role = 'owner' | 'admin' | 'member'

export interface LoginRequest {
  email?: string
  password?: string
  api_key?: string
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
  issued_at: string
  expires_at: string
}

// === Clientes ===

export interface Customer {
  id: string
  name: string
  email: string
  phone?: string
  document: string
  document_type: 'CPF' | 'CNPJ'
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
  document_type: 'CPF' | 'CNPJ'
  metadata?: Record<string, unknown>
  address?: CreateAddressRequest
}

export interface UpdateCustomerRequest {
  name: string
  email: string
  document: string
  document_type: 'CPF' | 'CNPJ'
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

// === Usuários ===

export interface User {
  id: string
  name: string
  email: string
  role: Role
  created_at: string
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

// === Erros da API ===

export interface ApiError {
  message: string
  status: number
  validationErrors?: Record<string, string>
}

export interface ApiValidationError {
  field: string
  rejected_value: string
  message: string
}
