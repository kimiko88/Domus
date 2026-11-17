-- 000021_predictive_climate_presence.down.sql
ALTER TABLE predictive_climate_zones DROP COLUMN away_temperature;
ALTER TABLE predictive_climate_zones DROP COLUMN presence_sensor_key;
