INSERT INTO dua_translations (dua_id, lang, text)
VALUES ('iftar', 'tg', 'Ташнагӣ рафт, рагҳо тар шуданд ва савоб собит гашт, агар Аллоҳ хоҳад.')
    ON CONFLICT (dua_id, lang) DO UPDATE SET text = EXCLUDED.text;