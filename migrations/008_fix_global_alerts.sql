-- Fix global alert configuration to enable alerts by default
UPDATE alert_configs
SET enabled = true,
    alert_on_down = true,
    alert_on_up = true
WHERE name = 'global' AND enabled = false;