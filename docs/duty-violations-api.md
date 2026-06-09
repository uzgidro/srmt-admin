# Прогулы дежурных (Duty-officer violations) — API для фронта

Учёт нарушений (прогулов) дежурных по объектам: фиксируем организацию,
интервал смены, имя дежурного, причину и (опционально) несколько
прикреплённых файлов.

## Эндпоинты

| Метод | Путь | Роли | Описание |
|---|---|---|---|
| `GET` | `/duty-violations` | `sc`, `rais` | Список нарушений с опц. фильтрами |
| `POST` | `/duty-violations` | `sc`, `rais` | Создать запись |
| `PATCH` | `/duty-violations/{id}` | `sc`, `rais` | Обновить запись (file_ids — **полная замена** списка файлов) |
| `DELETE` | `/duty-violations/{id}` | `sc`, `rais` | Удалить запись |

> Доступ к чужой `organization_id` блокируется на бэке — `auth.CheckOrgAccess`
> возвращает 403 ещё до того, как затрагивается БД. `sc`/`rais` имеют
> сквозной доступ ко всем организациям; для не-sc/rais ролей (когда они
> будут добавлены в группу маршрутов) действуют следующие правила:
>
> - **GET без `organization_id`** автоматически фильтруется по первой
>   организации из claims пользователя.
> - **GET с `organization_id`** на чужую организацию → 403.
> - **PATCH/DELETE** загружают существующую запись и проверяют доступ к
>   её ТЕКУЩЕЙ `organization_id` (защита от IDOR — нельзя «отнять»
>   запись передав свой `organization_id` в body).
> - **PATCH с переносом** (новый `organization_id` ≠ старого) требует
>   доступа и к старой, и к новой организации.

## Workflow с файлами

Двухшаговая загрузка — как у incidents/discharges:

1. Фронт загружает каждый файл через `POST /upload/files` (multipart),
   получает массив `{id, file_name, ...}`.
2. В теле `POST /duty-violations` передаёт массив этих id в поле
   `file_ids`: `"file_ids": [42, 43]`.
3. На `PATCH /duty-violations/{id}` отправляется **полный** массив
   `file_ids` (не дельта):
   - чтобы добавить файл — `[...старые_id, новый_id]`
   - чтобы убрать — `[...без_него]`
   - чтобы очистить — `[]` или вообще опустить поле

> Удаление записи через `DELETE` каскадно удаляет связи в junction-таблице,
> но **сами файлы из MinIO не удаляет** — они могут быть прикреплены к
> другим записям или нужны для аудита. Для удаления самого файла —
> существующий `DELETE /files/{fileID}`.

## Фильтры списка

`GET /duty-violations?organization_id=N&from=YYYY-MM-DD&to=YYYY-MM-DD`

Все три фильтра опциональны и независимы. Без фильтров возвращает все
записи. Сортировка — по `start_time DESC` (свежие сверху), затем по `id DESC`
(стабильный tie-breaker).

| Параметр | Тип | Описание |
|---|---|---|
| `organization_id` | int (>0) | Фильтр по организации |
| `from` | `YYYY-MM-DD` | start_time ≥ 05:00 этой даты (Asia/Tashkent) |
| `to` | `YYYY-MM-DD` | start_time < 05:00 СЛЕДУЮЩЕГО дня — полу-открытый интервал, события дня `to` ВХОДЯТ в выборку |

> **Operational day:** дни в системе начинаются в 05:00 Asia/Tashkent —
> единый паттерн со всеми ежедневными отчётами (incidents, discharges,
> ges-report, day rotation). Граничный случай: запись со
> `start_time = 05:00:00` относится к НАЧАЛУ нового op-day, не к концу
> предыдущего (совпадает с поведением dayrotation cutoff).
>
> Примеры:
>
> - **Ровно одни сутки 08.06:** `from=2026-06-08&to=2026-06-08` →
>   `[2026-06-08T05:00+05:00 .. 2026-06-09T05:00+05:00)` — включает в
>   том числе записи между полуночью и 05:00 утра 9 июня.
> - **Весь июнь 2026:** `from=2026-06-01&to=2026-06-30` →
>   `[2026-06-01T05:00 .. 2026-07-01T05:00)` (последний день включён).
>
> **Breaking change vs предыдущей версии:** до этого `from`/`to`
> парсились как UTC midnight, теперь — как op-day Asia/Tashkent. Сдвиг
> границы ≈ 5 часов; фронт скорее всего и хотел именно бизнес-день.

## TypeScript DTO

