package domain

type Role uint8

const (
	RoleClient Role = 1 << iota
	RoleAnalyst
	RoleAdmin
)

type Permission string

const (
	PermissionChangeRole  Permission = "role:change"
	PermissionGetUserList Permission = "user:list"
)

var rolePermission = map[Role][]Permission{
	RoleAdmin: {
		PermissionChangeRole,
		PermissionGetUserList,
	},
}

func (r Role) HasPermission(p Permission) bool {
	for role, perms := range rolePermission {
		if r&role != role {
			continue
		}

		for _, perm := range perms {
			if perm == p {
				return true
			}
		}
	}

	return false
}
