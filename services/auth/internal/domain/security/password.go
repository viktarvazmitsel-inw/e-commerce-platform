package security

import "regexp"

var passwordPattern = regexp.MustCompile(`^[A-Za-z0-9!@#$%^&*()+]+$`)

func IsPasswordStrong(p string) bool {
	if len(p) < 8 || len(p) > 20 {
		return false
	}
	return passwordPattern.MatchString(p)
}
