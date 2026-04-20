# Агрегаты ГЭС: working / repair / modernization / reserve

Операторы вводят по станции три числа: сколько агрегатов **работает**,
сколько **в ремонте** и сколько **в модернизации**. Четвёртое,
**резерв**, бэкенд считает сам. Сумма первых трёх не должна
превышать общее число агрегатов (`ges_config.total_aggregates`).

## Поля

- **`working_aggregates`** — рабочие агрегаты на станции.
- **`repair_aggregates`** — агрегаты в ремонте.
- **`modernization_aggregates`** — агрегаты в модернизации.
- **`reserve_aggregates`** — резерв. Вычисляется бэкендом,
  фронт **не вводит**.

Все четыре — целые числа `>= 0`. `total_aggregates` задаётся один
раз в `POST /ges-report/config` и используется как верхняя граница.

## Правила

- `reserve = total_aggregates - working - repair - modernization`,
  где `total_aggregates` — из `ges_config` для этой станции.
- Сумма `working + repair + modernization` не должна превышать
  `total_aggregates`. Иначе — **400**.
- Инвариант дублируется на уровне БД триггером
  `ges_daily_data_check_aggregates_trg` — защита от race conditions
  между параллельными апсертами.
- Если `ges_config` для станции ещё нет — проверка суммы
  пропускается (и в handler, и в триггере). Отдельные `CHECK (>= 0)`
  по колонкам остаются.
- Сервис отчёта клампит `reserve` к `0` для исторических / кривых
  данных (с лог-ворнингом и `organization_id`), чтобы UI не показывал
  отрицательный резерв.

### Три состояния полей (`Optional[int]`)

Каждое из трёх вводимых полей — `Optional[int]` с тремя состояниями:

| JSON в запросе | Что происходит |
| --- | --- |
| поле отсутствует | значение в БД не меняется |
| `"repair_aggregates": null` | пишется `0` (столбец `NOT NULL`) |
| `"repair_aggregates": 2` | пишется `2` (должно быть `>= 0`) |

Проверка суммы применяется к **эффективному** состоянию строки:
отсутствующие поля берутся из текущего значения в БД, присутствующие —
из запроса. Подробнее про контракт `Optional[T]`:
[ges-daily-data-partial-update.md](ges-daily-data-partial-update.md).

## API

### POST /ges-report/daily-data

```json
[
  {
    "organization_id": 10,
    "date": "2026-04-13",
    "working_aggregates": 3,
    "repair_aggregates": 1,
    "modernization_aggregates": 0
  }
]
```

Минимальный запрос для смены только «в ремонте»:

```json
[
  { "organization_id": 10, "date": "2026-04-13", "repair_aggregates": 2 }
]
```

Полный контракт эндпоинта — в
[GES_DAILY_REPORT_API.md](GES_DAILY_REPORT_API.md).

### GET /ges-report

Фрагмент ответа (станция, каскад, grand_total):

```json
{
  "cascades": [
    {
      "stations": [
        {
          "organization_id": 10,
          "config": { "total_aggregates": 4 },
          "current": {
            "working_aggregates": 3,
            "repair_aggregates": 1,
            "modernization_aggregates": 0,
            "reserve_aggregates": 0
          }
        }
      ],
      "summary": {
        "total_aggregates": 12,
        "working_aggregates": 8,
        "repair_aggregates": 2,
        "modernization_aggregates": 1,
        "reserve_aggregates": 1
      }
    }
  ],
  "grand_total": {
    "total_aggregates": 180,
    "working_aggregates": 120,
    "repair_aggregates": 8,
    "modernization_aggregates": 3,
    "reserve_aggregates": 49
  }
}
```

`reserve_aggregates` — только чтение. На `POST /daily-data` его
передавать не нужно (и поле игнорируется, если передать).

### GET /ges-report/daily-data

Возвращает сырую строку `ges_daily_data` — здесь есть
`working_aggregates`, `repair_aggregates`, `modernization_aggregates`,
но **нет** `reserve_aggregates` (его знает только сервис отчёта).

## Ошибки

| HTTP | Когда |
| --- | --- |
| 400 | Поле `< 0` (`working_aggregates`, `repair_aggregates` или `modernization_aggregates`) |
| 400 | `working + repair + modernization > total_aggregates` |
| 403 | cascade user пытается записать чужую станцию |

### Пример 400: отрицательное значение

```json
{
  "status": "Error",
  "error": "repair_aggregates must be >= 0 for organization_id=10, got -1"
}
```

### Пример 400: превышение суммы

```json
{
  "status": "Error",
  "error": "aggregates sum exceeds total for organization_id=10: 4+2+1=7 > 6"
}
```

Формат: `working+repair+modernization=sum > total`. Значения в
сообщении — **эффективные** (то есть с учётом текущих значений БД для
полей, не пришедших в запросе).

## Экспорт в Excel

Ячейки *Таъмирдаги* / *Модернизацияда* / *Заҳирадаги* заполняются
из `grand_total.repair_aggregates` / `.modernization_aggregates` /
`.reserve_aggregates`. Query params `repair` и `modernization` у
`GET /ges-report/export` **удалены** (миграция 000071) — см. блок
**Breaking change** в [ges-export.md](ges-export.md).

## Связанное

- [GES_DAILY_REPORT_API.md](GES_DAILY_REPORT_API.md) — полный контракт
  эндпоинтов отчёта ГЭС
- [ges-cascade-role.md](ges-cascade-role.md) — scope доступа для роли
  `cascade`
- [ges-daily-data-partial-update.md](ges-daily-data-partial-update.md) —
  трёхзначная семантика `Optional[T]`
- [ges-export.md](ges-export.md) — экспорт отчёта в Excel/PDF
