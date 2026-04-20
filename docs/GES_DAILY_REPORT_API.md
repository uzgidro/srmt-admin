# GES Daily Report Module — API Documentation

## Overview

Module automates daily GES (hydroelectric station) operational reporting. Operators enter raw data per station, and the system computes all derived values: power output, diffs from yesterday, month-to-date/year-to-date production, plan fulfillment, year-over-year comparison, and idle discharge integration.

**Base path:** `/ges-report`

**Authorization:** Requires JWT with roles `sc`, `rais`, or `cascade`. Access scope:

- **`sc` / `rais`** — full access to all endpoints
- **`cascade`** — restricted to own cascade (see [ges-cascade-role.md](ges-cascade-role.md))

**Related documentation:**

- [ges-cascade-role.md](ges-cascade-role.md) — `cascade` role, access scope per endpoint
- [ges-daily-data-partial-update.md](ges-daily-data-partial-update.md) — `Optional[T]` partial updates and bulk array body for `POST /daily-data`
- [ges-aggregates.md](ges-aggregates.md) — working / repair / modernization / reserve aggregates (rules, validation, 400 errors)
- [ges-cascade-weather.md](ges-cascade-weather.md) — automatic weather collection (background ticker)
- [ges-cascade-daily-weather.md](ges-cascade-daily-weather.md) — manual weather correction endpoint
- [ges-export.md](ges-export.md) — Excel/PDF export endpoint

---

## Database Schema

### `ges_config` — Static GES configuration

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | BIGSERIAL | PK | |
| `organization_id` | BIGINT | NOT NULL, UNIQUE | FK → organizations. One config per GES |
| `installed_capacity_mwt` | NUMERIC | NOT NULL, default 0 | Installed capacity in MW |
| `total_aggregates` | INT | NOT NULL, default 0 | Total turbine-generator units |
| `has_reservoir` | BOOLEAN | NOT NULL, default false | Whether station has a reservoir |
| `sort_order` | INT | NOT NULL, default 0 | Display order in report |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |

### `ges_daily_data` — Daily operational measurements

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | BIGSERIAL | PK | |
| `organization_id` | BIGINT | NOT NULL | FK → organizations |
| `date` | DATE | NOT NULL | Report date (YYYY-MM-DD) |
| `daily_production_mln_kwh` | NUMERIC | NOT NULL, default 0 | Daily electricity production (million kWh) |
| `working_aggregates` | INT | NOT NULL, default 0 | Currently operating units |
| `repair_aggregates` | INT | NOT NULL, default 0 | Units currently under repair. `CHECK (>= 0)` + trigger `working + repair + modernization <= ges_config.total_aggregates` |
| `modernization_aggregates` | INT | NOT NULL, default 0 | Units currently under modernization. Same CHECK + sum trigger |
| `water_level_m` | NUMERIC | YES | Upper pool water level (m above sea level) |
| `water_volume_mln_m3` | NUMERIC | YES | Reservoir water volume (million m³) |
| `water_head_m` | NUMERIC | YES | Net hydraulic head (m) |
| `reservoir_income_m3s` | NUMERIC | YES | Water inflow (m³/s) |
| `total_outflow_m3s` | NUMERIC | YES | Total water discharge (m³/s) |
| `ges_flow_m3s` | NUMERIC | YES | Water through turbines (m³/s) |
| `created_by_user_id` | BIGINT | YES | FK → users |
| `updated_by_user_id` | BIGINT | YES | FK → users |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |

**Unique constraint:** `(organization_id, date)` — supports upsert (re-entering data overwrites)

**Note:** `temperature` and `weather_condition` are **no longer** stored on `ges_daily_data`. Weather is per-cascade and lives in `cascade_daily_data` — see [ges-cascade-weather.md](ges-cascade-weather.md).

### `cascade_config` — Cascade configuration

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | BIGSERIAL | PK | |
| `organization_id` | BIGINT | NOT NULL, UNIQUE | FK → organizations (the cascade org) |
| `latitude` | DOUBLE PRECISION | YES | Cascade location latitude (for weather API) |
| `longitude` | DOUBLE PRECISION | YES | Cascade location longitude |
| `sort_order` | INT | NOT NULL, default 0 | Display order |
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |

A cascade is an organization that has child organizations (stations) via `organizations.parent_organization_id`.

