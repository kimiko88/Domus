// Package sqlite provides repository container and database helpers.
package sqlite

import (
	"database/sql"

	"github.com/domus-hub/domus/internal/domain/area"
	"github.com/domus-hub/domus/internal/domain/audit"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/chore"
	"github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/domain/dashboard"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/energy"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/floorplan"
	"github.com/domus-hub/domus/internal/domain/helper"
	"github.com/domus-hub/domus/internal/domain/history"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/domain/logbook"
	"github.com/domus-hub/domus/internal/domain/notification"
	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/domus-hub/domus/internal/domain/schedule"
	domscript "github.com/domus-hub/domus/internal/domain/script"
	"github.com/domus-hub/domus/internal/domain/template_entity"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/domain/webhook"
	"github.com/domus-hub/domus/internal/domain/zone"
)

// Repositories aggregates all initialized SQLite repositories.
type Repositories struct {
	DB                   *sql.DB
	Users                user.UserRepository
	Sessions             user.SessionRepository
	Audit                audit.Repository
	Areas                area.Repository
	Integrations         integration.Repository
	Devices              device.Repository
	Entities             entity.Repository
	Automations          automation.Repository
	Scenes               scene.Repository
	History              history.Repository
	Invites              user.InviteRepository
	Floors               floorplan.Repository
	Chores               chore.Repository
	Dashboards           dashboard.Repository
	Helpers              helper.Repository
	Shopping             *ShoppingRepository
	Calendar             *CalendarRepository
	Tags                 *TagRepository
	Media                *MediaRepository
	Cameras              *CameraRepository
	OTA                  *OTARepository
	Store                *StoreRepository
	Templates            template_entity.Repository
	Statistics           history.StatisticsRepository
	TwoFactor            *TwoFactorRepository
	BannedIPs            *BannedIPRepository
	Zones                zone.Repository
	UtilityMeters        energy.UtilityMeterRepository
	Scripts              domscript.Repository
	Logbook              logbook.Repository
	PushTokens           notification.PushTokenRepository
	WeeklySchedules      schedule.Repository
	ClimateCompensations climate.Repository
	OutboundWebhooks     webhook.OutboundRepository
	Finance              *FinanceRepository
}

// NewRepositories initializes and returns all SQLite repositories.
func NewRepositories(db *sql.DB) *Repositories {
	rawEntities := NewEntityRepository(db)
	cachedEntities := NewCachedEntityRepository(rawEntities)

	return &Repositories{
		DB:                   db,
		Users:                NewUserRepository(db),
		Sessions:             NewSessionRepository(db),
		Audit:                NewAuditRepository(db),
		Areas:                NewAreaRepository(db),
		Integrations:         NewIntegrationRepository(db),
		Devices:              NewDeviceRepository(db),
		Entities:             cachedEntities,
		Automations:          NewAutomationRepository(db),
		Scenes:               NewSceneRepository(db),
		History:              NewHistoryRepository(db),
		Invites:              NewInviteRepository(db),
		Floors:               NewFloorplanRepository(db),
		Chores:               NewChoreRepository(db),
		Dashboards:           NewDashboardRepository(db),
		Helpers:              NewHelperRepository(db),
		Shopping:             NewShoppingRepository(db),
		Calendar:             NewCalendarRepository(db),
		Tags:                 NewTagRepository(db),
		Media:                NewMediaRepository(db),
		Cameras:              NewCameraRepository(db),
		OTA:                  NewOTARepository(db),
		Store:                NewStoreRepository(db),
		Templates:            NewTemplateEntityRepository(db),
		Statistics:           NewStatisticsRepository(db),
		TwoFactor:            NewTwoFactorRepository(db),
		BannedIPs:            NewBannedIPRepository(db),
		Zones:                NewZoneRepository(db),
		UtilityMeters:        NewUtilityMeterRepository(db),
		Scripts:              NewScriptRepository(db),
		Logbook:              NewLogbookRepository(db),
		PushTokens:           NewPushTokenRepository(db),
		WeeklySchedules:      NewWeeklyScheduleRepository(db),
		ClimateCompensations: NewClimateCompensatorRepository(db),
		OutboundWebhooks:     NewOutboundWebhookRepository(db),
		Finance:              NewFinanceRepository(db),
	}
}
