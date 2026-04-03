# Manual Comparison API (Ручное сравнение фильтрации)

Модуль для ручного ввода текущих и исторических данных фильтрации/пьезометров **без привязки к автоматически найденной исторической дате**. Используется когда нет исторической даты с подобным уровнем водохранилища.

**Базовый путь:** `/manual-comparison`
**Авторизация:** JWT, роли `sc`, `rais`, `reservoir`

---

## 1. POST /measurements — Сохранить данные

Создаёт или обновляет текущие и исторические измерения для организации на указанную дату.

### Request

```
POST /manual-comparison/measurements
Content-Type: application/json
Authorization: Bearer <token>
```

```json
{
  "organization_id": 1,
  "date": "2026-03-25",
  "historical_filter_date": "2025-09-15",
  "historical_piezo_date": "2025-09-15",
  "filters": [
    {
      "location_id": 10,
      "flow_rate": 12.5,
      "historical_flow_rate": 11.8
    },
    {
      "location_id": 11,
      "flow_rate": 8.3,
      "historical_flow_rate": null
    }
  ],
  "piezos": [
    {
      "piezometer_id": 20,
      "level": 45.67,
      "anomaly": false,
      "historical_level": 44.12
    },
    {
      "piezometer_id": 21,
      "level": 32.10,
      "historical_level": 31.50
    }
  ]
}
```

### Поля запроса

| Поле | Тип | Обязательное | Описание |
|------|-----|:---:|----------|
| `organization_id` | int64 | да | ID организации (водохранилища) |
| `date` | string | да | Дата текущих измерений, `YYYY-MM-DD` |
| `historical_filter_date` | string | нет | Произвольная текстовая метка исторической даты для фильтрации (например `"2025-09-15"` или `""`) |
| `historical_piezo_date` | string | нет | Произвольная текстовая метка исторической даты для пьезометров |
| `filters` | array | нет | Массив измерений фильтрации (минимум 1 элемент в `filters` или `piezos`) |
| `filters[].location_id` | int64 | да | ID места фильтрации (из `/filtration/locations`) |
| `filters[].flow_rate` | float64? | нет | Текущий расход, л/с (`null` = не задано) |
| `filters[].historical_flow_rate` | float64? | нет | Исторический расход, л/с (`null` = не задано) |
| `piezos` | array | нет | Массив измерений пьезометров |
| `piezos[].piezometer_id` | int64 | да | ID пьезометра (из `/filtration/piezometers`) |
| `piezos[].level` | float64? | нет | Текущий уровень (`null` = не задано) |
| `piezos[].anomaly` | bool? | нет | Признак аномалии (`null` = сохранить предыдущее значение) |
| `piezos[].historical_level` | float64? | нет | Исторический уровень (`null` = не задано) |

### Response — 200 OK

```json
{
  "status": "OK"
}
```

### Ошибки

| Код | Когда |
|-----|-------|
| 400 | Невалидный JSON, отсутствует `organization_id`/`date`, неверный формат даты, пустые `filters` и `piezos` одновременно |
| 401 | Нет токена |
| 403 | Нет доступа к организации |
| 500 | Ошибка БД |

### Поведение

- **Upsert**: если данные на эту дату уже есть — перезапишет. Ключ уникальности: `(location_id, date)` / `(piezometer_id, date)`.
- **Частичное обновление**: можно отправить только `filters` без `piezos` и наоборот. Уже сохранённые данные другого типа не удаляются.
- `historical_filter_date` / `historical_piezo_date` — свободный текст. Можно `"2025-09-15"`, можно `"сентябрь 2025"`, можно `""`.

---

## 2. GET /measurements — Получить данные для формы

Возвращает данные ручного сравнения для одной организации на дату. Включает **все** места фильтрации и пьезометры организации (даже если данные ещё не введены — значения будут `null`).

### Request

```
GET /manual-comparison/measurements?organization_id=1&date=2026-03-25
Authorization: Bearer <token>
```

| Параметр | Тип | Обязательный | Описание |
|----------|-----|:---:|----------|
| `organization_id` | int64 | да | ID организации |
| `date` | string | да | Дата, `YYYY-MM-DD` |

### Response — 200 OK

