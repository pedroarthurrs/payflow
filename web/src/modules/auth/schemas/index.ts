import { emailSchema } from '@/core/utils/validators'
import { z } from 'zod'

export const loginSchema = z.object({
    email: emailSchema,
    password: z.string().min(8, 'Senha deve ter no mínimo 8 caracteres'),
})

export type LoginFormData = z.infer<typeof loginSchema>
