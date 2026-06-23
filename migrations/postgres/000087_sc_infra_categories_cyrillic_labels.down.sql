-- Revert to the original latin-script labels seeded in migration 000066.
-- Same UPDATE-by-slug shape — see 000087_*.up.sql for the rationale.

UPDATE sc_infra_event_categories SET label = 'Vaziyatlar markaziga integratsiya qilingan Tizim tashkilotlari Videokuzatuv tizimi holati' WHERE slug = 'video';
UPDATE sc_infra_event_categories SET label = 'Tizim tashkilotlari Aloqa tizimi holati' WHERE slug = 'comms';
UPDATE sc_infra_event_categories SET label = 'ASKUE holati to''g''risida ma''lumot' WHERE slug = 'ascue';
UPDATE sc_infra_event_categories SET label = 'ATNT holati to''g''risida ma''lumot' WHERE slug = 'atnt';
UPDATE sc_infra_event_categories SET label = 'Tizim tashkilotlarida kuzatilgan holatlar' WHERE slug = 'observation';
