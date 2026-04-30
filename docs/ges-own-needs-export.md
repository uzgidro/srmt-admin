# GET /ges-report/own-needs/export — экспорт ежедневного отчёта по СН/ХН

Эндпоинт генерирует Excel по шаблону `template/own-needs.xlsx` за указанную дату. Под капотом вызывает `BuildOwnNeedsReport(date)` — лёгкая проекция `BuildDailyReport`, дополнительных SQL-запросов нет.

**Доступ:** только роли `sc` и `rais`. Для роли `cascade` — **403 Forbidden** (как у существующего `/ges-report/export`).

## Запрос

```http
GET /ges-report/own-needs/export?date=YYYY-MM-DD
Authorization: Bearer <token>
```

### Query params

| Параметр | Тип | Обязательно | Описание |
| --- | --- | --- | --- |
| `date` | `string` | да | Дата отчёта в формате `YYYY-MM-DD` |

PDF-формат не поддерживается (в отличие от `/ges-report/export`). Если потребуется — добавим отдельной итерацией через тот же путь soffice → pdf.

## Что в файле

Один лист с именем `DD.MM.YY` (например `27.04.26`). Структура:

- **Шапка** (строки 2–5): название организации, дата (H3), год (D4 = "2026 йил"), месяц в подписях C5/D5 ("Ойлик режа (Апрель)", "Режа (Январь-Апрель)").
- **Тело**: для каждого каскада — одна строка-итог (`b`old, агрегаты по детям) и N строк-станций.
- **Grand-total** в последней строке: "«Ўзбекгидроэнерго» АЖ бўйича:" + суммы по всем каскадам.

### Колонки (A..P)

| Колонка | Что | Источник |
|---|---|---|
| A | Название каскада / станции | `organizations.name` |
| B | Установленная мощность, МВт | `ges_config.installed_capacity_mwt` |
| C | Месячный план, млн.кВт·ч | `ges_production_plan` за year+month |
| D | План с начала года (Январь–<месяц>), млн.кВт·ч | сумма `ges_production_plan` за year, month=1..N |
| E | Выработка за день, млн.кВт·ч | `ges_daily_data.daily_production_mln_kwh` |
| F | ∆ за день vs вчера | `today.daily_production - yesterday.daily_production` (nil если нет вчерашних данных) |
| G | С начала месяца (MTD production), млн.кВт·ч | `ProductionAggregation.MTD` |
| H | С начала года (YTD production), млн.кВт·ч | `ProductionAggregation.YTD` |
| I | Расход на СН/ХН за день, кВт·ч | `ges_daily_data.own_consumption_kwh` |
| J | ∆ I vs вчера | `today.own_consumption - yesterday.own_consumption` |
| K | СН/ХН с начала месяца (MTD), кВт·ч | `ProductionAggregation.MTDOwnConsumptionKWh` |
| L | СН/ХН с начала года (YTD), кВт·ч | `ProductionAggregation.YTDOwnConsumptionKWh` |
| M | На 1 кВт за день, Вт·ч | `I / B` (если `B = 0` → пусто) |
| N | ∆ M vs вчера | `J / B` (или пусто при `B = 0`) |
| O | На 1 кВт с начала месяца | `K / B` (или пусто при `B = 0`) |
| P | На 1 кВт с начала года | `L / B` (или пусто при `B = 0`) |

`B` тут — установленная мощность в МВт; деление на МВт даёт значение «Вт·ч на 1 установленный кВт» (так как `kWh / (capacity_kW) = kWh / (capacity_MW × 1000) × 1000 = kWh / capacity_MW`).

### Поведение nil-значений

- Если `own_consumption_kwh` в БД нет (`NULL`) — ячейки I и J оставлены пустыми, M и N тоже.
- Если у станции нет данных за вчера — ячейки F и J пусты (нет дельты).
- Если у станции `installed_capacity_mwt = 0` (например, водохранилища без выработки) — M, N, O, P пусты, чтобы не делить на ноль.

## Filename

`Own-Needs-YYYY-MM-DD.xlsx` (например `Own-Needs-2026-04-27.xlsx`).

## Пример

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://api.example.com/ges-report/own-needs/export?date=2026-04-27" \
  -o Own-Needs-2026-04-27.xlsx
```

Референсный заполненный пример (от заказчика) — `docs/2026-04-30/own-needs-temp.xlsx`. Сгенерированный нашим экспортом файл должен визуально соответствовать ему по составу колонок и порядку каскадов/станций.

## Ответ

- `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `Content-Disposition: attachment; filename="Own-Needs-YYYY-MM-DD.xlsx"`
- Тело: бинарный xlsx

## Ошибки

| HTTP | Причина |
| --- | --- |
| `400 Bad Request` | Отсутствует `date` или невалидный формат |
| `401 Unauthorized` | Нет/невалидный токен |
| `403 Forbidden` | Роль не `sc` и не `rais` |
| `500 Internal Server Error` | Ошибка БД или генерации Excel |

## Что НЕ реализовано (по решению на момент мерджа)

- **Микро-ГЭС сводный блок** — в референсном сэмпле есть отдельная сводка только по микро-ГЭС с прогнозами и фарқ. В текущей версии не воспроизводится; если потребуется — отдельный handler / sheet или флаг `?include_micro_summary=true`.
- **PDF** — только Excel. Конвертация через soffice добавится при необходимости (паттерн уже есть в `/ges-report/export`).
- **Cascade-проекция** — экспорт глобальный по всем каскадам. Отдельный отчёт «по своему каскаду» для роли `cascade` не сделан (контракт продукта — этот отчёт для sc/rais).

## Связанное

- [solar-and-own-consumption.md](solar-and-own-consumption.md) — где и как сохраняется `own_consumption_kwh`
- [ges-export.md](ges-export.md) — основной production-экспорт `/ges-report/export`
- [ges-cascade-role.md](ges-cascade-role.md) — почему `cascade` получает 403
