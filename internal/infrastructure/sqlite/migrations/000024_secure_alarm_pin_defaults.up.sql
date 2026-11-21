-- Clear the historical shared default PIN. Users must set a private PIN
-- before arming or disarming the alarm; custom legacy PINs are upgraded to
-- Argon2id by the alarm application service on first read.
UPDATE alarm_control_panels
SET pin_hash = NULL
WHERE pin_hash = '1234';
