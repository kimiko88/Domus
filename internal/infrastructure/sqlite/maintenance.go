// Package sqlite provides SQLite persistence, dual connection pooling, and maintenance tasks.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// MaintenanceStats reports database footprint and WAL checkpoint metrics.
type MaintenanceStats struct {
	Timestamp      time.Time `json:"timestamp"`
	DBSizeBytes    int64     `json:"db_size_bytes"`
	WALSizeBytes   int64     `json:"wal_size_bytes"`
	SHMSizeBytes   int64     `json:"shm_size_bytes"`
	Checkpointed   bool      `json:"checkpointed"`
	Busy           int       `json:"busy"`
	LogPages       int       `json:"log_pages"`
	CheckpointedPages int   `json:"checkpointed_pages"`
}

// MaintenanceConfig sets scheduling parameters for WAL checkpoint and storage optimization.
type MaintenanceConfig struct {
	CheckpointInterval time.Duration
	DBPath             string
}

// MaintenanceService manages scheduled checkpoints and flash wear protection.
type MaintenanceService struct {
	pool       *DBPool
	cfg        MaintenanceConfig
	mu         sync.Mutex
	lastStats  *MaintenanceStats
	shutdownCh chan struct{}
	stopOnce   sync.Once
}

// NewMaintenanceService creates an instance of the SQLite maintenance service.
func NewMaintenanceService(pool *DBPool, cfg MaintenanceConfig) *MaintenanceService {
	if cfg.CheckpointInterval <= 0 {
		cfg.CheckpointInterval = 4 * time.Hour
	}
	return &MaintenanceService{
		pool:       pool,
		cfg:        cfg,
		shutdownCh: make(chan struct{}),
	}
}

// RunCheckpoint executes PRAGMA wal_checkpoint(TRUNCATE) and incremental optimization.
func (m *MaintenanceService) RunCheckpoint(ctx context.Context) (MaintenanceStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := MaintenanceStats{
		Timestamp: time.Now(),
	}

	// Read file sizes on disk if not in-memory
	if m.cfg.DBPath != "" && m.cfg.DBPath != ":memory:" {
		if fi, err := os.Stat(m.cfg.DBPath); err == nil {
			stats.DBSizeBytes = fi.Size()
		}
		if fi, err := os.Stat(m.cfg.DBPath + "-wal"); err == nil {
			stats.WALSizeBytes = fi.Size()
		}
		if fi, err := os.Stat(m.cfg.DBPath + "-shm"); err == nil {
			stats.SHMSizeBytes = fi.Size()
		}
	}

	// Execute WAL checkpoint with TRUNCATE mode on writer connection
	row := m.pool.Write.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE);")
	var busy, logPages, checkpointedPages int
	if err := row.Scan(&busy, &logPages, &checkpointedPages); err != nil {
		slog.Warn("wal checkpoint warning (possible in-memory or busy lock)", "error", err)
	} else {
		stats.Busy = busy
		stats.LogPages = logPages
		stats.CheckpointedPages = checkpointedPages
		stats.Checkpointed = true
	}

	// Execute PRAGMA optimize for query planner efficiency
	_, _ = m.pool.Write.ExecContext(ctx, "PRAGMA optimize;")

	// Update WAL size post-truncate
	if m.cfg.DBPath != "" && m.cfg.DBPath != ":memory:" {
		if fi, err := os.Stat(m.cfg.DBPath + "-wal"); err == nil {
			stats.WALSizeBytes = fi.Size()
		} else {
			stats.WALSizeBytes = 0
		}
	}

	m.lastStats = &stats
	slog.Info("sqlite storage maintenance completed",
		"busy", stats.Busy,
		"log_pages", stats.LogPages,
		"checkpointed_pages", stats.CheckpointedPages,
		"db_bytes", stats.DBSizeBytes,
		"wal_bytes", stats.WALSizeBytes,
	)

	return stats, nil
}

// Start launches the background scheduler.
func (m *MaintenanceService) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.cfg.CheckpointInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.shutdownCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_, _ = m.RunCheckpoint(runCtx)
				cancel()
			}
		}
	}()
}

// Stop cleanly halts the maintenance scheduler.
func (m *MaintenanceService) Stop() {
	m.stopOnce.Do(func() {
		close(m.shutdownCh)
	})
}

// LastStats returns the most recent checkpoint statistics.
func (m *MaintenanceService) LastStats() *MaintenanceStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastStats
}

// DirectCheckpoint is a helper to run an immediate checkpoint on an arbitrary *sql.DB.
func DirectCheckpoint(ctx context.Context, db *sql.DB) error {
	row := db.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE);")
	var busy, logPages, checkpointedPages int
	if err := row.Scan(&busy, &logPages, &checkpointedPages); err != nil {
		return fmt.Errorf("direct wal checkpoint failed: %w", err)
	}
	return nil
}
