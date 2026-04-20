# GET /ges-report/export — экспорт ежедневного отчёта ГЭС в Excel/PDF

Эндпоинт генерирует Excel (или PDF) файл по шаблону
`template/ges-prod.xlsx` с данными за указанную дату. Под капотом
вызывает `BuildDailyReport(date)`, заполняет шаблон и отдаёт файл
для скачивания.

**Доступ:** только роли `sc` и `rais`. Для роли `cascade` —
**403 Forbidden** (см. [ges-cascade-role.md](ges-cascade-role.md)).

## Запрос

```http
GET /ges-report/export?date=YYYY-MM-DD&format=excel
Authorization: Bearer <token>
```

### Query params

| Параметр | Тип | Обязательно | По умолчанию | Описание |
| --- | --- | --- | --- | --- |
| `date` | `string` | да | — | Дата отчёта в формате `YYYY-MM-DD` |
| `format` | `string` | нет | `excel` | `excel` или `pdf` |

## Breaking change (миграция 000071)

Query params `modernization` и `repair` **удалены**. Сервер их **игнорирует** — не возвращает ошибку, но и не применяет. Значения для ячеек *Таъмирдаги* и *Модернизацияда* берутся из `ges_daily_data` (см. [ges-aggregates.md](ges-aggregates.md)).

Что должен сделать фронт:

- Убрать `?modernization=…&repair=…` из URL `GET /ges-report/export`.
- Если нужно изменить эти значения — отправлять `POST /ges-report/daily-data` с полями `repair_aggregates` и `modernization_aggregates` на соответствующие станции.

Проверка «резерв >= 0» на уровне handler тоже удалена: резерв вычисляется сервисом и клампится к `0`, инвариант `working + repair + modernization <= total` гарантирован триггером БД + early-валидацией в POST /daily-data.

## Что попадает в файл

- **AH3** — дата отчёта **+1 день** (отчёт строится за `date`,
  но в шапке стоит дата следующего дня — таково требование шаблона)
- **Имя листа** — формат `DD.MM.YY` (например `13.03.26`)
- **Каскады и станции** — данные за `date`, сгруппированы по
  каскадам (порядок из БД)
- **Прогнозы** — годовой план, месячный план, суточный, факт
- **Агрегаты** — кол-во ГЭС по типам, всего/рабочих/резерв/ремонт/модернизация
- **Дельты** — изменения уровня/объёма/расхода/мощности vs вчера
- **Прошлый год** — те же поля за `date - 1 год`
- **Погода** — температура (текстом) и иконка OpenWeatherMap (PNG)
  для каждого каскада

## Расчёт «резерв агрегатов»

```text
reserve = grand_total.total_aggregates
        - grand_total.working_aggregates
        - grand_total.repair_aggregates
        - grand_total.modernization_aggregates
```

Значения всех четырёх слагаемых берутся из `report.GrandTotal.*`,
которое сервис считает по `ges_daily_data` и `ges_config`. Реально
в Excel подставляется уже посчитанное `grand_total.reserve_aggregates`
(сервис клампит его к `0`, если данные операторов некорректны).
Query params на это не влияют — см. блок **Breaking change** выше.

## Примеры

### Базовый экспорт в Excel

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://api.example.com/ges-report/export?date=2026-04-13" \
  -o GES-2026-04-13.xlsx
```

### PDF

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://api.example.com/ges-report/export?date=2026-04-13&format=pdf" \
  -o GES-2026-04-13.pdf
```

PDF конвертируется из Excel через LibreOffice (`soffice`)
на сервере. Если LibreOffice не установлен — **500 Internal Server
Error**.

## Ответ

**Excel:**

- `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `Content-Disposition: attachment; filename="GES-YYYY-MM-DD.xlsx"`
- Тело: бинарный xlsx

**PDF:**

- `Content-Type: application/pdf`
- `Content-Disposition: attachment; filename="GES-YYYY-MM-DD.pdf"`
- Тело: бинарный pdf

## Ошибки

| HTTP | Причина |
| --- | --- |
| `400 Bad Request` | Отсутствует `date`, неверный формат даты, неверный `format` |
| `401 Unauthorized` | Нет/невалидный токен |
| `403 Forbidden` | Роль не `sc` и не `rais` |
| `500 Internal Server Error` | Ошибка БД, ошибка генерации Excel, ошибка LibreOffice (для PDF) |

## Иконки погоды

Иконки OpenWeatherMap встроены в Excel как изображения (PNG).
Файлы лежат в `template/weather-icons/{code}.png`, где `code` —
стандартный код OWM (`01d`, `01n`, `02d` …, `50n` — всего 18
иконок). Источник: <https://openweathermap.org/weather-conditions>.

В каждой ячейке погоды (колонка D и Z в шаблоне):

- Верхняя половина merge — температура (`18°С`)
- Нижняя половина merge — иконка (центрированная)
- Между ними — пунктирная граница

Если в каскаде только одна станция — погода занимает одну ячейку
без иконки (только температура).

## Связанное

- [ges-aggregates.md](ges-aggregates.md) — агрегаты working /
  repair / modernization / reserve: правила и 400-ка
- [ges-cascade-role.md](ges-cascade-role.md) — роль cascade,
  scope доступа
- [ges-daily-data-partial-update.md](ges-daily-data-partial-update.md) —
  ввод данных, partial updates
- [ges-cascade-daily-weather.md](ges-cascade-daily-weather.md) —
  ручная коррекция погоды
- [ges-cascade-weather.md](ges-cascade-weather.md) — автоматический
  сбор погоды (тикер 04:00)
