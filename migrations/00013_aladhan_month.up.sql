-- Кеш ответов Aladhan: один месяц одного города для метода и мазхаба.
CREATE TABLE aladhan_months
(
    city_id TEXT     NOT NULL REFERENCES cities (id) ON DELETE CASCADE,
    year    INT      NOT NULL,
    month   SMALLINT NOT NULL CHECK (month BETWEEN 1 AND 12
) ,
    method     SMALLINT    NOT NULL,
    school     SMALLINT    NOT NULL,
    days       JSONB       NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (city_id, year, month, method, school)
);

-- Все страны, кроме Таджикистана, по умолчанию — Карачи
UPDATE cities
SET suggested_method = 'karachi'
WHERE country <> 'TJ';