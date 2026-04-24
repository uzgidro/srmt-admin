# GES Frozen Defaults — Frontend Implementation Guide

Документ описывает что нужно изменить на фронте, чтобы реализовать новый функционал «замораживание значений полей» в `ges_daily_data` (sticky carry-forward defaults).

## Бизнес-контекст

Часть числовых полей в форме ежедневного отчёта (напор воды, уровень, объём водохранилища, количество агрегатов и т.п.) меняются крайне редко — могут неделями оставаться одинаковыми. Раньше каждый день приходилось вручную вводить их заново, иначе в отчёте они становились `0`/`null`.

**Решение.** Админ «замораживает» поле для конкретной станции (например `water_head_m = 45.0` для org=42). После этого:

- В дни, когда `daily_data` для этой пары `(org, date)` отсутствует ИЛИ nullable-поле в записи `null`, в отчёте подставляется замороженное значение.
- Когда оператор вводит фактическое значение через `POST /ges-report/daily-data` — оно записывается в `ges_daily_data` И **обновляет** `frozen_value` (sync-on-write на backend в одной транзакции). Семантика «sticky latest».
- Размораживание удаляет дефолт.

**Важное исключение** (см. §9): для NOT NULL числовых полей (`daily_production_mln_kwh`, `*_aggregates`) frozen НЕ применяется, если строка `daily_data` существует на эту дату — даже если значение в ней `0`. Это документированное поведение: явный 0 трактуется как «оператор реально ввёл 0», а не «забыл заполнить».

Backend уже выкачен (миграция `000074`, эндпоинты `/ges-report/frozen-defaults`, sync-on-write в `POST /ges-report/daily-data`). Дальше — задача фронта.

## TL;DR — что фронту нужно сделать

| # | Где | Что | Кто видит |
| --- | --- | --- | --- |
| 1 | Модель `FrozenDefault` (TS interface) | Добавить новый interface + helper `buildFrozenMap` | — |
| 2 | `loadData()` — волна 1 | Добавить 4-й параллельный запрос `GET /ges-report/frozen-defaults` | — |
| 3 | Форма ввода daily_data (10 числовых полей) | Иконка-замочек рядом с каждым; placeholder из frozen | sc, rais, cascade |
| 4 | Диалог управления замочком | freeze / update / unfreeze через PUT/DELETE | sc, rais, cascade |
| 5 | Обработчик ошибок (400/403) | Распарсить и показать понятное сообщение | sc, rais, cascade |

Существующие endpoint'ы (`GET /ges-report`, `/config`, `/cascade-config`, `POST /daily-data`, `/export`) **не меняются**.

## Список замораживаемых полей

10 полей `ges_daily_data`. `field_name` — точный ключ API (исходит в payload как есть).

| `field_name` | RU label | Тип | Nullable в БД | Когда применяется frozen |
| --- | --- | --- | --- | --- |
| `daily_production_mln_kwh` | Суточная выработка, млн кВт·ч | float | NOT NULL | только если строки `daily_data` нет на эту дату |
| `working_aggregates` | Работающих агрегатов | int | NOT NULL | только если строки `daily_data` нет |
| `repair_aggregates` | В ремонте | int | NOT NULL | только если строки `daily_data` нет |
| `modernization_aggregates` | На модернизации | int | NOT NULL | только если строки `daily_data` нет |
| `water_level_m` | Уровень воды, м | float | nullable | если значение в БД `null` |
| `water_volume_mln_m3` | Объём водохранилища, млн м³ | float | nullable | если значение в БД `null` |
| `water_head_m` | Напор, м | float | nullable | если значение в БД `null` |
| `reservoir_income_m3s` | Приток в водохранилище, м³/с | float | nullable | если значение в БД `null` |
| `total_outflow_m3s` | Суммарный расход, м³/с | float | nullable | если значение в БД `null` |
| `ges_flow_m3s` | Расход через ГЭС, м³/с | float | nullable | если значение в БД `null` |

Любое другое имя поля backend отклонит с 400.

## API-контракты

### `GET /ges-report/frozen-defaults`

**Roles:** `sc`, `rais`, `cascade`.
**Параметры:** нет.
**Response 200:**

```json
[
  {
    "organization_id": 100,
    "cascade_id": 10,
    "field_name": "water_head_m",
    "frozen_value": 45.0,
    "frozen_at": "2026-04-23T10:30:00Z",
    "updated_at": "2026-04-24T09:15:00Z"
  },
  {
    "organization_id": 100,
    "cascade_id": 10,
    "field_name": "working_aggregates",
    "frozen_value": 3.0,
    "frozen_at": "2026-04-20T14:00:00Z",
    "updated_at": "2026-04-20T14:00:00Z"
  }
]
```