### `cascade_daily_data` — Per-cascade daily weather

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | BIGSERIAL | PK | |
| `organization_id` | BIGINT | NOT NULL | FK → organizations (cascade) |
| `date` | DATE | NOT NULL | Report date |
| `temperature` | NUMERIC | YES | Air temperature (°C) |
| `weather_condition` | TEXT | YES | OWM icon code (`01d`, `10n`, …) |
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |

**Unique constraint:** `(organization_id, date)` — one weather record per cascade per day. Filled automatically by the 04:00 ticker via OpenWeatherMap, can be corrected manually via `POST /ges-report/cascade-daily-data`.

### `ges_production_plan` — Monthly production targets

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | BIGSERIAL | PK | |
| `organization_id` | BIGINT | NOT NULL | FK → organizations |
| `year` | INT | NOT NULL | Plan year (2020–2100) |
| `month` | INT | NOT NULL | Plan month (1–12) |
| `plan_mln_kwh` | NUMERIC | NOT NULL, default 0 | Production target (million kWh) |
| `created_by_user_id` | BIGINT | NOT NULL | FK → users |
| `updated_by_user_id` | BIGINT | NOT NULL | FK → users |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |

**Unique constraint:** `(organization_id, year, month)` — supports upsert

---

## Workflow

### 1. Initial Setup (once)

Register each cascade and station:

```text
POST /ges-report/cascade-config    # cascade with coordinates for weather
POST /ges-report/config            # each GES station
```

### 2. Set Production Plans (monthly/yearly)

Enter monthly production targets (bulk):

```text
POST /ges-report/plans
```

### 3. Daily Data Entry (per GES, bulk)

Operators enter daily measurements as **array** (supports partial updates via `Optional[T]` — see [ges-daily-data-partial-update.md](ges-daily-data-partial-update.md)):

```text
POST /ges-report/daily-data
```

### 4. View Report

Get the full computed report for any date:

```text
GET /ges-report?date=2026-03-13
```

For `cascade` role users, only their own cascade is returned.

### 5. Manual Weather Correction (optional)

Background ticker fills weather automatically at 04:00. To override:

```text
POST /ges-report/cascade-daily-data
```

See [ges-cascade-daily-weather.md](ges-cascade-daily-weather.md).

### 6. Excel/PDF Export

```text
GET /ges-report/export?date=2026-03-13
```

`sc`/`rais` only. Repair/modernization values come from `ges_daily_data` — see [ges-export.md](ges-export.md) for the breaking change and [ges-aggregates.md](ges-aggregates.md) for the data model.

---

## API Endpoints

### 1. `POST /ges-report/config` — Upsert GES Configuration

Creates or updates static configuration for a GES station.

**Request Body:**

