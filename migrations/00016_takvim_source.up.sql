INSERT INTO official_sources (id, name) VALUES
    ('takvim-tj', '{"tg": "takvim.tj (Маркази исломии ҶТ)", "ru": "takvim.tj (Исламский центр РТ)", "en": "takvim.tj (Islamic Centre of Tajikistan)"}')
    ON CONFLICT (id) DO NOTHING;