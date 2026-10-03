// Package config_test is the black-box test suite for the config package.
// Written BEFORE the implementation (TDD red phase).
package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domus-hub/domus/internal/infrastructure/config"
)

// ─── Load ────────────────────────────────────────────────────────────────────

func TestLoad_Defaults(t *testing.T) {
	t.Run("returns sensible defaults when path is empty", func(t *testing.T) {
		cfg, err := config.Load("")
		require.NoError(t, err)

		assert.Equal(t, "0.0.0.0", cfg.Server.Host)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.False(t, cfg.Server.CookieSecure)

		assert.Equal(t, "./data/domus.db", cfg.Database.Path)
		assert.True(t, cfg.Database.WALMode)
		assert.Equal(t, 5000, cfg.Database.BusyTimeoutMs)
		assert.Equal(t, 1, cfg.Database.MaxOpenConns)

		assert.Equal(t, "info", cfg.Logging.Level)
		assert.Equal(t, "json", cfg.Logging.Format)

		assert.Equal(t, 30, cfg.Retention.StateHistoryDays)
		assert.Equal(t, 90, cfg.Retention.AuditLogDays)
		assert.Equal(t, 30, cfg.Retention.AutomationRunDays)
	})

	t.Run("returns defaults when file does not exist", func(t *testing.T) {
		cfg, err := config.Load("/nonexistent/config.toml")
		require.NoError(t, err)
		assert.Equal(t, 8080, cfg.Server.Port)
	})
}

func TestLoad_FromTOML(t *testing.T) {
	t.Run("overrides server port and host from file", func(t *testing.T) {
		f := writeTempConfig(t, `
[server]
port = 9090
host = "127.0.0.1"
`)
		cfg, err := config.Load(f)
		require.NoError(t, err)

		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, "127.0.0.1", cfg.Server.Host)
		// Unspecified fields keep defaults
		assert.Equal(t, "./data/domus.db", cfg.Database.Path)
	})

	t.Run("overrides database path from file", func(t *testing.T) {
		f := writeTempConfig(t, `
[database]
path = "/srv/domus/domus.db"
busy_timeout_ms = 3000
`)
		cfg, err := config.Load(f)
		require.NoError(t, err)

		assert.Equal(t, "/srv/domus/domus.db", cfg.Database.Path)
		assert.Equal(t, 3000, cfg.Database.BusyTimeoutMs)
	})

	t.Run("overrides logging from file", func(t *testing.T) {
		f := writeTempConfig(t, `
[logging]
level = "debug"
format = "text"
`)
		cfg, err := config.Load(f)
		require.NoError(t, err)

		assert.Equal(t, "debug", cfg.Logging.Level)
		assert.Equal(t, "text", cfg.Logging.Format)
	})

	t.Run("parses session TTL duration from file", func(t *testing.T) {
		f := writeTempConfig(t, `
[auth]
session_ttl = "12h"
`)
		cfg, err := config.Load(f)
		require.NoError(t, err)

		assert.Equal(t, 12*time.Hour, cfg.Auth.SessionTTL.Duration)
	})

	t.Run("returns error for malformed TOML", func(t *testing.T) {
		f := writeTempConfig(t, `
[server
port = not_a_number
`)
		_, err := config.Load(f)
		assert.Error(t, err)
	})
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Run("DOMUS_SERVER_PORT overrides port", func(t *testing.T) {
		t.Setenv("DOMUS_SERVER_PORT", "7777")

		cfg, err := config.Load("")
		require.NoError(t, err)
		assert.Equal(t, 7777, cfg.Server.Port)
	})

	t.Run("DOMUS_SERVER_HOST overrides host", func(t *testing.T) {
		t.Setenv("DOMUS_SERVER_HOST", "192.168.1.100")

		cfg, err := config.Load("")
		require.NoError(t, err)
		assert.Equal(t, "192.168.1.100", cfg.Server.Host)
	})

	t.Run("DOMUS_DATABASE_PATH overrides db path", func(t *testing.T) {
		t.Setenv("DOMUS_DATABASE_PATH", "/tmp/test.db")

		cfg, err := config.Load("")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/test.db", cfg.Database.Path)
	})

	t.Run("DOMUS_LOG_LEVEL overrides log level", func(t *testing.T) {
		t.Setenv("DOMUS_LOG_LEVEL", "warn")

		cfg, err := config.Load("")
		require.NoError(t, err)
		assert.Equal(t, "warn", cfg.Logging.Level)
	})

	t.Run("DOMUS_LOG_FORMAT overrides log format", func(t *testing.T) {
		t.Setenv("DOMUS_LOG_FORMAT", "text")

		cfg, err := config.Load("")
		require.NoError(t, err)
		assert.Equal(t, "text", cfg.Logging.Format)
	})

	t.Run("invalid port env var is ignored", func(t *testing.T) {
		t.Setenv("DOMUS_SERVER_PORT", "not-a-number")

		cfg, err := config.Load("")
		require.NoError(t, err)
		assert.Equal(t, 8080, cfg.Server.Port) // keeps default
	})

	t.Run("env var takes precedence over file", func(t *testing.T) {
		t.Setenv("DOMUS_SERVER_PORT", "5000")

		f := writeTempConfig(t, "[server]\nport = 9000\n")
		cfg, err := config.Load(f)
		require.NoError(t, err)
		assert.Equal(t, 5000, cfg.Server.Port)
	})
}

// ─── ServerConfig ─────────────────────────────────────────────────────────────

func TestServerConfig_Addr(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{"default", "0.0.0.0", 8080, "0.0.0.0:8080"},
		{"localhost", "127.0.0.1", 9090, "127.0.0.1:9090"},
		{"empty host", "", 8080, ":8080"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.ServerConfig{Host: tc.host, Port: tc.port}
			assert.Equal(t, tc.want, cfg.Addr())
		})
	}
}

// ─── Auth defaults ────────────────────────────────────────────────────────────

func TestLoad_AuthDefaults(t *testing.T) {
	cfg, err := config.Load("")
	require.NoError(t, err)

	assert.Positive(t, cfg.Auth.Argon2Time, "Argon2 time must be > 0")
	assert.Positive(t, cfg.Auth.Argon2Memory, "Argon2 memory must be > 0")
	assert.Positive(t, cfg.Auth.Argon2Threads, "Argon2 threads must be > 0")
	assert.Greater(t, cfg.Auth.SessionTTL.Duration, time.Duration(0), "SessionTTL must be positive")
	assert.Positive(t, cfg.Auth.RateLimitLoginPerMin)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "domus.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
