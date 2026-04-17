# GET /ges-report/export — экспорт ежедневного отчёта ГЭС в Excel/PDF

Эндпоинт генерирует Excel (или PDF) файл по шаблону
`template/ges-prod.xlsx` с данными за указанную дату. Под капотом
вызывает `BuildDailyReport(date)`, заполняет шаблон и отдаёт файл
для скачивания.

**Доступ:** только роли `sc` и `rais`. Для роли `cascade` —
**403 Forbidden** (см. [ges-cascade-role.md](ges-cascade-role.md)).

## Запрос

```http
GET /ges-report/export?date=YYYY-MM-DD&format=excel&modernization=N&repair=N
Authorization: Bearer <token>
```

### Query params

| Параметр | Тип | Обязательно | По умолчанию | Описание |
| --- | --- | --- | --- | --- |
| `date` | `string` | да | — | Дата отчёта в формате `YYYY-MM-DD` |
| `format` | `string` | нет | `excel` | `excel` или `pdf` |
| `modernization` | `int` | нет | `0` | Количество агрегатов в модернизации |
| `repair` | `int` | нет | `0` | Количество агрегатов в ремонте |

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
        - modernization
        - repair
```

Если `reserve < 0` — запрос отклоняется с **400 Bad Request**:
`reserve aggregates cannot be negative`. Проверьте параметры
`modernization` и `repair`.

## Примеры

### Базовый экспорт в Excel

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://api.example.com/ges-report/export?date=2026-04-13" \
  -o GES-2026-04-13.xlsx
```

### С учётом модернизации/ремонта

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://api.example.com/ges-report/export?date=2026-04-13&modernization=4&repair=14" \
  -o GES-2026-04-13.xlsx
```

### PDF

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://api.example.com/ges-report/export?date=2026-04-13&format=pdf&modernization=4&repair=14" \
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
| `400 Bad Request` | Отсутствует `date`, неверный формат даты, неверный `format`, `reserve < 0` |
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

- [ges-cascade-role.md](ges-cascade-role.md) — роль cascade,
  scope доступа
- [ges-daily-data-partial-update.md](ges-daily-data-partial-update.md) —
  ввод данных, partial updates
- [ges-cascade-daily-weather.md](ges-cascade-daily-weather.md) —
  ручная коррекция погоды
- [ges-cascade-weather.md](ges-cascade-weather.md) — автоматический
  сбор погоды (тикер 04:00)
