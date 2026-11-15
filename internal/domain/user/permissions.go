package user

// UserPermissions describes the resources and actions available to one household member.
// A nil allow-list means unrestricted access; a non-nil empty list means no access.
type UserPermissions struct {
	UserID               string   `json:"user_id,omitempty"`
	Role                 Role     `json:"role"`
	AllowedAreaIDs       []string `json:"allowed_area_ids"`
	AllowedEntityIDs     []string `json:"allowed_entity_ids"`
	AllowedDashboardIDs  []string `json:"allowed_dashboard_ids"`
	AllowedViews         []string `json:"allowed_views"`
	CanControlDevices    bool     `json:"can_control_devices"`
	CanManageAutomations bool     `json:"can_manage_automations"`
	CanManageSettings    bool     `json:"can_manage_settings"`
	CanViewCameras       bool     `json:"can_view_cameras"`
}

func DefaultPermissions(userID string, role Role) *UserPermissions {
	p := &UserPermissions{UserID: userID, Role: role}
	switch role {
	case RoleAdmin:
		p.CanControlDevices = true
		p.CanManageAutomations = true
		p.CanManageSettings = true
		p.CanViewCameras = true
	case RoleOperator:
		p.CanControlDevices = true
		p.CanManageAutomations = true
		p.CanViewCameras = true
	case RoleViewer:
		p.CanViewCameras = true
	case RoleKid:
		p.AllowedAreaIDs = []string{}
		p.AllowedEntityIDs = []string{}
		p.AllowedDashboardIDs = []string{}
		p.AllowedViews = []string{"/", "/media", "/chores", "/custom-dashboards"}
	case RoleGuest:
		p.AllowedAreaIDs = []string{}
		p.AllowedEntityIDs = []string{}
		p.AllowedDashboardIDs = []string{}
		p.AllowedViews = []string{"/", "/custom-dashboards", "/media"}
	}
	return p
}

func DefaultAdminPermissions(userID string) *UserPermissions {
	return DefaultPermissions(userID, RoleAdmin)
}

func DefaultMemberPermissions(userID string) *UserPermissions {
	return DefaultPermissions(userID, RoleOperator)
}

func DefaultKidPermissions(userID string) *UserPermissions {
	return DefaultPermissions(userID, RoleKid)
}

func DefaultGuestPermissions(userID string) *UserPermissions {
	return DefaultPermissions(userID, RoleGuest)
}

func (p *UserPermissions) CanAccessArea(areaID string) bool {
	return p != nil && (p.Role == RoleAdmin || containsOrUnrestricted(p.AllowedAreaIDs, areaID))
}

func (p *UserPermissions) CanAccessEntity(entityID string) bool {
	return p != nil && (p.Role == RoleAdmin || containsOrUnrestricted(p.AllowedEntityIDs, entityID))
}

func (p *UserPermissions) CanAccessDashboard(dashboardID string) bool {
	return p != nil && (p.Role == RoleAdmin || containsOrUnrestricted(p.AllowedDashboardIDs, dashboardID))
}

func (p *UserPermissions) CanAccessView(viewPath string) bool {
	return p != nil && (p.Role == RoleAdmin || containsOrUnrestricted(p.AllowedViews, viewPath))
}

func containsOrUnrestricted(values []string, value string) bool {
	if values == nil {
		return true
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
