# План реализации недостающих HRM API Endpoints

## Резюме

**Цель:** Реализовать 45 недостающих endpoints по API документации

| Модуль | Endpoints | Приоритет |
|--------|-----------|-----------|
| Employee Cabinet (`/my-*`) | 16 | Высокий |
| Personnel Records | 11 | Средний |
| Analytics | 18 | Средний |

**Оценка:** ~45 новых endpoints, ~15 новых файлов

---

## Текущее состояние

### Уже готово к использованию:
- Связь `user_id` ↔ `employee_id` через `hrm_employees.user_id`
- Метод `GetEmployeeByUserID(ctx, userID)` в репозитории
- Все методы для данных: отпуска, зарплаты, обучение, компетенции, уведомления
- JWT authentication через `mwauth.ClaimsFromContext()`

### Не хватает:
- Handlers для `/my-*` endpoints (личный кабинет)
- Personnel Records CRUD endpoints
- Analytics endpoints и отчёты
- Repo методы для аналитики

---

## Фаза 1: Employee Cabinet (16 endpoints)

### 1.1 Создать файлы

```
internal/http-server/handlers/hrm/cabinet/
├── cabinet.go           # Общие интерфейсы и хелперы
├── profile.go           # GET/PATCH /my-profile
├── leave_balance.go     # GET /my-leave-balance
├── vacations.go         # GET/POST /my-vacations, POST /{id}/cancel
├── salary.go            # GET /my-salary, GET /payslip/{id}
├── training.go          # GET /my-training
├── competencies.go      # GET /my-competencies
├── notifications.go     # GET/PATCH/POST /my-notifications
├── tasks.go             # GET /my-tasks
└── documents.go         # GET /my-documents, GET /{id}/download

internal/lib/dto/hrm/cabinet.go  # DTO структуры
```

### 1.2 Endpoints

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/my-profile` | `GetProfile` | Профиль текущего сотрудника |
| PATCH | `/my-profile` | `UpdateProfile` | Обновить phone/email |
| GET | `/my-leave-balance` | `GetLeaveBalance` | Баланс отпусков |
| GET | `/my-vacations` | `GetMyVacations` | Список заявок |
| POST | `/my-vacations` | `CreateMyVacation` | Подать заявку |
| POST | `/my-vacations/{id}/cancel` | `CancelMyVacation` | Отменить заявку |
| GET | `/my-salary` | `GetMySalary` | Информация о зарплате |
| GET | `/my-salary/payslip/{id}` | `GetMyPayslip` | Скачать расчётный лист |
| GET | `/my-training` | `GetMyTraining` | Моё обучение |
| GET | `/my-competencies` | `GetMyCompetencies` | Мои компетенции |
| GET | `/my-notifications` | `GetMyNotifications` | Уведомления |
| PATCH | `/my-notifications/{id}/read` | `MarkRead` | Отметить прочитанным |
| POST | `/my-notifications/read-all` | `MarkAllRead` | Прочитать все |
| GET | `/my-tasks` | `GetMyTasks` | Мои задачи |
| GET | `/my-documents` | `GetMyDocuments` | Мои документы |
| GET | `/my-documents/{id}/download` | `DownloadDocument` | Скачать документ |

### 1.3 Ключевой паттерн

```go
func GetProfile(log *slog.Logger, repo ProfileRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Получить claims из JWT
        claims, ok := mwauth.ClaimsFromContext(r.Context())
        if !ok {
            render.Status(r, http.StatusUnauthorized)
            render.JSON(w, r, resp.Unauthorized("Unauthorized"))
            return
        }

        // 2. Найти employee по user_id
        employee, err := repo.GetEmployeeByUserID(r.Context(), claims.UserID)
        if err != nil {
            if errors.Is(err, storage.ErrNotFound) {
                render.Status(r, http.StatusNotFound)
                render.JSON(w, r, resp.NotFound("Employee profile not found"))
                return
            }
            // handle error
        }

        // 3. Вернуть данные
        render.JSON(w, r, toMyProfileResponse(employee))
    }
}
```

### 1.4 DTO структуры (`cabinet.go`)

```go
// MyProfileResponse - профиль для /my-profile
type MyProfileResponse struct {
    ID              int64   `json:"id"`
    EmployeeNumber  string  `json:"employee_id"`
    FullName        string  `json:"full_name"`
    FirstName       string  `json:"first_name"`
    LastName        string  `json:"last_name"`
    MiddleName      *string `json:"middle_name,omitempty"`
    Photo           *string `json:"photo,omitempty"`
    PositionID      *int64  `json:"position_id,omitempty"`
    PositionName    *string `json:"position_name,omitempty"`
    DepartmentID    *int64  `json:"department_id,omitempty"`
    DepartmentName  *string `json:"department_name,omitempty"`
    Email           *string `json:"email,omitempty"`
    Phone           *string `json:"phone,omitempty"`
    HireDate        string  `json:"hire_date"`
    EmploymentStatus string `json:"employment_status"`
    ManagerID       *int64  `json:"manager_id,omitempty"`
    ManagerName     *string `json:"manager_name,omitempty"`
}

