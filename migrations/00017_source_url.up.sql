-- Ссылка на сайт источника → sourceUrl в API
ALTER TABLE official_sources ADD COLUMN url TEXT;

UPDATE official_sources SET url = 'https://shuroiulamo.tj/ru/namaz' WHERE id = 'shuroiulamo-tj';
UPDATE official_sources SET url = 'https://takvim.tj' WHERE id = 'takvim-tj';