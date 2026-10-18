-- 000017_predictive_climate_profiles.up.sql
-- Learned thermal profiles for predictive climate zones (Grey-box 1R-1C thermal modeling with EMA self-learning).

CREATE TABLE IF NOT EXISTS predictive_climate_profiles (
    zone_id TEXT PRIMARY KEY,
    heat_rate_per_hr REAL NOT NULL DEFAULT 1.2,
    cool_rate_per_hr REAL NOT NULL DEFAULT 0.25,
    outdoor_loss_coeff REAL NOT NULL DEFAULT 0.02,
    solar_boost_coeff REAL NOT NULL DEFAULT 0.30,
    samples_count INTEGER NOT NULL DEFAULT 0,
    last_trained_at DATETIME NOT NULL,
    FOREIGN KEY(zone_id) REFERENCES predictive_climate_zones(id) ON DELETE CASCADE
);
