-- Полные диапазоны намазов из таблицы Шуро: начало — существующие колонки
-- (fajr, dhuhr, asr, maghrib, isha), конец — новые *_end. Плюс время макрух.
-- isha_end — время СЛЕДУЮЩЕГО дня (19:49 - 04:50).
-- NULL — в загруженной таблице нет (старый CSV): конец считается по правилу.

ALTER TABLE official_timetable_days
    ADD COLUMN fajr_end TIME,
ADD COLUMN dhuhr_end TIME,
ADD COLUMN asr_end TIME,
ADD COLUMN makruh_start TIME,
ADD COLUMN makruh_end TIME,
ADD COLUMN maghrib_end TIME,
ADD COLUMN isha_end TIME;