// MyProfileUpdateRequest - обновление профиля
type MyProfileUpdateRequest struct {
    Phone *string `json:"phone" validate:"omitempty,max=50"`
    Email *string `json:"email" validate:"omitempty,email,max=255"`
}

// MyLeaveBalanceResponse - баланс отпусков
type MyLeaveBalanceResponse struct {
    EmployeeID              int64 `json:"employee_id"`
    Year                    int   `json:"year"`
    AnnualLeaveTotal        int   `json:"annual_leave_total"`
    AnnualLeaveUsed         int   `json:"annual_leave_used"`
    AnnualLeaveRemaining    int   `json:"annual_leave_remaining"`
    SickLeaveUsedYear       int   `json:"sick_leave_used_year"`
    // ... остальные поля
}

// MyVacationRequest - создание заявки
type MyVacationRequest struct {
    VacationTypeID int64   `json:"type" validate:"required"`
    StartDate      string  `json:"start_date" validate:"required"`
    EndDate        string  `json:"end_date" validate:"required"`
    Reason         *string `json:"reason"`
    SubstituteID   *int64  `json:"substitute_id"`
}
```

---

## Фаза 2: Personnel Records (11 endpoints)

### 2.1 Архитектура

Personnel Records - это **композитный view** на данные:
- `hrm_employees` (основная информация)
- `hrm_personnel_documents` (документы)
- `hrm_transfers` (история переводов)

**Handlers для documents и transfers УЖЕ ЕСТЬ** - нужно только добавить:

### 2.2 Создать файл

```
internal/http-server/handlers/hrm/personnel-records/
└── personnel_records.go
```

### 2.3 Endpoints

| Method | Path | Handler | Описание |
|--------|------|---------|----------|
| GET | `/hrm/personnel-records` | `GetAll` | Список кадровых записей |
| GET | `/hrm/personnel-records/{id}` | `GetByID` | Запись по ID |
| GET | `/hrm/personnel-records/employee/{empId}` | `GetByEmployeeID` | По ID сотрудника |
| POST | `/hrm/personnel-records` | `Add` | Создать запись |
| PUT | `/hrm/personnel-records/{id}` | `Edit` | Обновить |
| DELETE | `/hrm/personnel-records/{id}` | `Delete` | Удалить |

**Примечание:** Documents и Transfers уже реализованы через `/hrm/employees/{id}/documents` и `/hrm/employees/{id}/transfers`. Можно добавить алиасы если нужно.

---

## Фаза 3: Analytics (18 endpoints)

### 3.1 Создать файлы

```
internal/http-server/handlers/hrm/analytics/
├── dashboard.go         # GET /hrm/analytics/dashboard
├── reports.go           # GET /hrm/analytics/reports/*
└── export.go            # POST /hrm/analytics/export/*

internal/lib/dto/hrm/analytics.go    # DTO структуры
internal/storage/repo/hrm_analytics.go  # Repo методы
```

### 3.2 Endpoints

| Method | Path | Handler |
|--------|------|---------|
| GET | `/hrm/analytics/dashboard` | `GetDashboard` |
| GET | `/hrm/analytics/reports/headcount` | `GetHeadcountReport` |
| GET | `/hrm/analytics/reports/headcount-trend` | `GetHeadcountTrend` |
| GET | `/hrm/analytics/reports/turnover` | `GetTurnoverReport` |
| GET | `/hrm/analytics/reports/turnover-trend` | `GetTurnoverTrend` |
| GET | `/hrm/analytics/reports/attendance` | `GetAttendanceReport` |
| GET | `/hrm/analytics/reports/salary` | `GetSalaryReport` |
| GET | `/hrm/analytics/reports/salary-trend` | `GetSalaryTrend` |
| GET | `/hrm/analytics/reports/performance` | `GetPerformanceReport` |
| GET | `/hrm/analytics/reports/training` | `GetTrainingReport` |
| GET | `/hrm/analytics/reports/demographics` | `GetDemographicsReport` |
| GET | `/hrm/analytics/reports/diversity` | `GetDiversityReport` |
| POST | `/hrm/analytics/reports/custom` | `GenerateCustomReport` |
| POST | `/hrm/analytics/export` | `Export` |
| POST | `/hrm/analytics/export/pdf` | `ExportPDF` |
| POST | `/hrm/analytics/export/excel` | `ExportExcel` |

### 3.3 Repo методы (`hrm_analytics.go`)

```go
// GetHeadcountStats - численность по отделам
func (r *Repo) GetHeadcountStats(ctx context.Context, filter AnalyticsFilter) (*HeadcountStats, error)

