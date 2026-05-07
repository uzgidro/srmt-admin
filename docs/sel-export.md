# Sel Export — «Тезкор маълумот» по водохранилищам в период паводков

Excel/PDF отчёт почасового состояния включённых в `reservoir_flood_config` водохранилищ. Подаёт сравнение «час−1 / час» по каждому параметру (уровень, объём, келиш, чиқиш, ГЭС орқали, мощность, салт ташлама), плюс погоду, температуру и ФИО дежурного.

## Endpoint

```
GET /reservoir-flood/export?date=YYYY-MM-DD&hour=HH&format=excel|pdf
```

| Параметр | Обязательный | Default | Описание |
|---|---|---|---|
| `date` | да | — | Дата отчёта (YYYY-MM-DD) в Asia/Tashkent. |
| `hour` | нет | `0` | Час среза (0..23). При `hour=0` «час−1» = 23:00 предыдущей даты. |
| `format` | нет | `excel` | `excel` или `pdf`. |

**Доступ**: `sc` + `rais`. Роль `reservoir_duty` к этому endpoint'у не имеет доступа (она ограничена одной организацией, а отчёт — сводный по всем включённым).

**Имя файла**: `ТЕЗКОР-МАЪЛУМОТ-YYYY-MM-DD-HH.{xlsx|pdf}`.

**Рабочее окно**: 21..08 (вечер–утро, через полночь). Endpoint технически принимает любой час; запрос с `hour ∉ {21..23, 0..8}` обрабатывается, но в лог пишется WARN — это сигнал, что запрос вне обычного режима.

## Источник данных

Отчёт читает только две таблицы:

1. `reservoir_flood_config` — список включённых организаций (`is_active=TRUE`) и порядок (`sort_order`).
2. `reservoir_flood_hourly` — две точки (час−1, час) на каждую включённую организацию.

Если для текущего часа записи нет — соответствующие ячейки в Excel показывают `"-"`. Если нет записи на час−1 — то же самое. Дельты в строке 7 (и аналогах для остальных блоков) автоматически становятся `"-"` благодаря формуле `=IFERROR(E6-D6, "-")` в шаблоне.

## Маппинг колонок шаблона (`template/sel.xlsx`)

Данные физически лежат в **B..T**. Колонки **A** (слева) и **U** (справа) — пустой padding, добавлены для надёжности fit-to-page при PDF-конвертации (см. ниже). Шаблон содержит один эталонный 2-рядный блок (ряды 6-7), который генератор клонирует под каждый резервуар. Подпись (F9:K9 хардкод + N9:R9 для имени оператора) автоматически уезжает вниз.

| Колонка | Header (ряд 4) | Значения ряд 6 (час−1 / час) | Источник из БД |
|---|---|---|---|
| A | — | пусто (left padding) | — |
| B | Т/р | порядковый номер | вычисляется генератором (1, 2, 3, …) |
| C | Сув омборлар номи | имя резервуара | `organizations.name` целиком |
| D/E | сатҳ, м +/− | level prev / curr | `reservoir_flood_hourly.water_level_m` |
| F/G | ҳажм, мlн.м³ +/− | volume prev / curr | `reservoir_flood_hourly.water_volume_mln_m3` |
| H/I | келиш, м³/с +/− | inflow prev / curr | `reservoir_flood_hourly.inflow_m3s` |
| J/K | чиқиш, м³/с +/− | outflow prev / curr | `reservoir_flood_hourly.outflow_m3s` |
| L/M | ГЭС орқали +/− | ges flow prev / curr | `reservoir_flood_hourly.ges_flow_m3s` |
| N/O | Қуввати +/− | capacity prev / curr | `reservoir_flood_hourly.capacity_mwt` |
| P/Q | салт ташлама +/− | idle discharge prev / curr | `reservoir_flood_hourly.idle_discharge_m3s` |
| R | Об-хаво холати | weather (current hour) | `reservoir_flood_hourly.weather_condition` |
| S | харорат | temperature °C (current hour) | `reservoir_flood_hourly.temperature_c` |
| T | Диспетчер Ф.И.Ш | duty (current, fallback prev) | `reservoir_flood_hourly.duty_name` |
| U | — | пусто (right padding) | — |

Ряд 7 (и клонированные `9, 11, …, 23`) — формулы дельты `=IFERROR(E6-D6, "-")` по парам колонок. Excelize при `DuplicateRowTo` копирует формулу буквально (без сдвига ссылок), поэтому генератор после клонирования **перезаписывает** дельту корректным `=IFERROR(E{value_row}-D{value_row}, "-")` для каждого скопированного блока.

**Print_area** перепрошивается генератором динамически в `'<sheet>'!$A$1:$U$<signer_row>` — обе padding-колонки в печати дают soffice достаточный запас, чтобы fit-to-width гарантированно сжимал таблицу до одного landscape-листа на Linux (стало воспроизводимой проблемой когда print_area был ровно по правой границе данных).

## Параметризация по часу

В шаблоне:

- `T2` хранит время — `HH:00`. Генератор пишет `time.Time` с нужным часом.
- `T3` хранит дату.
- Подзаголовки времени D5/F5/H5/J5/L5/N5/P5 — формулы `=MOD($T$2-TIME(1,0,0), 1)` (час−1, корректный wrap через полночь).
- Подзаголовки E5/G5/I5/K5/M5/O5/Q5 — `=$T$2`.

Это значит: запрос `?hour=15` приведёт к `T2 = 15:00`, а нечётные подзаголовки покажут `14:00`, чётные — `15:00`. При `?hour=0` → `T2 = 00:00`, формулы дают `23:00 / 00:00`.

## Отсутствующие данные

Для всех ячеек блока (числовые N/O + R/S/T и пары D/E…P/Q): если значение `nil`/пусто — генератор пишет строку `"-"`. Дельты в ряду 7 благодаря IFERROR тоже становятся `"-"`. Сценарий «нет записи на час−1» полностью покрыт: блок выводится с `"-"` в левых колонках пар, числами в правых, и `"-"` в дельте.

## Расширение списка резервуаров

Если в `reservoir_flood_config` сейчас N включённых организаций, отчёт выведет N блоков. Чтобы добавить водохранилище:

```bash
curl -X POST http://localhost:9010/reservoir-flood/config \
  -H 'Authorization: Bearer $JWT' \
  -H 'Content-Type: application/json' \
  -d '{"organization_id": 96, "sort_order": 1, "is_active": true}'
```

`sort_order` определяет порядок в отчёте.

## Подпись оператора

Внизу отчёта генератор пишет `ShortenName(claims.Name)` (`Иванов Иван Иванович` → `И. Иванов`) в ячейку `N{signer_row}` (top-left of the `N9:R9` merge in the template). `signer_row` сдвигается вниз вместе с количеством блоков: для 9 резервуаров — ряд 25.

## Примеры

```bash
# Сводка за 00:00 (стандартный режим)
curl -H "Authorization: Bearer $JWT" \
  "http://localhost:9010/reservoir-flood/export?date=2026-05-04&format=excel" \
  -o sel.xlsx

# Произвольный час, PDF
curl -H "Authorization: Bearer $JWT" \
  "http://localhost:9010/reservoir-flood/export?date=2026-05-04&hour=15&format=pdf" \
  -o sel-15.pdf
```

## Зависимости

- PDF-конвертация требует наличия `soffice` (LibreOffice headless) на хосте — тот же паттерн, что в остальных Excel/PDF endpoints.
- Шаблон загружается из `template/sel.xlsx`, путь конфигурируется через `Config.TemplatePath`.
