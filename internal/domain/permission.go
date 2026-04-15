package domain

// Permission represents a granular capability in the system.
// Format: {resource}:{action}
type Permission string

// Catalog of all available permissions.
const (
	PermUsersRead          Permission = "users:read"
	PermUsersInvite        Permission = "users:invite"
	PermUsersManage        Permission = "users:manage"
	PermCredentialsRead    Permission = "credentials:read"
	PermCredentialsManage  Permission = "credentials:manage"
	PermCustomersRead      Permission = "customers:read"
	PermCustomersWrite     Permission = "customers:write"
	PermCustomersDelete    Permission = "customers:delete"
	PermProductsRead       Permission = "products:read"
	PermProductsWrite      Permission = "products:write"
	PermPlansRead          Permission = "plans:read"
	PermPlansWrite         Permission = "plans:write"
	PermSubscriptionsRead  Permission = "subscriptions:read"
	PermSubscriptionsWrite Permission = "subscriptions:write"
	PermSubscriptionsItems Permission = "subscriptions:items"
	PermPaymentsRead       Permission = "payments:read"
	PermPaymentsCharge     Permission = "payments:charge"
	PermPaymentsRefund     Permission = "payments:refund"
	PermPaymentsCancel     Permission = "payments:cancel"
	PermWebhooksRead       Permission = "webhooks:read"
	PermWebhooksManage     Permission = "webhooks:manage"
	PermLogsRead           Permission = "logs:read"
)

// roleDefaultPermissions maps each role to its default permission set.
var roleDefaultPermissions = map[Role][]Permission{
	RoleOwner: {
		PermUsersRead, PermUsersInvite, PermUsersManage,
		PermCredentialsRead, PermCredentialsManage,
		PermCustomersRead, PermCustomersWrite, PermCustomersDelete,
		PermProductsRead, PermProductsWrite,
		PermPlansRead, PermPlansWrite,
		PermSubscriptionsRead, PermSubscriptionsWrite, PermSubscriptionsItems,
		PermPaymentsRead, PermPaymentsCharge, PermPaymentsRefund, PermPaymentsCancel,
		PermWebhooksRead, PermWebhooksManage,
		PermLogsRead,
	},
	RoleAdmin: {
		PermUsersRead, PermUsersInvite,
		PermCredentialsRead, PermCredentialsManage,
		PermCustomersRead, PermCustomersWrite, PermCustomersDelete,
		PermProductsRead, PermProductsWrite,
		PermPlansRead, PermPlansWrite,
		PermSubscriptionsRead, PermSubscriptionsWrite, PermSubscriptionsItems,
		PermPaymentsRead, PermPaymentsCharge, PermPaymentsRefund, PermPaymentsCancel,
		PermWebhooksRead, PermWebhooksManage,
		PermLogsRead,
	},
	RoleDeveloper: {
		PermCredentialsRead, PermCredentialsManage,
		PermCustomersRead, PermCustomersWrite,
		PermProductsRead,
		PermPlansRead,
		PermSubscriptionsRead,
		PermPaymentsRead, PermPaymentsCharge,
		PermWebhooksRead, PermWebhooksManage,
		PermLogsRead,
	},
	RoleSupport: {
		PermCustomersRead, PermCustomersWrite,
		PermProductsRead,
		PermPlansRead,
		PermSubscriptionsRead, PermSubscriptionsWrite,
		PermPaymentsRead, PermPaymentsRefund, PermPaymentsCancel,
	},
}

// DefaultPermissionsFor returns the default permission set for the given role.
func DefaultPermissionsFor(r Role) []Permission {
	return roleDefaultPermissions[r]
}

// HasDefaultPermission returns true if the role includes the given permission by default.
func HasDefaultPermission(r Role, p Permission) bool {
	for _, perm := range roleDefaultPermissions[r] {
		if perm == p {
			return true
		}
	}
	return false
}
