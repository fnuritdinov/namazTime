-- Города Таджикистана. Население — приблизительное (для сортировки «популярных»).
-- Официальное расписание: Шурои уламо (Душанбе + поправки по городам) — has_official_timetable = true.
INSERT INTO cities (id, country, lat, lon, time_zone, suggested_method, suggested_madhab, has_official_timetable, population) VALUES
                                                                                                                                  ('dushanbe',    'TJ', 38.5598, 68.7870, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true, 1200000),
                                                                                                                                  ('khujand',     'TJ', 40.2826, 69.6222, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,  183000),
                                                                                                                                  ('bokhtar',     'TJ', 37.8364, 68.7803, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,  120000),
                                                                                                                                  ('kulob',       'TJ', 37.9146, 69.7845, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,  110000),
                                                                                                                                  ('istaravshan', 'TJ', 39.9108, 69.0064, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   70000),
                                                                                                                                  ('tursunzoda',  'TJ', 38.5108, 68.2303, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   60000),
                                                                                                                                  ('konibodom',   'TJ', 40.2941, 70.4312, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   55000),
                                                                                                                                  ('vahdat',      'TJ', 38.5563, 69.0135, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   50000),
                                                                                                                                  ('isfara',      'TJ', 40.1265, 70.6253, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   50000),
                                                                                                                                  ('panjakent',   'TJ', 39.4952, 67.6093, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   45000),
                                                                                                                                  ('hisor',       'TJ', 38.5249, 68.5513, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   30000),
                                                                                                                                  ('norak',       'TJ', 38.3883, 69.3222, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   30000),
                                                                                                                                  ('khorugh',     'TJ', 37.4897, 71.5530, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   30000),
                                                                                                                                  ('danghara',    'TJ', 38.0950, 69.3389, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   25000),
                                                                                                                                  ('rasht',       'TJ', 39.0213, 70.3793, 'Asia/Dushanbe', 'muslimWorldLeague', 'hanafi', true,   10000);

INSERT INTO city_names (city_id, lang, name, region) VALUES
                                                         ('dushanbe', 'tg', 'Душанбе', 'Шаҳри Душанбе'),
                                                         ('dushanbe', 'ru', 'Душанбе', 'г. Душанбе'),
                                                         ('dushanbe', 'en', 'Dushanbe', 'Dushanbe'),
                                                         ('dushanbe', 'ar', 'دوشنبه', NULL),

                                                         ('khujand', 'tg', 'Хуҷанд', 'Вилояти Суғд'),
                                                         ('khujand', 'ru', 'Худжанд', 'Согдийская область'),
                                                         ('khujand', 'en', 'Khujand', 'Sughd'),
                                                         ('khujand', 'ar', 'خجند', NULL),

                                                         ('bokhtar', 'tg', 'Бохтар', 'Вилояти Хатлон'),
                                                         ('bokhtar', 'ru', 'Бохтар', 'Хатлонская область'),
                                                         ('bokhtar', 'en', 'Bokhtar', 'Khatlon'),

                                                         ('kulob', 'tg', 'Кӯлоб', 'Вилояти Хатлон'),
                                                         ('kulob', 'ru', 'Куляб', 'Хатлонская область'),
                                                         ('kulob', 'en', 'Kulob', 'Khatlon'),

                                                         ('istaravshan', 'tg', 'Истаравшан', 'Вилояти Суғд'),
                                                         ('istaravshan', 'ru', 'Истаравшан', 'Согдийская область'),
                                                         ('istaravshan', 'en', 'Istaravshan', 'Sughd'),

                                                         ('tursunzoda', 'tg', 'Турсунзода', 'Ноҳияҳои тобеи ҷумҳурӣ'),
                                                         ('tursunzoda', 'ru', 'Турсунзаде', 'Районы республиканского подчинения'),
                                                         ('tursunzoda', 'en', 'Tursunzoda', 'Districts of Republican Subordination'),

                                                         ('konibodom', 'tg', 'Конибодом', 'Вилояти Суғд'),
                                                         ('konibodom', 'ru', 'Канибадам', 'Согдийская область'),
                                                         ('konibodom', 'en', 'Konibodom', 'Sughd'),

                                                         ('vahdat', 'tg', 'Ваҳдат', 'Ноҳияҳои тобеи ҷумҳурӣ'),
                                                         ('vahdat', 'ru', 'Вахдат', 'Районы республиканского подчинения'),
                                                         ('vahdat', 'en', 'Vahdat', 'Districts of Republican Subordination'),

                                                         ('isfara', 'tg', 'Исфара', 'Вилояти Суғд'),
                                                         ('isfara', 'ru', 'Исфара', 'Согдийская область'),
                                                         ('isfara', 'en', 'Isfara', 'Sughd'),

                                                         ('panjakent', 'tg', 'Панҷакент', 'Вилояти Суғд'),
                                                         ('panjakent', 'ru', 'Пенджикент', 'Согдийская область'),
                                                         ('panjakent', 'en', 'Panjakent', 'Sughd'),

                                                         ('hisor', 'tg', 'Ҳисор', 'Ноҳияҳои тобеи ҷумҳурӣ'),
                                                         ('hisor', 'ru', 'Гиссар', 'Районы республиканского подчинения'),
                                                         ('hisor', 'en', 'Hisor', 'Districts of Republican Subordination'),

                                                         ('norak', 'tg', 'Норак', 'Вилояти Хатлон'),
                                                         ('norak', 'ru', 'Нурек', 'Хатлонская область'),
                                                         ('norak', 'en', 'Norak', 'Khatlon'),

                                                         ('khorugh', 'tg', 'Хоруғ', 'ВМКБ'),
                                                         ('khorugh', 'ru', 'Хорог', 'ГБАО'),
                                                         ('khorugh', 'en', 'Khorugh', 'Gorno-Badakhshan'),

                                                         ('danghara', 'tg', 'Данғара', 'Вилояти Хатлон'),
                                                         ('danghara', 'ru', 'Дангара', 'Хатлонская область'),
                                                         ('danghara', 'en', 'Danghara', 'Khatlon'),

                                                         ('rasht', 'tg', 'Рашт', 'Ноҳияҳои тобеи ҷумҳурӣ'),
                                                         ('rasht', 'ru', 'Рашт', 'Районы республиканского подчинения'),
                                                         ('rasht', 'en', 'Rasht', 'Districts of Republican Subordination');