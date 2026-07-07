/**
 * Role default permissions — mirrors backend domain/permission.go roleDefaultPermissions.
 * Used as fallback when /me hasn't been fetched yet.
 */
import type { Permission, Role } from './types'

export const ROLE_DEFAULTS: Record<Role, readonly Permission[]> = {
    owner: [
        'users:read', 'users:invite', 'users:manage',
        'credentials:read', 'credentials:manage',
        'customers:read', 'customers:write', 'customers:delete',
        'products:read', 'products:write',
        'plans:read', 'plans:write',
        'subscriptions:read', 'subscriptions:write', 'subscriptions:items',
        'payments:read', 'payments:charge', 'payments:refund', 'payments:cancel',
        'webhooks:read', 'webhooks:manage',
        'logs:read',
        'wallets:read', 'wallets:payout', 'wallets:adjust',
        'commissions:read', 'commissions:manage',
        'invoices:read', 'invoices:write', 'invoices:manage',
    ],
    admin: [
        'users:read', 'users:invite',
        'credentials:read', 'credentials:manage',
        'customers:read', 'customers:write', 'customers:delete',
        'products:read', 'products:write',
        'plans:read', 'plans:write',
        'subscriptions:read', 'subscriptions:write', 'subscriptions:items',
        'payments:read', 'payments:charge', 'payments:refund', 'payments:cancel',
        'webhooks:read', 'webhooks:manage',
        'logs:read',
        'wallets:read', 'wallets:payout',
        'commissions:read', 'commissions:manage',
        'invoices:read', 'invoices:write', 'invoices:manage',
    ],
    developer: [
        'credentials:read', 'credentials:manage',
        'customers:read', 'customers:write',
        'products:read',
        'plans:read',
        'subscriptions:read',
        'payments:read', 'payments:charge',
        'webhooks:read', 'webhooks:manage',
        'logs:read',
        'wallets:read',
        'commissions:read',
        'invoices:read',
    ],
    support: [
        'customers:read', 'customers:write',
        'products:read',
        'plans:read',
        'subscriptions:read', 'subscriptions:write',
        'payments:read', 'payments:refund', 'payments:cancel',
    ],
} as const
