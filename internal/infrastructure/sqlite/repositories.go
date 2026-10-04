// Package sqlite provides repository container and database helpers.
package sqlite

import (
	"database/sql"

	"github.com/domus-hub/domus/internal/domain/area"
	"github.com/domus-hub/domus/internal/domain/audit"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/history"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/domus-hub/domus/internal/domain/user"
)

// Repositories aggregates all initialized SQLite repositories.
type Repositories struct {
	Users        user.UserRepository
	Sessions     user.SessionRepository
	Audit        audit.Repository
	Areas        area.Repository
	Integrations integration.Repository
	Devices      device.Repository
	Entities     entity.Repository
	Automations  automation.Repository
	Scenes       scene.Repository
	History      history.Repository
	Invites      user.InviteRepository
}

// NewRepositories initializes and returns all SQLite repositories.
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Users:        NewUserRepository(db),
		Sessions:     NewSessionRepository(db),
		Audit:        NewAuditRepository(db),
		Areas:        NewAreaRepository(db),
		Integrations: NewIntegrationRepository(db),
		Devices:      NewDeviceRepository(db),
		Entities:     NewEntityRepository(db),
		Automations:  NewAutomationRepository(db),
		Scenes:       NewSceneRepository(db),
		History:      NewHistoryRepository(db),
		Invites:      NewInviteRepository(db),
	}
}
