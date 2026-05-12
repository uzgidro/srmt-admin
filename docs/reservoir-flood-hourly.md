# reservoir-flood-hourly — API for frontend

## 1. Общее

- **TZ-policy**: `date` и `hour` интерпретируются в зоне Asia/Tashkent (server `Config.Location`). Поле `recorded_at` в ответе/запросе — RFC3339 в UTC. На записи backend нормализует время через `.UTC().Truncate(time.Hour)` (см. `internal/http-server/handlers/reservoir-flood/hourly_upsert.go`).
- **Auth**: Bearer JWT (`Authorization: Bearer <token>`). Глобальный `mwauth.Authenticator` обязателен для всей `/reservoir-flood/*` группы.
- **Access-control**: на master реализован только role-based middleware `mwauth.RequireAnyRole(...)` (см. `internal/http-server/middleware/auth/auth.go`). Permission-строк (`RequirePermission*`) в коде нет. Распределение по тирам:

| Tier | Endpoints | `RequireAnyRole(...)` |
|---|---|---|
| 1 | `GET /reservoir-flood/hourly`, `POST /reservoir-flood/hourly`, `GET /reservoir-flood/config` | `sc`, `rais`, `reservoir_duty` |
| 2 | `POST /reservoir-flood/config`, `DELETE /reservoir-flood/config` | `sc`, `rais` |
| 3 | `GET /reservoir-flood/export` | `sc`, `rais` |

Дополнительно в хэндлерах есть defence-in-depth:
- `callerIsAdmin` (sc/rais) проверяется в `config_upsert.go` и `config_delete.go` — если запрос дошёл, но роли нет → `403`.
- В `GetHourly` и `GetConfigs` non-admin без `claims.OrganizationID` → `403` ("user has no organization assigned").
- Ответы `GetHourly`/`GetConfigs` фильтруются для не-admin: `reservoir_duty` видит только свою организацию.
- В `UpsertHourly` каждый item проходит `auth.CheckOrgAccessBatch`; чужой item → `403` для всего батча (атомарно, без частичных записей).

## 2. GET /reservoir-flood/hourly

Возвращает массив `HourlyRecord` за указанный день (или конкретный час).

| Параметр | Обязательный | Тип/формат | Описание |
|---|---|---|---|
| `date` | да | `YYYY-MM-DD` | Дата в Asia/Tashkent. Окно — сутки `[00:00, 24:00)` локального дня. |
| `hour` | нет | `0..23` (ведущий ноль `08` допустим) | Час в Asia/Tashkent. Если задан — окно сужается до `[hour:00, hour:00+1h)`. |
| `organization_id` | нет | `int64` | Доп. фильтр. Для `reservoir_duty` всё равно ограничивается своей org через post-filter. |

Коды:

| Код | Когда |
|---|---|
| `200` | OK, тело — JSON-массив `HourlyRecord[]` (может быть пустым). |
| `400` | Нет `date`, неверный формат, `hour` вне `0..23` или нечисловой `organization_id`. |
| `401` | Нет/невалидный JWT (срабатывает в `Authenticator`). |
| `403` | Роль не в `{sc, rais, reservoir_duty}`, либо non-admin без `OrganizationID`. |
| `500` | Ошибка БД. |

Пример ответа:

```json
[
  {
    "id": 142,
    "organization_id": 96,
    "organization_name": "Туямуюнское в/х",
    "recorded_at": "2026-05-12T07:00:00Z",
    "water_level_m": 130.45,
    "water_volume_mln_m3": 4520.1,
    "inflow_m3s": 312.0,
    "outflow_m3s": 280.0,
    "ges_flow_m3s": 240.0,
    "filtration_m3s": null,
    "idle_discharge_m3s": 0,
    "duty_name": "Каримов А.К.",
    "capacity_mwt": 64.5,
    "weather_condition": "ясно",
    "temperature_c": 22.0,
    "created_by_user_id": 17,
    "updated_at": "2026-05-12T12:03:14Z"
  }
]
```

## 3. POST /reservoir-flood/hourly

Bulk-upsert. Тело — JSON-массив `UpsertHourlyRequest`. Минимум 1 элемент (пустой массив → `400`).

