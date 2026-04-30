# Solar + Own Consumption — Frontend Implementation Guide

Документ описывает что нужно сделать на фронте для двух связанных изменений: (1) новый модуль учёта солнечных панелей (`/solar/*`, 7 endpoint'ов) и (2) новое поле «собственные нужды» в существующем `ges-report/daily-data` плюс расширение его агрегаций MTD/YTD.

## 1. Бизнес-цель и семантика

**Solar.** Часть станций каскада оборудована солнечными панелями. Их выработку и сдачу в сеть нужно вести так же системно, как ГЭС-выработку: с конфигом установленной мощности, ежедневными показаниями и помесячным планом. Данные хранятся отдельно от ges-report (новые таблицы `solar_config`, `solar_daily_data`, `solar_production_plan`), потому что станция может быть solar-only без записи в `ges_config`. По авторизации — те же роли, что у ges-report (`sc`, `rais`, `cascade`); новых ролей нет.

**Own consumption.** На каждой станции (ges/mini/micro) есть собственные нужды — потребление электроэнергии для эксплуатации самой станции (освещение, подогрев, насосы, КИПиА). Это не отдельный модуль, а одна дополнительная числовая колонка в существующей таблице `ges_daily_data`, которая уже общая для всех типов станций. Соответственно, на фронте — это новая колонка в существующей форме / таблице ges-report daily-data.

**MTD/YTD own_consumption.** Накопительные суммы по собственным нуждам (с начала месяца и с начала года) приходят в существующий ответ ges-report — никаких новых endpoint'ов для этого нет. Просто в `aggregations` появились два дополнительных поля.

## 2. Список endpoint'ов

| Метод | Путь | Роли | Цель |
|---|---|---|---|
| POST | `/solar/daily-data` | sc, rais, cascade | Bulk upsert ежедневной выработки солнечных панелей |
| GET | `/solar/daily-data` | sc, rais, cascade | Список записей за дату |
| POST | `/solar/config` | sc, rais | Upsert конфига солнечных панелей (мощность, sort_order) |
| GET | `/solar/config` | sc, rais, cascade | Список конфигов |
| DELETE | `/solar/config` | sc, rais | Удалить конфиг по `organization_id` |
| POST | `/solar/plans` | sc, rais | Bulk upsert помесячного плана выработки |
| GET | `/solar/plans` | sc, rais, cascade | План на год |
| POST | `/ges-report/daily-data` | sc, rais, cascade | **Расширено**: новое Optional-поле `own_consumption_kwh` в каждом item массива |
| GET | `/ges-report/...` | sc, rais, cascade | **Расширено**: 2 новых поля в `aggregations` (`mtd_own_consumption_kwh`, `ytd_own_consumption_kwh`) |

## 3. Матрица доступа

| Действие | `sc` | `rais` | `cascade` |
|---|---|---|---|
| GET `/solar/daily-data` (любая org) | OK (все) | OK (все) | только своя org |
| POST `/solar/daily-data` (своя org) | OK | OK | OK |
| POST `/solar/daily-data` (чужая org) | OK | OK | **403** |
| POST `/solar/daily-data` (mixed batch — есть чужая org) | OK | OK | **403** на весь батч (атомарно) |
| GET `/solar/config` | OK (все) | OK (все) | OK (все — только чтение) |
| POST `/solar/config` | OK | OK | **403** (route-level) |
| DELETE `/solar/config` | OK | OK | **403** (route-level) |
| GET `/solar/plans` | OK | OK | OK |
| POST `/solar/plans` | OK | OK | **403** (route-level) |
| POST `/ges-report/daily-data` с `own_consumption_kwh` | OK | OK | OK для своей org; 403 для чужой |
| GET `/ges-report` (новые aggregations) | все org | все org | только своя org (фильтр на сервере) |

Защита `/solar/config` и `/solar/plans` от cascade — на уровне роутера через `RequireAnyRole("sc", "rais")`. Запрос даже не доходит до handler'а. Фронту полагаться на 403 не нужно — кнопки/формы должны быть скрыты или дизейблены для cascade-юзера. 403 — fallback на случай DevTools.

Защита `/solar/daily-data` POST от cross-org write — на уровне handler'а через `auth.CheckOrgAccessBatch`. Mixed-batch (часть своя, часть чужая org) откатывается целиком.

cascade-юзер без `claims.OrganizationID` (broken account state — например, если организацию удалили, но юзер остался) получает **403** ещё до похода в БД, как на POST, так и на GET.

## 4. Полные API-контракты

### 4.1. POST `/solar/daily-data`

**Roles:** `sc`, `rais`, `cascade`.

**Body** — массив объектов. Даже на одну запись — оборачивайте в `[...]`.

```json
[
  {
    "organization_id": 42,
    "date": "2026-04-28",
    "generation_kwh": 620.5,
    "grid_export_kwh": 540.0
  }
]
```

`organization_id` и `date` (формат `YYYY-MM-DD`) обязательны. `generation_kwh`, `grid_export_kwh` — Optional (см. §5).

**Response 200:**

```json
{ "status": "OK" }
```

**Errors:**

| Code | Когда | Тело |
|---|---|---|
| 400 | Невалидный JSON / пустой массив / отсутствует `organization_id` или `date` / неверный формат `date` / отрицательное `generation_kwh` или `grid_export_kwh` | `{"error": "invalid request: <details>"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | cascade пишет в чужую org (single или mixed batch) / cascade без `claims.OrganizationID` | `{"error": "access to target organization denied"}` |
| 500 | Ошибка БД | `{"error": "internal server error"}` |

При ошибке батч откатывается целиком — в БД ничего не запишется.

### 4.2. GET `/solar/daily-data`

**Roles:** `sc`, `rais`, `cascade`.

**Query:**

- `date` — обязателен, формат `YYYY-MM-DD`.
- `organization_id` — опционален. Для cascade это hint: даже если задать чужую org, фильтр на сервере вернёт только свою.

```
GET /solar/daily-data?date=2026-04-28
GET /solar/daily-data?date=2026-04-28&organization_id=42
```

**Response 200** — массив:

```json
[
  {
    "id": 1001,
    "organization_id": 42,
    "organization_name": "ГЭС Alpha",
    "date": "2026-04-28T00:00:00Z",
    "generation_kwh": 620.5,
    "grid_export_kwh": 540.0,
    "updated_at": "2026-04-28T18:02:11Z"
  }
]
```

Если на дату записей нет — пустой массив. Организации без записи в `solar_config` в ответ не попадают.

**Errors:**

| Code | Когда | Тело |
|---|---|---|
| 400 | `date` отсутствует или невалидный | `{"error": "invalid date format, expected YYYY-MM-DD"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | cascade без `claims.OrganizationID` | `{"error": "access to target organization denied"}` |
| 500 | Ошибка БД | `{"error": "internal server error"}` |

### 4.3. POST `/solar/config`

**Roles:** `sc`, `rais` (route-level).

**Body** — одиночный объект (НЕ массив):

```json
{
  "organization_id": 42,
  "installed_capacity_kw": 150.0,
  "sort_order": 1
}
```

Семантика — upsert по `organization_id`. Единица `installed_capacity_kw` — **кВт** (не МВт; солнечные панели обычно порядка десятков-сотен кВт).

**Response 200:**

```json
{ "status": "OK" }
```

**Errors:**

| Code | Когда | Тело |
|---|---|---|
| 400 | Невалидный JSON / отсутствует `organization_id` / `installed_capacity_kw` < 0 / `sort_order` < 0 | `{"error": "invalid request: <details>"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | Роль не sc/rais | `{"error": "forbidden"}` |
| 500 | Ошибка БД (включая FK violation, если в payload `organization_id` не существует в `organizations`) | `{"error": "internal server error"}` |

> **Примечание.** Если `organization_id` не существует, бэк сейчас возвращает 500, а не 404 — FK violation от Postgres попадает в общий error-handler. Фронту: показать generic-toast «не удалось сохранить» и при необходимости проверить корректность org-id перед отправкой.

### 4.4. GET `/solar/config`

**Roles:** `sc`, `rais`, `cascade`. Параметров нет.

**Response 200:**

```json
[
  {
    "id": 1,
    "organization_id": 42,
    "organization_name": "ГЭС Alpha",
    "installed_capacity_kw": 150.0,
    "sort_order": 1,
    "updated_at": "2026-04-25T08:00:00Z"
  }
]
```

cascade видит **все** конфиги (read-only — это нужно, чтобы понимать состав солнечного парка). Запретить cascade писать config — задача route middleware на POST/DELETE, GET остаётся открытым.

**Errors:** 401, 500 — стандартные.

### 4.5. DELETE `/solar/config`

**Roles:** `sc`, `rais` (route-level).

**Query:** `organization_id` обязателен.

```
DELETE /solar/config?organization_id=42
```

**Response 200:** `{"status": "OK"}` (или 204 — фронту обработать оба).

**Errors:**

| Code | Когда | Тело |
|---|---|---|
| 400 | `organization_id` отсутствует или невалидный | `{"error": "invalid organization_id"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | Роль не sc/rais | `{"error": "forbidden"}` |
| 404 | Записи нет | `{"error": "config not found"}` |
| 500 | Ошибка БД (включая FK RESTRICT — если есть записи в `solar_daily_data` или `solar_production_plan`) | `{"error": "internal server error"}` |

> **Примечание.** DELETE не идемпотентен — повторный запрос вернёт 404.

### 4.6. POST `/solar/plans`

**Roles:** `sc`, `rais` (route-level).

**Body** — обёртка `{ "plans": [...] }`:

```json
{
  "plans": [
    { "organization_id": 42, "year": 2026, "month": 4, "plan_thousand_kwh": 18.5 },
    { "organization_id": 42, "year": 2026, "month": 5, "plan_thousand_kwh": 22.0 }
  ]
}
```

Семантика — upsert по `(organization_id, year, month)`. Единица `plan_thousand_kwh` — **тысячи кВтч** (НЕ млн как у ГЭС-плана; солнечные масштабы меньше).

**Response 200:** `{"status": "OK"}`.

**Errors:**

| Code | Когда | Тело |
|---|---|---|
| 400 | Невалидный JSON / пустой `plans` / `month` вне 1–12 / `year` вне 2020–2100 / `plan_thousand_kwh` < 0 | `{"error": "invalid request: <details>"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | Роль не sc/rais | `{"error": "forbidden"}` |
| 500 | Ошибка БД (включая FK violation, если `organization_id` не существует) | `{"error": "internal server error"}` |

### 4.7. GET `/solar/plans`

**Roles:** `sc`, `rais`, `cascade`.

**Query:** `year` обязателен.

```
GET /solar/plans?year=2026
```

**Response 200:**

```json
[
  {
    "id": 10,
    "organization_id": 42,
    "organization_name": "ГЭС Alpha",
    "year": 2026,
    "month": 4,
    "plan_thousand_kwh": 18.5
  }
]
```

cascade видит **все** планы за указанный год — это справочные данные, доступные всем ролям с read-доступом. Если фронту нужно показать только план «своей» org для cascade-пользователя — фильтровать на клиенте по `organization_id == claims.organization_id`.

**Errors:**

| Code | Когда | Тело |
|---|---|---|
| 400 | `year` отсутствует или невалидный | `{"error": "invalid year"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 500 | Ошибка БД | `{"error": "internal server error"}` |

### 4.8. Дельта: POST `/ges-report/daily-data`

API то же самое, добавлено одно Optional-поле в каждый item:

```json
[
  {
    "organization_id": 42,
    "date": "2026-04-28",
    "daily_production_mln_kwh": 1.250,
    "working_aggregates": 3,
    "water_head_m": 45.0,
    "own_consumption_kwh": 1250.0
  }
]
```

`own_consumption_kwh` — Optional[float64], CHECK ≥ 0. Подчиняется тем же правилам Optional-семантики, что и остальные nullable-поля (см. §5). Старый фронт без этого поля продолжит работать.

Errors — те же, что у уже существующего endpoint'а; добавилось одно условие 400: `own_consumption_kwh < 0` → `{"error": "invalid request: own_consumption_kwh must be >= 0"}`.

> **Партиал-апдейт только с `own_consumption_kwh` гарантированно проходит**, независимо от того, какие исторические значения лежат в БД для `daily_production_mln_kwh` или агрегатных полей. Бэк валидирует cap (`max_daily_production_mln_kwh`) и sum (`working+repair+modernization ≤ total_aggregates`) **только** для тех полей, которые юзер реально шлёт. Это поведение нужно для парсеров, которые шлют разные типы метрик отдельными запросами. Подробнее: [ges-max-daily-production.md](ges-max-daily-production.md), [ges-aggregates.md](ges-aggregates.md).

### 4.9. Дельта: GET `/ges-report?date=...`

В `station.aggregations` каждой станции отчёта появились два новых поля. Этот блок раньше содержал только production-метрики; теперь к ним добавлены own_consumption-аналоги:

```json
{
  "stations": [
    {
      "organization_id": 42,
      "aggregations": {
        "mtd_production_mln_kwh": 12.5,
        "ytd_production_mln_kwh": 124.0,
        "mtd_own_consumption_kwh": 8420.0,
        "ytd_own_consumption_kwh": 95300.0
      },
      "previous_year": {
        "mtd_production_mln_kwh": 11.8,
        "ytd_production_mln_kwh": 119.4
      }
    }
  ]
}
```

`mtd_own_consumption_kwh` — сумма `own_consumption_kwh` с начала месяца до запрошенной даты включительно. `ytd_own_consumption_kwh` — то же с начала года. Единица — **кВтч** (НЕ млн кВтч). Для cascade суммы посчитаны только по их org. `NULL`-значения в БД считаются как 0.

> **Где живут `prev_year_*` поля.** Прошлогодние MTD/YTD по production живут в **`station.previous_year`** (`mtd_production_mln_kwh` / `ytd_production_mln_kwh`), а **не** в `aggregations`. Для own_consumption аналогов прошлого года в response нет — это out-of-scope текущей итерации.

## 5. Optional-семантика — критично

Все Optional-поля (в `/solar/daily-data` это `generation_kwh`, `grid_export_kwh`; в `/ges-report/daily-data` — все nullable метрики, включая новое `own_consumption_kwh`) различают **три** состояния, а не два:

| JSON состояние | Backend интерпретирует как | Что произойдёт в БД |
|---|---|---|
| **поле отсутствует** в payload | `Set=false` | колонка НЕ трогается (preserve existing) |
| `"field": null` | `Set=true, Value=nil` | колонка пишется в `NULL` |
| `"field": <value>` | `Set=true, Value=&value` | колонка пишется в `value` |

### Примеры на `/solar/daily-data`

**(a) Обновить только generation_kwh — grid_export_kwh сохранить:**

```json
[{ "organization_id": 42, "date": "2026-04-28", "generation_kwh": 620.5 }]
```

Если в БД на эту дату уже было `grid_export_kwh = 540.0` — оно **останется**. Запишется только `generation_kwh = 620.5`.

**(b) Явно очистить grid_export_kwh:**

```json
[{ "organization_id": 42, "date": "2026-04-28", "grid_export_kwh": null }]
```

`grid_export_kwh` → NULL. `generation_kwh` не трогается.

**(c) Полная перезапись:**

```json
[{
  "organization_id": 42,
  "date": "2026-04-28",
  "generation_kwh": 620.5,
  "grid_export_kwh": 540.0
}]
```

Оба поля запишутся.

### Пример на `own_consumption_kwh`

```json
[{ "organization_id": 42, "date": "2026-04-28", "own_consumption_kwh": 1250.0 }]
```

Запишется только `own_consumption_kwh`. Все остальные поля строки `ges_daily_data` — `daily_production_mln_kwh`, `water_head_m` и т.д. — **сохранятся** в БД (preserve), потому что их нет в payload.

> **Внимание фронту.** Если на форме все поля хранятся как `null` для пустых ячеек, и вы при сохранении сериализуете весь объект — вы случайно затрёте уже введённые коллегой данные. Используйте сериализатор, который пропускает поля по правилу «не отправлять, если пользователь не трогал ячейку». Стандартный `JSON.stringify` без фильтра включает `undefined` как отсутствующее свойство — это удобно. См. helper `buildSolarPayload` в §7.

## 6. Единицы измерения — критично

Самая частая ошибка — путаница масштабов. Solar plan в **тысячах** кВтч, GES plan в **млн** кВтч; daily generation/own_consumption — в **чистых** кВтч; GES daily production — в **млн** кВтч.

| Поле | Endpoint | Единица |
|---|---|---|
| `generation_kwh` | solar/daily-data | **кВтч** |
| `grid_export_kwh` | solar/daily-data | **кВтч** |
| `own_consumption_kwh` | ges-report/daily-data | **кВтч** |
| `mtd_own_consumption_kwh` | ges-report response.aggregations | **кВтч** |
| `ytd_own_consumption_kwh` | ges-report response.aggregations | **кВтч** |
| `installed_capacity_kw` | solar/config | **кВт** (мощность) |
| `plan_thousand_kwh` | solar/plans | **тысячи кВтч** |
| `daily_production_mln_kwh` | ges-report/daily-data | **млн кВтч** (без изменений) |
| `plan_mln_kwh` | ges-report/plans | **млн кВтч** (без изменений) |
| `installed_capacity_mwt` | ges-report/config | **МВт** (без изменений) |
| `mtd_production_mln_kwh` | ges-report response.aggregations | **млн кВтч** (без изменений) |
| `ytd_production_mln_kwh` | ges-report response.aggregations | **млн кВтч** (без изменений) |

Для отображения в одной строке отчёта рядом — приводите к одной единице на стороне фронта (например, всё к кВтч умножением).

## 7. TS-модели + helpers

```typescript
// ========== Solar ==========

export interface SolarConfig {
  id: number;
  organization_id: number;
  organization_name?: string;
  installed_capacity_kw: number;     // кВт
  sort_order: number;
  updated_at: string;
}

export interface UpsertSolarConfigRequest {
  organization_id: number;
  installed_capacity_kw: number;     // кВт, >= 0
  sort_order: number;                // >= 0
}

export interface SolarDailyData {
  id: number;
  organization_id: number;
  organization_name?: string;
  date: string;                      // ISO 8601
  generation_kwh: number | null;     // кВтч
  grid_export_kwh: number | null;    // кВтч
  updated_at: string;
}

// Optional<T> — три состояния. Если поле отсутствует в payload — backend
// не трогает соответствующую колонку. Если null — пишет NULL. Если value —
// пишет value.
export interface UpsertSolarDailyDataItem {
  organization_id: number;
  date: string;                      // YYYY-MM-DD
  generation_kwh?: number | null;    // кВтч, >= 0
  grid_export_kwh?: number | null;   // кВтч, >= 0
}

export interface SolarProductionPlan {
  id: number;
  organization_id: number;
  organization_name?: string;
  year: number;
  month: number;                     // 1..12
  plan_thousand_kwh: number;         // тысячи кВтч (НЕ млн)
}

export interface UpsertSolarPlanItem {
  organization_id: number;
  year: number;                      // 2020..2100
  month: number;                     // 1..12
  plan_thousand_kwh: number;         // >= 0
}

export interface BulkUpsertSolarPlanRequest {
  plans: UpsertSolarPlanItem[];
}

// ========== GES report — расширения ==========

// В UpsertDailyDataRequest добавить (рядом с existing полями):
//   own_consumption_kwh?: number | null;     // кВтч, >= 0; Optional

// В response DailyData добавить:
//   own_consumption_kwh: number | null;

// В station.aggregations response GET /ges-report добавить
// (prev_year_* поля живут в station.previous_year, а НЕ здесь):
export interface GESAggregations {
  mtd_production_mln_kwh: number;
  ytd_production_mln_kwh: number;
  mtd_own_consumption_kwh: number;   // NEW, кВтч
  ytd_own_consumption_kwh: number;   // NEW, кВтч
}

// ========== Roles & helpers ==========

export type Role = "sc" | "rais" | "cascade";

export function canEditSolarConfig(roles: Role[]): boolean {
  return roles.includes("sc") || roles.includes("rais");
}

export function canEditSolarPlan(roles: Role[]): boolean {
  return roles.includes("sc") || roles.includes("rais");
}

export function canEditOrg(
  roles: Role[],
  userOrgID: number,
  recordOrgID: number,
): boolean {
  if (roles.includes("sc") || roles.includes("rais")) return true;
  if (roles.includes("cascade")) return userOrgID === recordOrgID;
  return false;
}

// Helper: сериализация формы solar daily-data с учётом Optional-семантики.
// touched — Set имён полей, которые юзер фактически менял.
// Поля не из touched НЕ попадают в payload (preserve в БД).
export function buildSolarPayload(
  form: Partial<UpsertSolarDailyDataItem>,
  touched: Set<keyof UpsertSolarDailyDataItem>,
): UpsertSolarDailyDataItem {
  const out: any = {
    organization_id: form.organization_id,
    date: form.date,
  };
  for (const key of touched) {
    if (key === "organization_id" || key === "date") continue;
    out[key] = form[key] ?? null;   // null — это «очистить»
  }
  return out as UpsertSolarDailyDataItem;
}
```

## 8. UX-паттерн

**Solar — отдельная секция UI.** Не смешивайте solar-станции с обычной ges-report таблицей. Solar — это отдельная вкладка / страница / раздел дашборда: своя таблица «дата × станция», свой раздел «помесячный план», свой экран управления конфигом.

**own_consumption_kwh — колонка в существующей ges-report daily-data таблице.** Это просто ещё одно поле формы рядом с `daily_production_mln_kwh`, `water_head_m` и т.д. Никакого отдельного экрана не нужно. Подпись — «Собственные нужды, кВтч».

**MTD/YTD own_consumption — рядом с production-агрегациями.** В дашборде / отчёте, где сейчас отображаются `mtd_production_mln_kwh` и `ytd_production_mln_kwh`, добавьте рядом `mtd_own_consumption_kwh` и `ytd_own_consumption_kwh`. Логичная пара: «Выработка (млн кВтч) — Собственные нужды (кВтч)». Не забыть про разницу в единицах при отображении.

**cascade — фильтрация UI.**
- На solar daily-data таблице селектор организаций показывает только свою. Это поверх серверной фильтрации (defence-in-depth) — UI просто не даёт юзеру даже выбрать чужую org.
- Кнопки «Добавить станцию» / «Удалить станцию» в solar config, кнопка «Сохранить план» в solar plans — **скрыты или дизейблены** через `canEditSolarConfig(roles)` / `canEditSolarPlan(roles)`. Если оставить кликабельными — сервер вернёт 403, фронту обработать как fallback toast «Нет прав».
- На ges-report — own_consumption_kwh колонка для cascade редактируется только в строках своей org (стандартная поведение, как у остальных полей).

**Auto-save vs батч.** Для solar daily-data — рекомендуется auto-save при потере фокуса ячейки (одна запись `[{...}]`, только тронутые поля). Уменьшает риск потери данных. Альтернатива — кнопка «Сохранить всё» с батч-отправкой.

**Подсветка пустых дней.** Если на дату записи нет — соответствующая ячейка серая («нет данных»). Помогает оператору визуально найти пропуски.

**Единицы — подписи у полей.** Каждое поле в форме обязательно подписать единицей (`кВт`, `кВтч`, `тыс. кВтч`, `млн кВтч`). Это защита от ошибок ввода — порядок величин разный.

## 9. End-to-end сценарии

### Сценарий 1 — sc включает org A в solar и задаёт план

Юзер `sc-1` (роль `sc`) открывает страницу управления solar.

1. Кликает «Добавить станцию», выбирает организацию A (id=42), вводит мощность 150 кВт, sort_order = 1.

   `POST /solar/config`

   ```json
   { "organization_id": 42, "installed_capacity_kw": 150.0, "sort_order": 1 }
   ```

   Ответ 200: `{"status": "OK"}`. UI обновляет список конфигов.

2. Открывает экран «План на 2026 год», заполняет апрель = 18.5 тыс. кВтч, кликает «Сохранить».

   `POST /solar/plans`

   ```json
   { "plans": [{ "organization_id": 42, "year": 2026, "month": 4, "plan_thousand_kwh": 18.5 }] }
   ```

   Ответ 200: `{"status": "OK"}`. Org A теперь видна всем cascade-юзерам этой org в режиме «солнечная станция».

### Сценарий 2 — cascade-юзер своей org A вводит solar-выработку за день

Юзер `cascade-10` (роль `cascade`, `claims.OrganizationID = 42`) открывает экран solar daily-data на дату 2026-04-28.

1. Видит свою org A в селекторе (единственный вариант — UI отфильтровал остальные).
2. Заполняет ячейку `generation_kwh = 620.5`, `grid_export_kwh = 540.0`. На потере фокуса — auto-save.

   `POST /solar/daily-data`

   ```json
   [{
     "organization_id": 42,
     "date": "2026-04-28",
     "generation_kwh": 620.5,
     "grid_export_kwh": 540.0
   }]
   ```

3. Backend: 200 OK. `{"status": "OK"}`. UI помечает ячейки как сохранённые.

### Сценарий 3 — тот же юзер пробует ввести данные за чужую org B

Юзер `cascade-10` (org A=42) каким-то образом (DevTools, прямой POST) пробует записать в org B (id=99).

1. `POST /solar/daily-data`

   ```json
   [{
     "organization_id": 99,
     "date": "2026-04-28",
     "generation_kwh": 100.0
   }]
   ```

2. Backend: `auth.CheckOrgAccessBatch` отбивает.

   ```json
   { "error": "access to target organization denied" }
   ```

   HTTP 403. **В БД ничего не записано** — батч атомарен. Если в том же массиве была валидная запись по org=42 — она тоже не запишется.

3. Фронт показывает toast «Нет доступа к организации id=99».

### Сценарий 4 — cascade вводит own_consumption через расширенный POST `/ges-report/daily-data`

Юзер `cascade-10` (org=42) на странице ges-report заполняет колонку «Собственные нужды» для своей строки.

1. Вводит `own_consumption_kwh = 1250.0` за 2026-04-28, кликает «Сохранить» (или auto-save).

   `POST /ges-report/daily-data`

   ```json
   [{
     "organization_id": 42,
     "date": "2026-04-28",
     "own_consumption_kwh": 1250.0
   }]
   ```

   Все остальные поля строки (`daily_production_mln_kwh`, `water_head_m` и т.д.) — НЕ в payload, они сохранятся в БД.

2. Backend: 200 OK. Поле записано в `ges_daily_data.own_consumption_kwh`.

3. Юзер обновляет страницу. Фронт делает `GET /ges-report?date=2026-04-28`. В ответе:

   ```json
   {
     "aggregations": {
       "mtd_production_mln_kwh": 12.5,
       "ytd_production_mln_kwh": 124.0,
       "mtd_own_consumption_kwh": 8420.0,
       "ytd_own_consumption_kwh": 95300.0
     }
   }
   ```

   `mtd_own_consumption_kwh` увеличилось на 1250 по сравнению с предыдущим запросом (если день ещё не входил в MTD-окно). `ytd` — аналогично. Цифры посчитаны только по org=42, потому что юзер cascade.

## 10. Чек-лист готовности фронта

- [ ] Добавлены TS-типы `SolarConfig`, `UpsertSolarConfigRequest`, `SolarDailyData`, `UpsertSolarDailyDataItem`, `SolarProductionPlan`, `UpsertSolarPlanItem`, `BulkUpsertSolarPlanRequest`.
- [ ] Расширены TS-типы ges-report: `own_consumption_kwh?: number | null` в Upsert-payload и в response `DailyData`; `mtd_own_consumption_kwh` / `ytd_own_consumption_kwh` в `Aggregations`.
- [ ] Реализован client для всех 7 solar endpoint'ов (`POST/GET /solar/daily-data`, `GET/POST/DELETE /solar/config`, `GET/POST /solar/plans`).
- [ ] Обновлён client `POST /ges-report/daily-data` — добавлено новое Optional-поле в типизацию запроса.
- [ ] Обновлён парсинг ответа `GET /ges-report` — два новых поля aggregations отображаются в дашборде.
- [ ] Role-based UI: для cascade скрыты/дизейблены кнопки управления solar config и solar plans; селектор org показывает только свою на solar daily-data.
- [ ] Сериализатор формы solar daily-data соблюдает Optional-семантику: не тронутые поля — НЕ в payload (см. §5 и helper `buildSolarPayload`). Аналогично для own_consumption_kwh в ges-report.
- [ ] Все числовые поля подписаны единицами в UI: `кВт`, `кВтч`, `тыс. кВтч`, `млн кВтч`. Не путать solar plan (тысячи) и GES plan (миллионы).
- [ ] Обработка 403 (toast «Нет доступа к организации» / «Нет прав»).
- [ ] Обработка 400 (показать details, для батча — подсветить упавшую строку, если backend вернёт `details.item_index`).
- [ ] Подсветка пустых дней в solar daily-data таблице.
- [ ] MTD/YTD own_consumption отображается в дашборде/отчёте рядом с production-агрегациями, в кВтч.
- [ ] Solar — отдельная вкладка/секция UI; не смешивается с ges-report таблицей.
- [ ] own_consumption_kwh — колонка в существующей форме ges-report daily-data.
- [ ] Ручное тестирование 4 сценариев из §9.

## 11. Changelog / Совместимость

- **Solar** — additive: 7 новых endpoint'ов под `/solar/*`. Никаких breaking-изменений в существующем API. Новых ролей нет — используются существующие `sc`, `rais`, `cascade`. Если фронт не обновляется — модуль для него просто не существует, остальные экраны не затронуты.
- **`own_consumption_kwh`** в `POST /ges-report/daily-data` — additive Optional-поле в request. Старый фронт без этого поля продолжит работать: при сохранении ges-report-формы поле просто не попадёт в payload (Optional.Set=false) → существующее значение в БД (или NULL по умолчанию после миграции) сохранится. Записи `ges_daily_data` после применения миграции имеют `own_consumption_kwh = NULL` для всех существующих строк — это корректно и ожидаемо.
- **`own_consumption_kwh`** в response `DailyData` ges-report — additive поле; старый фронт его проигнорирует.
- **`mtd_own_consumption_kwh` / `ytd_own_consumption_kwh`** в response `aggregations` ges-report — additive поля; старый фронт их проигнорирует. До накопления данных за период значения = 0.
- **MTD/YTD расчёт** перенесён в существующий SQL `GetGESProductionAggregations` (две дополнительные SUM по той же таблице). Производительность изменения незаметна. Никаких новых endpoint'ов или view не появилось.
- **Solar Excel/PDF export** — отложен на следующую итерацию; в текущей версии нет.
- **Own-Needs Excel export** — добавлен endpoint `GET /ges-report/own-needs/export?date=...` (sc/rais), отдающий отдельный xlsx-отчёт по СН/ХН с MTD/YTD по `own_consumption_kwh`. Шаблон `template/own-needs.xlsx`. Подробности — [ges-own-needs-export.md](ges-own-needs-export.md).