`cascade_id` — id родительской организации (каскада); может быть `null` если станция не привязана к каскаду. Используется backend'ом для фильтрации по cascade-роли (фронту, как правило, читать его не нужно — фильтрация происходит на сервере).

**Видимость:**
- `sc`/`rais` — все записи.
- `cascade` (org_id = X) — записи, где `organization_id == X` ИЛИ `cascade_id == X`.

### `PUT /ges-report/frozen-defaults`

**Roles:** `sc`, `rais`, `cascade`.
**Body:**

```json
{
  "organization_id": 100,
  "field_name": "water_head_m",
  "frozen_value": 45.0
}
```

**Валидация:**
- `organization_id` — required, int64.
- `field_name` — required, один из 10 имён выше.
- `frozen_value` — required, `>= 0`.
- Для `working_aggregates`, `repair_aggregates`, `modernization_aggregates` — дополнительно: `frozen_value` должно быть **целым** (3.0 OK, 3.7 → 400).

**Response 200:** `{"status":"OK"}`.

**Errors:**

| Code | Когда | Что показать |
| --- | --- | --- |
| 400 | invalid `field_name` / отрицательное значение / non-integer для агрегатов / невалидный JSON | «Неверные данные: <details>» |
| 401 | нет/протух токен | «Сессия истекла, перелогиньтесь» |
| 403 | cascade-юзер пытается заморозить чужую станцию | «Нет прав на заморозку этой станции» |
| 500 | ошибка БД | «Ошибка сервера, попробуйте позже» |

### `DELETE /ges-report/frozen-defaults`

**Roles:** `sc`, `rais`, `cascade`.
**Body:**

```json
{
  "organization_id": 100,
  "field_name": "water_head_m"
}
```

Идемпотентно: даже если записи не было — возвращает успех (204). Это упрощает UI flow «снять заморозку» — не нужно знать состояние заранее.

**Response 204** No Content (тело может содержать `{"status":"deleted"}`, фронт принимает оба).

**Errors:** 400 / 401 / 403 / 500 — те же что у PUT.

### `POST /ges-report/daily-data` — без изменений API, новый side-effect

Если в payload поле X для org Y присутствует с не-`null` значением, и для пары `(Y, X)` уже есть запись в `ges_frozen_defaults` — `frozen_value` автоматически обновится до этого значения в той же транзакции. **Фронту ничего делать не нужно**, sync прозрачен.

Если frozen-записи для `(Y, X)` ещё нет — sync — no-op (новые frozen через POST `/daily-data` не создаются, только через PUT `/frozen-defaults`).

## Интеграция в существующий loadData flow

Сегодня страница ges-report делает 2 параллельные волны:

- **Волна 1** (одновременно): `GET /ges-report/config`, `GET /ges-report/cascade-config`, `GET /ges-report?date=...`.
- **Волна 2** (одновременно по N станциям): `GET /ges-report/daily-data?organization_id=...&date=...`.

Изменения:

- В **волну 1** добавить 4-й параллельный запрос — `GET /ges-report/frozen-defaults`. Сохранить ответ в стейт через helper `buildFrozenMap` (см. §TS-модель).
- В **волну 2** изменений нет.
- На сохранении (`POST /ges-report/daily-data`) изменений нет — sync прозрачен.
- На экспорте (`GET /ges-report/export`) изменений нет.

Псевдокод (RxJS-стиль):

```typescript
loadData(date: string) {
  // Волна 1
  forkJoin({
    configs: gesApi.getConfigs(),
    cascadeConfigs: gesApi.getCascadeConfigs(),
    report: gesApi.getReport(date),
    frozen: gesApi.listFrozenDefaults(),  // ← новое
  }).subscribe(({configs, cascadeConfigs, report, frozen}) => {
    state.configs = configs;
    state.cascadeConfigs = cascadeConfigs;
    state.report = report;
    state.frozenMap = buildFrozenMap(frozen);  // ← O(1) lookup в форме

    // Волна 2 — как раньше
    forkJoin(orgIds.map(id => gesApi.getDailyData(id, date)))...
  });
}
```

## UX-паттерн замочка

Иконка-замочек рисуется **рядом с каждым из 10 числовых полей** в форме редактирования daily_data.

**Состояния иконки:**

| Состояние | Иконка | Tooltip |
| --- | --- | --- |
| Не заморожено | пустой/открытый замочек, серый | «Заморозить значение для станции» |
| Заморожено | закрытый замочек, синий | «Заморожено: {frozen_value} {ед.}, обновлено {updated_at}» |

**Клик по замочку:**

