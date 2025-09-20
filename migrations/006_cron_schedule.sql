-- Add cron schedule support to site configurations
DO $$ 
BEGIN
    -- Add cron_schedule column for cron expressions
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'cron_schedule') THEN
        ALTER TABLE site_configs ADD COLUMN cron_schedule VARCHAR(100) DEFAULT '';
    END IF;
    
    -- Add schedule_enabled flag to choose between interval and cron
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'schedule_enabled') THEN
        ALTER TABLE site_configs ADD COLUMN schedule_enabled BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

-- Update existing configurations with default cron schedules based on intervals
UPDATE site_configs 
SET 
    cron_schedule = CASE 
        WHEN check_interval <= 60 THEN '* * * * *'  -- каждую минуту
        WHEN check_interval <= 300 THEN '*/5 * * * *'  -- каждые 5 минут  
        WHEN check_interval <= 900 THEN '*/15 * * * *'  -- каждые 15 минут
        WHEN check_interval <= 1800 THEN '*/30 * * * *'  -- каждые 30 минут
        WHEN check_interval <= 3600 THEN '0 * * * *'  -- каждый час
        ELSE '0 */6 * * *'  -- каждые 6 часов
    END,
    schedule_enabled = FALSE,  -- По умолчанию используем интервалы
    updated_at = CURRENT_TIMESTAMP
WHERE cron_schedule = '' OR cron_schedule IS NULL;