```ts
interface FileMeta {
  id: number;
  file_name: string;
  category_id: number;
  mime_type: string;
  size_bytes: number;
  created_at: string;   // ISO 8601
}

interface DutyViolation {
  id: number;
  organization_id: number;
  organization_name?: string;  // JOIN из organizations.name
  start_time: string;          // ISO 8601 с timezone
  end_time: string;            // > start_time
  duty_officer_name: string;
  reason: string;
  files: FileMeta[];           // всегда массив (возможно пустой), не null
  created_at: string;
  created_by_user_id?: number; // null если пользователь удалён
  updated_at: string;
}

interface CreateDutyViolationRequest {
  organization_id: number;   // > 0
  start_time: string;        // ISO 8601
  end_time: string;          // должно быть > start_time
  duty_officer_name: string; // 1..200 символов, не пустая после trim
  reason: string;            // 1..2000 символов, не пустая после trim
  file_ids?: number[];       // опц., каждый > 0
}

type UpdateDutyViolationRequest = CreateDutyViolationRequest;
```

## Коды ответов

| Эндпоинт | Успех | Ошибки |
|---|---|---|
| `POST` | `200` + `DutyViolation` | `400` invalid JSON / validation, `401` not auth, `403` foreign org, `422` org/file не существует, `500` server |
| `GET` | `200` + `DutyViolation[]` | `400` invalid filters, `500` |
| `PATCH` | `200` + обновлённый `DutyViolation` | `400`, `403`, `404` not found, `422`, `500` |
| `DELETE` | `204` + `{"status":"Deleted"}` | `400` invalid id, `404`, `500` |

Тело ошибки — стандартный формат:

```json
{ "status": "Error", "message": "Access denied" }
```

или (для валидации) — массив ошибок поля:

```json
{ "status": "Error", "errors": [{"field":"end_time","message":"..."}] }
```

## Примеры

### Создание

```http
POST /duty-violations
Authorization: Bearer ...
Content-Type: application/json

{
  "organization_id": 103,
  "start_time": "2026-06-08T08:00:00+05:00",
  "end_time":   "2026-06-08T20:00:00+05:00",
  "duty_officer_name": "Иванов И.И.",
  "reason": "Не вышел на смену; уведомление не передал.",
  "file_ids": [42, 43]
}
```

```http
200 OK
{
  "id": 7,
  "organization_id": 103,
  "organization_name": "Пском",
  "start_time": "2026-06-08T08:00:00+05:00",
  "end_time":   "2026-06-08T20:00:00+05:00",
  "duty_officer_name": "Иванов И.И.",
  "reason": "Не вышел на смену; уведомление не передал.",
  "files": [
    {"id": 42, "file_name": "act.pdf", ...},
    {"id": 43, "file_name": "report.docx", ...}
  ],
  "created_at": "2026-06-09T10:00:00Z",
  "created_by_user_id": 7,
  "updated_at": "2026-06-09T10:00:00Z"
}
```

### Список с фильтрами

```http
GET /duty-violations?organization_id=103&from=2026-06-01&to=2026-07-01
→ 200 OK
[
  { "id": 7, "organization_id": 103, ... },
  { "id": 3, "organization_id": 103, ... }
]
```

### Обновление — добавить ещё один файл

```http
PATCH /duty-violations/7
Content-Type: application/json

{
  "organization_id": 103,
  "start_time": "2026-06-08T08:00:00+05:00",
  "end_time":   "2026-06-08T20:00:00+05:00",
  "duty_officer_name": "Иванов И.И.",
  "reason": "Не вышел на смену; уведомление не передал.",
  "file_ids": [42, 43, 99]
}
```

### Обновление — отвязать все файлы

```http
PATCH /duty-violations/7
{
  "organization_id": 103,
  "start_time": "...",
  "end_time": "...",
  "duty_officer_name": "Иванов И.И.",
  "reason": "Не вышел на смену.",
  "file_ids": []
}
```

### Удаление

```http
DELETE /duty-violations/7
→ 204 No Content
```

## UI-рекомендации

- **Кнопка "Прикрепить файл"** — двухшаговая: upload → запоминаем
  возвращённый `id` → отправляем все накопленные id в `POST/PATCH`.
- **Форма редактирования** должна показывать текущий список файлов
  (из последнего GET'а) и при сохранении прислать **полный** массив
  `file_ids` — иначе все привязки сбросятся.
- **Datetime-picker** для `start_time` / `end_time` — обязательно
  валидировать `end > start` на клиенте до сабмита (бэк всё равно
  отобьёт 400, но клиентская проверка экономит round-trip).
- **403 Forbidden** на запросе своей же организации — означает что
  токен утратил привязку (relogin) или роль изменили; форма должна
  показать осмысленное сообщение.
- **422 Unprocessable Entity** на POST/PATCH — `organization_id` или
  один из `file_ids` указывает на несуществующую сущность. Покажите
  «организация или файл не найдены» и предложите проверить список.

## Связанные endpoints

- `POST /upload/files` (multipart) — загрузка файла, возвращает id для
  использования в `file_ids`.
- `GET /files/{fileID}/download` — скачивание файла из приложенных.
- `DELETE /files/{fileID}` — физическое удаление файла (если он не
  нужен ни в одной записи).
