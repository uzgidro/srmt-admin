# Роль `cascade` для отчёта ГЭС — разделение ответственности

## Назначение

Роль `cascade` даёт оператору каскада ГЭС возможность вводить
суточные данные только по станциям своего каскада и видеть отчёт
только за свой каскад. Роли `sc` и `rais` сохраняют полный доступ
ко всем каскадам.

## Создание пользователя каскада

При создании пользователя в БД установить:

- `users.organization_id` = ID организации-каскада (та, которая
  есть в `cascade_config`)
- В `users_roles` назначить роль `cascade`

Пример:

```sql
INSERT INTO users_roles (user_id, role_id)
SELECT 42, id FROM roles WHERE name = 'cascade';
```

## Scope доступа

| Endpoint | sc/rais | cascade |
| --- | --- | --- |
| `GET /ges-report` | Все каскады | Только свой каскад |
| `GET /ges-report/daily-data` | Любая станция | Только свои станции |
| `POST /ges-report/daily-data` | Любые станции | Только свои станции |
| `GET /ges-report/cascade-daily-data` | Любой каскад | Только свой каскад |
| `POST /ges-report/cascade-daily-data` | Любые каскады | Только свой каскад |
| `GET /ges-report/export` | Да | **403 Forbidden** |
| `GET/POST/DELETE /ges-report/config` | Да | **403 Forbidden** |
| `GET/POST /ges-report/plans` | Да | **403 Forbidden** |
| `GET/POST/DELETE /ges-report/cascade-config` | Да | **403 Forbidden** |

## Как работает фильтрация

### GET /ges-report

Если у пользователя роль `cascade` (без `sc`/`rais`), сервис
возвращает `DailyReport` только с одним элементом в `cascades` —
тем, чей `cascade_id` совпадает с `claims.organization_id`.
`grand_total` пересчитывается = `summary` этого каскада.

### POST /ges-report/daily-data

Для каждого `item.organization_id` в массиве проверяется:

- если станция: `parent_organization_id` должна быть равна
  `claims.organization_id` (т.е. станция принадлежит каскаду
  пользователя)
- если сам каскад: `organization_id` равно `claims.organization_id`

Если хоть одна станция чужая — весь запрос отклоняется с **403**.

### POST /ges-report/cascade-daily-data

Каждый `item.organization_id` должен быть равен
`claims.organization_id` (cascade user может править погоду
только своего каскада).

## Пример

Пользователь `dispatcher_x` имеет:

- `organization_id` = 10 (каскад X)
- роль `cascade`

В системе:

- Каскад X (org_id=10) с тремя станциями: org_id 101, 102, 103,
  все с `parent_organization_id=10`
- Каскад Y (org_id=20) с двумя станциями: org_id 201, 202

Что может `dispatcher_x`:

- ✅ `GET /ges-report?date=2026-04-13` → видит только каскад X
- ✅ `POST /ges-report/daily-data` с item.organization_id=101 → 200
- ❌ `POST /ges-report/daily-data` с item.organization_id=201 → 403
- ✅ `POST /ges-report/cascade-daily-data` с
  organization_id=10 → 200
- ❌ `POST /ges-report/cascade-daily-data` с
  organization_id=20 → 403
- ❌ `GET /ges-report/export` → 403
- ❌ `GET /ges-report/config` → 403

## Ошибки

- **403 Forbidden** — попытка доступа к чужой станции/каскаду
  или к sc/rais-only endpoint
- **401 Unauthorized** — нет валидного токена

## Совместимость

- `sc`/`rais` пользователи: поведение без изменений
- Существующие пользователи без роли `cascade`: работают как
  раньше через `CheckOrgAccess` (только своя org для остальных
  ролей)
