-- Android-приложение: разрешаем platform = 'android'
ALTER TABLE devices DROP CONSTRAINT devices_platform_check;
ALTER TABLE devices ADD CONSTRAINT devices_platform_check CHECK (platform IN ('ios', 'android'));