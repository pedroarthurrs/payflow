import { api } from '@/core/api/http'
import type { PagedResponse } from '@/core/types'
import type {
    Address,
    CreateAddressRequest,
    CreateCustomerRequest,
    Customer,
    UpdateAddressRequest,
    UpdateCustomerRequest,
} from '@/modules/customers/types'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'

// --- Filters ---

export interface CustomerFilters {
    search?: string
    status?: string
    limit?: number
    offset?: number
}

// --- Queries ---

export function useCustomerList(filters?: Ref<CustomerFilters>) {
    const query = useQuery({
        queryKey: ['customers', 'list', filters] as const,
        queryFn: () => api.get<PagedResponse<Customer>>('/v1/customers', { params: filters?.value }),
    })

    return {
        ...query,
        isPending: query.isPending,
        isError: query.isError,
    }
}

export function useCustomerDetail(id: Ref<string>) {
    const query = useQuery({
        queryKey: computed(() => ['customers', 'detail', id.value] as const),
        queryFn: () => api.get<Customer>(`/v1/customers/${id.value}`),
        enabled: computed(() => !!id.value),
    })

    return {
        ...query,
        isPending: query.isPending,
        isError: query.isError,
    }
}

// --- Mutations ---

export function useCreateCustomer() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (data: CreateCustomerRequest) =>
            api.post<Customer>('/v1/customers', data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['customers', 'list'] })
        },
    })
}

export function useUpdateCustomer() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateCustomerRequest }) =>
            api.put<Customer>(`/v1/customers/${id}`, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['customers', 'list'] })
        },
    })
}

export function useDeleteCustomer() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (id: string) => api.delete(`/v1/customers/${id}`),
        onSuccess: (_data, id) => {
            queryClient.invalidateQueries({ queryKey: ['customers', 'list'] })
            queryClient.invalidateQueries({ queryKey: ['customers', 'detail', id] })
        },
    })
}

// --- Address Queries ---

export function useAddressList(customerId: Ref<string>) {
    const query = useQuery({
        queryKey: computed(() => ['customers', customerId.value, 'addresses'] as const),
        queryFn: () =>
            api.get<PagedResponse<Address>>(`/v1/customers/${customerId.value}/addresses`),
        enabled: computed(() => !!customerId.value),
    })

    return {
        ...query,
        isPending: query.isPending,
        isError: query.isError,
    }
}

// --- Address Mutations ---

export function useCreateAddress(customerId: Ref<string>) {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (data: CreateAddressRequest) =>
            api.post<Address>(`/v1/customers/${customerId.value}/addresses`, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['customers', customerId.value, 'addresses'] })
        },
    })
}

export function useUpdateAddress(customerId: Ref<string>) {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ addressId, data }: { addressId: string; data: UpdateAddressRequest }) =>
            api.put<Address>(`/v1/customers/${customerId.value}/addresses/${addressId}`, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['customers', customerId.value, 'addresses'] })
        },
    })
}

export function useDeleteAddress(customerId: Ref<string>) {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (addressId: string) =>
            api.delete(`/v1/customers/${customerId.value}/addresses/${addressId}`),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['customers', customerId.value, 'addresses'] })
        },
    })
}
