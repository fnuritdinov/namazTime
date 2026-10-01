UPDATE cities SET has_official_timetable = true WHERE country = 'TJ';
ALTER TABLE cities DROP COLUMN IF EXISTS official_base_city_id;
DROP TABLE IF EXISTS time_adjustments;
DROP TABLE IF EXISTS official_timetable_days;
DROP TABLE IF EXISTS official_timetables;
DROP TABLE IF EXISTS official_sources;