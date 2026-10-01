-- Поиск с опечатками и по части слова (§15 ТЗ)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE cities (
                        id                     TEXT PRIMARY KEY,                 -- "dushanbe"
                        country                CHAR(2)          NOT NULL,        -- "TJ"
                        lat                    DOUBLE PRECISION NOT NULL,
                        lon                    DOUBLE PRECISION NOT NULL,
                        time_zone              TEXT             NOT NULL,        -- "Asia/Dushanbe"
                        suggested_method       TEXT             NOT NULL,        -- "muslimWorldLeague"
                        suggested_madhab       TEXT             NOT NULL CHECK (suggested_madhab IN ('hanafi', 'shafii')),
                        has_official_timetable BOOLEAN          NOT NULL DEFAULT false,
                        population             INTEGER,
                        geonames_id            INTEGER
);

-- «Популярные города страны» = сортировка по населению
CREATE INDEX cities_country_population_idx ON cities (country, population DESC NULLS LAST);

CREATE TABLE city_names (
                            city_id     TEXT NOT NULL REFERENCES cities (id) ON DELETE CASCADE,
                            lang        TEXT NOT NULL,                                   -- tg / ru / en / ar
                            name        TEXT NOT NULL,
                            region      TEXT,
    -- Нормализованное имя для поиска: нижний регистр + таджикские буквы → русские (§15)
                            name_search TEXT GENERATED ALWAYS AS (
                                translate(lower(name), 'ҳҷӣӯқғё', 'хчиукге')
                                ) STORED,

                            PRIMARY KEY (city_id, lang)
);

-- Триграммный индекс: быстрый поиск по подстроке ILIKE '%худж%'
CREATE INDEX city_names_search_trgm_idx ON city_names USING gin (name_search gin_trgm_ops);