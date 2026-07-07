import { z } from 'zod'

// ---------------------------------------------------------------------------
// Pure validation functions
// ---------------------------------------------------------------------------

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function isValidEmail(value: string): boolean {
    return EMAIL_RE.test(value)
}

export function isValidCPF(value: string): boolean {
    const digits = value.replace(/\D/g, '')
    if (digits.length !== 11) return false

    // Reject all-same-digit sequences (e.g. 111.111.111-11)
    if (/^(\d)\1{10}$/.test(digits)) return false

    // First check digit
    let sum = 0
    for (let i = 0; i < 9; i++) {
        sum += Number(digits[i]) * (10 - i)
    }
    let remainder = (sum * 10) % 11
    if (remainder === 10) remainder = 0
    if (remainder !== Number(digits[9])) return false

    // Second check digit
    sum = 0
    for (let i = 0; i < 10; i++) {
        sum += Number(digits[i]) * (11 - i)
    }
    remainder = (sum * 10) % 11
    if (remainder === 10) remainder = 0
    if (remainder !== Number(digits[10])) return false

    return true
}

export function isValidCNPJ(value: string): boolean {
    const digits = value.replace(/\D/g, '')
    if (digits.length !== 14) return false

    // Reject all-same-digit sequences
    if (/^(\d)\1{13}$/.test(digits)) return false

    const weights1 = [5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2]
    const weights2 = [6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2]

    // First check digit
    let sum = 0
    for (let i = 0; i < 12; i++) {
        sum += Number(digits[i]) * weights1[i]
    }
    let remainder = sum % 11
    const first = remainder < 2 ? 0 : 11 - remainder
    if (first !== Number(digits[12])) return false

    // Second check digit
    sum = 0
    for (let i = 0; i < 13; i++) {
        sum += Number(digits[i]) * weights2[i]
    }
    remainder = sum % 11
    const second = remainder < 2 ? 0 : 11 - remainder
    if (second !== Number(digits[13])) return false

    return true
}

export function isValidPhone(value: string): boolean {
    const digits = value.replace(/\D/g, '')
    // Brazilian phone: DDD (2 digits) + number (8 or 9 digits) = 10 or 11 digits
    return digits.length === 10 || digits.length === 11
}

export function isValidCEP(value: string): boolean {
    const digits = value.replace(/\D/g, '')
    return /^\d{8}$/.test(digits)
}

// ---------------------------------------------------------------------------
// Zod primitive schemas
// ---------------------------------------------------------------------------

export const emailSchema = z
    .string()
    .refine(isValidEmail, { message: 'E-mail inválido' })

export const cpfSchema = z
    .string()
    .refine(isValidCPF, { message: 'CPF inválido' })

export const cnpjSchema = z
    .string()
    .refine(isValidCNPJ, { message: 'CNPJ inválido' })

export const phoneSchema = z
    .string()
    .refine(isValidPhone, { message: 'Telefone inválido' })

export const cepSchema = z
    .string()
    .refine(isValidCEP, { message: 'CEP inválido' })
