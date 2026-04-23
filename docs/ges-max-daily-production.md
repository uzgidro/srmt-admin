# GES Max Daily Production — Frontend Implementation Guide

Документ описывает что нужно изменить на фронте, чтобы реализовать новый функционал «максимальная суточная выработка» в `ges_config` с автоматическим ограничением ввода фактической выработки в форме `daily_data`.

## Бизнес-контекст

Каждой ГЭС (`ges_config`) теперь можно задать **верхнюю границу** суточной выработки в млн кВт·ч. Когда оператор вводит фактическую `daily_production_mln_kwh` в форме отчёта за день, значение не должно превышать этот потолок.

- Если потолок не задан (значение `0` — состояние по умолчанию для ещё не сконфигурированных станций) → ограничения нет, ввод любого числа.
- Если потолок задан (`> 0`) → ввод бо́льшего числа отклоняется и на клиенте, и на сервере.

Backend уже выкачен (миграция `000073`, валидаторы на `POST /ges-report/config` и `POST /ges-report/daily-data`). Дальше — задача фронта.

## TL;DR — что фронту нужно сделать

| # | Где | Что | Кто видит |
| --- | --- | --- | --- |
| 1 | Модель `Config` (TS interface) | Добавить поле `maxDailyProductionMlnKwh: number` | — |
| 2 | Форма редактирования конфига ГЭС | Добавить numeric-input «Макс. суточная выработка» | sc, rais |
| 3 | Форма ввода daily_data (таблица отчёта) | На колонке «Выработка»: `<input max>` динамически из `cfg.maxDailyProductionMlnKwh`, инлайн-валидация | sc, rais, cascade |
| 4 | Обработчик ошибок при `POST /ges-report/daily-data` | Распарсить новый `400` и подсветить виновную строку | sc, rais, cascade |
| 5 | Документация компонента / Storybook | Обновить пример пропсов, если форма реюзается | dev |

Никаких новых HTTP-запросов, никаких новых эндпоинтов. Backend отдаёт поле в существующий `getConfigs()` (волна 1 в `loadData()`).

## Шаг 1. Расширить TS-модель `Config`

Скорее всего интерфейс уже есть в `models/ges-config.ts` (или похожий путь). Добавить поле:

```ts
export interface GesConfig {
  id: number;
  organizationId: number;
  organizationName: string;
  cascadeId: number | null;
  cascadeName: string | null;
  installedCapacityMwt: number;
  totalAggregates: number;
  hasReservoir: boolean;
  sortOrder: number;
  maxDailyProductionMlnKwh: number;  // ← NEW. Always present, default 0.
}
```

Если сервис маппит snake_case → camelCase автоматически (через interceptor / `httpClientWithCase`), достаточно добавить только поле в интерфейс. Если маппинг ручной — добавить строку в маппере: `maxDailyProductionMlnKwh: dto.max_daily_production_mln_kwh ?? 0`.

**Безопасный fallback на rollout-окно:** до того как backend выкатится в нужное окружение, поле может отсутствовать. Используй `?? 0`:

```ts
maxDailyProductionMlnKwh: dto.max_daily_production_mln_kwh ?? 0,
```

После выкатки бэка поле приходит всегда (backend пишет без `omitempty`).

## Шаг 2. Форма редактирования `ges_config` (sc, rais)

UI-элемент — обычный numeric input рядом с `installed_capacity_mwt`, `total_aggregates`, `has_reservoir`, `sort_order`. Доступ к этой форме у вас уже только под `sc`/`rais` — не меняется.

ASCII-мокап секции формы:

```
┌─ Конфигурация ГЭС ──────────────────────────────────┐
│                                                     │
│ Организация:        [ Зомин микроГЭС-1,2     ▾ ]    │
│ Установленная мощность (МВт): [ 50.0       ]        │
│ Кол-во агрегатов:             [ 4          ]        │
│ Есть водохранилище:           [ ☑ ]                 │
│ Порядок сортировки:           [ 1          ]        │
│ Макс. суточная выработка      [ 12.5       ] млн    │  ← NEW
│  (0 = без ограничения)                       кВт·ч  │
│                                                     │
│            [ Отмена ]   [ Сохранить ]               │
└─────────────────────────────────────────────────────┘
```

Если используется Reactive Forms (Angular):

