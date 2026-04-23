# GES Max Daily Production — Frontend Integration Guide

## TL;DR

`ges_config` got a new field `max_daily_production_mln_kwh` (NUMERIC NOT NULL DEFAULT 0, CHECK >= 0). When > 0 it's the daily upper bound on `daily_production_mln_kwh` accepted by `POST /ges-report/daily-data`. Value `0` ≡ "no cap" — backwards-compat for stations that aren't configured yet.

Backend changes: migration 000073, `model.Config` + `model.UpsertConfigRequest` extended, repo round-trips the column, `daily_data` upsert validates effective production ≤ cap with the same Optional semantics as aggregates.

## What changed in the API

### `GET /ges-report/config` — response gains a field

Each item in the array now includes:

```json
{
  "id": 1,
  "organization_id": 10,
  "organization_name": "ГЭС-1",
  "cascade_id": 5,
  "cascade_name": "Ўрта Чирчиқ каскади",
  "installed_capacity_mwt": 50.0,
  "total_aggregates": 4,
  "has_reservoir": true,
  "sort_order": 1,
  "max_daily_production_mln_kwh": 12.5
}
```

The field is **always present** (no `omitempty`). For stations that haven't been configured yet, the value is `0`.

### `POST /ges-report/config` — request accepts the new field

```json
{
  "organization_id": 10,
  "installed_capacity_mwt": 50.0,
  "total_aggregates": 4,
  "has_reservoir": true,
  "sort_order": 1,
  "max_daily_production_mln_kwh": 12.5
}
```

| Field | Validation |
| --- | --- |
| `max_daily_production_mln_kwh` | optional, default `0`, must be `>= 0` (server validator + DB CHECK) |

If you send a negative value → `400 Bad Request` with `validation_errors`. If validator is bypassed somehow, the DB CHECK fires and you also get a `400` (not 500).

### `POST /ges-report/daily-data` — new validation error

When the upsert payload pushes a `daily_production_mln_kwh` that exceeds the configured cap, you get:

```json
{
  "status": "Error",
  "error": "daily_production_mln_kwh exceeds max for organization_id=10: 13.0 > 12.5"
}
```

Status code: `400`. **No row is written** when this fires (atomic, like the aggregates-sum violation).

The check applies effective semantics:

- payload contains `daily_production_mln_kwh: 10.0` → checks `10.0 > cap`;
- payload omits the field → backend reads current DB value and checks **that** against the cap (so a partial update can't bypass the cap by leaving the field out);
- payload contains explicit `daily_production_mln_kwh: null` → treated as `0` (passes any cap).

If a station has `max_daily_production_mln_kwh == 0` (or no `ges_config` row at all) — no cap is enforced.

## What the frontend should do

### 1. Edit `ges_config` form (sc/rais only)

Add a numeric input bound to `max_daily_production_mln_kwh`:

- type=number
- min=0
- step is a UX choice (0.01 is fine for mln kWh)
- placeholder/hint: «макс. суточная выработка, млн кВт·ч (0 = без ограничения)»

POST it together with the existing fields.

### 2. Daily-data input form (sc/rais and cascade)

You already load `getConfigs()` in the first wave (forkJoin) of `loadData()`. For each station's row:

```ts
const cap = cfg.maxDailyProductionMlnKwh ?? 0;
if (cap > 0) {
  // dynamic <input max={cap}> on daily_production_mln_kwh
  // optional tooltip: «не более ${cap} млн кВт·ч»
}
// if cap === 0 → no client-side limit; backend will accept anything
```

### 3. Handle the `400` from daily-data save

When `POST /ges-report/daily-data` returns 400 with `error` matching:

```
daily_production_mln_kwh exceeds max for organization_id=N: D > M
```

show an inline error on the row for `organization_id=N`: «выработка D превышает максимум M». Highlight only the offending field; other rows in the batch saved fine? **No** — the upsert is atomic, the whole batch is rolled back. Either fix the offending row and resubmit, or split the batch.

### 4. Safe fallback before backend rollout

Use the nullish-coalesce so the form doesn't crash if you point at an old backend:

```ts
const cap = cfg.maxDailyProductionMlnKwh ?? 0;  // post-deploy: always present
```

After the migration is applied in prod, the field is always there.

## What the frontend does NOT need to do

- No new endpoint. No change to the forkJoin wave count.
- No change to the `cascade-config` / `report?date=...` / `daily-data?orgId=...` / `export` calls.
- No change to JWT / `auth.interceptor.ts`.

## Backend cross-references

- Migration: [`migrations/postgres/000073_ges_config_max_daily_production.up.sql`](../migrations/postgres/000073_ges_config_max_daily_production.up.sql)
- Model: [`internal/lib/model/ges-report/model.go`](../internal/lib/model/ges-report/model.go) — `Config`, `UpsertConfigRequest`
- Repo: [`internal/storage/repo/ges_report.go`](../internal/storage/repo/ges_report.go) — `UpsertGESConfig`, `GetAllGESConfigs`, `GetGESConfigsMaxDailyProduction`, `GetGESDailyProductionsBatch`
- Handler upsert: [`internal/http-server/handlers/ges-report/config.go`](../internal/http-server/handlers/ges-report/config.go)
- Handler validation: [`internal/http-server/handlers/ges-report/daily_data.go`](../internal/http-server/handlers/ges-report/daily_data.go) — `validateProductionCap`
- Full API spec: [`GES_DAILY_REPORT_API.md`](GES_DAILY_REPORT_API.md)
