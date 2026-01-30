# HRM Module Implementation Plan

## Overview

Реализация бэкенда для HRM модуля с 19 подмодулями, следуя существующей архитектуре проекта.

**Scope:** Полная реализация всех 19 модулей
**Integration:** hrm_employees интегрирован с существующими contacts/users

**Принципы:** Clean Architecture, SOLID, DRY, KISS
**Паттерны:** Factory handlers, Interface-based repositories, Google Wire DI

---

## Architecture Summary

```
internal/
├── lib/model/hrm/           # Domain models
├── lib/dto/hrm/             # Request/Response DTOs
├── http-server/handlers/hrm/ # HTTP handlers
├── storage/repo/hrm_*.go    # PostgreSQL repositories
migrations/postgres/
├── 000039-000051            # HRM database migrations
```

---

## Implementation Phases

### Phase 1: Foundation
1. Database migrations (12 файлов)
2. Core models (`internal/lib/model/hrm/`)
3. Employee module (CRUD)
4. Route setup with HRM roles

### Phase 2: Core HR Operations
1. Personnel Records (документы, переводы)
2. Vacation Management (заявки, баланс, календарь)
3. Salary Management (расчёт, бонусы, удержания)
4. Timesheet (табель, корректировки)
5. Employee Cabinet (/my-* endpoints)

### Phase 3: Talent Management
1. Recruiting (вакансии, кандидаты, собеседования)
2. Training (курсы, сертификаты, планы развития)

### Phase 4: Performance & Assessment
1. Competency Assessment (компетенции, матрицы)
2. Performance Management (reviews, goals, KPIs)

### Phase 5: Documents & Access
1. HR Documents (шаблоны, подписи)
2. Access Control (карты, зоны, логи)

### Phase 6: Analytics & Dashboard
1. Dashboard с метриками
2. Analytics reports
3. Export (PDF, Excel)

---

## Database Schema (New Tables)

### Core Tables
| Table | Description |
|-------|-------------|
| `hrm_employees` | Extended employee info (links to contacts, users) |
| `hrm_personnel_documents` | Passport, diplomas, certificates |
| `hrm_transfers` | Transfer history |

### Vacation
| Table | Description |
|-------|-------------|
| `hrm_vacation_types` | Leave types (annual, sick, study, etc.) |
| `hrm_vacation_balances` | Balance per employee/year/type |
| `hrm_vacations` | Leave requests with approval flow |

### Salary
| Table | Description |
|-------|-------------|
| `hrm_salary_structures` | Base salary + allowances |
| `hrm_salaries` | Monthly payroll records |
| `hrm_salary_bonuses` | Bonus entries |
| `hrm_salary_deductions` | Deduction entries |

### Timesheet
| Table | Description |
|-------|-------------|
| `hrm_timesheets` | Monthly summary |
| `hrm_timesheet_entries` | Daily entries |
| `hrm_timesheet_corrections` | Correction requests |
| `hrm_holidays` | Public holidays |

### Recruiting
| Table | Description |
|-------|-------------|
| `hrm_vacancies` | Job openings |
| `hrm_candidates` | Applicants |
| `hrm_interviews` | Interview records |

### Training
| Table | Description |
|-------|-------------|
| `hrm_trainings` | Training courses |
| `hrm_training_participants` | Enrollments |
| `hrm_certificates` | Employee certificates |
| `hrm_development_plans` | IDP |
| `hrm_development_goals` | Goals in plans |

### Competency
| Table | Description |
|-------|-------------|
| `hrm_competency_categories` | Categories |
| `hrm_competencies` | Competency definitions |
| `hrm_competency_levels` | Level descriptions |
| `hrm_competency_matrices` | Required per position |
| `hrm_competency_assessments` | Assessments |
| `hrm_competency_scores` | Individual scores |

### Performance
| Table | Description |
|-------|-------------|
| `hrm_performance_reviews` | Review cycles |
| `hrm_performance_goals` | Employee goals |
| `hrm_kpis` | KPI tracking |

### Documents
| Table | Description |
|-------|-------------|
| `hrm_document_types` | Document types |
| `hrm_documents` | HR documents |
| `hrm_document_signatures` | Signature workflow |
| `hrm_document_templates` | Templates |

### Access Control
| Table | Description |
|-------|-------------|
| `hrm_access_zones` | Access zones |
| `hrm_access_cards` | Employee cards |
| `hrm_card_zone_access` | Card-zone permissions |
| `hrm_access_logs` | Access logs |

### Notifications
| Table | Description |
|-------|-------------|
| `hrm_notifications` | User notifications |

---

## Route Structure

