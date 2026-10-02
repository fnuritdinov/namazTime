-- Хиджра и Рамадан по странам (§7, §14 ТЗ)

-- Начала месяцев хиджры. Зависят от наблюдения луны и различаются по странам.
CREATE TABLE hijri_months
(
    country    CHAR(2)  NOT NULL,
    hijri_year INTEGER  NOT NULL,
    month      SMALLINT NOT NULL CHECK (month BETWEEN 1 AND 12
) ,
    start_date   DATE        NOT NULL,
    status       TEXT        NOT NULL CHECK (status IN ('expected', 'confirmed')),
    confirmed_at TIMESTAMPTZ,
    source_id    TEXT        REFERENCES official_sources (id),

    PRIMARY KEY (country, hijri_year, month),
    -- подтверждённый месяц обязан иметь дату подтверждения
    CHECK (status = 'expected' OR confirmed_at IS NOT NULL)
);

-- Быстрый поиск «какой месяц идёт на эту дату» для страны
CREATE INDEX hijri_months_country_start_idx ON hijri_months (country, start_date);

-- Дуа
CREATE TABLE duas
(
    id              TEXT PRIMARY KEY, -- "iftar"
    category        TEXT    NOT NULL, -- "ramadan"
    arabic          TEXT    NOT NULL,
    transliteration TEXT,
    source          TEXT,             -- "Sunan Abi Dawud 2357"
    sort_order      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE dua_translations
(
    dua_id TEXT NOT NULL REFERENCES duas (id) ON DELETE CASCADE,
    lang   TEXT NOT NULL,
    text   TEXT NOT NULL,
    PRIMARY KEY (dua_id, lang)
);

-- Известно точно из таблицы Шурои уламо:
--   01.10.2026 = 19.04.1448  →  месяц 4 начался 13.09.2026
--   12.10.2026 = 01.05.1448  →  месяц 5 начался 12.10.2026
INSERT INTO hijri_months (country, hijri_year, month, start_date, status, confirmed_at, source_id)
VALUES ('TJ', 1448, 4, '2026-09-13', 'confirmed', now(), 'shuroiulamo-tj'),
       ('TJ', 1448, 5, '2026-10-12', 'confirmed', now(), 'shuroiulamo-tj');

-- Дуа для ифтара (текст и источник — из ТЗ §7)
INSERT INTO duas (id, category, arabic, transliteration, source, sort_order)
VALUES ('iftar', 'ramadan',
        'ذَهَبَ الظَّمَأُ وَابْتَلَّتِ الْعُرُوقُ وَثَبَتَ الأَجْرُ إِنْ شَاءَ اللَّهُ',
        'Dhahaba-ẓ-ẓamaʾu wa-btallati-l-ʿurūqu wa thabata-l-ajru in shāʾ Allāh',
        'Sunan Abi Dawud 2357', 1);

INSERT INTO dua_translations (dua_id, lang, text)
VALUES ('iftar', 'ru', 'Ушла жажда, увлажнились жилы, и утвердилась награда, если пожелает Аллах.'),
       ('iftar', 'en', 'The thirst has gone, the veins are moistened, and the reward is confirmed, if Allah wills.');