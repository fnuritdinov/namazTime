DELETE FROM time_adjustments
WHERE city_id IN ('istaravshan', 'kulob', 'khujand', 'rasht', 'konibodom', 'isfara',
                  'khorugh', 'bokhtar', 'panjakent', 'tursunzoda');

UPDATE cities
SET official_base_city_id = NULL,
    has_official_timetable = (id = 'dushanbe')
WHERE country = 'TJ';