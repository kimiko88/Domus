package history_test

import (
	"context"
	"testing"
	"time"

	apphistory "github.com/domus-hub/domus/internal/application/history"
	domaudit "github.com/domus-hub/domus/internal/domain/audit"
	domauto "github.com/domus-hub/domus/internal/domain/automation"
)

type mockAuditRepo struct {
	purgedCutoff time.Time
	purgedCount  int64
}

func (m *mockAuditRepo) Record(ctx context.Context, entry *domaudit.Entry) error { return nil }
func (m *mockAuditRepo) List(ctx context.Context, limit int) ([]*domaudit.Entry, error) {
	return nil, nil
}
func (m *mockAuditRepo) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	m.purgedCutoff = cutoff
	m.purgedCount = 10
	return 10, nil
}

type mockAutoRepo struct {
	purgedCutoff time.Time
	purgedCount  int64
}

func (m *mockAutoRepo) Create(ctx context.Context, a *domauto.Automation) error { return nil }
func (m *mockAutoRepo) GetByID(ctx context.Context, id string) (*domauto.Automation, error) {
	return nil, nil
}
func (m *mockAutoRepo) List(ctx context.Context, filter domauto.ListFilter) ([]*domauto.Automation, error) {
	return nil, nil
}
func (m *mockAutoRepo) Update(ctx context.Context, a *domauto.Automation) error { return nil }
func (m *mockAutoRepo) UpdateLastRun(ctx context.Context, id string, status string) error {
	return nil
}
func (m *mockAutoRepo) Delete(ctx context.Context, id string) error                     { return nil }
func (m *mockAutoRepo) CreateRun(ctx context.Context, run *domauto.AutomationRun) error { return nil }
func (m *mockAutoRepo) UpdateRun(ctx context.Context, run *domauto.AutomationRun) error { return nil }
func (m *mockAutoRepo) ListRuns(ctx context.Context, automationID string, limit int) ([]*domauto.AutomationRun, error) {
	return nil, nil
}
func (m *mockAutoRepo) PurgeRunsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	m.purgedCutoff = cutoff
	m.purgedCount = 5
	return 5, nil
}

type mockHistoryRetentionRepo struct {
	mockHistoryRepo
	purgedCutoff time.Time
	purgedCount  int64
}

func (m *mockHistoryRetentionRepo) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	m.purgedCutoff = cutoff
	m.purgedCount = 42
	return 42, nil
}

func TestRetentionService_RunCleanup(t *testing.T) {
	ctx := context.Background()

	hRepo := &mockHistoryRetentionRepo{}
	auditRepo := &mockAuditRepo{}
	autoRepo := &mockAutoRepo{}

	cfg := apphistory.RetentionConfig{
		StateHistoryDays:  30,
		AuditLogDays:      90,
		AutomationRunDays: 14,
	}

	service := apphistory.NewRetentionService(hRepo, auditRepo, autoRepo, cfg)
	summary, err := service.RunCleanup(ctx)
	if err != nil {
		t.Fatalf("unexpected RunCleanup error: %v", err)
	}

	if summary.StateHistoryPurged != 42 {
		t.Fatalf("expected 42 state records purged, got %d", summary.StateHistoryPurged)
	}
	if summary.AuditLogPurged != 10 {
		t.Fatalf("expected 10 audit records purged, got %d", summary.AuditLogPurged)
	}
	if summary.AutomationRunsPurged != 5 {
		t.Fatalf("expected 5 automation runs purged, got %d", summary.AutomationRunsPurged)
	}

	// Verify cutoffs roughly match days
	now := time.Now().UTC()
	if now.Sub(hRepo.purgedCutoff) < 29*24*time.Hour {
		t.Fatalf("history cutoff too recent: %v", hRepo.purgedCutoff)
	}
	if now.Sub(auditRepo.purgedCutoff) < 89*24*time.Hour {
		t.Fatalf("audit cutoff too recent: %v", auditRepo.purgedCutoff)
	}
	if now.Sub(autoRepo.purgedCutoff) < 13*24*time.Hour {
		t.Fatalf("auto cutoff too recent: %v", autoRepo.purgedCutoff)
	}
}
