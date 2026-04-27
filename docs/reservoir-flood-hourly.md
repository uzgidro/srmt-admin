# Reservoir Flood Hourly — Frontend Implementation Guide

Документ описывает что нужно сделать на фронте, чтобы реализовать новый модуль почасовой отчётности по водохранилищам в период паводков (`reservoir_flood_hourly` + `reservoir_flood_config`).

## Бизнес-цель и семантика

В период паводков диспетчеры обязаны вести почасовой журнал состояния каждого водохранилища, включённого в паводковый список: уровень, объём, приток, расход через ГЭС, фильтрация, холостой сброс и ФИО дежурного. Это отдельный, более детальный поток данных, чем регулярная телеметрия `reservoir_data` — он:

- работает с шагом **1 час**, а не «текущая точка по запросу»;
- допускает **ручной ввод** оператором, а не только автоматическую запись с датчиков;
- закрывает только организации, явно включённые в `reservoir_flood_config` (не все водохранилища, а только те, что в паводковом периоде);
- хранит ФИО дежурного как поле `duty_name` (TEXT, свободный ввод).

**Кто пользуется:**

- `sc` и `rais` — все организации, полный read+write доступ, включая управление конфигом.
- `reservoir_duty` — новая роль «дежурный по водохранилищу». Доступ только к своей организации. Может писать почасовые данные, но не может управлять конфигом.

Backend уже выкачен (миграция, `/reservoir-flood/*` endpoints, доступ к `/level-volume` для `reservoir_duty`). Дальше — задача фронта.

## Затронутые/новые endpoints

| # | Endpoint | Роли | Что делает |
| --- | --- | --- | --- |
| 1 | `POST /reservoir-flood/hourly` | `sc`, `rais`, `reservoir_duty` | Bulk upsert почасовых записей (массив, даже на одну). |
| 2 | `GET /reservoir-flood/hourly?date=YYYY-MM-DD&organization_id=X` | `sc`, `rais`, `reservoir_duty` | Чтение записей за день. |
| 3 | `GET /reservoir-flood/config` | `sc`, `rais`, `reservoir_duty` | Список включённых организаций. |
| 4 | `POST /reservoir-flood/config` | `sc`, `rais` | Upsert одной строки конфига. |
| 5 | `DELETE /reservoir-flood/config?organization_id=X` | `sc`, `rais` | Удаление организации из конфига. |
| 6 | `GET /level-volume?id=&level=` | `sc`, `rais`, `reservoir`, `reservoir_duty` | Уже существующий эндпоинт. Сигнатура **не меняется**. Теперь к нему добавлен доступ для роли `reservoir_duty`. |

## Роли и матрица доступа

| Действие | `sc` | `rais` | `reservoir_duty` |
| --- | --- | --- | --- |
| GET `/reservoir-flood/hourly` (любая org) | OK | OK | только своя org |
| POST `/reservoir-flood/hourly` (своя org) | OK | OK | OK |
| POST `/reservoir-flood/hourly` (чужая org) | OK | OK | **403** |
| GET `/reservoir-flood/config` | OK (все строки) | OK (все строки) | только своя org в ответе |
| POST `/reservoir-flood/config` | OK | OK | **403** (route + handler defence) |
| DELETE `/reservoir-flood/config` | OK | OK | **403** (route + handler defence) |
| GET `/level-volume` | OK | OK | OK |

Защита от `reservoir_duty` на конфиг — двойная: роль не пропустит middleware, плюс handler ещё раз проверяет claims (defence-in-depth). Фронт может полагаться на 403, но должен заранее скрывать или дизейблить элементы управления конфигом для `reservoir_duty`-юзеров.

## Полные API-контракты

### 1. `POST /reservoir-flood/hourly`

**Roles:** `sc`, `rais`, `reservoir_duty`.

**Body** — массив объектов `UpsertHourlyRequest`. Даже если запись одна, оборачивайте в `[...]`.

```json
[
  {
    "organization_id": 42,
    "recorded_at": "2026-04-27T15:00:00Z",
    "water_level_m": 815.4,
    "water_volume_mln_m3": 1234.5,
    "inflow_m3s": 320.0,
    "outflow_m3s": 280.0,
    "ges_flow_m3s": 250.0,
    "filtration_m3s": 5.0,
    "idle_discharge_m3s": 25.0,
    "duty_name": "Иванов И.И."
  }
]
```

