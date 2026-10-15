// Package backup provides automated scheduled backup orchestration and offsite replication.
package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// SchedulerConfig defines configuration for automated night-time backups.
type SchedulerConfig struct {
	Enabled       bool                `json:"enabled"`
	Interval      time.Duration       `json:"interval"` // e.g. 24 * time.Hour
	DBPath        string              `json:"db_path"`
	ConfigPath    string              `json:"config_path"`
	OutputDir     string              `json:"output_dir"`
	Passphrase    string              `json:"passphrase"` // AES-256-GCM encryption key
	RetentionDays int                 `json:"retention_days"`
	Offsite       *OffsiteDestination `json:"offsite,omitempty"`
}

// Scheduler handles periodic backup execution and offsite export.
type Scheduler struct {
	service *Service
	cfg     SchedulerConfig
	bus     events.Bus
	logger  *slog.Logger

	mu     sync.Mutex
	stopCh chan struct{}
}

// NewScheduler creates a backup Scheduler.
func NewScheduler(service *Service, cfg SchedulerConfig, bus events.Bus, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 24 * time.Hour
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 7
	}
	return &Scheduler{
		service: service,
		cfg:     cfg,
		bus:     bus,
		logger:  logger,
	}
}

// RunOnce executes a single backup, encrypts it, exports offsite, and cleans old files.
func (s *Scheduler) RunOnce(ctx context.Context) (*CreateResult, error) {
	s.logger.InfoContext(ctx, "executing scheduled backup run", "output_dir", s.cfg.OutputDir)

	res, err := s.service.Create(CreateOptions{
		DBPath:     s.cfg.DBPath,
		ConfigPath: s.cfg.ConfigPath,
		OutputDir:  s.cfg.OutputDir,
		Passphrase: s.cfg.Passphrase,
	})
	if err != nil {
		return nil, fmt.Errorf("scheduled backup failed: %w", err)
	}

	// Offsite replication if configured
	if s.cfg.Offsite != nil && s.cfg.Offsite.Type != "" {
		s.logger.InfoContext(ctx, "exporting backup to offsite destination",
			"type", s.cfg.Offsite.Type, "endpoint", s.cfg.Offsite.Endpoint)
		if err := s.service.ExportOffsite(ctx, res, *s.cfg.Offsite); err != nil {
			s.logger.ErrorContext(ctx, "offsite backup export failed", "error", err)
		}
	}

	// Prune local backups older than retention days
	s.pruneOldBackups(ctx)

	// Emit event to EventBus
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EventType("backup.created"),
			OccurredAt: time.Now().UTC(),
			Source:     "backup:scheduler",
			Payload: map[string]any{
				"file_path":  res.FilePath,
				"size_bytes": res.SizeBytes,
				"encrypted":  res.Encrypted,
			},
		})
	}

	return &res, nil
}

// Start begins periodic background execution.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.stopCh != nil {
		s.mu.Unlock()
		return
	}
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.cfg.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.RunOnce(ctx)
			}
		}
	}()
}

// Stop stops the background scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
}

func (s *Scheduler) pruneOldBackups(ctx context.Context) {
	cutoff := time.Now().Add(-time.Duration(s.cfg.RetentionDays) * 24 * time.Hour)
	entries, err := os.ReadDir(s.cfg.OutputDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(s.cfg.OutputDir, entry.Name())
			_ = os.Remove(path)
			s.logger.InfoContext(ctx, "pruned expired backup", "path", path)
		}
	}
}