```ts
this.form = this.fb.group({
  organizationId:           [null,  [Validators.required]],
  installedCapacityMwt:     [0,     [Validators.min(0)]],
  totalAggregates:          [0,     [Validators.min(0)]],
  hasReservoir:             [false],
  sortOrder:                [0,     [Validators.min(0)]],
  maxDailyProductionMlnKwh: [0,     [Validators.min(0)]],   // ← NEW
});
```

В шаблоне:

```html
<label>
  Макс. суточная выработка, млн кВт·ч
  <input type="number"
         min="0"
         step="0.01"
         formControlName="maxDailyProductionMlnKwh">
  <small>0 = без ограничения</small>
</label>
```

При сохранении тело `POST /ges-report/config` уже содержит `max_daily_production_mln_kwh: <число>`. Серверная валидация: `validate:"gte=0"` + DB CHECK `>= 0`. Если как-то прошло отрицательное значение — придёт `400` с понятным сообщением (см. секцию «Обработка ошибок»).

## Шаг 3. Форма ввода `daily_data` (sc, rais, cascade)

Это табличная форма редактирования суточных данных. По одной строке на ГЭС, видимых в волне 2 forkJoin'а. Колонка «Выработка, млн кВт·ч» (`daily_production_mln_kwh`) — целевая.

### Что меняется визуально

```
№   ГЭС                Выработка, млн кВт·ч        Аггрегаты ...
────────────────────────────────────────────────────────────────
1   Зомин микроГЭС-1,2 [ 12.30 ] / 12.5 макс       [ 4 ] ...
2   Чарвакская ГЭС     [ 950.0 ]                   [ 8 ] ...
3   Туполанг ГЭС       [ 5.0  ] ⚠ макс 4.0          [ 2 ] ...
                            └── красная подсветка + tooltip
```

- Если у станции `cap > 0` → справа от инпута показать `/ {cap} макс` (или подобное), на инпут навесить `max="{cap}"`.
- Если `cap === 0` → ничего не показывать, никакой клиентской валидации.
- Превышение → красная подсветка ячейки + сообщение.

### Реализация (RxJS, Angular)

В `loadData()` после волны 1 у вас уже есть `Map<organizationId, GesConfig>` (или массив). Когда строится строка таблицы для станции:

```ts
const cfg = this.configsByOrgId.get(row.organizationId);
const cap = cfg?.maxDailyProductionMlnKwh ?? 0;

const productionControl = this.fb.control(
  row.dailyProductionMlnKwh,
  cap > 0
    ? [Validators.min(0), Validators.max(cap)]
    : [Validators.min(0)],
);
```

В шаблоне (упрощённо):

```html
<td>
  <input type="number"
         min="0"
         step="0.001"
         [attr.max]="cap > 0 ? cap : null"
         [formControl]="productionControl"
         [class.is-invalid]="productionControl.errors?.['max']">

  <span *ngIf="cap > 0" class="cap-hint">
    / {{ cap | number:'1.0-3' }} макс
  </span>

  <small *ngIf="productionControl.errors?.['max']" class="text-danger">
    Не более {{ cap | number:'1.0-3' }} млн кВт·ч
  </small>
</td>
```

При `saveRow()` / `saveAll()` если форма `invalid` — не отправлять, показать пользователю что есть проблемные ячейки. Если как-то отправили (race / programmatic) — backend отобьёт `400` (см. шаг 4).

### Семантика на бэке (важно для понимания)

Backend проверяет **эффективное** значение:

| Что в payload'е | Что бэк сравнивает с cap |
| --- | --- |
| `daily_production_mln_kwh: 10.0` | 10.0 |
| Поле опущено (partial update) | текущее значение из БД |
| Поле явно `null` | 0 |

Это значит: **partial-update без поля не обходит проверку** — если в БД уже лежит 100 при cap=50, любой PATCH без `daily_production_mln_kwh` вернёт `400`. Фронту обычно это не страшно, потому что вы либо шлёте полный набор полей, либо реактивная форма не позволяет «забыть» поле. Но если ваш `saveRow()` шлёт только dirty-поля, помни про эту семантику.

## Шаг 4. Обработка `400` от `POST /ges-report/daily-data`

Бэк возвращает:

```json
{
  "status": "Error",
  "error": "daily_production_mln_kwh exceeds max for organization_id=10: 13.0 > 12.5"
}
```