- **Если не заморожено** → мини-диалог:
  «Заморозить значение поля **{label}** = **{текущее_введённое}**? Это значение будет автоматически использоваться в дни, когда оно не заполнено.»
  → Кнопка «Заморозить» → `PUT /ges-report/frozen-defaults` → обновление UI.

- **Если заморожено** → диалог с двумя действиями:
  - «Обновить заморозку до текущего значения» → `PUT` с новым `frozen_value`.
  - «Снять заморозку» → `DELETE`.

**Поведение поля при наличии заморозки:**

- Если поле пустое — показать `frozen_value` как **placeholder** (НЕ как `value`!), плюс надпись маленьким шрифтом «(значение из заморозки)».
- При сохранении формы пустое поле НЕ попадает в payload (`Optional.Set=false`), и backend подтянет frozen при построении следующего отчёта.

**Предупреждение для NOT NULL полей** (`daily_production_mln_kwh`, `*_aggregates`):

> Чтобы сработала заморозка, оставьте это поле незаполненным и не сохраняйте строку для этой даты. Если в строке вы введёте 0 явно — заморозка проигнорируется.

Это можно показывать как inline-hint или как тултип над предупреждающим значком.

## Коды ошибок — что показать пользователю

| HTTP code | Условие (распознать по `details`/тексту) | Сообщение |
| --- | --- | --- |
| 400 | `field_name` invalid | «Неверное имя поля для заморозки» |
| 400 | `frozen_value` < 0 | «Значение должно быть ≥ 0» |
| 400 | non-integer для агрегатов | «Для агрегатов значение должно быть целым» |
| 400 | прочее | «Неверные данные» + raw details |
| 401 | — | «Сессия истекла, перелогиньтесь» |
| 403 | — | «Нет прав на заморозку этой станции» |
| 500 | — | «Ошибка сервера, попробуйте позже» |

## TS-модель

```typescript
export type FreezableField =
  | "daily_production_mln_kwh"
  | "working_aggregates"
  | "repair_aggregates"
  | "modernization_aggregates"
  | "water_level_m"
  | "water_volume_mln_m3"
  | "water_head_m"
  | "reservoir_income_m3s"
  | "total_outflow_m3s"
  | "ges_flow_m3s";

export interface FrozenDefault {
  organization_id: number;
  cascade_id: number | null;
  field_name: FreezableField;
  frozen_value: number;
  frozen_at: string;     // ISO 8601
  updated_at: string;
}

// Группировка для O(1) lookup в форме daily-data:
export type FrozenMap = Record<number, Partial<Record<FreezableField, FrozenDefault>>>;
// Использование: frozenMap[orgId]?.[fieldName] — undefined если не заморожено.

export interface UpsertFrozenDefaultRequest {
  organization_id: number;
  field_name: FreezableField;
  frozen_value: number;
}

export interface DeleteFrozenDefaultRequest {
  organization_id: number;
  field_name: FreezableField;
}

// Helper: построить FrozenMap из массива.
export function buildFrozenMap(list: FrozenDefault[]): FrozenMap {
  return list.reduce<FrozenMap>((acc, fd) => {
    if (!acc[fd.organization_id]) acc[fd.organization_id] = {};
    acc[fd.organization_id]![fd.field_name] = fd;
    return acc;
  }, {});
}

// Helper: проверка является ли поле integer-ным (для UX-валидации перед PUT).
export const INTEGER_FREEZABLE_FIELDS: ReadonlySet<FreezableField> = new Set([
  "working_aggregates",
  "repair_aggregates",
  "modernization_aggregates",
]);
```

## End-to-end сценарии

### Сценарий 1 — заморозить напор воды для станции Alpha

1. Юзер открывает форму daily_data, видит water_head_m = 45.0 (только что ввёл).
2. Кликает на замочек рядом с полем → диалог «Заморозить water_head_m = 45.0?» → подтверждает.
3. Фронт: `PUT /ges-report/frozen-defaults` с `{organization_id: 100, field_name: "water_head_m", frozen_value: 45.0}`.
4. Backend: 200 OK. Запись создана.
5. Фронт обновляет локальный `frozenMap`: `frozenMap[100]["water_head_m"] = {...}`. Иконка перекрашивается в синий.

### Сценарий 2 — ввести новое значение поверх заморозки

