// Package roles defines the merchant team role hierarchy.
package roles

import "errors"

// Role values — stored as text in DB.
const (
	RoleOwner = "owner"
	RoleAdmin = "admin"
	RoleStaff = "staff"
)

// rank maps roles to comparable integers. Higher = more access.
var rank = map[string]int{
	RoleOwner: 3,
	RoleAdmin: 2,
	RoleStaff: 1,
}

// HasMinRole reports whether actorRole meets or exceeds the required minimum role.
func HasMinRole(actorRole, minRole string) bool {
	return rank[actorRole] >= rank[minRole]
}

// RequireRole returns an error if actorRole is below minRole.
func RequireRole(actorRole, minRole string) error {
	if !HasMinRole(actorRole, minRole) {
		return errors.New("insufficient role: " + minRole + " required, have " + actorRole)
	}
	return nil
}
