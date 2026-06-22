-- Adds modsnow_enabled flag to control whether an organization shows
-- modsnow data in /reservoir-summary report. When FALSE: Excel cell is
-- left empty, JSON response zeroes Modsnow.Current/YearAgo, frontend
-- renders the modsnow field as read-only.
--
-- Default TRUE keeps existing behaviour for every reservoir EXCEPT
-- Sardoba, which historically had modsnow skipped via a hardcoded
-- "if i == 2 { continue }" in the Excel generator. That hardcode is
-- removed in this release; the UPDATE below preserves the old visual
-- (no modsnow shown for Sardoba) by explicitly disabling the flag.
--
-- Sardoba is identified by organization name to stay portable across
-- envs (id varies between dev/prod). If no organization matches, the
-- UPDATE no-ops — safe.

ALTER TABLE reservoir_summary_config
    ADD COLUMN modsnow_enabled BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE reservoir_summary_config rsc
SET modsnow_enabled = FALSE
WHERE organization_id IN (
    SELECT id FROM organizations WHERE name ILIKE 'Сардоба%'
);
