import { describe, expect, it } from 'vitest'
import {
    cepSchema,
    cnpjSchema,
    cpfSchema,
    emailSchema,
    isValidCEP,
    isValidCNPJ,
    isValidCPF,
    isValidEmail,
    isValidPhone,
    phoneSchema,
} from './validators'

// ---------------------------------------------------------------------------
// Pure functions
// ---------------------------------------------------------------------------

describe('isValidEmail', () => {
    it('accepts valid emails', () => {
        expect(isValidEmail('user@example.com')).toBe(true)
        expect(isValidEmail('a@b.co')).toBe(true)
    })
    it('rejects invalid emails', () => {
        expect(isValidEmail('')).toBe(false)
        expect(isValidEmail('no-at-sign')).toBe(false)
        expect(isValidEmail('@missing-local.com')).toBe(false)
    })
})

describe('isValidCPF', () => {
    it('accepts a valid CPF', () => {
        // 529.982.247-25 is a well-known valid CPF
        expect(isValidCPF('52998224725')).toBe(true)
        expect(isValidCPF('529.982.247-25')).toBe(true)
    })
    it('rejects all-same-digit CPFs', () => {
        expect(isValidCPF('11111111111')).toBe(false)
        expect(isValidCPF('00000000000')).toBe(false)
    })
    it('rejects wrong check digits', () => {
        expect(isValidCPF('52998224720')).toBe(false)
    })
    it('rejects wrong length', () => {
        expect(isValidCPF('123')).toBe(false)
    })
})

describe('isValidCNPJ', () => {
    it('accepts a valid CNPJ', () => {
        // 11.222.333/0001-81 is a well-known valid CNPJ
        expect(isValidCNPJ('11222333000181')).toBe(true)
        expect(isValidCNPJ('11.222.333/0001-81')).toBe(true)
    })
    it('rejects all-same-digit CNPJs', () => {
        expect(isValidCNPJ('11111111111111')).toBe(false)
    })
    it('rejects wrong check digits', () => {
        expect(isValidCNPJ('11222333000100')).toBe(false)
    })
    it('rejects wrong length', () => {
        expect(isValidCNPJ('123')).toBe(false)
    })
})

describe('isValidPhone', () => {
    it('accepts 10-digit landline', () => {
        expect(isValidPhone('1133334444')).toBe(true)
    })
    it('accepts 11-digit mobile', () => {
        expect(isValidPhone('11999998888')).toBe(true)
    })
    it('accepts formatted phone', () => {
        expect(isValidPhone('(11) 99999-8888')).toBe(true)
    })
    it('rejects too short', () => {
        expect(isValidPhone('123')).toBe(false)
    })
})

describe('isValidCEP', () => {
    it('accepts 8-digit CEP', () => {
        expect(isValidCEP('01001000')).toBe(true)
        expect(isValidCEP('01001-000')).toBe(true)
    })
    it('rejects wrong length', () => {
        expect(isValidCEP('123')).toBe(false)
        expect(isValidCEP('123456789')).toBe(false)
    })
})

// ---------------------------------------------------------------------------
// Zod schemas
// ---------------------------------------------------------------------------

describe('emailSchema', () => {
    it('parses valid email', () => {
        expect(emailSchema.parse('user@example.com')).toBe('user@example.com')
    })
    it('rejects invalid email with Portuguese message', () => {
        const result = emailSchema.safeParse('bad')
        expect(result.success).toBe(false)
        if (!result.success) {
            expect(result.error.issues[0].message).toBe('E-mail inválido')
        }
    })
})

describe('cpfSchema', () => {
    it('parses valid CPF', () => {
        expect(cpfSchema.parse('52998224725')).toBe('52998224725')
    })
    it('rejects invalid CPF with Portuguese message', () => {
        const result = cpfSchema.safeParse('00000000000')
        expect(result.success).toBe(false)
        if (!result.success) {
            expect(result.error.issues[0].message).toBe('CPF inválido')
        }
    })
})

describe('cnpjSchema', () => {
    it('parses valid CNPJ', () => {
        expect(cnpjSchema.parse('11222333000181')).toBe('11222333000181')
    })
    it('rejects invalid CNPJ with Portuguese message', () => {
        const result = cnpjSchema.safeParse('00000000000000')
        expect(result.success).toBe(false)
        if (!result.success) {
            expect(result.error.issues[0].message).toBe('CNPJ inválido')
        }
    })
})

describe('phoneSchema', () => {
    it('parses valid phone', () => {
        expect(phoneSchema.parse('11999998888')).toBe('11999998888')
    })
    it('rejects invalid phone with Portuguese message', () => {
        const result = phoneSchema.safeParse('123')
        expect(result.success).toBe(false)
        if (!result.success) {
            expect(result.error.issues[0].message).toBe('Telefone inválido')
        }
    })
})

describe('cepSchema', () => {
    it('parses valid CEP', () => {
        expect(cepSchema.parse('01001000')).toBe('01001000')
    })
    it('rejects invalid CEP with Portuguese message', () => {
        const result = cepSchema.safeParse('123')
        expect(result.success).toBe(false)
        if (!result.success) {
            expect(result.error.issues[0].message).toBe('CEP inválido')
        }
    })
})
