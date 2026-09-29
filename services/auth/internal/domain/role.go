package domain

type Role uint8

const (
	RoleClient Role = 1 << iota
	RoleAnalyst
	RoleAdmin
)

type Permission string

const (
	PermissionChangeRole Permission = "role:change"
)

var rolePermission = map[Role][]Permission{
	RoleAdmin: {
		PermissionChangeRole,
	},
}

func (r Role) HasPermission(p Permission) bool {
	perms, exists := rolePermission[r]

	if !exists {
		return false
	}

	for _, perm := range perms {
		if perm == p {
			return true
		}
	}

	return false
}
