-- Официальные расписания намазов (§6, §14 ТЗ)

-- Кто публикует расписание (Шурои уламо, муфтият и т.д.)
CREATE TABLE official_sources (
                                  id   TEXT PRIMARY KEY,         -- "shuroiulamo-tj"
                                  name JSONB NOT NULL            -- {"tg": "...", "ru": "...", "en": "..."} → sourceName в API
);

-- Таблица расписания: город + источник + год
CREATE TABLE official_timetables (
                                     id          BIGSERIAL PRIMARY KEY,
                                     city_id     TEXT        NOT NULL REFERENCES cities (id) ON DELETE CASCADE,
                                     source_id   TEXT        NOT NULL REFERENCES official_sources (id),
                                     year        INTEGER     NOT NULL,
                                     method_note TEXT,
                                     updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

                                     CONSTRAINT official_timetables_unique UNIQUE (city_id, source_id, year)
);

-- Дни расписания. Время — местное (в часовом поясе города).
CREATE TABLE official_timetable_days (
                                         timetable_id BIGINT   NOT NULL REFERENCES official_timetables (id) ON DELETE CASCADE,
                                         date         DATE     NOT NULL,
                                         fajr         TIME     NOT NULL,
                                         sunrise      TIME     NOT NULL,
                                         dhuhr        TIME     NOT NULL,
                                         asr          TIME     NOT NULL,
                                         maghrib      TIME     NOT NULL,
                                         isha         TIME     NOT NULL,
                                         imsak        TIME,                -- если источник не публикует — посчитаем как Фаджр − 10 мин
                                         hijri_day    SMALLINT,            -- официальная дата хиджры, если есть в таблице
                                         hijri_month  SMALLINT,
                                         hijri_year   SMALLINT,

                                         PRIMARY KEY (timetable_id, date)
);

-- Ручные поправки ± минут на каждый намаз (§6: «админка… + ручные поправки»)
CREATE TABLE time_adjustments (
                                  city_id TEXT    NOT NULL REFERENCES cities (id) ON DELETE CASCADE,
                                  prayer  TEXT    NOT NULL CHECK (prayer IN ('imsak', 'fajr', 'sunrise', 'dhuhr', 'asr', 'maghrib', 'isha')),
                                  minutes INTEGER NOT NULL CHECK (minutes BETWEEN -120 AND 120),

                                  PRIMARY KEY (city_id, prayer)
);

-- Город может брать официальное расписание ДРУГОГО города + свои поправки.
-- Так публикует Шурои уламо: таблица для Душанбе, для остальных городов — «±N минут».
ALTER TABLE cities
    ADD COLUMN official_base_city_id TEXT REFERENCES cities (id);

-- Источник для Таджикистана
INSERT INTO official_sources (id, name) VALUES
    ('shuroiulamo-tj', '{"tg": "Шӯрои уламои Тоҷикистон", "ru": "Совет улемов Таджикистана", "en": "Council of Ulema of Tajikistan"}');

-- Пока официальное расписание будет только у Душанбе.
-- Остальные города получат его в D.3.4, когда внесём поправки Шурои уламо.
UPDATE cities SET has_official_timetable = (id = 'dushanbe') WHERE country = 'TJ';