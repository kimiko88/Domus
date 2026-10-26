package user

type UserPermissions struct {
	UserID               string   `json:"user_id"`
	Role                 Role     `json:"role"`
	AllowedAreaIDs       []string `json:"allowed_area_ids"`      // Empty slice means access to all areas
	AllowedEntityIDs     []string `json:"allowed_entity_ids"`    // Empty slice means access to all entities
	AllowedDashboardIDs  []string `json:"allowed_dashboard_ids"` // Empty slice means access to all dashboards
	AllowedViews         []string `json:"allowed_views"`         // Empty slice means access to all views
	CanManageAutomations bool     `json:"can_manage_automations"`
	CanManageSettings    bool     `json:"can_manage_settings"`
	CanViewCameras       bool     `json:"can_view_cameras"`
}

func DefaultAdminPermissions(userID string) *UserPermissions {
	return &UserPermissions{
		UserID:               userID,
		Role:                 RoleAdmin,
		AllowedAreaIDs:       nil,
		AllowedEntityIDs:     nil,
		AllowedDashboardIDs:  nil,
		AllowedViews:         nil,
		CanManageAutomations: true,
		CanManageSettings:    true,
		CanViewCameras:       true,
	}
}

func DefaultMemberPermissions(userID string) *UserPermissions {
	return &UserPermissions{
		UserID:               userID,
		Role:                 RoleOperator,
		AllowedAreaIDs:       nil,
		AllowedEntityIDs:     nil,
		AllowedDashboardIDs:  nil,
		AllowedViews:         nil,
		CanManageAutomations: true,
		CanManageSettings:    false,
		CanViewCameras:       true,
	}
}

func DefaultKidPermissions(userID string) *UserPermissions {
	return &UserPermissions{
		UserID:               userID,
		Role:                 RoleKid,
		AllowedAreaIDs:       []string{},
		AllowedEntityIDs:     []string{},
		AllowedDashboardIDs:  []string{},
		AllowedViews:         []string{"/", "/media", "/chores", "/family", "/custom-dashboards"},
		CanManageAutomations: false,
		CanManageSettings:    false,
		CanViewCameras:       false,
	}
}

func DefaultGuestPermissions(userID string) *UserPermissions {
	return &UserPermissions{
		UserID:               userID,
		Role:                 RoleGuest,
		AllowedAreaIDs:       []string{},
		AllowedEntityIDs:     []string{},
		AllowedDashboardIDs:  []string{},
		AllowedViews:         []string{"/", "/custom-dashboards", "/media"},
		CanManageAutomations: false,
		CanManageSettings:    false,
		CanViewCameras:       false,
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

func (p *UserPermissions) CanAccessEntity(entityID string) bool {
	if p.Role == RoleAdmin || len(p.AllowedEntityIDs) == 0 {
		return true
	}
	for _, id := range p.AllowedEntityIDs {
		if id == entityID {
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

func (p *UserPermissions) CanAccessView(viewPath string) bool {
	if p.Role == RoleAdmin || len(p.AllowedViews) == 0 {
		return true
	}
	for _, v := range p.AllowedViews {
		if v == viewPath {
			return true
		}
	}
	return false
}
