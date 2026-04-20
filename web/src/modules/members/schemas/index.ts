import { emailSchema } from '@/core/utils/validators'
import { z } from 'zod'

export const createMemberSchema = z.object({
    name: z.string().min(2, 'Nome deve ter no mínimo 2 caracteres'),
    email: emailSchema,
    password: z.string().min(8, 'Senha deve ter no mínimo 8 caracteres'),
    role: z.enum(['admin', 'member']),
})

export const updateMemberSchema = z.object({
    name: z.string().min(2, 'Nome deve ter no mínimo 2 caracteres'),
    role: z.enum(['admin', 'member']),
})

export type CreateMemberFormData = z.infer<typeof createMemberSchema>
export type UpdateMemberFormData = z.infer<typeof updateMemberSchema>