```json
{
  "organization_id": 1,
  "organization_name": "Чорвок",
  "date": "2026-03-25",
  "historical_filter_date": "2025-09-15",
  "historical_piezo_date": "2025-09-15",
  "filters": [
    {
      "location_id": 10,
      "location_name": "Левый берег",
      "norm": 15.0,
      "sort_order": 1,
      "flow_rate": 12.5,
      "historical_flow_rate": 11.8
    },
    {
      "location_id": 11,
      "location_name": "Правый берег",
      "norm": 10.0,
      "sort_order": 2,
      "flow_rate": 8.3,
      "historical_flow_rate": null
    }
  ],
  "piezometers": [
    {
      "piezometer_id": 20,
      "piezometer_name": "ПК-1",
      "norm": 50.0,
      "sort_order": 1,
      "level": 45.67,
      "anomaly": false,
      "historical_level": 44.12
    },
    {
      "piezometer_id": 21,
      "piezometer_name": "ПК-2",
      "norm": null,
      "sort_order": 2,
      "level": null,
      "anomaly": false,
      "historical_level": null
    }
  ]
}
```

### Поля ответа

| Поле | Тип | Описание |
|------|-----|----------|
| `organization_id` | int64 | ID организации |
| `organization_name` | string | Название организации |
| `date` | string | Дата |
| `historical_filter_date` | string | Текстовая метка исторической даты фильтрации (пустая строка если не задана) |
| `historical_piezo_date` | string | Текстовая метка исторической даты пьезометров |
| `filters` | array | Все места фильтрации организации |
| `filters[].location_id` | int64 | ID места |
| `filters[].location_name` | string | Название |
| `filters[].norm` | float64? | Норматив (мб `null`) |
| `filters[].sort_order` | int | Порядок сортировки |
| `filters[].flow_rate` | float64? | Текущий расход (`null` если не введён) |
| `filters[].historical_flow_rate` | float64? | Исторический расход (`null` если не введён) |
| `piezometers` | array | Все пьезометры организации |
| `piezometers[].piezometer_id` | int64 | ID пьезометра |
| `piezometers[].piezometer_name` | string | Название |
| `piezometers[].norm` | float64? | Норматив (`null` если не задан) |
| `piezometers[].sort_order` | int | Порядок сортировки |
| `piezometers[].level` | float64? | Текущий уровень (`null` если не введён) |
| `piezometers[].anomaly` | bool | Признак аномалии |
| `piezometers[].historical_level` | float64? | Исторический уровень (`null` если не введён) |

### Важно

- Ответ **всегда** содержит все location-ы и пьезометры организации (LEFT JOIN). Если данные не были введены — значения `null`.
- `historical_filter_date` / `historical_piezo_date` — пустая строка `""` если не было POST-запроса на эту дату.

---

## 3. DELETE /measurements — Удалить данные

Удаляет все данные ручного сравнения для организации на дату.

### Request

```
DELETE /manual-comparison/measurements?organization_id=1&date=2026-03-25
Authorization: Bearer <token>
```

| Параметр | Тип | Обязательный | Описание |
|----------|-----|:---:|----------|
| `organization_id` | int64 | да | ID организации |
| `date` | string | да | Дата, `YYYY-MM-DD` |

### Response — 200 OK

```json
{
  "status": "Deleted"
}
```

### Ошибки

| Код | Когда |
|-----|-------|
| 400 | Отсутствует параметр, неверный формат |
| 403 | Нет доступа к организации |
| 500 | Ошибка БД |

Если данных на эту дату не было — всё равно возвращает 200 (идемпотентный DELETE).

---

## 4. GET /data — Данные сравнения всех организаций

Возвращает данные ручного сравнения **всех организаций** в формате, совместимом с существующим UI сравнения фильтрации (`OrgComparisonV2`). Организации без введённых данных не включаются.

### Request

```
GET /manual-comparison/data?date=2026-03-25
Authorization: Bearer <token>
```

| Параметр | Тип | Обязательный | Описание |
|----------|-----|:---:|----------|
| `date` | string | да | Дата, `YYYY-MM-DD` |

### Response — 200 OK

Массив объектов `OrgComparisonV2` (тот же формат что у `GET /filtration/comparison/data`):

```json
[
  {
    "organization_id": 1,
    "organization_name": "Чорвок",
    "current": {
      "date": "2026-03-25",
      "level": 897.5,
      "volume": 1250.3,
      "locations": [
        {
          "id": 10,
          "organization_id": 1,
          "name": "Левый берег",
          "norm": 15.0,
          "sort_order": 1,
          "created_at": "...",
          "updated_at": "...",
          "flow_rate": 12.5
        }
      ],
      "piezometers": [
        {
          "id": 20,
          "organization_id": 1,
          "name": "ПК-1",
          "norm": 50.0,
          "sort_order": 1,
          "created_at": "...",
          "updated_at": "...",
          "level": 45.67,
          "anomaly": false
        }
      ],
      "piezometer_counts": {
        "pressure": 5,
        "non_pressure": 3
      }
    },
    "historical_filter": {
      "date": "2025-09-15",
      "level": null,
      "volume": null,
      "locations": [
        {
          "id": 10,
          "name": "Левый берег",
          "norm": 15.0,
          "sort_order": 1,
          "flow_rate": 11.8
        }
      ],
      "piezometers": [
        {
          "id": 20,
          "name": "ПК-1",
          "norm": 50.0,
          "sort_order": 1,
          "level": 44.12,
          "anomaly": false
        }
      ],
      "piezometer_counts": {
        "pressure": 5,
        "non_pressure": 3
      }
    },
    "historical_piezo": {
      "date": "2025-09-15",
      "level": null,
      "volume": null,
      "locations": [...],
      "piezometers": [...],
      "piezometer_counts": {...}
    }
  }
]
```

