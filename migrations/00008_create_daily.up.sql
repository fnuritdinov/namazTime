-- Контент дня (§10 ТЗ): аят дня и напоминание дня.
-- По умолчанию — ротация по кругу; на конкретную дату редактор может задать своё.
-- Хадис дня добавится во второй итерации вместе с хадисами.

-- Аяты для ротации: только ссылка, текст Корана есть в приложении
CREATE TABLE daily_ayahs
(
    position SMALLINT PRIMARY KEY, -- порядок в круге
    surah    SMALLINT NOT NULL CHECK (surah BETWEEN 1 AND 114),
    ayah     SMALLINT NOT NULL CHECK (ayah >= 1)
);

-- Напоминания для ротации: {"ru": "...", "en": "...", "tg": "..."}
CREATE TABLE daily_reminders
(
    id    SMALLINT PRIMARY KEY,
    texts JSONB NOT NULL
);

-- Редакционный выбор на конкретную дату (перекрывает ротацию; NULL — взять из ротации)
CREATE TABLE daily_content
(
    date        DATE PRIMARY KEY,
    surah       SMALLINT CHECK (surah BETWEEN 1 AND 114),
    ayah        SMALLINT CHECK (ayah >= 1),
    reminder_id SMALLINT REFERENCES daily_reminders (id),
    CHECK ((surah IS NULL) = (ayah IS NULL)) -- аят задаётся целиком или никак
);

INSERT INTO daily_ayahs (position, surah, ayah)
VALUES (1, 2, 255),
       (2, 2, 286),
       (3, 2, 152),
       (4, 2, 153),
       (5, 2, 186),
       (6, 2, 201),
       (7, 3, 8),
       (8, 3, 139),
       (9, 3, 173),
       (10, 3, 190),
       (11, 3, 200),
       (12, 13, 28),
       (13, 14, 7),
       (14, 16, 97),
       (15, 17, 23),
       (16, 20, 114),
       (17, 21, 87),
       (18, 29, 69),
       (19, 39, 53),
       (20, 40, 60),
       (21, 49, 13),
       (22, 55, 13),
       (23, 57, 4),
       (24, 65, 2),
       (25, 65, 3),
       (26, 93, 5),
       (27, 94, 5),
       (28, 94, 6),
       (29, 103, 1),
       (30, 112, 1);

-- tg добавит редактор (как и другие таджикские переводы)
INSERT INTO daily_reminders (id, texts)
VALUES (1, '{
  "ru": "Начни день с «Бисмиллях» и намерения сделать доброе дело.",
  "en": "Start your day with Bismillah and an intention to do good."
}'),
       (2, '{
         "ru": "Посмотри время намазов на сегодня и спланируй дела вокруг них.",
         "en": "Check today''s prayer times and plan your day around them."
       }'),
       (3, '{
         "ru": "Прочитай сегодня хотя бы одну страницу Корана.",
         "en": "Read at least one page of the Quran today."
       }'),
       (4, '{
         "ru": "Дай сегодня садака — пусть даже совсем небольшую.",
         "en": "Give some sadaqah today, even a small amount."
       }'),
       (5, '{
         "ru": "Позвони родителям или родственникам.",
         "en": "Call your parents or relatives."
       }'),
       (6, '{
         "ru": "Перед сном вспомни, за что ты благодарен сегодня.",
         "en": "Before sleep, remember what you are grateful for today."
       }'),
       (7, '{
         "ru": "Сделай дуа за близких и за всех мусульман.",
         "en": "Make dua for your family and for all Muslims."
       }');