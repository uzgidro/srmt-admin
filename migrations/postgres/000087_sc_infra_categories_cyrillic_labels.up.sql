-- Replace latin-script labels for sc_infra_event_categories with their
-- Uzbek-Cyrillic equivalents. The labels are what end up in the SC daily
-- report Excel header rows (rendered via processInfraEvents in
-- internal/lib/service/excel/sc/generator.go), and the report is consumed
-- by Uzbek-Cyrillic readers — latin script was the wrong locale for them.
--
-- Match by slug (UNIQUE) so this is safe to re-run even if id values
-- diverge between dev and prod. display_name and sort_order are left
-- alone — they are display-only Russian labels for the admin UI and are
-- already correct.

UPDATE sc_infra_event_categories SET label = 'Вазиятлар марказига интеграция қилинган Тизим ташкилотлари Видеокузатув тизими ҳолати' WHERE slug = 'video';
UPDATE sc_infra_event_categories SET label = 'Тизим ташкилотлари Алоқа тизими ҳолати' WHERE slug = 'comms';
UPDATE sc_infra_event_categories SET label = 'АСКУЭ ҳолати тўғрисида маълумот' WHERE slug = 'ascue';
UPDATE sc_infra_event_categories SET label = 'АТНТ ҳолати тўғрисида маълумот' WHERE slug = 'atnt';
UPDATE sc_infra_event_categories SET label = 'Тизим ташкилотларида кузатилган ҳолатлар' WHERE slug = 'observation';
