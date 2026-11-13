package domain

import "strconv"

const (
	RoleClient  = 1
	RoleAnalyst = 2
	RoleAdmin   = 4
)

type Principal struct {
	UserID string
	Role   string
}

func CanReadNotifications(role string) bool {
	value, err := strconv.Atoi(role)
	if err != nil || value <= 0 {
		return false
	}
	return value&(RoleAnalyst|RoleAdmin) != 0
}
