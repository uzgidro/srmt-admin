# GES Daily Report — error contract & `consumption_m3_s`

This page describes the structured-error envelope returned by the GES daily-report endpoints, the catalog of stable error codes, and the new `consumption_m3_s` field on `ges_daily_data`.

Audience: frontend developers integrating with `POST /ges/daily-data`, `GET /ges/daily-data`, `GET /ges/daily-report`, and `GET /ges/daily-report/export`.

## 1. Response envelope

All error responses use the same envelope. The shape is backwards compatible — old clients that read only `error` continue to work.

```json
{
  "error":  "human-readable fallback message",
  "code":   "stable.machine.identifier",
  "details": [ /* optional, structured per-violation entries */ ]
}
```

| Field | Type | When present | Purpose |
|---|---|---|---|
| `error` | string | always for non-2xx | Human-readable fallback. Always set so legacy clients keep working. |
| `code` | string | structured errors only | Stable identifier — frontend keys off this for localization and field binding. Absent on legacy plain responses. |
| `details` | array of objects | structured errors only | Per-violation context. Each entry is a free-form object whose keys depend on `code`. See §3 for the full key catalog per code. |

Success responses are unchanged (status 200/204 with the existing payload).

## 2. New field: `consumption_m3_s`

`consumption_m3_s` is a per-day, per-station value representing **useful water consumption** in cubic meters per second — water diverted from the reservoir/discharge to irrigation, drinking water supply, and other beneficial uses.

| Aspect | Value |
|---|---|
| Backend column | `ges_daily_data.consumption_m3_s` (NUMERIC, NULLABLE, CHECK ≥ 0) |
| JSON key | `consumption_m3_s` |
| Type | `number` (float) or `null` |
| Save validation | Independently non-negative when present. Null and missing are both allowed. |
| Effect on report | Subtracted from idle discharge: `idle = total_outflow_m3s - ges_flow_m3s - consumption_m3_s`. |
| Effect on save | None — save does NOT cross-validate against `total_outflow_m3s` / `ges_flow_m3s`. |

### Save semantics (POST /ges/daily-data)

The field uses the same three-state partial-update semantics as every other optional field on this endpoint:

| Payload | Meaning |
|---|---|
| Field absent | Preserve existing DB value. |
| `"consumption_m3_s": null` | Explicitly clear — write SQL NULL. |
| `"consumption_m3_s": 1.5` | Write the value. Must be ≥ 0 or 400 with `code = save.field_negative`. |

### Report semantics (GET /ges/daily-report, GET /ges/daily-report/export)

For the operational day being rendered, every station with a non-null `consumption_m3_s` MUST satisfy:

```
consumption_m3_s ≤ total_outflow_m3s - ges_flow_m3s
```

If any station violates this, the report is **not built** — endpoint returns `400` with `code = report.consumption_exceeds_idle` and `details` listing every violation. The user must fix the offending row(s) on `POST /ges/daily-data` first, then retry the report.

When validation passes, the response field `current.idle_discharge_m3s` reflects the adjusted value (idle minus consumption). The original `consumption_m3_s` is also returned in `current.consumption_m3_s` so the frontend can show both numbers.

When `total_outflow_m3s` OR `ges_flow_m3s` is null, idle is uncomputable and the consumption check is skipped (not a violation). `current.idle_discharge_m3s` is null in that case.

## 3. Error code catalog

| Code | HTTP | Endpoint | Trigger |
|---|---|---|---|
| `save.field_negative` | 400 | POST /ges/daily-data | A numeric field was sent with a negative value. Applies to `working_aggregates`, `repair_aggregates`, `modernization_aggregates`, `own_consumption_kwh`, `consumption_m3_s`. |
| `save.aggregates_exceed_total` | 400 | POST /ges/daily-data | `working + repair + modernization` (after merging the request with the current DB row) exceeds the configured `ges_config.total_aggregates`. |
| `save.production_exceeds_max` | 400 | POST /ges/daily-data | `daily_production_mln_kwh` exceeds the configured `ges_config.max_daily_production_mln_kwh` for the station. |
| `report.consumption_exceeds_idle` | 400 | GET /ges/daily-report, GET /ges/daily-report/export | One or more stations have `consumption_m3_s > (total_outflow_m3s - ges_flow_m3s)` for the requested day. |

