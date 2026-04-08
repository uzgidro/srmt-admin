# Руководство по миграции загрузки файлов

## Общий принцип

**Было:** Файлы загружались вместе с данными в одном multipart-запросе.
**Стало:** Файлы загружаются отдельно через `POST /upload/files`, получаем ID, передаём в JSON.

Multipart пока работает (обратная совместимость), но новый код должен использовать JSON-подход.

---

## Шаг 1: Загрузка файлов

### `POST /api/v3/upload/files`

Загружает один или несколько файлов. Возвращает их ID.

**Требуется роль:** `sc`

**Request:** `multipart/form-data`

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| `file` | file | Да* | Один файл |
| `files` | file[] | Да* | Несколько файлов |
| `category_id` | int | Да | ID категории файла |
| `date` | string | Нет | Дата в формате YYYY-MM-DD (по умолчанию сегодня) |

*Нужен хотя бы один из `file` или `files`.

**Ответ (один файл):** `201`
```json
{
  "id": 42,
  "uploaded_files": [
    {"id": 42, "file_name": "photo.jpg"}
  ]
}
```

**Ответ (несколько файлов):** `201`
```json
{
  "ids": [42, 43, 44],
  "uploaded_files": [
    {"id": 42, "file_name": "photo1.jpg"},
    {"id": 43, "file_name": "photo2.jpg"},
    {"id": 44, "file_name": "doc.pdf"}
  ]
}
```

---

## Шаг 2: Передача ID файлов в JSON

После загрузки файлов используйте полученные ID в JSON-запросах.

---

## API по модулям

### SC-модуль (сбросы, остановки, инциденты, визиты)

Все эндпоинты уже поддерживают `file_ids` в JSON.

#### Сбросы (Discharges)

**POST /api/v3/discharges**
```json
{
  "organization_id": 1,
  "started_at": "2026-04-08T09:00:00+05:00",
  "flow_rate": 5.5,
  "reason": "Превышение уровня",
  "file_ids": [42, 43]
}
```

**PATCH /api/v3/discharges/{id}**
```json
{
  "reason": "Обновлённая причина",
  "file_ids": [42, 44]
}
```
> `file_ids` заменяет все файлы. `file_ids: []` — удалить все. Без `file_ids` — не трогать.

#### Аварийные отключения (Shutdowns)

**POST /api/v3/shutdowns**
```json
{
  "organization_id": 1,
  "start_time": "2026-04-08T09:00:00+05:00",
  "reason": "Перегрев подшипника",
  "generation_loss": 1500.0,
  "file_ids": [42]
}
```

**PATCH /api/v3/shutdowns/{id}**
```json
{
  "end_time": "2026-04-08T13:00:00+05:00",
  "file_ids": [42, 43]
}
```

#### Инциденты (Incidents)

**POST /api/v3/incidents**
```json
{
  "organization_id": 1,
  "incident_time": "2026-04-08T09:00:00+05:00",
  "description": "Землетрясение 3.5 балла",
  "file_ids": [42]
}
```

**PATCH /api/v3/incidents/{id}**
```json
{
  "description": "Обновлённое описание",
  "file_ids": [42, 43]
}
```

#### Визиты (Visits)

**POST /api/v3/visits**
```json
{
  "organization_id": 1,
  "visit_date": "2026-04-08T10:00:00+05:00",
  "description": "Плановая проверка",
  "responsible_name": "Иванов И.И.",
  "file_ids": [42]
}
```

**PATCH /api/v3/visits/{id}**
```json
{
  "description": "Обновлено",
  "file_ids": [42, 43]
}
```

---

### Инфраструктурные события (Infra Events)

**Только JSON** (multipart не поддерживается).

**POST /api/v3/infra-events**
```json
{
  "category_id": 1,
  "organization_id": 42,
  "occurred_at": "2026-04-08T09:00:00+05:00",
  "description": "Камера оффлайн",
  "remediation": "Специалист направлен",
  "file_ids": [42, 43]
}
```

**PATCH /api/v3/infra-events/{id}**
```json
{
  "restored_at": "2026-04-08T13:00:00+05:00",
  "file_ids": [42, 43]
}
```

**Переоткрытие события:**
```json
{
  "clear_restored_at": true
}
```

---

### Документооборот (Рапорты, Письма, Инструкции, НПА, Приказы, Инвестиции)

Все эндпоинты уже поддерживают `file_ids` в JSON. Паттерн одинаковый.

**POST /api/v3/reports** (аналогично для `/letters`, `/instructions`, `/legal-documents`, `/decrees`, `/investments`)
```json
{
  "name": "Рапорт №123",
  "number": "123",
  "document_date": "2026-04-08",
  "type_id": 1,
  "description": "Описание",
  "file_ids": [42, 43]
}
```

**PATCH /api/v3/reports/{id}**
```json
{
  "description": "Обновлено",
  "file_ids": [42, 44]
}
```

---

### Мероприятия (Events)

**POST /api/v3/events**
```json
{
  "name": "Совещание",
  "event_date": "2026-04-08T14:00:00+05:00",
  "event_type_id": 1,
  "description": "Плановое совещание",
  "file_ids": [42]
}
```

**PATCH /api/v3/events/{id}**
```json
{
  "description": "Обновлено",
  "file_ids": [42, 43]
}
```

---

### Пользователи и Контакты (Users / Contacts)

Для аватарок используется `icon_id` вместо `file_ids`.

#### Контакты

**POST /api/v3/contacts**
```json
{
  "name": "Иванов Иван",
  "phone": "+998901234567",
  "organization_id": 1,
  "icon_id": 42
}
```

**PATCH /api/v3/contacts/{id}**
```json
{
  "phone": "+998907654321",
  "icon_id": 43
}
```

#### Пользователи

**POST /api/v3/users** (с новым контактом)
```json
{
  "login": "ivanov",
  "password": "securepass123",
  "role_ids": [1, 2],
  "contact": {
    "name": "Иванов Иван",
    "phone": "+998901234567",
    "icon_id": 42
  }
}
```

**POST /api/v3/users** (с существующим контактом)
```json
{
  "login": "ivanov",
  "password": "securepass123",
  "role_ids": [1, 2],
  "contact_id": 5
}
```

**PATCH /api/v3/users/{id}**
```json
{
  "password": "newpass456",
  "icon_id": 43
}
```

---

## Логика file_ids в PATCH-запросах

| Что отправлено | Результат |
|----------------|-----------|
| `file_ids` отсутствует | Файлы не трогаются |
| `"file_ids": []` | Все файлы удаляются |
| `"file_ids": [1, 2, 3]` | Файлы заменяются на указанные |

Чтобы **добавить** файл к существующим, фронт должен:
1. Взять текущие file_ids из GET-ответа
2. Добавить новый ID
3. Отправить полный массив

---

## Категории файлов

При загрузке через `POST /upload/files` нужно указать `category_id`. Список категорий:

```
GET /api/v3/files/categories
```

---

## Обратная совместимость

Multipart-загрузка **пока работает** для всех эндпоинтов кроме `infra-events`. После полной миграции фронта multipart будет удалён.
