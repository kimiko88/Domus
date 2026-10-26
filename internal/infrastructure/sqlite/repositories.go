// Package sqlite provides repository container and database helpers.
package sqlite

import (
	"context"
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
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
)

// Repositories aggregates all initialized SQLite repositories.
type Repositories struct {
	DB                   *sql.DB
	ReadDB               *sql.DB
	Pool                 *DBPool
	Cipher               *crypto.EnvelopeCipher
	RawEntities          *EntityRepository
	CachedEntities       *CachedEntityRepository
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
	Pantry               *PantryRepository
	Medicines            *MedicineRepository
	PredictiveClimate    *PredictiveClimateRepository
	NVR                  *NVRRepository
	Anomaly              *AnomalyRepository
	Safety               *SafetyRepository
	Backup               *BackupRepository
	Vacation             *VacationRepository
	Commute              *CommuteRepository
	Wallbox              *WallboxRepository
}

// NewRepositories initializes and returns all SQLite repositories with a single database connection.
func NewRepositories(db *sql.DB) *Repositories {
	rawEntities := NewEntityRepository(db)
	cachedEntities := NewCachedEntityRepository(rawEntities)

	return &Repositories{
		DB:                   db,
		ReadDB:               db,
		RawEntities:          rawEntities,
		CachedEntities:       cachedEntities,
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
		Pantry:               NewPantryRepository(db),
		Medicines:            NewMedicineRepository(db),
		PredictiveClimate:    NewPredictiveClimateRepository(db),
		NVR:                  NewNVRRepository(db),
		Anomaly:              NewAnomalyRepository(db),
		Safety:               NewSafetyRepository(db),
		Backup:               NewBackupRepository(db),
		Vacation:             NewVacationRepository(db),
		Commute:              NewCommuteRepository(db),
		Wallbox:              NewWallboxRepository(db),
	}
}

// NewRepositoriesWithPool initializes all SQLite repositories using separate write and read database pools.
func NewRepositoriesWithPool(pool *DBPool) *Repositories {
	if pool == nil {
		return NewRepositories(nil)
	}
	writeDB := pool.Write
	readDB := pool.Read
	if readDB == nil {
		readDB = writeDB
	}

	rawEntities := NewEntityRepository(writeDB, readDB)
	cachedEntities := NewCachedEntityRepository(rawEntities)

	return &Repositories{
		DB:                   writeDB,
		ReadDB:               readDB,
		Pool:                 pool,
		RawEntities:          rawEntities,
		CachedEntities:       cachedEntities,
		Users:                NewUserRepository(writeDB),
		Sessions:             NewSessionRepository(writeDB),
		Audit:                NewAuditRepository(writeDB),
		Areas:                NewAreaRepository(writeDB),
		Integrations:         NewIntegrationRepository(writeDB),
		Devices:              NewDeviceRepository(writeDB),
		Entities:             cachedEntities,
		Automations:          NewAutomationRepository(writeDB),
		Scenes:               NewSceneRepository(writeDB),
		History:              NewHistoryRepository(writeDB, readDB),
		Invites:              NewInviteRepository(writeDB),
		Floors:               NewFloorplanRepository(writeDB),
		Chores:               NewChoreRepository(writeDB),
		Dashboards:           NewDashboardRepository(writeDB),
		Helpers:              NewHelperRepository(writeDB),
		Shopping:             NewShoppingRepository(writeDB),
		Calendar:             NewCalendarRepository(writeDB),
		Tags:                 NewTagRepository(writeDB),
		Media:                NewMediaRepository(writeDB),
		Cameras:              NewCameraRepository(writeDB),
		OTA:                  NewOTARepository(writeDB),
		Store:                NewStoreRepository(writeDB),
		Templates:            NewTemplateEntityRepository(writeDB),
		Statistics:           NewStatisticsRepository(writeDB, readDB),
		TwoFactor:            NewTwoFactorRepository(writeDB),
		BannedIPs:            NewBannedIPRepository(writeDB),
		Zones:                NewZoneRepository(writeDB),
		UtilityMeters:        NewUtilityMeterRepository(writeDB),
		Scripts:              NewScriptRepository(writeDB),
		Logbook:              NewLogbookRepository(writeDB, readDB),
		PushTokens:           NewPushTokenRepository(writeDB),
		WeeklySchedules:      NewWeeklyScheduleRepository(writeDB),
		ClimateCompensations: NewClimateCompensatorRepository(writeDB),
		OutboundWebhooks:     NewOutboundWebhookRepository(writeDB),
		Finance:              NewFinanceRepository(writeDB),
		Pantry:               NewPantryRepository(writeDB),
		Medicines:            NewMedicineRepository(writeDB),
		PredictiveClimate:    NewPredictiveClimateRepository(writeDB),
		NVR:                  NewNVRRepository(writeDB),
		Anomaly:              NewAnomalyRepository(writeDB),
		Safety:               NewSafetyRepository(writeDB),
		Backup:               NewBackupRepository(writeDB),
		Vacation:             NewVacationRepository(writeDB),
		Commute:              NewCommuteRepository(writeDB),
		Wallbox:              NewWallboxRepository(writeDB),
	}
}

// WarmUp preloads cacheable repositories (such as entities) into RAM.
func (r *Repositories) WarmUp(ctx context.Context) error {
	if r.CachedEntities != nil {
		return r.CachedEntities.WarmUp(ctx)
	}
	return nil
}

// WithCipher configures envelope encryption on all supported repositories storing sensitive data at rest.
func (r *Repositories) WithCipher(c *crypto.EnvelopeCipher) *Repositories {
	r.Cipher = c
	if whRepo, ok := r.OutboundWebhooks.(*OutboundWebhookRepository); ok && whRepo != nil {
		whRepo.WithCipher(c)
	}
	return r
}

// DatabaseSizeBytes returns the real storage size in bytes of the SQLite database.
func (r *Repositories) DatabaseSizeBytes(ctx context.Context) int64 {
	db := r.DB
	if db == nil {
		db = r.ReadDB
	}
	if db == nil {
		return 0
	}
	var pageCount, pageSize int64
	if err := db.QueryRowContext(ctx, "PRAGMA page_count;").Scan(&pageCount); err != nil {
		return 0
	}
	if err := db.QueryRowContext(ctx, "PRAGMA page_size;").Scan(&pageSize); err != nil {
		return 0
	}
	return pageCount * pageSize
}
