package user

type UserPermissions struct {
	UserID               string   `json:"user_id"`
	Role                 Role     `json:"role"`
	AllowedAreaIDs       []string `json:"allowed_area_ids"`      // Empty slice means access to all areas
	AllowedDashboardIDs  []string `json:"allowed_dashboard_ids"` // Empty slice means access to all dashboards
	CanManageAutomations bool     `json:"can_manage_automations"`
	CanManageSettings    bool     `json:"can_manage_settings"`
}

func DefaultAdminPermissions(userID string) *UserPermissions {
	return &UserPermissions{
		UserID:               userID,
		Role:                 RoleAdmin,
		AllowedAreaIDs:       nil,
		AllowedDashboardIDs:  nil,
		CanManageAutomations: true,
		CanManageSettings:    true,
	}
}

func DefaultMemberPermissions(userID string) *UserPermissions {
	return &UserPermissions{
		UserID:               userID,
		Role:                 RoleOperator,
		AllowedAreaIDs:       nil,
		AllowedDashboardIDs:  nil,
		CanManageAutomations: true,
		CanManageSettings:    false,
	}
}

func (p *UserPermissions) CanAccessArea(areaID string) bool {
	if p.Role == RoleAdmin || len(p.AllowedAreaIDs) == 0 {
		return true
	}
	for _, id := range p.AllowedAreaIDs {
		if id == areaID {
			return true
		}
	}
	return false
}

func (p *UserPermissions) CanAccessDashboard(dashboardID string) bool {
	if p.Role == RoleAdmin || len(p.AllowedDashboardIDs) == 0 {
		return true
	}
	for _, id := range p.AllowedDashboardIDs {
		if id == dashboardID {
			return true
		}
	}
	return false
}