HTTP `400`. **Вся batch-операция отменяется** (атомарный upsert). Если в payload'е было 5 строк и одна нарушила cap — ни одна не сохранилась.

Парсинг ошибки и подсветка:

```ts
const RX_CAP = /organization_id=(\d+):\s*([\d.]+)\s*>\s*([\d.]+)/;

this.api.upsertDailyData(payload).subscribe({
  next: () => this.toast.success('Сохранено'),
  error: (err: HttpErrorResponse) => {
    if (err.status === 400 && err.error?.error?.startsWith('daily_production_mln_kwh exceeds max')) {
      const m = RX_CAP.exec(err.error.error);
      if (m) {
        const orgId = +m[1];
        const got = +m[2];
        const cap = +m[3];
        this.highlightRow(orgId, `Выработка ${got} превышает максимум ${cap}`);
        return;
      }
    }
    this.toast.error('Ошибка сохранения: ' + (err.error?.error ?? err.message));
  },
});
```

Если оператор нажимает «Сохранить всё» и одна строка побила cap — не повторяй с теми же данными, дождись правки. Альтернатива (если UX это позволяет) — разбить batch и переотправить только валидные строки.

## Шаг 5. Тесты на фронте

Минимум, что стоит добавить:

- **Unit на маппер**: `dto.max_daily_production_mln_kwh: 12.5` → `model.maxDailyProductionMlnKwh: 12.5`. И `undefined` → `0`.
- **Unit на форму config**: `Validators.min(0)` отбивает `-1`; `0` валидно; положительное валидно.
- **Unit на форму daily_data**: с `cap=10` и значением `15` — control becomes `invalid` с ошибкой `max`. С `cap=0` — control valid при любом значении.
- **Integration / e2e (если есть)**: создать конфиг с `cap=5`, попытаться ввести `7` в daily_data, проверить что Submit заблокирован, или (если удалось послать) показывается toast с распарсенной ошибкой бэка.

## Что **не** меняется

- Структура forkJoin'а (волна 1 / волна 2) — то же количество запросов.
- Эндпоинты `/cascade-config`, `/ges-report?date=...`, `/ges-report/daily-data?organization_id=...&date=...`, `/ges-report/export?date=...&format=...` — без изменений.
- Авторизация / `auth.interceptor.ts` — без изменений. Bearer-JWT по-прежнему.
- Excel/PDF экспорт — никак не зависит от cap'а.

## Чеклист готовности

- [ ] TS-интерфейс `GesConfig` содержит `maxDailyProductionMlnKwh: number`.
- [ ] Маппер DTO→модель ставит `?? 0` для backwards-compat.
- [ ] Форма редактирования config: новый input с `Validators.min(0)`.
- [ ] Сабмит конфига шлёт snake_case-поле в payload.
- [ ] Форма daily_data: динамический `Validators.max(cap)` когда `cap > 0`.
- [ ] Подсказка «/ N макс» рядом с инпутом выработки.
- [ ] Сабмит daily_data блокируется при `form.invalid`.
- [ ] Обработчик `400` распознаёт новое сообщение и подсвечивает строку.
- [ ] Юнит-тесты на маппер / валидаторы.
- [ ] Smoke на dev-стенде: создать cap=10 → ввод 11 заблокирован, ввод 9 проходит, cap=0 ничем не ограничивает.

## Backend cross-references (если нужно)

- API spec: [`GES_DAILY_REPORT_API.md`](GES_DAILY_REPORT_API.md) — таблица схемы + примеры payload/response.
- Migration: [`migrations/postgres/000073_ges_config_max_daily_production.up.sql`](../migrations/postgres/000073_ges_config_max_daily_production.up.sql).
- Model: [`internal/lib/model/ges-report/model.go`](../internal/lib/model/ges-report/model.go) — `Config`, `UpsertConfigRequest`.
- Repo: [`internal/storage/repo/ges_report.go`](../internal/storage/repo/ges_report.go) — `UpsertGESConfig`, `GetAllGESConfigs`, `GetGESConfigsMaxDailyProduction`, `GetGESDailyProductionsBatch`.
- Handler upsert: [`internal/http-server/handlers/ges-report/config.go`](../internal/http-server/handlers/ges-report/config.go).
- Handler validation: [`internal/http-server/handlers/ges-report/daily_data.go`](../internal/http-server/handlers/ges-report/daily_data.go) — функция `validateProductionCap`.
