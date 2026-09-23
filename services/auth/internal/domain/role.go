package domain

type Role int

const (
	RoleClient    Role = 1
	RoleAnalycist Role = 2
	RoleAdmin     Role = 3
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