Все метрики — **nullable + Optional** (см. §5). `organization_id` и `recorded_at` обязательны.

**Response 200:**

```json
{ "status": "OK" }
```

**Errors:**

| Code | Когда | Тело |
| --- | --- | --- |
| 400 | Невалидный JSON / отсутствует `organization_id` или `recorded_at` / неверный формат `recorded_at` / пустой массив | `{"error": "invalid request: <details>"}`, опционально `details: { item_index: N, field: "..." }` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | Роль `reservoir_duty` пишет в чужую `organization_id` | `{"error": "access to target organization denied"}` |
| 500 | Ошибка БД | `{"error": "internal server error"}` |

При 400 на батче backend возвращает индекс упавшего элемента в `details.item_index` — фронту нужно его распарсить, чтобы показать «ошибка в строке 3».

### 2. `GET /reservoir-flood/hourly`

**Roles:** `sc`, `rais`, `reservoir_duty`.

**Query:**

- `date` — обязателен, формат `YYYY-MM-DD`. Backend трактует как окно `[date 00:00:00 UTC, date+1 00:00:00 UTC)`.
- `organization_id` — опционален. Если не задан — возвращаются все организации, к которым у текущего юзера есть доступ. Для `reservoir_duty` — только своя org (фильтрация на стороне сервера).

```
GET /reservoir-flood/hourly?date=2026-04-27&organization_id=42
```

**Response 200** — массив `HourlyRecord`:

```json
[
  {
    "id": 1001,
    "organization_id": 42,
    "organization_name": "Чарвакское водохранилище",
    "recorded_at": "2026-04-27T15:00:00Z",
    "water_level_m": 815.4,
    "water_volume_mln_m3": 1234.5,
    "inflow_m3s": 320.0,
    "outflow_m3s": 280.0,
    "ges_flow_m3s": 250.0,
    "filtration_m3s": 5.0,
    "idle_discharge_m3s": 25.0,
    "duty_name": "Иванов И.И.",
    "created_by_user_id": 10,
    "updated_at": "2026-04-27T15:02:11Z"
  }
]
```

Если на час записи нет — её просто не будет в массиве. Фронт сам решает, как помечать пропущенные часы (см. §8).

**Errors:**

