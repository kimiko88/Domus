-- 000021_predictive_climate_presence.up.sql
-- Optional occupancy-aware away setpoints for predictive climate zones.
ALTER TABLE predictive_climate_zones
    ADD COLUMN presence_sensor_key TEXT NOT NULL DEFAULT '';
ALTER TABLE predictive_climate_zones
    ADD COLUMN away_temperature REAL NOT NULL DEFAULT 17.0;