### Поля ответа

| Поле | Тип | Описание |
|------|-----|----------|
| `organization_id` | int64 | ID организации |
| `organization_name` | string | Название |
| `current` | ComparisonSnapshot | Текущие измерения |
| `current.date` | string | Дата текущих измерений |
| `current.level` | float64? | Уровень водохранилища на эту дату (из `reservoir_data`) |
| `current.volume` | float64? | Объём водохранилища на эту дату |
| `current.locations` | array | Места фильтрации с текущими расходами |
| `current.piezometers` | array | Пьезометры с текущими уровнями |
| `current.piezometer_counts` | object | Количество пьезометров (напорных/безнапорных) |
| `historical_filter` | ComparisonSnapshot? | Исторический снимок фильтрации. `null` если `historical_filter_date` не задана |
| `historical_filter.date` | string | Текстовая метка исторической даты |
| `historical_filter.level` | null | Всегда `null` (данные ручные, уровень водохранилища неизвестен) |
| `historical_filter.volume` | null | Всегда `null` |
| `historical_filter.locations[].flow_rate` | float64? | Вручную введённый исторический расход |
| `historical_piezo` | ComparisonSnapshot? | Исторический снимок пьезометров. `null` если `historical_piezo_date` не задана |
| `historical_piezo.piezometers[].level` | float64? | Вручную введённый исторический уровень |

### Логика доступа

- Роли `sc` / `rais` — видят все организации
- Роль `reservoir` — только свою организацию

---

## 5. GET /export — Экспорт в Excel/PDF

Генерирует Excel или PDF файл с таблицей сравнения фильтрации. Использует тот же шаблон и генератор, что и `/filter/export`.

### Request

```
GET /manual-comparison/export?date=2026-03-26&format=excel
Authorization: Bearer <token>
```

| Параметр | Тип | Обязательный | Описание |
|----------|-----|:---:|----------|
| `date` | string | да | Дата сводки (данные фильтрации берутся за `date - 1 день`) |
| `format` | string | нет | `excel` (по умолчанию) или `pdf` |

### Response

- **Excel**: `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, файл `ManualComparison-YYYY-MM-DD.xlsx`
- **PDF**: `Content-Type: application/pdf`, файл `ManualComparison-YYYY-MM-DD.pdf`

### Важно

- `date` — это дата **сводки** (как в существующем `/filter/export`). Данные фильтрации берутся за **предыдущий день** (`date - 1`).
- Файл содержит: сводку водохранилищ (Section 1) + блоки фильтрации/пьезометров (Section 2).
- Организации без введённых данных не включаются.

---

## Отличие от существующего модуля `/filtration`

| Аспект | `/filtration` (существующий) | `/manual-comparison` (новый) |
|--------|------------------------------|------------------------------|
| Историческая дата | Выбирается из `similar-dates` (точное совпадение уровня) | Вводится вручную (свободный текст) |
| Исторические данные | Реальные измерения из БД за выбранную дату | Вводятся вручную пользователем |
| Хранение | `filtration_measurements.comparison_date` FK | Отдельные таблицы `manual_comparison_*` |
| Когда использовать | Есть историческая дата с таким же уровнем | Нет подходящей исторической даты |
| Места фильтрации / Пьезометры | Общие (одни и те же `filtration_locations`, `piezometers`) | Общие |
| Excel-генератор | `FillFiltrationBlocks` | Тот же самый |

---

## Типичный сценарий использования

1. Фронтенд загружает список мест фильтрации и пьезометров: `GET /filtration/locations?organization_id=1` и `GET /filtration/piezometers?organization_id=1`
2. Фронтенд загружает существующие данные (если есть): `GET /manual-comparison/measurements?organization_id=1&date=2026-03-25`
3. Пользователь заполняет форму: текущие значения, исторические значения, исторические даты
4. Фронтенд сохраняет: `POST /manual-comparison/measurements`
5. Для просмотра сравнения всех оргов: `GET /manual-comparison/data?date=2026-03-25`
6. Для экспорта: `GET /manual-comparison/export?date=2026-03-26&format=excel`
