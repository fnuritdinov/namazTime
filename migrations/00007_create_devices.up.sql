-- Анонимные устройства и push-подписки (§12 ТЗ). Ни телефона, ни email.

CREATE TABLE devices
(
    id           TEXT PRIMARY KEY,            -- dev_5f1c0a9e...
    token_hash   TEXT        NOT NULL UNIQUE, -- SHA-256 токена; сам токен не храним
    platform     TEXT        NOT NULL CHECK (platform IN ('ios')),
    app_version  TEXT        NOT NULL,
    locale       TEXT,
    time_zone_id TEXT,
    country      CHAR(2),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Для удаления неактивных больше 12 месяцев (§16 ТЗ)
CREATE INDEX devices_last_seen_idx ON devices (last_seen_at);

CREATE TABLE device_push
(
    device_id   TEXT PRIMARY KEY REFERENCES devices (id) ON DELETE CASCADE,
    apns_token  TEXT        NOT NULL UNIQUE,
    environment TEXT        NOT NULL CHECK (environment IN ('production', 'sandbox')),
    topics      TEXT[]      NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- GIN-индекс по массиву: быстро найти всех подписанных на тему
--   WHERE topics @> ARRAY['ramadan-TJ']
CREATE INDEX device_push_topics_idx ON device_push USING GIN (topics);