**Нормализация `recorded_at`**: каждый item парсится как `time.RFC3339`, затем `t.UTC().Truncate(time.Hour)` и переписывается обратно в строку перед уходом в репо. Так что `2026-05-12T07:34:21+05:00` → `2026-05-12T02:00:00Z` (см. `hourly_upsert.go:74`). UNIQUE на `(organization_id, recorded_at)` гарантирует одну запись на пару `(org, час)`.

**Optional[T] семантика** (поля-метрики и `duty_name`/`weather_condition`):

| JSON | Backend | Эффект на колонку |
|---|---|---|
| ключ отсутствует | `Set=false` | колонка НЕ трогается (preserve) |
| `"field": null` | `Set=true, Value=nil` | колонка пишется в `NULL` |
| `"field": <value>` | `Set=true, Value=&v` | колонка пишется в `value` |

**Negative-value rejection** (`negativeMetric()` в `hourly_upsert.go`): если `Value != nil && *Value < 0` для одного из:
`water_level_m`, `water_volume_mln_m3`, `inflow_m3s`, `outflow_m3s`, `ges_flow_m3s`, `filtration_m3s`, `idle_discharge_m3s`, `capacity_mwt` — `400` с указанием `item_index` и имени поля. **`temperature_c` исключён** — зимние значения легитимно ниже нуля.

**Org-bound write**: `auth.CheckOrgAccessBatch(ctx, orgIDs)` — `sc/rais` пропускают; `reservoir_duty` обязан совпасть с собственной `OrganizationID`. Любой чужой item → `403` для всего батча.

Коды:

| Код | Когда |
|---|---|
| `200` | OK (`{"status":"ok"}`). |
| `400` | Невалидный JSON, пустой массив, validator-ошибка по item, неверный `recorded_at`, отрицательная метрика. Тело включает `item_index`. |
| `401` | Нет user_id в context (broken JWT). |
| `403` | Роль не в `{sc, rais, reservoir_duty}`, либо item-organization вне зоны доступа. |
| `500` | Ошибка БД. |

## 4. Config endpoints

### GET /reservoir-flood/config

Список `Config[]`. Tier 1. Для `reservoir_duty` — только своя организация (post-filter в хэндлере; пустой массив, если своя org не включена в config).

Коды: `200`, `401`, `403` (нет роли или нет `OrganizationID` у не-admin), `500`.

### POST /reservoir-flood/config

Body — `UpsertConfigRequest` (одиночный объект, НЕ массив). Tier 2 (`sc`, `rais`). Хэндлер дополнительно проверяет `callerIsAdmin` → если нет, `403` ("only sc/rais may modify config"). Validator: `OrganizationID` required, `SortOrder >= 0`. Семантика — upsert по `organization_id`.

Коды: `200`, `400` (невалидный JSON / validator / `ErrCheckConstraintViolation`), `401`, `403`, `500`.

### DELETE /reservoir-flood/config

Query: `organization_id` (int64, > 0). Tier 2. Хэндлер дополнительно проверяет `callerIsAdmin`. НЕ идемпотентен — повторный запрос на уже удалённый id вернёт `404`.

Коды: `204` (`resp.Delete()`), `400` (нет/невалидный `organization_id`), `401`, `403`, `404` (`ErrNotFound`), `500`.

## 5. GET /reservoir-flood/export

Excel/PDF «Тезкор маълумот». Подробности — см. [docs/sel-export.md](sel-export.md).

## 6. curl examples

