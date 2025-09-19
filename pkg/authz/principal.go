package authz

type Principal struct {
	UserID   int64
	RoleID   int64
	RoleName string
	Perms    map[string]struct{}
}

func (p *Principal) HasRole(role string) bool {
	return p != nil && (p.RoleName == "super-admin" || p.RoleName == role)
}
func (p *Principal) HasAny(perms ...string) bool {
	if p == nil {
		return false
	}
	if _, ok := p.Perms["root_access"]; ok {
		return true
	}
	for _, n := range perms {
		if _, ok := p.Perms[n]; ok {
			return true
		}
	}
	return false
}
func (p *Principal) HasAll(perms ...string) bool {
	if p == nil {
		return false
	}
	if _, ok := p.Perms["root_access"]; ok {
		return true
	}
	for _, n := range perms {
		if _, ok := p.Perms[n]; !ok {
			return false
		}
	}
	return true
}