| Code | Когда | Тело |
| --- | --- | --- |
| 400 | `date` отсутствует или невалидного формата | `{"error": "invalid date format, expected YYYY-MM-DD"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | `reservoir_duty` запросил чужую `organization_id` | `{"error": "access to target organization denied"}` |
| 500 | Ошибка БД | `{"error": "internal server error"}` |

### 3. `GET /reservoir-flood/config`

**Roles:** `sc`, `rais`, `reservoir_duty`.

Параметров нет.

**Response 200** — массив `Config`:

```json
[
  {
    "id": 1,
    "organization_id": 42,
    "organization_name": "Чарвакское водохранилище",
    "sort_order": 10,
    "is_active": true,
    "updated_at": "2026-04-25T08:00:00Z"
  },
  {
    "id": 2,
    "organization_id": 43,
    "organization_name": "Тюямуюнское водохранилище",
    "sort_order": 20,
    "is_active": true,
    "updated_at": "2026-04-25T08:05:00Z"
  }
]
```

Для `reservoir_duty`-юзера ответ отфильтрован сервером — вернётся максимум одна запись (его собственная org), либо пустой массив, если его org не включена в паводковый список.

**Errors:**

| Code | Когда | Тело |
| --- | --- | --- |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 500 | Ошибка БД | `{"error": "internal server error"}` |

### 4. `POST /reservoir-flood/config`

**Roles:** `sc`, `rais` (роут защищён `RequireAnyRole("sc","rais")`).

**Body** — одиночный объект (НЕ массив):

```json
{
  "organization_id": 42,
  "sort_order": 10,
  "is_active": true
}
```

Семантика — upsert по `organization_id`: если строка есть, обновляется; если нет — создаётся.

**Response 200:**

```json
{ "status": "OK" }
```

**Errors:**

| Code | Когда | Тело |
| --- | --- | --- |
| 400 | Невалидный JSON / отсутствует `organization_id` / `sort_order` < 0 | `{"error": "invalid request: <details>"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | Роль `reservoir_duty` (или любая другая, кроме sc/rais) | `{"error": "forbidden"}` |
| 404 | `organization_id` не существует в `organizations` | `{"error": "organization not found"}` |
| 500 | Ошибка БД | `{"error": "internal server error"}` |

### 5. `DELETE /reservoir-flood/config`

**Roles:** `sc`, `rais`.

**Query:** `organization_id` — обязателен.

```
DELETE /reservoir-flood/config?organization_id=42
```

**Response 204** No Content. Тело пустое.

**Errors:**

| Code | Когда | Тело |
| --- | --- | --- |
| 400 | `organization_id` отсутствует или невалидный | `{"error": "invalid organization_id"}` |
| 401 | Нет/протух токен | `{"error": "unauthorized"}` |
| 403 | Не sc/rais | `{"error": "forbidden"}` |
| 404 | Записи с таким `organization_id` нет | `{"error": "config not found"}` |
| 500 | Ошибка БД | `{"error": "internal server error"}` |

> **Примечание:** DELETE НЕ идемпотентен — повторный запрос на уже удалённый `organization_id` вернёт **404**. Фронту обработать этот кейс (показать toast «уже удалено» или просто скрыть строку из UI).

### 6. `GET /level-volume`

**Roles:** `sc`, `rais`, `reservoir`, `reservoir_duty`.

API не меняется — это уже существующий эндпоинт. Единственное изменение — в список разрешённых ролей добавлена `reservoir_duty`, чтобы дежурный мог пользоваться auto-fill объёма по уровню (см. §8).

```
GET /level-volume?id=42&level=815.4
```

**Response 200:**

```json
{ "volume": 1234.5 }
```

Все коды ошибок и формат — как раньше. Фронту менять ничего не нужно, кроме того, что вызов теперь работает и для `reservoir_duty`.

## Optional-семантика для POST /reservoir-flood/hourly

**Критически важно для понимания.** Все метрики (`water_level_m`, `water_volume_mln_m3`, `inflow_m3s`, `outflow_m3s`, `ges_flow_m3s`, `filtration_m3s`, `idle_discharge_m3s`, `duty_name`) на бэкенде представлены типом `Optional[T]` с тремя состояниями:

| JSON состояние | Backend интерпретирует как | Что произойдёт в БД |
| --- | --- | --- |
| **поле отсутствует** в payload | `Set=false` | колонка НЕ трогается (preserve existing) |
| `"field": null` | `Set=true, Value=nil` | колонка пишется в `NULL` |
| `"field": <value>` | `Set=true, Value=&value` | колонка пишется в `value` |

### Примеры

**(a) Обновить только уровень — остальные сохранить:**

```json
{
  "organization_id": 42,
  "recorded_at": "2026-04-27T15:00:00Z",
  "water_level_m": 815.4
}
```

Только `water_level_m` будет записан. Если в БД на этот час уже стояли `inflow_m3s = 320`, `outflow_m3s = 280` — они **останутся** на месте.

**(b) Явно очистить уровень:**

```json
{
  "organization_id": 42,
  "recorded_at": "2026-04-27T15:00:00Z",
  "water_level_m": null
}
```

`water_level_m` будет записан в `NULL`. Остальные поля не трогаются.

**(c) Полная перезапись часа:**

```json
{
  "organization_id": 42,
  "recorded_at": "2026-04-27T15:00:00Z",
  "water_level_m": 815.4,
  "water_volume_mln_m3": 1234.5,
  "inflow_m3s": 320.0,
  "outflow_m3s": 280.0,
  "ges_flow_m3s": 250.0,
  "filtration_m3s": 5.0,
  "idle_discharge_m3s": 25.0,
  "duty_name": "Иванов И.И."
}
```

Все восемь полей запишутся.

> **Внимание фронту.** Если хотите ОБНОВИТЬ только одно поле — шлите только его. Если на каждый клик по ячейке отправлять полный объект формы со всеми значениями, вы случайно затрёте уже введённые коллегой данные. Стандартный JSON-сериализатор (`JSON.stringify` без фильтра) включает `undefined`-поля как отсутствующие — это удобно. Но если у вас на форме хранится `null` для пустых ячеек — это будет означать «очистить» на бэкенде, что почти никогда не то, что вы хотели. Используйте сериализатор, который пропускает поля по правилу «не отправлять, если пользователь не трогал ячейку».

## Time-нормализация

Поле `recorded_at` нормализуется на сервере к началу часа в UTC. UNIQUE constraint на `(organization_id, recorded_at)` гарантирует, что на каждую пару `(org, час)` будет ровно одна запись.

Примеры нормализации:

| Что прислал фронт | Что положено в БД |
| --- | --- |
| `2026-04-27T15:42:18Z` | `2026-04-27T15:00:00Z` |
| `2026-04-27T15:00:00.999Z` | `2026-04-27T15:00:00Z` |
| `2026-04-27T15:59:59Z` | `2026-04-27T15:00:00Z` |
| `2026-04-27T15:00:00+05:00` (Asia/Tashkent) | `2026-04-27T10:00:00Z` (тот же абсолютный момент, нормализован в UTC до начала часа) |

Фронт может слать любой timestamp, попадающий в нужный час — backend сам обрежет минуты и секунды и переведёт в UTC. Удобный паттерн на фронте — отправлять `new Date(year, month, day, hour, 0, 0).toISOString()`, либо просто `new Date().toISOString()` если запись «сейчас».

При чтении (`GET /reservoir-flood/hourly`) `recorded_at` всегда возвращается как ISO-8601 в UTC, ровно по началу часа.

## TS-модель + helpers

```typescript
interface Config {
  id: number;
  organization_id: number;
  organization_name?: string;
  sort_order: number;
  is_active: boolean;
  updated_at: string;
}

interface HourlyRecord {
  id: number;
  organization_id: number;
  organization_name?: string;
  recorded_at: string;        // ISO 8601 UTC, normalized to hour
  water_level_m: number | null;
  water_volume_mln_m3: number | null;
  inflow_m3s: number | null;
  outflow_m3s: number | null;
  ges_flow_m3s: number | null;
  filtration_m3s: number | null;
  idle_discharge_m3s: number | null;
  duty_name: string | null;
  created_by_user_id?: number | null;
  updated_at: string;
}

// Optional<T> — три состояния. Если поле отсутствует в payload — backend
// не трогает соответствующую колонку. Если null — пишет NULL. Если value —
// пишет value.
interface UpsertHourlyRequest {
  organization_id: number;
  recorded_at: string;       // ISO-8601; backend normalizes to hour
  water_level_m?: number | null;
  water_volume_mln_m3?: number | null;
  inflow_m3s?: number | null;
  outflow_m3s?: number | null;
  ges_flow_m3s?: number | null;
  filtration_m3s?: number | null;
  idle_discharge_m3s?: number | null;
  duty_name?: string | null;
}

interface UpsertConfigRequest {
  organization_id: number;
  sort_order: number;
  is_active: boolean;
}

type Role = "sc" | "rais" | "reservoir_duty";

function canEditConfig(roles: Role[]): boolean {
  return roles.includes("sc") || roles.includes("rais");
}

function canEditOrg(roles: Role[], userOrgID: number, recordOrgID: number): boolean {
  if (roles.includes("sc") || roles.includes("rais")) return true;
  if (roles.includes("reservoir_duty")) return userOrgID === recordOrgID;
  return false;
}

// Helper: нормализация ISO-строки к началу часа в UTC (необязательно — бэкенд
// и сам обрежет, но удобно для отображения и для дедупа на клиенте).
function toHourStartUTC(iso: string): string {
  const d = new Date(iso);
  d.setUTCMinutes(0, 0, 0);
  return d.toISOString();
}

// Helper: сериализация формы с учётом Optional-семантики.
// touched — Set имён полей, которые юзер фактически менял.
// Поля не из touched НЕ попадают в payload (preserve в БД).
function buildHourlyPayload(
  form: Partial<UpsertHourlyRequest>,
  touched: Set<keyof UpsertHourlyRequest>,
): UpsertHourlyRequest {
  const out: any = {
    organization_id: form.organization_id,
    recorded_at: form.recorded_at,
  };
  for (const key of touched) {
    if (key === "organization_id" || key === "recorded_at") continue;
    out[key] = form[key] ?? null;  // null — это «очистить»
  }
  return out as UpsertHourlyRequest;
}
```

## UX-паттерн

Базовый layout страницы — таблица **24 × N** (24 часа по вертикали × организации по горизонтали), либо **N × 24** на усмотрение дизайнера. Каждая ячейка раскрывается в редактируемую форму с 8 полями (`water_level_m`, `water_volume_mln_m3`, `inflow_m3s`, `outflow_m3s`, `ges_flow_m3s`, `filtration_m3s`, `idle_discharge_m3s`, `duty_name`).

**Ключевые UX-правила:**

- **Auto-fill объёма по уровню.** При вводе `water_level_m` для конкретной организации — отправить `GET /level-volume?id={orgID}&level={value}` и предзаполнить `water_volume_mln_m3` из ответа (`{"volume": ...}`). Поле остаётся редактируемым — пользователь может вручную поменять значение, если auto-fill не подходит. Дебаунсить запрос (например, 400 мс после остановки ввода).
- **Подсветка пропущенных часов.** Если `GET /reservoir-flood/hourly` не вернул запись на конкретный час либо в записи метрика — `null`, ячейка отображается с маркером «нет данных» (например, серый фон или пунктирная рамка). Это помогает диспетчеру визуально найти пропуски в журнале.
- **Селектор организаций для `reservoir_duty`.** Должен показывать только собственную организацию пользователя. Это дополнительная защита поверх серверной фильтрации (defence-in-depth) — UI просто не даёт юзеру даже выбрать чужую org.
- **Скрытие управления конфигом для `reservoir_duty`.** Кнопки «добавить организацию», «удалить организацию», «изменить sort_order», «вкл/выкл паводковый режим» — должны быть скрыты или дизейблены через `canEditConfig(roles)`. Если оставлены кликабельными для отладки — обработать 403 как fallback.
- **Поле `duty_name`.** Свободный ввод (TEXT). Удобно подсказывать последний введённый дежурным текст (autocomplete по локальному стейту или по последним записям из ответа `GET /hourly`). Но это лишь UX-помощь, валидации формата на бэкенде нет.
- **Сохранение по ячейке vs батчем.** Два паттерна:
  1. Auto-save при потере фокуса ячейки — отправлять одну запись `[{...}]`, обновлять только тронутые поля (см. §5).
  2. Кнопка «Сохранить всё» — собрать все изменённые ячейки в один батч и отправить массивом. При ошибке — распарсить `details.item_index` из 400-ответа и подсветить строку.

  Рекомендация — auto-save: уменьшает риск потери данных при перезагрузке вкладки.

## End-to-end сценарии

### Сценарий 1 — sc включает org A в паводковый список

1. SC-админ открывает страницу управления конфигом, видит список организаций.
2. Кликает «Добавить» рядом с org A (`organization_id = 42`), задаёт `sort_order = 10`, `is_active = true`.
3. Фронт: `POST /reservoir-flood/config`:

   ```json
   { "organization_id": 42, "sort_order": 10, "is_active": true }
   ```

4. Backend: 200 OK.

   ```json
   { "status": "OK" }
   ```

5. UI обновляет список конфига; org A теперь видна всем `reservoir_duty`-юзерам этой org.

### Сценарий 2 — reservoir_duty юзер 10 (org A) вводит данные за 15:00

1. Юзер 10 открывает страницу почасового ввода, видит свою org A в селекторе (единственный вариант).
2. В строке 15:00 заполняет `water_level_m = 815.4`. Auto-fetch: `GET /level-volume?id=42&level=815.4` → `{"volume": 1234.5}` → поле `water_volume_mln_m3` предзаполнено.
3. Заполняет `duty_name = "Иванов И.И."`, оставляет остальные пустыми.
4. Кликает «Сохранить» (либо срабатывает auto-save). Фронт: `POST /reservoir-flood/hourly`:

   ```json
   [
     {
       "organization_id": 42,
       "recorded_at": "2026-04-27T15:00:00Z",
       "water_level_m": 815.4,
       "water_volume_mln_m3": 1234.5,
       "duty_name": "Иванов И.И."
     }
   ]
   ```

5. Backend: 200 OK.

   ```json
   { "status": "OK" }
   ```

6. UI помечает ячейку как сохранённую. `inflow_m3s`, `outflow_m3s` и т.д. — остались `null` в БД (preserve, т.к. их не было в payload).

### Сценарий 3 — тот же юзер пробует ввести данные за чужую org B

1. Юзер 10 (роль `reservoir_duty`, org A = 42) каким-то образом (DevTools, прямой POST) пробует записать в org B (`organization_id = 99`).
2. Фронт (теоретически, через DevTools): `POST /reservoir-flood/hourly`:

   ```json
   [
     {
       "organization_id": 99,
       "recorded_at": "2026-04-27T15:00:00Z",
       "water_level_m": 700.0
     }
   ]
   ```

3. Backend: 403.

   ```json
   { "error": "access to target organization denied" }
   ```

4. В БД ничего не записано — батч атомарен, при первом же недопустимом элементе вся транзакция откатывается.
5. Фронт показывает toast «Нет доступа к организации id=99».

### Сценарий 4 — sc редактирует данные org A через bulk POST

1. SC-админ открывает страницу за `2026-04-27`, видит, что в часах 12:00–14:00 у org A пропущены `inflow_m3s` и `outflow_m3s`.
2. Заполняет три ячейки `inflow_m3s` для 12:00, 13:00, 14:00 одновременно, кликает «Сохранить всё».
3. Фронт: `POST /reservoir-flood/hourly`:

   ```json
   [
     { "organization_id": 42, "recorded_at": "2026-04-27T12:00:00Z", "inflow_m3s": 310.0 },
     { "organization_id": 42, "recorded_at": "2026-04-27T13:00:00Z", "inflow_m3s": 315.0 },
     { "organization_id": 42, "recorded_at": "2026-04-27T14:00:00Z", "inflow_m3s": 320.0 }
   ]
   ```

4. Backend: 200 OK.

   ```json
   { "status": "OK" }
   ```

5. UI обновляет три ячейки. `outflow_m3s`, `water_level_m` и прочие поля во всех трёх часах — preserve (не трогались).

## Чек-лист готовности фронта

- [ ] Добавлены TS-типы `Config`, `HourlyRecord`, `UpsertHourlyRequest`, `UpsertConfigRequest`.
- [ ] Реализован client для всех 5 новых endpoint'ов (`POST/GET /hourly`, `GET/POST/DELETE /config`).
- [ ] Auto-fetch `GET /level-volume?id=&level=` при вводе уровня (с дебаунсом ~400 мс).
- [ ] Role-based UI: для `reservoir_duty` скрыты/дизейблены кнопки управления конфигом, селектор org показывает только свою.
- [ ] Сериализатор формы соблюдает Optional-семантику: не тронутые поля — НЕ в payload (см. §5, helper `buildHourlyPayload`).
- [ ] Обработка 403 (toast «нет доступа к этой организации»).
- [ ] Обработка 400 на батче: распарсить `details.item_index`, подсветить упавшую строку.
- [ ] Подсветка пропущенных часов в таблице (нет записи или `null`-метрика).
- [ ] Ручное тестирование 4 сценариев из §9.

## Changelog / Совместимость

- 5 новых endpoint'ов (`POST/GET /reservoir-flood/hourly`, `GET/POST/DELETE /reservoir-flood/config`) — additive, ничего из существующего API не ломают.
- `GET /level-volume` — сигнатура и формат ответа **без изменений**, добавлена только новая роль `reservoir_duty` в whitelist. Существующие потребители (`reservoir`, `sc`, `rais`) работают как раньше.
- Роль `reservoir_duty` назначается через существующую систему ролей (assign в админке). Назначение роли не требует переезда пользователя в новую организацию — он остаётся в той же org, просто получает дополнительный уровень доступа к новому модулю.
- Старый фронт без обновления продолжит работать в полном объёме — модуль `reservoir-flood` для него просто не существует. Никакие из существующих экранов (телеметрия, отчёты ГЭС, shutdowns и т.д.) не затронуты.
- Новая роль `reservoir_duty` — только write-доступ к новому модулю. Если юзеру случайно назначили эту роль и забыли — у него не появится никаких прав за пределами `/reservoir-flood/*` и `/level-volume`.