Other 4xx/5xx responses (validation tag failures, missing date, auth, internal errors) currently return only `error` — no `code` / `details`. Treat the absence of `code` as "generic error, show the `error` text as-is."

## 4. Per-code `details` shape

### `save.field_negative`

Exactly one entry, for the first negative value found in the batch:

```json
{
  "error": "consumption_m3_s must be >= 0 for organization_id=16, got -1.5",
  "code":  "save.field_negative",
  "details": [
    {
      "organization_id": 16,
      "field":           "consumption_m3_s",
      "value":           -1.5
    }
  ]
}
```

`field` is one of: `working_aggregates`, `repair_aggregates`, `modernization_aggregates`, `own_consumption_kwh`, `consumption_m3_s`. Use it to highlight the input control on the form.

### `save.aggregates_exceed_total`

Exactly one entry for the first row whose merged aggregate sum exceeds the cap:

```json
{
  "error": "aggregates sum exceeds total for organization_id=10: 4+1+0=5 > 4",
  "code":  "save.aggregates_exceed_total",
  "details": [
    {
      "organization_id": 10,
      "date":            "2026-04-22",
      "working":         4,
      "repair":          1,
      "modernization":   0,
      "sum":             5,
      "total":           4
    }
  ]
}
```

The integers in `working`/`repair`/`modernization` are the **effective** values — a request that omits a field uses the current DB value. Show all three to make the math obvious.

### `save.production_exceeds_max`

```json
{
  "error": "daily_production_mln_kwh exceeds max for organization_id=1: 10 > 5",
  "code":  "save.production_exceeds_max",
  "details": [
    {
      "organization_id": 1,
      "date":            "2026-04-22",
      "field":           "daily_production_mln_kwh",
      "value":           10.0,
      "max":             5.0
    }
  ]
}
```

### `report.consumption_exceeds_idle`

One entry per offending station, in the order they appear in the day's data. The full list is returned — the frontend should render all of them so the user can fix everything in one round.

```json
{
  "error":  "useful consumption exceeds idle discharge for: organization_id=16 (ГЭС-1): consumption=5 > idle=2 on 2026-04-22; ...",
  "code":   "report.consumption_exceeds_idle",
  "details": [
    {
      "organization_id":   16,
      "organization_name": "ГЭС-1",
      "date":              "2026-04-22",
      "idle_m3_s":         2.0,
      "consumption_m3_s":  5.0
    },
    {
      "organization_id":   17,
      "organization_name": "ГЭС-2",
      "date":              "2026-04-22",
      "idle_m3_s":         1.0,
      "consumption_m3_s":  3.0
    }
  ]
}
```

`idle_m3_s` is the pre-consumption idle (`total_outflow_m3s - ges_flow_m3s`). `consumption_m3_s` is the saved value that violates `consumption ≤ idle`.

## 5. Frontend handling pattern

```ts
type ErrorEnvelope = {
  error: string;
  code?: string;
  details?: Record<string, unknown>[];
};

async function handleResponse(res: Response) {
  if (res.ok) return res.json();
  const body: ErrorEnvelope = await res.json();
  if (body.code) {
    showLocalizedError(body.code, body.details ?? []);
  } else {
    showRawMessage(body.error);
  }
}

function showLocalizedError(code: string, details: Record<string, unknown>[]) {
  switch (code) {
    case "save.field_negative":
      // Highlight details[0].field on the form, show: i18n("err.field_negative", details[0]).
      break;
    case "save.aggregates_exceed_total":
      // Highlight all three aggregate inputs; show explanation with effective sum/total.
      break;
    case "save.production_exceeds_max":
      // Highlight daily_production_mln_kwh input; show value vs max.
      break;
    case "report.consumption_exceeds_idle":
      // Render a list of violating stations; for each link to its edit page.
      break;
  }
}
```

## 6. What the report response looks like on success

`current` (and `previous_day`) carries both fields:

```json
{
  "current": {
    "total_outflow_m3s":  10.0,
    "ges_flow_m3s":        5.0,
    "consumption_m3_s":    2.0,
    "idle_discharge_m3s":  3.0
  }
}
```

`idle_discharge_m3s` is post-adjustment. The frontend can recompute pre-adjustment idle as `total_outflow - ges_flow` if needed for display.
