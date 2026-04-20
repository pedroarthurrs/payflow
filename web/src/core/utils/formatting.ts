import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs))
}

export function formatDocument(value: string, type: string): string {
    if (!value) return ''
    const d = value.replace(/\D/g, '')
    const t = type.toUpperCase()
    if (t === 'CPF') {
        return d.replace(/^(\d{3})(\d{3})(\d{3})(\d{2})$/, '$1.$2.$3-$4')
    }
    if (t === 'CNPJ') {
        return d.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, '$1.$2.$3/$4-$5')
    }
    return value
}

export function formatPhone(value: string): string {
    if (!value) return ''
    let d = value.replace(/\D/g, '')
    // strip country code 55
    if (d.length === 13 || d.length === 12) {
        d = d.slice(2)
    }
    if (d.length === 11) {
        return d.replace(/^(\d{2})(\d{5})(\d{4})$/, '($1) $2-$3')
    }
    if (d.length === 10) {
        return d.replace(/^(\d{2})(\d{4})(\d{4})$/, '($1) $2-$3')
    }
    return value
}

export function formatMetadata(metadata?: Record<string, unknown>): Array<{ key: string; value: string }> {
    if (!metadata) return []
    return Object.entries(metadata).map(([key, val]) => ({
        key,
        value: typeof val === 'object' && val !== null ? JSON.stringify(val) : String(val ?? ''),
    }))
}
