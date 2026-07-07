/**
 * Centralized RBAC types — mirrors backend domain/permission.go and domain/user.go.
 *
 * Role hierarchy: owner (4) > admin (3) > developer (2) > support (1)
 * Frontend "member" alias kept for backward compat but maps to support (1).
 */

export type Role = 'owner' | 'admin' | 'developer' | 'support'

export const ROLE_HIERARCHY: Record<Role, number> = {
    owner: 4,
    admin: 3,
    developer: 2,
    support: 1,
} as const

/** All 30 backend permissions in {resource}:{action} format. */
export type Permission =
    | 'users:read'
    | 'users:invite'
    | 'users:manage'
    | 'credentials:read'
    | 'credentials:manage'
    | 'customers:read'
    | 'customers:write'
    | 'customers:delete'
    | 'products:read'
    | 'products:write'
    | 'plans:read'
    | 'plans:write'
    | 'subscriptions:read'
    | 'subscriptions:write'
    | 'subscriptions:items'
    | 'payments:read'
    | 'payments:charge'
    | 'payments:refund'
    | 'payments:cancel'
    | 'webhooks:read'
    | 'webhooks:manage'
    | 'logs:read'
    | 'wallets:read'
    | 'wallets:payout'
    | 'wallets:adjust'
    | 'commissions:read'
    | 'commissions:manage'
    | 'invoices:read'
    | 'invoices:write'
    | 'invoices:manage'
