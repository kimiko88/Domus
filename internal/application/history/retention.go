// Package history provides application services for state history recording and retention.
package history

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	domaudit "github.com/domus-hub/domus/internal/domain/audit"
	domauto "github.com/domus-hub/domus/internal/domain/automation"
	domhistory "github.com/domus-hub/domus/internal/domain/history"
)

// RetentionConfig configures retention policies in days for historical tables.
type RetentionConfig struct {
	StateHistoryDays  int
	AuditLogDays      int
	AutomationRunDays int
}

// PurgeSummary reports how many records were cleaned up across tables.
type PurgeSummary struct {
	StateHistoryPurged   int64 `json:"state_history_purged"`
	AuditLogPurged       int64 `json:"audit_log_purged"`
	AutomationRunsPurged int64 `json:"automation_runs_purged"`
}

// RetentionService periodically purges stale records from history, audit, and automation runs.
type RetentionService struct {
	historyRepo domhistory.Repository
	auditRepo   domaudit.Repository
	autoRepo    domauto.Repository
	cfg         RetentionConfig
	stopCh      chan struct{}
	mu          sync.Mutex
	running     bool
}

// NewRetentionService constructs a RetentionService.
func NewRetentionService(
	historyRepo domhistory.Repository,
	auditRepo domaudit.Repository,
	autoRepo domauto.Repository,
	cfg RetentionConfig,
) *RetentionService {
	return &RetentionService{
		historyRepo: historyRepo,
		auditRepo:   auditRepo,
		autoRepo:    autoRepo,
		cfg:         cfg,
		stopCh:      make(chan struct{}),
	}
}

// RunCleanup performs a single cleanup pass based on configured retention durations.
func (s *RetentionService) RunCleanup(ctx context.Context) (*PurgeSummary, error) {
	summary := &PurgeSummary{}
	now := time.Now().UTC()

	// 1. Purge state history
	if s.cfg.StateHistoryDays > 0 && s.historyRepo != nil {
		cutoff := now.AddDate(0, 0, -s.cfg.StateHistoryDays)
		n, err := s.historyRepo.PurgeOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to purge state history: %w", err)
		}
		summary.StateHistoryPurged = n
	}

	// 2. Purge audit logs
	if s.cfg.AuditLogDays > 0 && s.auditRepo != nil {
		cutoff := now.AddDate(0, 0, -s.cfg.AuditLogDays)
		n, err := s.auditRepo.PurgeOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to purge audit log: %w", err)
		}
		summary.AuditLogPurged = n
	}

	// 3. Purge automation runs
	if s.cfg.AutomationRunDays > 0 && s.autoRepo != nil {
		cutoff := now.AddDate(0, 0, -s.cfg.AutomationRunDays)
		n, err := s.autoRepo.PurgeRunsOlderThan(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to purge automation runs: %w", err)
		}
		summary.AutomationRunsPurged = n
	}

	return summary, nil
}

// Start begins a background ticker executing periodic retention purges.
func (s *RetentionService) Start(ctx context.Context, interval time.Duration) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		// Run initial cleanup pass on startup
		if summary, err := s.RunCleanup(ctx); err == nil {
			slog.Info("initial retention cleanup completed",
				"state_purged", summary.StateHistoryPurged,
				"audit_purged", summary.AuditLogPurged,
				"runs_purged", summary.AutomationRunsPurged,
			)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if summary, err := s.RunCleanup(ctx); err != nil {
					slog.Error("periodic retention cleanup error", "error", err)
				} else if summary.StateHistoryPurged > 0 || summary.AuditLogPurged > 0 || summary.AutomationRunsPurged > 0 {
					slog.Info("periodic retention cleanup finished",
						"state_purged", summary.StateHistoryPurged,
						"audit_purged", summary.AuditLogPurged,
						"runs_purged", summary.AutomationRunsPurged,
					)
				}
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the background retention cleanup loop.
func (s *RetentionService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}
