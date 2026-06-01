CREATE TABLE reservoir_summary_config (
    id               BIGSERIAL PRIMARY KEY,
    organization_id  BIGINT NOT NULL UNIQUE
                     REFERENCES organizations(id) ON DELETE RESTRICT,
    sort_order       INT NOT NULL DEFAULT 0,
    include_in_total BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reservoir_summary_config_sort ON reservoir_summary_config(sort_order);

-- Initial seed: matches the current layout of res-summary.xlsx (8 slots).
-- Чотқол is summed into ИТОГО; Пском is shown below the totals row and is
-- excluded from the sum. Organization IDs are resolved by name to stay
-- stable across environments. If a name does not exist in `organizations`
-- the row is silently skipped — verify post-deploy that COUNT(*) = 8.
INSERT INTO reservoir_summary_config (organization_id, sort_order, include_in_total)
SELECT o.id, seed.ord, seed.in_total
FROM (VALUES
    ('Андижон сув омбори',   1, TRUE),
    ('Охангарон сув омбори', 2, TRUE),
    ('Сардоба сув омбори',   3, TRUE),
    ('Хисорак сув омбори',   4, TRUE),
    ('Топаланг сув омбори',  5, TRUE),
    ('Чорвок сув омбори',    6, TRUE),
    ('Қуйи Чоткол',          7, TRUE),
    ('Пском',                8, FALSE)
) AS seed(org_name, ord, in_total)
JOIN organizations o ON o.name = seed.org_name
ON CONFLICT (organization_id) DO NOTHING;
