// Package config loads and validates Domus configuration.
// Sources are applied in order of increasing priority:
//  1. Built-in defaults
//  2. TOML configuration file
//  3. Environment variables (DOMUS_* prefix)
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
)

// ─── Top-level config ─────────────────────────────────────────────────────────

// Config is the root configuration for Domus.
type Config struct {
	Server    ServerConfig    `toml:"server"`
	Database  DatabaseConfig  `toml:"database"`
	Auth      AuthConfig      `toml:"auth"`
	Logging   LoggingConfig   `toml:"logging"`
	Retention RetentionConfig `toml:"retention"`
	Presence  PresenceConfig  `toml:"presence"`
}

// ─── Sub-configs ──────────────────────────────────────────────────────────────

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	CookieSecure bool   `toml:"cookie_secure"`
	WebDir       string `toml:"web_dir"`
}

// Addr returns the "host:port" string used by net/http.
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig holds SQLite connection settings.
type DatabaseConfig struct {
	Path          string `toml:"path"`
	WALMode       bool   `toml:"wal_mode"`
	BusyTimeoutMs int    `toml:"busy_timeout_ms"`
	MaxOpenConns  int    `toml:"max_open_conns"`
}

// AuthConfig holds authentication and session settings.
type AuthConfig struct {
	SessionTTL           Duration `toml:"session_ttl"`
	RateLimitLoginPerMin int      `toml:"rate_limit_login_per_min"`
	Argon2Time           uint32   `toml:"argon2_time"`
	Argon2Memory         uint32   `toml:"argon2_memory"`
	Argon2Threads        uint8    `toml:"argon2_threads"`
}

// PresenceConfig configures the physical home location used for geofencing and solar calculations.
type PresenceConfig struct {
	HomeLatitude    float64 `toml:"home_latitude"`
	HomeLongitude   float64 `toml:"home_longitude"`
	GeofenceRadiusM float64 `toml:"geofence_radius_m"`
	GeofenceEnabled bool    `toml:"geofence_enabled"`
}

// LoggingConfig controls structured logging output.
type LoggingConfig struct {
	Level  string `toml:"level"`  // debug | info | warn | error
	Format string `toml:"format"` // json | text
}

// RetentionConfig controls how long historical data is kept.
type RetentionConfig struct {
	StateHistoryDays  int `toml:"state_history_days"`
	AuditLogDays      int `toml:"audit_log_days"`
	AutomationRunDays int `toml:"automation_run_days"`
}

// ─── Duration ─────────────────────────────────────────────────────────────────

// Duration wraps time.Duration to support TOML string unmarshaling (e.g. "24h").
type Duration struct {
	time.Duration
}

// UnmarshalText implements encoding.TextUnmarshaler for TOML/JSON.
func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", string(text), err)
	}
	d.Duration = dur
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// ─── Loader ───────────────────────────────────────────────────────────────────

// Load loads configuration from the given TOML file path, then applies
// environment variable overrides on top.
// If path is empty or the file does not exist, built-in defaults are used.
func Load(path string) (Config, error) {
	cfg := defaults()

	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &cfg); err != nil {
				return Config{}, fmt.Errorf("parse config %q: %w", path, err)
			}
		}
		// file not found → silently use defaults (not an error)
	}

	applyEnvOverrides(&cfg)
	return cfg, nil
}

// defaults returns the built-in default configuration.
func defaults() Config {
	return Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			CookieSecure: false,
		},
		Database: DatabaseConfig{
			Path:          "./data/domus.db",
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Auth: AuthConfig{
			SessionTTL:           Duration{24 * time.Hour},
			RateLimitLoginPerMin: 10,
			Argon2Time:           1,
			Argon2Memory:         64 * 1024, // 64 MiB
			Argon2Threads:        4,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Retention: RetentionConfig{
			StateHistoryDays:  30,
			AuditLogDays:      90,
			AutomationRunDays: 30,
		},
		Presence: PresenceConfig{
			GeofenceRadiusM: 150,
			GeofenceEnabled: false,
		},
	}
}

// applyEnvOverrides applies DOMUS_* environment variables on top of cfg.
// Invalid values are silently ignored (keeping the existing value).
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DOMUS_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("DOMUS_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("DOMUS_SERVER_WEB_DIR"); v != "" {
		cfg.Server.WebDir = v
	}
	if v := os.Getenv("DOMUS_DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("DOMUS_DATABASE_WAL_MODE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Database.WALMode = b
		}
	}
	if v := os.Getenv("DOMUS_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("DOMUS_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}
	if v := os.Getenv("DOMUS_PRESENCE_HOME_LATITUDE"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Presence.HomeLatitude = n
		}
	}
	if v := os.Getenv("DOMUS_PRESENCE_HOME_LONGITUDE"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Presence.HomeLongitude = n
		}
	}
	if v := os.Getenv("DOMUS_PRESENCE_GEOFENCE_RADIUS_M"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Presence.GeofenceRadiusM = n
		}
	}
	if v := os.Getenv("DOMUS_PRESENCE_GEOFENCE_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Presence.GeofenceEnabled = b
		}
	}
}
