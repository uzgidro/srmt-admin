# Reservoir-summary API (для фронта)

Описание эндпоинтов `GET /reservoir-summary` и `POST /reservoir-summary`,
включая правила доступа по ролям.

Параллельный документ `docs/reservoir-summary-config-api.md` покрывает
CRUD конфига `reservoir_summary_config`, который определяет состав отчёта.

## Эндпоинты

| Метод | Путь | Роли | Описание |
|---|---|---|---|
| `GET` | `/reservoir-summary?date=YYYY-MM-DD` | `sc`, `rais`, `reservoir` | Список объектов отчёта на указанную дату |
| `POST` | `/reservoir-summary` | `sc`, `rais`, `reservoir` | Внести/обновить данные на одну или несколько дат (bulk) |
| `GET` | `/reservoir-summary/export?date=YYYY-MM-DD` | `sc`, `rais` (НЕ `reservoir`) | Excel-экспорт всей картины |

## GET — поведение по ролям

- **`sc` / `rais`** → массив всех объектов из конфига + строка `ИТОГО`
  (`organization_id == null`).
- **`reservoir`** → **только** объекты, чьи `organization_id` есть в
  claims юзера (его собственные организации). Строка `ИТОГО`
  **не возвращается** — сумма по всем не имеет смысла для юзера,
  видящего только своё.
- **`reservoir` без `OrganizationIDs` в claims** → пустой массив `[]`.

### Пример

Запрос `reservoir`-юзера (`OrganizationIDs=[103]`):

```http
GET /reservoir-summary?date=2026-06-01
→ 200 OK
[
  {"organization_id": 103, "organization_name": "Пском", "volume": {...}, ...}
]
```

Тот же запрос от `sc`-юзера: 8 объектов + ИТОГО как раньше.

## POST — поведение по ролям

Тело запроса — массив `ReservoirDataItem[]` (см. модель
`reservoir-data`). Семантика:

- **`sc` / `rais`** могут писать в любую организацию.
- **`reservoir`** может писать **только** в свои `OrganizationIDs`.
  Попытка записи в чужой `organization_id` → `403 Forbidden`
  `"Access denied"`. Если в массиве смешаны свои+чужие — отказ ВСЕМУ
  запросу (не частичный).

### Успешный ответ

```http
POST /reservoir-summary
[{"organization_id": 103, "date": "2026-06-01", ...}]
→ 200 OK
{"status": "OK", "processed_count": 1}
```

### Отказ по доступу

```http
POST /reservoir-summary   (jwt: roles=[reservoir], orgs=[103])
[{"organization_id": 99, "date": "2026-06-01", ...}]
→ 403 Forbidden
{"status": "Error", "message": "Access denied"}
```

## UI-следствия

- **Кнопка "Экспорт в Excel"** должна быть скрыта для роли `reservoir`
  — эндпоинт `/reservoir-summary/export` вернёт 403.
- **Отрисовка таблицы** у `reservoir`-юзера: ожидать отсутствие строки
  `ИТОГО`, не падать на этом (не делать `summaries.find(s => s.organization_id === null)` обязательным).
- **Сабмит формы** у `reservoir`-юзера: проставлять `organization_id`
  из его claims автоматически, не давать выбирать чужие — даже если
  бэк защитит, явная UX-блокировка лучше silently-403.
- **`reservoir`-юзер без организаций** в claims увидит пустой массив —
  показывать осмысленное "У вашей учётной записи нет привязки к
  организации" вместо пустой таблицы.

## Связанные документы

- [`docs/reservoir-summary-config-api.md`](reservoir-summary-config-api.md)
  — CRUD конфига, который определяет какие организации входят в отчёт
  и в каком порядке.
