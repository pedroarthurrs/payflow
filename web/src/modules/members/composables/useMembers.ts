import { api } from '@/core/api/http'
import type { PagedResponse } from '@/core/types'
import type { CreateUserRequest, UpdateUserRequest, User } from '@/modules/members/types'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'

// --- Queries ---

export function useMemberList() {
    const query = useQuery({
        queryKey: ['members', 'list'] as const,
        queryFn: () => api.get<PagedResponse<User>>('/v1/members'),
    })

    return {
        ...query,
        isPending: query.isPending,
        isError: query.isError,
    }
}

// --- Mutations ---

export function useCreateMember() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (data: CreateUserRequest) =>
            api.post<User>('/v1/members', data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['members', 'list'] })
        },
    })
}

export function useUpdateMember() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateUserRequest }) =>
            api.put<User>(`/v1/members/${id}`, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['members', 'list'] })
        },
    })
}

export function useDeleteMember() {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: (id: string) => api.delete(`/v1/members/${id}`),
        onSuccess: (_data, id) => {
            queryClient.invalidateQueries({ queryKey: ['members', 'list'] })
            queryClient.invalidateQueries({ queryKey: ['members', 'detail', id] })
        },
    })
}
