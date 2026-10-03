-- Откат: та же таблица, что создавала миграция 00001 (пустая)
CREATE TABLE prayer_times (
                              id          BIGSERIAL PRIMARY KEY,
                              lat         NUMERIC(5, 2) NOT NULL,
                              lon         NUMERIC(5, 2) NOT NULL,
                              date        DATE          NOT NULL,
                              method      SMALLINT      NOT NULL,
                              school      SMALLINT      NOT NULL,
                              source      TEXT          NOT NULL,
                              hijri_date  TEXT,
                              fajr        TIME          NOT NULL,
                              sunrise     TIME          NOT NULL,
                              dhuhr       TIME          NOT NULL,
                              asr         TIME          NOT NULL,
                              maghrib     TIME          NOT NULL,
                              isha        TIME          NOT NULL,
                              created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),

                              CONSTRAINT prayer_times_unique UNIQUE (lat, lon, date, method, school)
);