```
/hrm/dashboard                    # Dashboard
/hrm/employees                    # Employee management
/hrm/personnel-records            # Personnel records
/hrm/recruiting/*                 # Recruiting
/hrm/vacations                    # Vacation management
/hrm/salaries                     # Salary management
/hrm/timesheets                   # Timesheet
/hrm/training/*                   # Training
/hrm/competencies                 # Competencies
/hrm/performance/*                # Performance
/hrm/documents                    # HR Documents
/hrm/analytics/*                  # Analytics
/hrm/access-control/*             # Access control

/my/profile                       # Employee cabinet
/my/leave-balance
/my/vacations
/my/salary
/my/training
/my/competencies
/my/notifications
/my/tasks
/my/documents
```

---

## Key Files to Modify/Create

### Migrations (create)
- `migrations/postgres/000039_hrm_employee.up.sql`
- `migrations/postgres/000040_hrm_personnel_records.up.sql`
- `migrations/postgres/000041_hrm_vacations.up.sql`
- `migrations/postgres/000042_hrm_salaries.up.sql`
- `migrations/postgres/000043_hrm_timesheets.up.sql`
- `migrations/postgres/000044_hrm_recruiting.up.sql`
- `migrations/postgres/000045_hrm_training.up.sql`
- `migrations/postgres/000046_hrm_competencies.up.sql`
- `migrations/postgres/000047_hrm_performance.up.sql`
- `migrations/postgres/000048_hrm_documents.up.sql`
- `migrations/postgres/000049_hrm_access_control.up.sql`
- `migrations/postgres/000050_hrm_notifications.up.sql`

### Models (create)
- `internal/lib/model/hrm/employee.go`
- `internal/lib/model/hrm/vacation.go`
- `internal/lib/model/hrm/salary.go`
- `internal/lib/model/hrm/timesheet.go`
- `internal/lib/model/hrm/recruiting.go`
- `internal/lib/model/hrm/training.go`
- `internal/lib/model/hrm/competency.go`
- `internal/lib/model/hrm/performance.go`
- `internal/lib/model/hrm/document.go`
- `internal/lib/model/hrm/access.go`
- `internal/lib/model/hrm/notification.go`
- `internal/lib/model/hrm/dashboard.go`

### DTOs (create)
- `internal/lib/dto/hrm/*.go`

### Repositories (create)
- `internal/storage/repo/hrm_employee.go`
- `internal/storage/repo/hrm_vacation.go`
- `internal/storage/repo/hrm_salary.go`
- `internal/storage/repo/hrm_timesheet.go`
- `internal/storage/repo/hrm_recruiting.go`
- `internal/storage/repo/hrm_training.go`
- `internal/storage/repo/hrm_competency.go`
- `internal/storage/repo/hrm_performance.go`
- `internal/storage/repo/hrm_document.go`
- `internal/storage/repo/hrm_access.go`
- `internal/storage/repo/hrm_notification.go`
- `internal/storage/repo/hrm_dashboard.go`

### Handlers (create)
- `internal/http-server/handlers/hrm/**/*.go`
- `internal/http-server/handlers/my-cabinet/**/*.go`

### Router (modify)
- `internal/http-server/router/router.go` - Add HRM route groups

### Storage Errors (modify)
- `internal/storage/storage.go` - Add HRM-specific errors

---

## Verification

1. **Database:** Run migrations, verify tables created
2. **API:** Test each endpoint with Postman/curl
3. **Integration:** Verify foreign key relationships work
4. **Auth:** Test role-based access control
5. **Files:** Test document uploads via MinIO

---

## Testing Strategy (Parallel Development)

### Approach
- **TDD-like:** Пишем тесты параллельно с реализацией
- **Unit tests:** Для каждого repository метода
- **Handler tests:** Для каждого HTTP handler

### Test Files Structure
```
internal/storage/repo/
├── hrm_employee.go
├── hrm_employee_test.go      # Unit tests for repo
├── hrm_vacation.go
├── hrm_vacation_test.go
└── ...

internal/http-server/handlers/hrm/
├── employee/
│   ├── add/
│   │   ├── add.go
│   │   └── add_test.go       # Handler tests
│   └── ...
└── ...
```

### Test Patterns (from existing codebase)
1. **Repository tests:** Use real test DB or sqlmock
2. **Handler tests:** Use httptest.NewRecorder + mock interfaces
3. **Validation tests:** Test request validation edge cases

### Priority Test Coverage
1. Employee CRUD - базовые операции
2. Vacation flow - request → approve/reject → balance update
3. Salary calculation - base + allowances - deductions
4. Timesheet entries - daily → monthly aggregation

---

## Implementation Workflow

Для каждого модуля:
1. **Migration** - создать таблицы
2. **Model** - определить структуры
3. **DTO** - request/response/filter
4. **Repository** - SQL queries + **тесты**
5. **Handler** - HTTP endpoints + **тесты**
6. **Router** - регистрация routes

---

## Notes

- Reuse existing: `organization`, `department`, `position`, `user`, `role`, `contact`
- File uploads: Use existing MinIO integration pattern
- Validation: Use `go-playground/validator` tags
- Errors: Translate PostgreSQL errors to domain errors