1. Юзер открывает форму daily_data на 2026-04-25. Поле water_head_m пустое, placeholder показывает 45.0 (frozen).
2. Юзер вводит 46.0 (станция изменила режим), нажимает «Сохранить строку».
3. Фронт: `POST /ges-report/daily-data` с `[{organization_id: 100, date: "2026-04-25", water_head_m: 46.0}]`.
4. Backend: в одной транзакции — INSERT в ges_daily_data + UPDATE ges_frozen_defaults SET frozen_value=46.0. 200 OK.
5. Фронт перезагружает frozenMap (или оптимистично обновляет): `frozenMap[100]["water_head_m"].frozen_value = 46.0`. Tooltip на замочке теперь «Заморожено: 46.0 м, обновлено сегодня».

### Сценарий 3 — снять заморозку

1. Юзер кликает на синий (заморожённый) замочек → диалог с двумя кнопками.
2. Кликает «Снять заморозку».
3. Фронт: `DELETE /ges-report/frozen-defaults` с `{organization_id: 100, field_name: "water_head_m"}`.
4. Backend: 204 No Content.
5. Фронт удаляет ключ из frozenMap: `delete frozenMap[100]["water_head_m"]`. Иконка серая. Placeholder в поле исчезает.

### Сценарий 4 — массовое редактирование с частично замороженными полями

Состояние:
- water_head_m заморожен на 45.0.
- water_level_m заморожен на 12.0.
- working_aggregates НЕ заморожен.

Форма для 2026-04-25 пустая (новая дата). Юзер заполняет только working_aggregates = 4, остальные оставляет пустыми.

1. Фронт: `POST /ges-report/daily-data` с `[{organization_id: 100, date: "2026-04-25", working_aggregates: 4}]`. Поля water_head_m и water_level_m НЕ в payload (Optional.Set=false).
2. Backend: INSERT (working_aggregates=4, water_head_m=NULL, water_level_m=NULL); sync для working_aggregates — no-op (нет frozen-записи); sync для water_head_m / water_level_m — НЕ запускается (т.к. в payload их нет).
3. Следующий `GET /ges-report?date=2026-04-25`: backend подтянет water_head_m=45.0 и water_level_m=12.0 из frozen, working_aggregates=4 из ges_daily_data. В отчёте отображается полный набор.

## Что гарантируется и что нет

| Сценарий | Поведение |
| --- | --- |
| Nullable поле = `null` в БД + frozen существует | Подставляется frozen в отчёт |
| Nullable поле имеет явное значение + frozen существует | Используется явное значение; frozen игнорируется при чтении (но обновляется при записи через POST `/daily-data`) |
| NOT NULL поле, daily_data строки нет (HasRowForDate=false) + frozen существует | Подставляется frozen |
| **NOT NULL поле, daily_data строка есть, поле = 0 + frozen существует** | **НЕ подставляется frozen** — 0 трактуется как явный ввод. Если хотите чтобы сработала заморозка — не создавайте строку для этой даты вовсе |
| Несколько строк daily_data на одну (org, date) | Невозможно — UNIQUE constraint в БД |
| `frozen_value < 0` | Невозможно — `validate:"gte=0"` + DB CHECK |
| Frozen для несуществующего поля | Невозможно — `validate:"oneof=..."` + DB CHECK enum |
| Удаление организации | ges_frozen_defaults записи каскадно удаляются (FK ON DELETE CASCADE) |
| Удаление пользователя, который замораживал | `frozen_by` обнуляется, frozen-запись остаётся (FK ON DELETE SET NULL) |
| Дробное значение для агрегата | Невозможно — handler-валидация + DB не имеет CHECK на это, защищает только server-side проверка |

## Changelog / Совместимость

- 3 новых endpoint'а (`PUT/DELETE/GET /ges-report/frozen-defaults`) — additive, не ломают существующих API.
- `POST /ges-report/daily-data` — то же API, новое побочное действие (sync frozen). Старый фронт продолжит работать без изменений.
- `GET /ges-report` (`BuildDailyReport`) — тот же ответ shape, но nullable поля могут стать non-null благодаря frozen. Фронт уже умеет рендерить non-null значения — никаких регрессий.
- Если фронт не обновляется — функционал будет невидим (нет иконки-замочка), но отчёты автоматически начнут использовать frozen-значения. Это плюс, не баг.

### Чеклист готовности фронта

- [ ] `FrozenDefault` interface добавлен в models.
- [ ] `loadData()` дёргает `GET /frozen-defaults` параллельно в волне 1.
- [ ] Стейт хранит `frozenMap: FrozenMap`.
- [ ] Иконка-замочек отрисована рядом с каждым из 10 полей.
- [ ] Placeholder из frozen для пустых полей.
- [ ] Диалоги freeze / update / unfreeze.
- [ ] Обработчик 400/403 показывает корректное сообщение.
- [ ] Предупреждение про NOT NULL поля (опционально, по UX).
- [ ] Манульное тестирование: 4 сценария из §End-to-end.