// GetTurnoverStats - текучесть кадров
func (r *Repo) GetTurnoverStats(ctx context.Context, filter AnalyticsFilter) (*TurnoverStats, error)

// GetAttendanceStats - посещаемость
func (r *Repo) GetAttendanceStats(ctx context.Context, filter AnalyticsFilter) (*AttendanceStats, error)

// GetSalaryStats - статистика зарплат
func (r *Repo) GetSalaryStats(ctx context.Context, filter AnalyticsFilter) (*SalaryStats, error)

// GetDemographicsStats - демография
func (r *Repo) GetDemographicsStats(ctx context.Context) (*DemographicsStats, error)
```

### 3.4 SQL запросы для аналитики

```sql
-- Численность по отделам
SELECT
    d.id, d.name,
    COUNT(e.id) as employee_count,
    COUNT(CASE WHEN e.employment_status = 'active' THEN 1 END) as active_count
FROM hrm_employees e
JOIN departments d ON e.department_id = d.id
WHERE e.employment_status != 'terminated'
GROUP BY d.id, d.name;

-- Текучесть за период
SELECT
    COUNT(CASE WHEN hire_date BETWEEN $1 AND $2 THEN 1 END) as hired,
    COUNT(CASE WHEN termination_date BETWEEN $1 AND $2 THEN 1 END) as terminated
FROM hrm_employees;

-- Демография
SELECT
    EXTRACT(YEAR FROM AGE(c.birth_date)) as age,
    c.gender,
    COUNT(*) as count
FROM hrm_employees e
JOIN contacts c ON e.contact_id = c.id
WHERE e.employment_status = 'active'
GROUP BY age, c.gender;
```

---

## Фаза 4: Обновление Router

### 4.1 Добавить импорты

```go
import (
    hrmCabinet "srmt-admin/internal/http-server/handlers/hrm/cabinet"
    hrmPersonnelRecords "srmt-admin/internal/http-server/handlers/hrm/personnel-records"
    hrmAnalytics "srmt-admin/internal/http-server/handlers/hrm/analytics"
)
```

### 4.2 Добавить routes

```go
// Employee Cabinet (доступно всем авторизованным)
r.Group(func(r chi.Router) {
    r.Use(mwauth.Authenticator(deps.Token))

    r.Route("/my-profile", func(r chi.Router) {
        r.Get("/", hrmCabinet.GetProfile(deps.Log, deps.PgRepo))
        r.Patch("/", hrmCabinet.UpdateProfile(deps.Log, deps.PgRepo))
    })

    r.Get("/my-leave-balance", hrmCabinet.GetLeaveBalance(deps.Log, deps.PgRepo))

    r.Route("/my-vacations", func(r chi.Router) {
        r.Get("/", hrmCabinet.GetMyVacations(deps.Log, deps.PgRepo))
        r.Post("/", hrmCabinet.CreateMyVacation(deps.Log, deps.PgRepo))
        r.Post("/{id}/cancel", hrmCabinet.CancelMyVacation(deps.Log, deps.PgRepo))
    })

    r.Route("/my-salary", func(r chi.Router) {
        r.Get("/", hrmCabinet.GetMySalary(deps.Log, deps.PgRepo))
        r.Get("/payslip/{id}", hrmCabinet.GetMyPayslip(deps.Log, deps.PgRepo))
    })

    r.Get("/my-training", hrmCabinet.GetMyTraining(deps.Log, deps.PgRepo))
    r.Get("/my-competencies", hrmCabinet.GetMyCompetencies(deps.Log, deps.PgRepo))

    r.Route("/my-notifications", func(r chi.Router) {
        r.Get("/", hrmCabinet.GetMyNotifications(deps.Log, deps.PgRepo))
        r.Patch("/{id}/read", hrmCabinet.MarkNotificationRead(deps.Log, deps.PgRepo))
        r.Post("/read-all", hrmCabinet.MarkAllNotificationsRead(deps.Log, deps.PgRepo))
    })

    r.Get("/my-tasks", hrmCabinet.GetMyTasks(deps.Log, deps.PgRepo))

    r.Route("/my-documents", func(r chi.Router) {
        r.Get("/", hrmCabinet.GetMyDocuments(deps.Log, deps.PgRepo))
        r.Get("/{id}/download", hrmCabinet.DownloadMyDocument(deps.Log, deps.PgRepo))
    })
})