```json
{
  "organization_id": 10,
  "installed_capacity_mwt": 50.0,
  "total_aggregates": 4,
  "has_reservoir": true,
  "sort_order": 1
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `organization_id` | int64 | Yes | > 0 | Organization (GES) ID |
| `installed_capacity_mwt` | float64 | No | >= 0 | Installed power capacity (MW) |
| `total_aggregates` | int | No | >= 0 | Number of turbine units |
| `has_reservoir` | bool | No | | Whether station has reservoir |
| `sort_order` | int | No | >= 0 | Display order |

**Responses:**

| Status | Body | When |
|--------|------|------|
| 200 | `{"status": "OK"}` | Config saved/updated |
| 400 | `{"status": "Error", "message": "..."}` | Invalid JSON or validation failure |
| 400 | `{"status": "Error", "validation_errors": [...]}` | Validation errors |
| 500 | `{"status": "Error", "message": "..."}` | Database error |

**Behavior:** If config exists for `organization_id`, it updates all fields. If not, creates new record. `updated_at` is set to `NOW()` on update.

---

### 2. `GET /ges-report/config` — List All Configs

Returns all configured GES stations with organization and cascade names.

**Parameters:** None

**Response (200):**

```json
[
  {
    "id": 1,
    "organization_id": 10,
    "organization_name": "ГЭС-1",
    "cascade_id": 5,
    "cascade_name": "Ўрта Чирчиқ каскади",
    "installed_capacity_mwt": 50.0,
    "total_aggregates": 4,
    "has_reservoir": true,
    "sort_order": 1
  }
]
```

Returns empty array `[]` if no configs exist.

---

### 3. `DELETE /ges-report/config?organization_id=N` — Delete Config

**Query Parameters:**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `organization_id` | int64 | Yes | GES to remove from report |

**Responses:**

| Status | When |
|--------|------|
| 204 | Config deleted |
| 400 | Missing or invalid `organization_id` |
| 404 | Config not found for this organization |
| 500 | Database error |

---

### 4. `POST /ges-report/daily-data` — Upsert Daily Data (bulk)

Enter or update daily operational data for **multiple GES stations** (body is an array). Supports partial updates via `Optional[T]` — see [ges-daily-data-partial-update.md](ges-daily-data-partial-update.md) for the full three-state contract.

**Request Body** (array, even for one item):

```json
[
  {
    "organization_id": 10,
    "date": "2026-03-13",
    "daily_production_mln_kwh": 3.389,
    "working_aggregates": 3,
    "repair_aggregates": 1,
    "modernization_aggregates": 0,
    "water_level_m": 846.05,
    "water_volume_mln_m3": 634.0,
    "water_head_m": 104.0,
    "reservoir_income_m3s": 85.5,
    "total_outflow_m3s": 95.0,
    "ges_flow_m3s": 85.0
  }
]
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `organization_id` | int64 | Yes | GES ID (must belong to user's cascade if role is `cascade`) |
| `date` | string | Yes | YYYY-MM-DD |
| `daily_production_mln_kwh` | `Optional[float64]` | No | Daily production (million kWh). NOT NULL in DB — `null` writes 0 |
| `working_aggregates` | `Optional[int]` | No | Operating units. `gte=0`. NOT NULL in DB — `null` writes 0 |
| `repair_aggregates` | `Optional[int]` | No | Units under repair. `gte=0`. NOT NULL in DB — `null` writes 0. See [ges-aggregates.md](ges-aggregates.md) |
| `modernization_aggregates` | `Optional[int]` | No | Units under modernization. `gte=0`. NOT NULL in DB — `null` writes 0. See [ges-aggregates.md](ges-aggregates.md) |
| `water_level_m` | `Optional[float64]` | No | Reservoir level (m), nullable |
| `water_volume_mln_m3` | `Optional[float64]` | No | Reservoir volume, nullable |
| `water_head_m` | `Optional[float64]` | No | Hydraulic head, nullable |
| `reservoir_income_m3s` | `Optional[float64]` | No | Inflow, nullable |
| `total_outflow_m3s` | `Optional[float64]` | No | Total outflow, nullable |
| `ges_flow_m3s` | `Optional[float64]` | No | Turbine flow, nullable |

**Note:** `temperature` and `weather_condition` are **not accepted** here — they live on `cascade_daily_data` and are managed via the ticker or [POST /cascade-daily-data](ges-cascade-daily-weather.md).

**Responses:**

| Status | When |
|--------|------|
| 200 | All items saved (atomic transaction) |
| 400 | Empty array, invalid JSON, validation failure, invalid date, or `item_index` error |
| 400 | Aggregate field `< 0` — `"{field} must be >= 0 for organization_id=N, got X"` |
| 400 | Effective `working + repair + modernization > ges_config.total_aggregates` — `"aggregates sum exceeds total for organization_id=N: W+R+M=S > T"` |
| 401 | Not authenticated |
| 403 | `cascade` user tried to write a station outside their cascade |
| 500 | Database error |

**Behavior:** Atomic bulk upsert on `(organization_id, date)`. Partial updates per the `Optional[T]` contract.

**Aggregate sum validation.** Handler checks `working + repair + modernization <= ges_config.total_aggregates` per `(organization_id, date)` tuple before writing, using current DB values for fields not present in the request. Missing `ges_config` row → check skipped (matches the DB trigger). A DB-level trigger re-runs the same check inside the transaction as a race-safety net. Example 400 body:

```json
{
  "status": "Error",
  "error": "aggregates sum exceeds total for organization_id=10: 4+2+1=7 > 6"
}
```

See [ges-aggregates.md](ges-aggregates.md) for the full rules.

---

### 5. `GET /ges-report/daily-data?organization_id=N&date=YYYY-MM-DD` — Get Daily Data

Returns raw daily data for a single GES on a specific date.

**Query Parameters:**

| Param | Type | Required |
|-------|------|----------|
| `organization_id` | int64 | Yes |
| `date` | string | Yes (YYYY-MM-DD) |

**Response (200):**

```json
{
  "id": 42,
  "organization_id": 10,
  "date": "2026-03-13",
  "daily_production_mln_kwh": 3.389,
  "working_aggregates": 3,
  "repair_aggregates": 1,
  "modernization_aggregates": 0,
  "water_level_m": 846.05,
  "water_volume_mln_m3": 634.0,
  "water_head_m": 104.0,
  "reservoir_income_m3s": 85.5,
  "total_outflow_m3s": 95.0,
  "ges_flow_m3s": 85.0
}
```

`reserve_aggregates` is **not** returned here — it is computed by the report service, see [GET /ges-report](#8-get-ges-reportdateyyyy-mm-dd--full-daily-report).

**Note:** weather fields removed — see [GET /ges-report response](#7-get-ges-reportdateyyyy-mm-dd--full-daily-report) where weather is on `cascades[].weather`.

**Errors:** 400 (missing/invalid params), 403 (`cascade` user requesting foreign station), 404 (no data for this org+date), 500

---

### 6. `POST /ges-report/plans` — Bulk Upsert Production Plans

Enter or update monthly production targets for multiple GES at once.

**Request Body:**

```json
{
  "plans": [
    { "organization_id": 10, "year": 2026, "month": 1, "plan_mln_kwh": 100.0 },
    { "organization_id": 10, "year": 2026, "month": 2, "plan_mln_kwh": 110.0 },
    { "organization_id": 10, "year": 2026, "month": 3, "plan_mln_kwh": 120.0 },
    { "organization_id": 20, "year": 2026, "month": 3, "plan_mln_kwh": 80.0 }
  ]
}
```

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `plans` | array | Yes | min 1 item |
| `plans[].organization_id` | int64 | Yes | > 0 |
| `plans[].year` | int | Yes | 2020–2100 |
| `plans[].month` | int | Yes | 1–12 |
| `plans[].plan_mln_kwh` | float64 | No | >= 0 |

**Responses:** 200 (saved), 400 (validation), 401 (not authenticated), 500 (database error)

**Behavior:** Atomic transaction. Upsert on `(organization_id, year, month)`.

---

### 7. `GET /ges-report/plans?year=N` — Get Plans for Year

Returns all production plans for the given year.

**Response (200):**

```json
[
  { "id": 1, "organization_id": 10, "year": 2026, "month": 1, "plan_mln_kwh": 100.0 },
  { "id": 2, "organization_id": 10, "year": 2026, "month": 2, "plan_mln_kwh": 110.0 }
]
```

---

### 8. `GET /ges-report?date=YYYY-MM-DD` — Full Daily Report

**The main endpoint.** Returns the complete daily report with all computed values, grouped by cascade.

**Query Parameters:**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `date` | string | Yes | Report date (YYYY-MM-DD) |

**Response (200):**

```json
{
  "date": "2026-03-13",
  "cascades": [
    {
      "cascade_id": 5,
      "cascade_name": "Ўрта Чирчиқ ГЭСлар каскади",
      "weather": {
        "temperature": 1.2,
        "weather_condition": "01d",
        "prev_year_temperature": 3.5,
        "prev_year_condition": "04d"
      },
      "summary": { "..." },
      "stations": [
        {
          "organization_id": 10,
          "name": "ГЭС-1",
          "config": {
            "installed_capacity_mwt": 50.0,
            "total_aggregates": 4,
            "has_reservoir": true
          },
          "current": {
            "daily_production_mln_kwh": 3.389,
            "power_mwt": 141.208,
            "working_aggregates": 3,
            "repair_aggregates": 1,
            "modernization_aggregates": 0,
            "reserve_aggregates": 0,
            "water_level_m": 846.05,
            "water_volume_mln_m3": 634.0,
            "water_head_m": 104.0,
            "reservoir_income_m3s": 85.5,
            "total_outflow_m3s": 95.0,
            "ges_flow_m3s": 85.0,
            "idle_discharge_m3s": 10.0
          },
          "diffs": {
            "level_change_cm": 8.0,
            "volume_change_mln_m3": -1.5,
            "income_change_m3s": 5.0,
            "ges_flow_change_m3s": -2.0,
            "power_change_mwt": 12.5,
            "production_change_mln_kwh": 0.3
          },
          "aggregations": {
            "mtd_production_mln_kwh": 42.5,
            "ytd_production_mln_kwh": 280.0
          },
          "plan": {
            "monthly_plan_mln_kwh": 120.0,
            "quarterly_plan_mln_kwh": 330.0,
            "fulfillment_pct": 0.8485,
            "difference_mln_kwh": -50.0
          },
          "previous_day": {
            "daily_production_mln_kwh": 3.2,
            "power_mwt": 133.3,
            "working_aggregates": 3,
            "repair_aggregates": 1,
            "modernization_aggregates": 0,
            "reserve_aggregates": 2,
            "water_level_m": 845.9,
            "water_volume_mln_m3": 632.5,
            "water_head_m": 103.8,
            "reservoir_income_m3s": 83.0,
            "total_outflow_m3s": 92.0,
            "ges_flow_m3s": 82.0,
            "idle_discharge_m3s": 10.0
          },
          "previous_year": {
            "water_level_m": 840.2,
            "water_volume_mln_m3": 580.0,
            "water_head_m": 100.0,
            "reservoir_income_m3s": 70.0,
            "ges_flow_m3s": 75.0,
            "power_mwt": 130.0,
            "daily_production_mln_kwh": 3.12,
            "mtd_production_mln_kwh": 40.0,
            "ytd_production_mln_kwh": 310.0
          },
          "yoy": {
            "growth_rate": -0.0968,
            "difference_mln_kwh": -30.0
          },
          "idle_discharge": {
            "flow_rate_m3s": 530.0,
            "volume_mln_m3": 1.5,
            "reason": "Паводок",
            "is_ongoing": true
          }
        }
      ]
    }
  ],
  "grand_total": {
    "installed_capacity_mwt": 2413.362,
    "total_aggregates": 180,
    "working_aggregates": 120,
    "repair_aggregates": 8,
    "modernization_aggregates": 3,
    "reserve_aggregates": 49,
    "power_mwt": 712.0,
    "daily_production_mln_kwh": 17.087,
    "production_change_mln_kwh": 0.5,
    "mtd_production_mln_kwh": 200.0,
    "ytd_production_mln_kwh": 1022.172,
    "monthly_plan_mln_kwh": 489.85,
    "quarterly_plan_mln_kwh": 1400.898,
    "fulfillment_pct": 0.7297,
    "difference_mln_kwh": -378.726,
    "prev_year_ytd_mln_kwh": 1307.863,
    "yoy_growth_rate": -0.218,
    "yoy_difference_mln_kwh": -285.691,
    "idle_discharge_total_m3s": 540.0
  }
}
```

---

## Computed Fields — Formulas

All computations happen in the service layer (Go code), not in the database.

### Per-Station Computations

| Field | Formula | Unit | Notes |
|-------|---------|------|-------|
| `power_mwt` | `daily_production * 1000 / 24` | МВт | Average power from daily production |
| `idle_discharge_m3s` | `total_outflow - ges_flow` | м³/с | `null` if either is null |
| `level_change_cm` | `(today.level - yesterday.level) * 100` | см | `null` if either is null |
| `volume_change_mln_m3` | `today.volume - yesterday.volume` | млн.м³ | `null` if either is null |
| `income_change_m3s` | `today.income - yesterday.income` | м³/с | `null` if either is null |
| `ges_flow_change_m3s` | `today.ges_flow - yesterday.ges_flow` | м³/с | `null` if either is null |
| `power_change_mwt` | `today.power - yesterday.power` | МВт | `null` if no yesterday data |
| `production_change` | `today.production - yesterday.production` | млн.кВт.ч | `null` if no yesterday data |
| `mtd_production` | `SUM(daily_production) WHERE date IN [month_start..today]` | млн.кВт.ч | |
| `ytd_production` | `SUM(daily_production) WHERE date IN [year_start..today]` | млн.кВт.ч | |
| `fulfillment_pct` | `ytd / quarterly_plan` | ratio | `null` if quarterly_plan = 0 |
| `difference` | `ytd - quarterly_plan` | млн.кВт.ч | Negative = behind plan |
| `prev_year.*` | Same-date data from previous year | | `null` if no data |
| `prev_year.mtd/ytd` | Aggregated from prev year's data | млн.кВт.ч | |
| `yoy.growth_rate` | `(ytd / prev_year_ytd) - 1` | ratio | `null` if prev_year_ytd = 0 |
| `yoy.difference` | `ytd - prev_year_ytd` | млн.кВт.ч | Negative = decline |

### Cascade & Grand Total

**SUM** across all stations for: `installed_capacity_mwt`, `total_aggregates`, `working_aggregates`, `repair_aggregates`, `modernization_aggregates`, `power_mwt`, `daily_production`, `production_change`, `mtd`, `ytd`, `monthly_plan`, `quarterly_plan`, `prev_year_ytd`, `idle_discharge_m3s`.

Then derived fields computed from the sums: `fulfillment_pct`, `difference`, `yoy_growth_rate`, `yoy_difference`, `reserve_aggregates`.

### Reserve Aggregates (computed, not stored)

```text
reserve_aggregates = total_aggregates - working_aggregates - repair_aggregates - modernization_aggregates
```

Present in `stations[].current`, `cascades[].summary`, and `grand_total`. The frontend never sends this field — it is recomputed on every `GET /ges-report`. Clamped to `0` if the operator's inputs make it negative (service logs a warning with `organization_id`). The `working + repair + modernization <= total_aggregates` invariant is enforced at write time (see [POST /daily-data](#4-post-ges-reportdaily-data--upsert-daily-data-bulk) + [ges-aggregates.md](ges-aggregates.md)).

### Previous day snapshot

`stations[].previous_day` — **полный слепок** данных за `date − 1` (день ранее),
симметричный `current`. Добавлен для того, чтобы фронт мог показывать виджеты
«сегодня vs вчера» и сам считать любые дельты, не ограничиваясь фиксированным
набором из `diffs`.

Содержит те же поля, что и `current`:

| Поле | Тип | Nullable | Описание |
| --- | --- | --- | --- |
| `daily_production_mln_kwh` | float64 | No | Производство за вчерашний день (млн.кВт.ч) |
| `power_mwt` | float64 | No | Средняя мощность = `daily_production * 1000 / 24` |
| `working_aggregates` | int | No | Работавших агрегатов |
| `repair_aggregates` | int | No | Агрегатов в ремонте |
| `modernization_aggregates` | int | No | Агрегатов в модернизации |
| `reserve_aggregates` | int | No | Резерв = `total - working - repair - modernization`, clamp к 0 |
| `water_level_m` | float64 | Yes | Уровень воды в верхнем бьефе (м) |
| `water_volume_mln_m3` | float64 | Yes | Объём воды в водохранилище (млн.м³) |
| `water_head_m` | float64 | Yes | Напор (м) |
| `reservoir_income_m3s` | float64 | Yes | Приток (м³/с) |
| `total_outflow_m3s` | float64 | Yes | Общий сброс (м³/с) |
| `ges_flow_m3s` | float64 | Yes | Расход через турбины (м³/с) |
| `idle_discharge_m3s` | float64 | Yes | Холостой сброс = `total_outflow - ges_flow`, `null` если одно из слагаемых отсутствует |

Всё поле `previous_day` становится `null`, если в `ges_daily_data` нет строки
за `date − 1` для этой станции. На уровне cascade summary и grand total
`previous_day` **не добавлен** — только в `stations[]`.

**Отличие от `diffs`.** Поле `diffs` содержит уже вычисленные дельты
(`level_change_cm`, `volume_change_mln_m3`, `income_change_m3s`,
`ges_flow_change_m3s`, `power_change_mwt`, `production_change_mln_kwh`) и
остаётся в ответе для обратной совместимости и для генератора Excel.
`previous_day` — сами значения за вчера; фронт может вычислять **любые**
дельты, в том числе те, которых нет в `diffs` (например, по агрегатам
или по холостому сбросу).

Пример расчёта дельт на клиенте:

```js
const levelChangeCm = (current.water_level_m - previous_day.water_level_m) * 100;
const productionChange = current.daily_production_mln_kwh - previous_day.daily_production_mln_kwh;
const workingDelta = current.working_aggregates - previous_day.working_aggregates;
```

Reserve для `previous_day` вычисляется по той же формуле, что и для `current`
— `total - working - repair - modernization` с clamp к 0 (см. [Reserve
Aggregates](#reserve-aggregates-computed-not-stored) и
[ges-aggregates.md](ges-aggregates.md)).

### Idle Discharge

Two sources of idle discharge data:

1. **Computed** (`current.idle_discharge_m3s`): `total_outflow - ges_flow` from daily data. Shows instantaneous flow rate bypass.

2. **From discharge records** (`idle_discharge`): Pulled from existing `idle_water_discharges` table (via `v_idle_water_discharges_with_volume` view). Shows tracked discharge events with volume, reason, and status.

If multiple discharge records exist for the same GES on the same operational day:
- Flow rates and volumes are **summed**
- First reason is kept
- `is_ongoing = true` if **any** discharge is ongoing

**Operational day:** 05:00 local time (Asia/Tashkent) to 05:00 next day.

---

## Error Responses

All errors follow the standard response format:

```json
{
  "status": "Error",
  "message": "description of the error"
}
```

Validation errors include details:

```json
{
  "status": "Error",
  "validation_errors": [
    {
      "field": "DailyProductionMlnKWh",
      "tag": "gte",
      "value": "-5"
    }
  ]
}
```

### Common Error Scenarios

| Scenario | Status | Message |
|----------|--------|---------|
| Missing required field | 400 | Validation error with field details |
| Invalid date format | 400 | `"invalid date format, expected YYYY-MM-DD"` |
| Missing query parameter | 400 | `"date is required (YYYY-MM-DD)"` / `"organization_id is required"` |
| Data not found | 404 | `"daily data not found"` / `"config not found"` |
| Not authenticated | 401 | `"not authenticated"` |
| FK violation (bad org_id) | 500 | `"failed to save ..."` (org doesn't exist in organizations table) |
| Database error | 500 | `"failed to ..."` |

---

## Data Flow Diagram

```
                    ┌─────────────────┐
                    │   Operator UI    │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
     POST /config    POST /daily-data  POST /plans
              │              │              │
              ▼              ▼              ▼
        ges_config     ges_daily_data  ges_production_plan
              │              │              │
              └──────────────┼──────────────┘
                             │
                    GET /ges-report?date=...
                             │
                             ▼
                    ┌────────────────┐
                    │  Service Layer │
                    │                │
                    │ 6 batch queries│
                    │ ┌────────────┐ │
                    │ │ today data │ │
                    │ │ yest. data │ │
                    │ │ prev. year │ │
                    │ │ MTD/YTD    │ │
                    │ │ plans      │ │
                    │ │ discharges │◄──── v_idle_water_discharges_with_volume
                    │ └────────────┘ │
                    │                │
                    │ Compute:       │
                    │  • power       │
                    │  • diffs       │
                    │  • aggregates  │
                    │  • plan %      │
                    │  • YoY         │
                    │  • cascades    │
                    │  • grand total │
                    └───────┬────────┘
                            │
                            ▼
                    JSON Response
                    (DailyReport)
```

---

## Mapping to Excel (template/ges-prod.xlsx)

The Excel template has 37 columns (A–AK). Cell AH3 holds the report date **+1 day**. Sheet name = `DD.MM.YY`. See [ges-export.md](ges-export.md) for the export endpoint and [internal/lib/service/excel/ges/generator.go](../internal/lib/service/excel/ges/generator.go) for the column-fill code.

**Today block (C4:Y4 in template):**

| Excel Column | JSON Path |
|--------------|-----------|
| A — ГЭС номи | `stations[].name` |
| B — Ўрн. қуввати, МВт | `stations[].config.installed_capacity_mwt` |
| C — Режа (Январь→Месяц) | sum of plans Jan→current month |
| D — Ҳарорат | `cascades[].weather.temperature` (per cascade) + icon |
| E — Сув сатҳи | `stations[].current.water_level_m` |
| F — Δ сатҳи, см | `stations[].diffs.level_change_cm` |
| G — Сув ҳажми, млн.м³ | `stations[].current.water_volume_mln_m3` |
| H — Δ ҳажми | `stations[].diffs.volume_change_mln_m3` |
| I — Сув босими, м | `stations[].current.water_head_m` |
| J — Келаётган сув, м³/с | `stations[].current.reservoir_income_m3s` |
| K — Δ келиш | `stations[].diffs.income_change_m3s` |
| L — Чиқаётган сув, м³/с | `stations[].current.total_outflow_m3s` |
| M — ГЭС орқали, м³/с | `stations[].current.ges_flow_m3s` |
| N — Δ ГЭС орқали | `stations[].diffs.ges_flow_change_m3s` |
| O — Салт ташлама, м³/с | `stations[].current.idle_discharge_m3s` |
| P — Агрегатлар сони | `stations[].config.total_aggregates` |
| Q — Ишлаётган | `stations[].current.working_aggregates` |
| R — Қуввати, МВт | `stations[].current.power_mwt` (= daily × 1000 / 24) |
| S — Δ қуввати | `stations[].diffs.power_change_mwt` |
| T — 1 кунда, млн.кВт.ч | `stations[].current.daily_production_mln_kwh` |
| U — Δ выработки | `stations[].diffs.production_change_mln_kwh` |
| V — Ой бошидан (MTD) | `stations[].aggregations.mtd_production_mln_kwh` |
| W — Йил бошидан (YTD) | `stations[].aggregations.ytd_production_mln_kwh` |
| X — Бажарилди, % | **Excel formula** `=IFERROR(W/C, 0)` |
| Y — Фарқи +/- | **Excel formula** `=W-C` |

**Previous year block (Z4:AK4):**

| Excel Column | JSON Path |
|--------------|-----------|
| Z — Ҳарорат (пр. год) | `cascades[].weather.prev_year_temperature` + icon |
| AA — Сув сатҳи | `stations[].previous_year.water_level_m` |
| AB — Сув ҳажми | `stations[].previous_year.water_volume_mln_m3` |
| AC — Сув босими | `stations[].previous_year.water_head_m` |
| AD — Келаётган сув | `stations[].previous_year.reservoir_income_m3s` |
| AE — ГЭС орқали | `stations[].previous_year.ges_flow_m3s` |
| AF — Қуввати | `stations[].previous_year.power_mwt` |
| AG — 1 кунда | `stations[].previous_year.daily_production_mln_kwh` |
| AH — MTD | `stations[].previous_year.mtd_production_mln_kwh` |
| AI — YTD | `stations[].previous_year.ytd_production_mln_kwh` |
| AJ — Ўсиш суръати % | **Excel formula** `=IFERROR(W/AI-1, 0)` |
| AK — Фарқи +/- | **Excel formula** `=W-AI` |

**Cascade summary rows** → cascade row gets sums of station fields B, C (YTD plan), P-W, AF-AI; X/Y/AJ/AK are Excel formulas.

**Grand total row** → `grand_total` (same fields as cascade summary).

**Aggregate rows** (after grand total):

- Умумий ГЭСлар сони — count of stations by type (`ges`/`mini`/`micro`)
- Умумий агрегатлар = `=+P{grandRow}` (formula)
- Ишлаётган = `=+Q{grandRow}` (formula)
- Заҳирадаги = `grand_total.reserve_aggregates`
- Таъмирдаги = `grand_total.repair_aggregates`
- Модернизацияда = `grand_total.modernization_aggregates`

Values come from the aggregated report (service layer, from `ges_daily_data`). The legacy `repair` / `modernization` query params on `GET /ges-report/export` were removed — see [ges-export.md](ges-export.md).

---

## File Structure

```text
internal/
├── lib/
│   ├── model/ges-report/
│   │   └── model.go              # All types + Optional[T], CascadeWeather, etc.
│   └── service/
│       ├── ges-report/
│       │   ├── service.go        # BuildDailyReport(ctx, date, cascadeOrgID) + filtering
│       │   └── service_test.go
│       └── excel/ges/
│           ├── generator.go      # Excel generator (template fill, formulas, weather icons)
│           ├── generator_test.go
│           └── generator_integration_test.go
├── storage/repo/
│   ├── ges_report.go             # CRUD + batch queries
│   └── organization.go           # GetOrganizationParentID (cascade access)
├── lib/service/auth/
│   └── auth.go                   # CheckCascadeStationAccess + Batch
├── http-server/handlers/ges-report/
│   ├── config.go                 # UpsertConfig, GetConfigs, DeleteConfig
│   ├── cascade_config.go         # cascade-config CRUD
│   ├── daily_data.go             # UpsertDailyData, GetDailyData (cascade-aware)
│   ├── cascade_daily_data.go     # cascade-daily-data (manual weather correction)
│   ├── plan.go                   # BulkUpsertPlan, GetPlans
│   ├── report.go                 # GetReport (filters by cascade)
│   └── export.go                 # Excel/PDF export
├── providers/
│   └── http.go                   # GESExcelTemplatePath, WeatherIconsPath
└── http-server/router/
    └── router.go                 # /ges-report tier 1 (sc/rais/cascade) + tier 2 (sc/rais)

template/
├── ges-prod.xlsx                 # Excel template
└── weather-icons/{01d,01n,...,50n}.png  # 18 OWM weather icons

migrations/postgres/
├── 000063_ges_daily_report.up.sql
├── 000065_cascade_config.up.sql
├── 000067_ges_daily_nullable_user.up.sql
├── 000068_cascade_daily_data.up.sql
├── 000069_role_cascade.up.sql
└── 000071_ges_daily_aggregates.up.sql
```