```bash
# 1) Полные сутки 2026-05-12 (Asia/Tashkent)
curl -H "Authorization: Bearer $JWT" \
  "http://localhost:9010/reservoir-flood/hourly?date=2026-05-12"

# 2) Один час (07:00 локального времени Asia/Tashkent)
curl -H "Authorization: Bearer $JWT" \
  "http://localhost:9010/reservoir-flood/hourly?date=2026-05-12&hour=07"

# 3) POST с локальным offset +05:00 (нормализуется в UTC и обрезается до часа)
curl -X POST -H "Authorization: Bearer $JWT" -H "Content-Type: application/json" \
  "http://localhost:9010/reservoir-flood/hourly" \
  -d '[
    {
      "organization_id": 96,
      "recorded_at": "2026-05-12T12:00:00+05:00",
      "water_level_m": 130.45,
      "water_volume_mln_m3": 4520.1,
      "inflow_m3s": 312,
      "outflow_m3s": 280,
      "ges_flow_m3s": 240,
      "idle_discharge_m3s": 0,
      "capacity_mwt": 64.5,
      "duty_name": "Каримов А.К.",
      "weather_condition": "ясно",
      "temperature_c": 22.0
    }
  ]'

# 4) POST с UTC (Z), явный сброс filtration_m3s в NULL, отрицательная температура
curl -X POST -H "Authorization: Bearer $JWT" -H "Content-Type: application/json" \
  "http://localhost:9010/reservoir-flood/hourly" \
  -d '[
    {
      "organization_id": 96,
      "recorded_at": "2026-05-12T07:00:00Z",
      "inflow_m3s": 318,
      "outflow_m3s": 285,
      "filtration_m3s": null,
      "temperature_c": -1.5
    }
  ]'
```

## 7. Models

Verbatim из `internal/lib/model/reservoir-flood/model.go`.

```go
type HourlyRecord struct {
    ID               int64     `json:"id"`
    OrganizationID   int64     `json:"organization_id"`
    OrganizationName string    `json:"organization_name,omitempty"`
    RecordedAt       time.Time `json:"recorded_at"`
    WaterLevelM      *float64  `json:"water_level_m"`
    WaterVolumeMlnM3 *float64  `json:"water_volume_mln_m3"`
    InflowM3s        *float64  `json:"inflow_m3s"`
    OutflowM3s       *float64  `json:"outflow_m3s"`
    GESFlowM3s       *float64  `json:"ges_flow_m3s"`
    FiltrationM3s    *float64  `json:"filtration_m3s"`
    IdleDischargeM3s *float64  `json:"idle_discharge_m3s"`
    DutyName         *string   `json:"duty_name"`
    CapacityMwt      *float64  `json:"capacity_mwt"`
    WeatherCondition *string   `json:"weather_condition"`
    TemperatureC     *float64  `json:"temperature_c"`
    CreatedByUserID  *int64    `json:"created_by_user_id,omitempty"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

```go
type UpsertHourlyRequest struct {
    OrganizationID   int64                      `json:"organization_id" validate:"required"`
    RecordedAt       string                     `json:"recorded_at"     validate:"required"`
    WaterLevelM      optional.Optional[float64] `json:"water_level_m"       validate:"omitempty"`
    WaterVolumeMlnM3 optional.Optional[float64] `json:"water_volume_mln_m3" validate:"omitempty"`
    InflowM3s        optional.Optional[float64] `json:"inflow_m3s"          validate:"omitempty"`
    OutflowM3s       optional.Optional[float64] `json:"outflow_m3s"         validate:"omitempty"`
    GESFlowM3s       optional.Optional[float64] `json:"ges_flow_m3s"        validate:"omitempty"`
    FiltrationM3s    optional.Optional[float64] `json:"filtration_m3s"      validate:"omitempty"`
    IdleDischargeM3s optional.Optional[float64] `json:"idle_discharge_m3s"  validate:"omitempty"`
    DutyName         optional.Optional[string]  `json:"duty_name"`
    CapacityMwt      optional.Optional[float64] `json:"capacity_mwt"        validate:"omitempty"`
    WeatherCondition optional.Optional[string]  `json:"weather_condition"`
    TemperatureC     optional.Optional[float64] `json:"temperature_c"       validate:"omitempty"`
}
```

```go
type Config struct {
    ID               int64     `json:"id"`
    OrganizationID   int64     `json:"organization_id"`
    OrganizationName string    `json:"organization_name,omitempty"`
    SortOrder        int       `json:"sort_order"`
    IsActive         bool      `json:"is_active"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

```go
type UpsertConfigRequest struct {
    OrganizationID int64 `json:"organization_id" validate:"required"`
    SortOrder      int   `json:"sort_order"      validate:"gte=0"`
    IsActive       bool  `json:"is_active"`
}
```
