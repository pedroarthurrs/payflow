import { cepSchema, cnpjSchema, cpfSchema, emailSchema, phoneSchema } from '@/core/utils/validators'
import { z } from 'zod'

export const createCustomerSchema = z.object({
    name: z.string().min(2, 'Nome deve ter no mínimo 2 caracteres'),
    email: emailSchema,
    phone: phoneSchema.optional(),
    document: z.union([cpfSchema, cnpjSchema]),
    document_type: z.enum(['cpf', 'cnpj']),
    metadata: z.record(z.unknown()).optional(),
})

export const updateCustomerSchema = z.object({
    name: z.string().min(2, 'Nome deve ter no mínimo 2 caracteres'),
    email: emailSchema,
    document: z.union([cpfSchema, cnpjSchema]),
    document_type: z.enum(['cpf', 'cnpj']),
    phone: phoneSchema.optional(),
    metadata: z.record(z.unknown()).optional(),
})

export const addressSchema = z.object({
    type: z.enum(['billing', 'shipping', 'home']).optional(),
    postal_code: cepSchema,
    street: z.string().min(1, 'Logradouro é obrigatório'),
    number: z.string().optional(),
    complement: z.string().optional(),
    neighborhood: z.string().optional(),
    city: z.string().min(1, 'Cidade é obrigatória'),
    state: z.string().length(2, 'Estado deve ter 2 caracteres'),
    country: z.string().optional().default('BR'),
    is_main: z.boolean().optional().default(false),
})

export type CreateCustomerFormData = z.infer<typeof createCustomerSchema>
export type UpdateCustomerFormData = z.infer<typeof updateCustomerSchema>
export type AddressFormData = z.infer<typeof addressSchema>
