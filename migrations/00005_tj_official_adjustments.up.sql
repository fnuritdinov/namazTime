-- Поправки Шурои уламо: «Тақвим мувофиқ ба шаҳри Душанбе гирифта шудааст».
-- Минус — раньше Душанбе (пеш аз), плюс — позже (баъд аз). Одинаково для всех намазов.
--
-- Пеш аз Душанбе: Истаравшан 1, Кӯлоб 4, Хуҷанд 3, Рашт 6, Конибодом 6, Исфара 7, Хоруғ 11
--                 (+ Ашт 6, Мурғоб 20, Ҳамадонӣ 3, Ш. Шоҳин 5, Муминобод 5 — этих городов пока нет в справочнике)
-- Баъд аз Душанбе: Бохтар 4, Панҷакент 5, Турсунзода 3
--                 (+ Шаҳритус 3, Айнӣ 1, Н. Хусрав 4 — пока нет в справочнике)
-- Ваҳдат, Ҳисор, Норак, Данғара в списке нет → время как в Душанбе (поправка 0).

-- Все города Таджикистана берут таблицу Душанбе
UPDATE cities
SET official_base_city_id = 'dushanbe',
    has_official_timetable = true
WHERE country = 'TJ' AND id <> 'dushanbe';

-- Поправка города × каждый намаз
INSERT INTO time_adjustments (city_id, prayer, minutes)
SELECT c.city_id, p.prayer, c.minutes
FROM (VALUES
          ('istaravshan', -1),
          ('kulob',       -4),
          ('khujand',     -3),
          ('rasht',       -6),
          ('konibodom',   -6),
          ('isfara',      -7),
          ('khorugh',    -11),
          ('bokhtar',      4),
          ('panjakent',    5),
          ('tursunzoda',   3)
     ) AS c (city_id, minutes)
         CROSS JOIN (VALUES ('imsak'), ('fajr'), ('sunrise'), ('dhuhr'), ('asr'), ('maghrib'), ('isha')) AS p (prayer);