# Reservoir Summary — пересчёт Volume через level_volume

## Зачем

В сводке по водохранилищам (`/reservoir-summary` и `/reservoir-summary/export`) поле `Volume.Current` исторически могло заполняться значением `size` из внешнего API `static.uz`. Операторы сообщили, что для нескольких водохранилищ `size` не совпадает с реальной кривой `level→volume`. Чтобы устранить расхождение, Volume теперь пересчитывается на сервере из `Level` через справочник `level_volume`, и только если калибровочной таблицы нет — оставляется старое поведение (fallback на `static.size`).

## Порядок применения источников Volume.Current

Для каждой организации в сводке Volume берётся в следующем приоритете:

1. **Из БД** (`reservoir_data.volume_mln_m3`) — если значение в БД ненулевое, ничего не подменяем.
2. **Computed** (`level_volume`) — если `Volume.Current == 0` и есть `Level.Current` (из БД или из static.uz fallback), вызываем `Repo.GetVolumeByLevelByOrg(orgID, level)` с линейной интерполяцией между двумя ближайшими точками. На успехе ставим `Volume.Current = computed`, `Volume.IsEdited = true`.
3. **static.uz `size`** — если в `level_volume` для этой организации нет ни одной строки (`storage.ErrLevelVolumeNotConfigured`), используем `*val.Data.Volume` из ответа `static.uz`. Старое поведение, сохранено для совместимости с водохранилищами без калибровки.

Если `level_volume` для организации есть, но `Level` оказался вне кривой — `storage.ErrLevelOutOfCurveRange`, лог уровня `WARN`, fallback на static (как в шаге 3).

## Где в коде

| Что | Файл |
|---|---|
| Sentinel `ErrLevelVolumeNotConfigured` | `internal/storage/storage.go` |
| `Repo.GetVolumeByLevelByOrg(orgID, level)` (новый, с интерполяцией) | `internal/storage/repo/reservoir.go` |
| `computeVolumeFromLevel`, `applyStaticFallbacks`, интерфейс `volumeByLevelByOrg` | `internal/http-server/handlers/reservoir-summary/volume_compute.go` |
| Использование в GET handler | `internal/http-server/handlers/reservoir-summary/get.go` |
| Использование в Export handler (Excel/PDF) | `internal/http-server/handlers/reservoir-summary/export.go` |
| Регистрация роутов (с прокидыванием `ReservoirFetcher` в Export) | `internal/http-server/router/router.go` |

## Поведение IsEdited

`Volume.IsEdited = true` ставится в любом случае, когда исходное значение в БД было `0` и мы его восстановили — независимо от того, computed или static. UI использует этот флаг, чтобы пометить ячейку как «не введено вручную».

## Покрытие в Excel/PDF экспорте

До этого изменения `/reservoir-summary/export` вообще не вызывал `FetchDataAtDayBegin`, поэтому Excel/PDF мог показать `Volume = 0` там, где UI показывал восстановленное значение. Теперь обе ручки используют общий helper `applyStaticFallbacks`, поэтому отчёт совпадает с экраном.

## Тесты

- `compute_volume_test.go` — unit-тесты на `computeVolumeFromLevel` (OK / not configured / out of range / generic error).
- `get_test.go` — расширен 4 сценариями:
  - `TestGet_VolumeRecomputedFromLevel` — Level в БД → Volume пересчитан.
  - `TestGet_VolumeRecomputedFromStaticLevel` — Level из static.uz → Volume пересчитан, обе ячейки `IsEdited=true`.
  - `TestGet_FallbackToStaticVolumeWhenCurveNotConfigured` — пустая таблица для org → fallback на `static.size`.
  - `TestGet_NoFallbackWhenDBVolumeNonZero` — DB Volume ненулевой, кривая не вызывается.

## Не затронуто

- Старый `Repo.GetVolumeByLevel(resID, level)` (вызывается из `data/set/set.go`) сломан после миграции `000022` (хардкод `WHERE res_id = $1`, а таблица перешла на `organization_id`). Намеренно не чиним в этом изменении — это отдельная задача с другим scope (затрагивает write-путь датчика, а не отчётный read-путь).
- Income / Release / Level fallback из static.uz не меняется — поведение идентично прежнему.

## Ручная проверка

```bash
# Заполнить кривую для тестовой org (96):
psql ... -c "INSERT INTO level_volume (level, volume, organization_id) VALUES (200.0, 100.0, 96), (210.0, 200.0, 96)"

# Сводка — Volume должен быть computed (для level=205 → ≈150, IsEdited=true)
curl "http://localhost:9010/reservoir-summary?date=2026-05-01" | jq '.[] | select(.organization_id == 96)'

# Excel — то же значение в C-колонке
curl "http://localhost:9010/reservoir-summary/export?date=2026-05-01" -o test.xlsx

# Org без level_volume — старое поведение (static.size)
curl "http://localhost:9010/reservoir-summary?date=2026-05-01" | jq '.[] | select(.organization_id == 99)'
```
