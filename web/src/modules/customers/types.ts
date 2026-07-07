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