// HRM Analytics (требует роль hrm)
r.Route("/hrm/analytics", func(r chi.Router) {
    r.Get("/dashboard", hrmAnalytics.GetDashboard(deps.Log, deps.PgRepo))

    r.Route("/reports", func(r chi.Router) {
        r.Get("/headcount", hrmAnalytics.GetHeadcountReport(deps.Log, deps.PgRepo))
        r.Get("/headcount-trend", hrmAnalytics.GetHeadcountTrend(deps.Log, deps.PgRepo))
        r.Get("/turnover", hrmAnalytics.GetTurnoverReport(deps.Log, deps.PgRepo))
        r.Get("/turnover-trend", hrmAnalytics.GetTurnoverTrend(deps.Log, deps.PgRepo))
        r.Get("/attendance", hrmAnalytics.GetAttendanceReport(deps.Log, deps.PgRepo))
        r.Get("/salary", hrmAnalytics.GetSalaryReport(deps.Log, deps.PgRepo))
        r.Get("/salary-trend", hrmAnalytics.GetSalaryTrend(deps.Log, deps.PgRepo))
        r.Get("/performance", hrmAnalytics.GetPerformanceReport(deps.Log, deps.PgRepo))
        r.Get("/training", hrmAnalytics.GetTrainingReport(deps.Log, deps.PgRepo))
        r.Get("/demographics", hrmAnalytics.GetDemographicsReport(deps.Log, deps.PgRepo))
        r.Post("/custom", hrmAnalytics.GenerateCustomReport(deps.Log, deps.PgRepo))
    })

    r.Route("/export", func(r chi.Router) {
        r.Post("/pdf", hrmAnalytics.ExportPDF(deps.Log, deps.PgRepo))
        r.Post("/excel", hrmAnalytics.ExportExcel(deps.Log, deps.PgRepo))
    })
})
```

---

## Файлы для модификации

### Новые файлы (создать)

| Файл | Описание |
|------|----------|
| `internal/http-server/handlers/hrm/cabinet/cabinet.go` | Общие интерфейсы |
| `internal/http-server/handlers/hrm/cabinet/profile.go` | Profile handlers |
| `internal/http-server/handlers/hrm/cabinet/leave_balance.go` | Leave balance |
| `internal/http-server/handlers/hrm/cabinet/vacations.go` | Vacations |
| `internal/http-server/handlers/hrm/cabinet/salary.go` | Salary |
| `internal/http-server/handlers/hrm/cabinet/training.go` | Training |
| `internal/http-server/handlers/hrm/cabinet/competencies.go` | Competencies |
| `internal/http-server/handlers/hrm/cabinet/notifications.go` | Notifications |
| `internal/http-server/handlers/hrm/cabinet/tasks.go` | Tasks |
| `internal/http-server/handlers/hrm/cabinet/documents.go` | Documents |
| `internal/http-server/handlers/hrm/personnel-records/personnel_records.go` | Personnel Records |
| `internal/http-server/handlers/hrm/analytics/dashboard.go` | Analytics Dashboard |
| `internal/http-server/handlers/hrm/analytics/reports.go` | Reports |
| `internal/http-server/handlers/hrm/analytics/export.go` | Export |
| `internal/lib/dto/hrm/cabinet.go` | Cabinet DTOs |
| `internal/lib/dto/hrm/analytics.go` | Analytics DTOs |
| `internal/storage/repo/hrm_analytics.go` | Analytics repo methods |

### Существующие файлы (изменить)

| Файл | Изменения |
|------|-----------|
| `internal/http-server/router/router.go` | Добавить routes для /my-* и /hrm/analytics/* |

---

## Верификация

### 1. Компиляция
```bash
go build ./...
```

### 2. Запуск сервера
```bash
go run cmd/api/main.go
```

### 3. Тестирование endpoints

```bash
# Получить JWT токен
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}' | jq -r '.token')

# Employee Cabinet
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/my-profile
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/my-leave-balance
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/my-vacations
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/my-salary
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/my-training
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/my-competencies
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/my-notifications

# Analytics
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/hrm/analytics/dashboard
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/hrm/analytics/reports/headcount
```

### 4. Проверки безопасности

- [ ] `/my-*` доступны только авторизованным
- [ ] `/my-*` возвращают только данные текущего пользователя
- [ ] Нельзя получить данные другого сотрудника
- [ ] `/hrm/analytics/*` требуют роль hrm

---

## Порядок выполнения

1. **cabinet.go** + **profile.go** - базовый паттерн
2. **leave_balance.go** + **vacations.go** - работа с отпусками
3. **salary.go** - зарплата
4. **training.go** + **competencies.go** - обучение и компетенции
5. **notifications.go** - уведомления
6. **tasks.go** + **documents.go** - задачи и документы
7. **router.go** - подключить Cabinet routes
8. **hrm_analytics.go** (repo) - методы аналитики
9. **analytics/*.go** (handlers) - endpoints аналитики
10. **router.go** - подключить Analytics routes
11. **Тестирование**
