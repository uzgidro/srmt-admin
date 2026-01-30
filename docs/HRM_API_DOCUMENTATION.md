# HRM API Documentation

## Overview

This document provides comprehensive API documentation for the Human Resource Management (HRM) module. The HRM system consists of 19 interconnected modules that cover all aspects of human resource management.

**Base URL:** `/api/v1`
**Authentication:** Bearer Token (JWT)
**Content-Type:** `application/json`

---

## Table of Contents

1. [Dashboard](#1-dashboard)
2. [Employee Cabinet](#2-employee-cabinet)
3. [Personnel Records](#3-personnel-records)
4. [Recruiting](#4-recruiting)
5. [Vacation Management](#5-vacation-management)
6. [Salary Management](#6-salary-management)
7. [Timesheet](#7-timesheet)
8. [Training](#8-training)
9. [Competency Assessment](#9-competency-assessment)
10. [Performance Management](#10-performance-management)
11. [HR Documents](#11-hr-documents)
12. [Analytics](#12-analytics)
13. [Access Control](#13-access-control)
14. [Org Structure](#14-org-structure)
15. [Employee](#15-employee)
16. [Department](#16-department)
17. [Position](#17-position)
18. [Users](#18-users)
19. [Roles](#19-roles)
20. [Data Models](#data-models)
21. [Error Handling](#error-handling)
22. [Authentication](#authentication)

---

## 1. Dashboard

**Description:** Главная панель HRM с ключевыми метриками и виджетами.

**Base URL:** `/hrm/dashboard`

### Endpoints

#### GET /hrm/dashboard
Получить данные дашборда HRM.

**Response:**
```json
{
    "total_employees": 150,
    "active_employees": 145,
    "on_leave": 5,
    "new_hires_this_month": 3,
    "pending_approvals": {
        "vacations": 5,
        "documents": 2,
        "salary_reviews": 1
    },
    "upcoming_events": [
        {
            "type": "birthday",
            "employee_name": "Иванов И.И.",
            "date": "2025-02-01"
        }
    ],
    "department_stats": [
        {
            "department_id": 1,
            "department_name": "IT-отдел",
            "employee_count": 25,
            "on_leave": 2
        }
    ],
    "quick_actions": [
        "approve_vacation",
        "create_employee",
        "view_reports"
    ]
}
```

#### GET /hrm/dashboard/widgets
Получить настраиваемые виджеты дашборда.

**Response:**
```json
{
    "widgets": [
        {
            "id": "headcount",
            "title": "Численность персонала",
            "type": "chart",
            "data": {}
        },
        {
            "id": "turnover",
            "title": "Текучесть кадров",
            "type": "metric",
            "value": 5.2,
            "unit": "%"
        }
    ]
}
```

---

## 2. Employee Cabinet

**Description:** Личный кабинет сотрудника для просмотра и управления личной информацией.

**Base URL:** `/employee` or `/my-*`

### Endpoints

#### GET /my-profile
Получить профиль текущего сотрудника.

**Response:**
```json
{
    "id": 1,
    "employee_id": "EMP-2019-0234",
    "full_name": "Абдуллаев Азамбай Ахмадович",
    "first_name": "Азамбай",
    "last_name": "Абдуллаев",
    "middle_name": "Ахмадович",
    "photo": "https://example.com/photos/emp-234.jpg",
    "position_id": 15,
    "position_name": "Инженер по автоматизации",
    "department_id": 1,
    "department_name": "IT-отдел",
    "email": "abdulaev.a@ministry.uz",
    "phone": "+998 (71) 123-45-67",
    "internal_phone": "1234",
    "hire_date": "2019-02-15",
    "birth_date": "1990-05-20",
    "employment_status": "active",
    "contract_type": "permanent",
    "manager_id": 5,
    "manager_name": "Каримов Бахтиёр Рустамович"
}
```

#### PATCH /my-profile
Обновить профиль текущего сотрудника.

**Request Body:**
```json
{
    "phone": "+998 (71) 123-45-68",
    "email": "new.email@ministry.uz"
}
```

**Response:** Updated `EmployeeProfile` object

#### GET /my-leave-balance
Получить баланс отпусков текущего сотрудника.

**Query Parameters:**
- `year` (optional): Год (default: текущий год)

**Response:**
```json
{
    "employee_id": 1,
    "year": 2025,
    "annual_leave_total": 15,
    "annual_leave_used": 3,
    "annual_leave_remaining": 12,
    "additional_leave_total": 5,
    "additional_leave_used": 3,
    "additional_leave_remaining": 2,
    "study_leave_total": 3,
    "study_leave_used": 3,
    "study_leave_remaining": 0,
    "sick_leave_used_month": 1,
    "sick_leave_used_year": 5,
    "comp_days_available": 2
}
```

#### GET /my-vacations
Получить заявки на отпуск текущего сотрудника.

**Query Parameters:**
- `status` (optional): Фильтр по статусу
- `year` (optional): Фильтр по году

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "type": "annual",
            "start_date": "2025-02-01",
            "end_date": "2025-02-05",
            "days_count": 5,
            "status": "approved",
            "reason": "Семейные обстоятельства",
            "submitted_at": "2025-01-20T10:00:00Z",
            "approved_by": 5,
            "approved_by_name": "Каримов Б.Р.",
            "approved_at": "2025-01-24T10:30:00Z"
        }
    ]
}
```

#### POST /my-vacations
Создать новую заявку на отпуск.

**Request Body:**
```json
{
    "type": "annual",
    "start_date": "2025-02-01",
    "end_date": "2025-02-05",
    "reason": "Семейные обстоятельства",
    "substitute_id": 10
}
```

**Response:** `201 Created` with `MyVacationRequest` object

#### POST /my-vacations/{id}/cancel
Отменить заявку на отпуск.

**Response:**
```json
{
    "id": 1,
    "status": "cancelled"
}
```

#### GET /my-salary
Получить информацию о зарплате.

**Response:**
```json
{
    "employee_id": 1,
    "current_salary": {
        "base_salary": 3000000,
        "total_allowances": 500000,
        "gross_salary": 3500000
    },
    "last_payment": {
        "id": 12,
        "period_month": 12,
        "period_year": 2024,
        "gross_salary": 3500000,
        "total_deductions": 525000,
        "net_salary": 2975000,
        "paid_at": "2025-01-02T00:00:00Z",
        "status": "paid"
    },
    "payment_history": []
}
```

#### GET /my-salary/payslip/{paymentId}
Скачать расчётный лист.

**Response:** `application/pdf` file

#### GET /my-training
Получить информацию об обучении.

**Response:**
```json
{
    "completed": [
        {
            "id": 1,
            "course_id": 10,
            "course_name": "Управление проектами",
            "course_type": "Профессиональное развитие",
            "completed_at": "2024-11-15",
            "score": 92,
            "certificate_number": "ПМ-2024-001234"
        }
    ],
    "in_progress": [
        {
            "id": 3,
            "course_id": 15,
            "course_name": "Цифровая трансформация",
            "started_at": "2025-01-20",
            "deadline": "2025-03-20",
            "progress_percent": 20
        }
    ],
    "assigned": []
}
```

#### GET /my-competencies
Получить оценки компетенций.

**Response:**
```json
{
    "employee_id": 1,
    "last_assessment_date": "2024-10-15",
    "next_assessment_date": "2025-04-15",
    "average_score": 3.5,
    "competencies": [
        {
            "competency_id": 1,
            "competency_name": "Лидерство",
            "category": "Управленческие",
            "current_level": 3,
            "max_level": 5,
            "target_level": 4
        }
    ]
}
```

#### GET /my-notifications
Получить уведомления.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "type": "vacation_approved",
            "title": "Отпуск одобрен",
            "message": "Ваша заявка на отпуск одобрена",
            "created_at": "2025-01-24T10:30:00Z",
            "read": false,
            "severity": "success"
        }
    ]
}
```

#### PATCH /my-notifications/{id}/read
Отметить уведомление как прочитанное.

#### POST /my-notifications/read-all
Отметить все уведомления как прочитанные.

#### GET /my-tasks
Получить назначенные задачи.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "type": "training",
            "title": "Пройти курс \"Цифровая трансформация\"",
            "due_date": "2025-03-20",
            "priority": "medium",
            "status": "in_progress"
        }
    ]
}
```

#### GET /my-documents
Получить личные документы.

#### GET /my-documents/{id}/download
Скачать документ.

---

## 3. Personnel Records

**Description:** Кадровый учёт сотрудников.

**Base URL:** `/hrm/personnel-records`

### Endpoints

#### GET /hrm/personnel-records
Получить список кадровых записей.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "tab_number": "TAB-001",
            "hire_date": "2019-02-15",
            "department_id": 1,
            "department_name": "IT-отдел",
            "position_id": 15,
            "position_name": "Инженер",
            "contract_type": "permanent",
            "contract_number": "TD-2019-001",
            "contract_date": "2019-02-15",
            "status": "active"
        }
    ]
}
```

#### GET /hrm/personnel-records/{id}
Получить кадровую запись по ID.

#### GET /hrm/personnel-records/employee/{employeeId}
Получить кадровую запись по ID сотрудника.

#### POST /hrm/personnel-records
Создать кадровую запись.

**Request Body:**
```json
{
    "employee_id": 1,
    "tab_number": "TAB-001",
    "hire_date": "2019-02-15",
    "department_id": 1,
    "position_id": 15,
    "contract_type": "permanent",
    "contract_number": "TD-2019-001",
    "contract_date": "2019-02-15"
}
```

#### PUT /hrm/personnel-records/{id}
Обновить кадровую запись.

#### DELETE /hrm/personnel-records/{id}
Удалить кадровую запись.

#### GET /hrm/personnel-records/{recordId}/documents
Получить документы кадровой записи.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "document_type": "passport",
            "document_number": "AA1234567",
            "issue_date": "2015-01-15",
            "expiry_date": "2025-01-15",
            "issuing_authority": "МВД",
            "file_url": "/documents/passport.pdf"
        }
    ]
}
```

#### POST /hrm/personnel-records/{recordId}/documents
Добавить документ к кадровой записи.

**Request:** `multipart/form-data`
- `file`: Файл документа
- `document_type`: Тип документа
- `document_number`: Номер документа
- `issue_date`: Дата выдачи
- `expiry_date`: Дата окончания
- `issuing_authority`: Кем выдан

#### DELETE /hrm/personnel-records/{recordId}/documents/{documentId}
Удалить документ.

#### GET /hrm/personnel-records/{recordId}/transfers
Получить историю переводов.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "from_department_id": 1,
            "from_department_name": "IT-отдел",
            "to_department_id": 2,
            "to_department_name": "Отдел разработки",
            "from_position_id": 15,
            "from_position_name": "Инженер",
            "to_position_id": 16,
            "to_position_name": "Старший инженер",
            "order_number": "ORD-2024-001",
            "order_date": "2024-06-01",
            "effective_date": "2024-06-15",
            "reason": "Повышение"
        }
    ]
}
```

#### POST /hrm/personnel-records/{recordId}/transfers
Создать запись о переводе.

---

## 4. Recruiting

**Description:** Управление вакансиями и кандидатами.

**Base URL:** `/hrm/recruiting`

### Vacancies

#### GET /hrm/recruiting/vacancies
Получить список вакансий.

**Query Parameters:**
- `status` (optional): Статус вакансии
- `department_id` (optional): ID отдела

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "title": "Frontend Developer",
            "department_id": 1,
            "department_name": "IT-отдел",
            "position_id": 15,
            "position_name": "Разработчик",
            "description": "Описание вакансии...",
            "requirements": "Требования...",
            "salary_min": 5000000,
            "salary_max": 8000000,
            "employment_type": "full_time",
            "status": "published",
            "published_at": "2025-01-15T10:00:00Z",
            "candidates_count": 5,
            "created_by": 1,
            "created_at": "2025-01-10T10:00:00Z"
        }
    ]
}
```

#### GET /hrm/recruiting/vacancies/{id}
Получить вакансию по ID.

#### POST /hrm/recruiting/vacancies
Создать вакансию.

**Request Body:**
```json
{
    "title": "Frontend Developer",
    "department_id": 1,
    "position_id": 15,
    "description": "Описание вакансии...",
    "requirements": "Требования...",
    "responsibilities": "Обязанности...",
    "salary_min": 5000000,
    "salary_max": 8000000,
    "employment_type": "full_time",
    "experience_required": "3-5 лет",
    "education_required": "Высшее"
}
```

#### PUT /hrm/recruiting/vacancies/{id}
Обновить вакансию.

#### DELETE /hrm/recruiting/vacancies/{id}
Удалить вакансию.

#### POST /hrm/recruiting/vacancies/{id}/publish
Опубликовать вакансию.

#### POST /hrm/recruiting/vacancies/{id}/close
Закрыть вакансию.

### Candidates

#### GET /hrm/recruiting/candidates
Получить список кандидатов.

**Query Parameters:**
- `vacancy_id` (optional): ID вакансии
- `status` (optional): Статус кандидата

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "vacancy_id": 1,
            "vacancy_title": "Frontend Developer",
            "first_name": "Иван",
            "last_name": "Петров",
            "email": "ivan.petrov@mail.com",
            "phone": "+998901234567",
            "resume_url": "/resumes/resume-1.pdf",
            "source": "hh.uz",
            "status": "interview_scheduled",
            "rating": 4,
            "skills": ["JavaScript", "TypeScript", "Angular"],
            "experience_years": 4,
            "created_at": "2025-01-20T10:00:00Z"
        }
    ]
}
```

#### GET /hrm/recruiting/candidates/{id}
Получить кандидата по ID.

#### POST /hrm/recruiting/candidates
Добавить кандидата.

**Request:** `multipart/form-data`
```
vacancy_id: 1
first_name: "Иван"
last_name: "Петров"
email: "ivan.petrov@mail.com"
phone: "+998901234567"
source: "hh.uz"
cover_letter: "Сопроводительное письмо..."
resume: [file]
```

#### PUT /hrm/recruiting/candidates/{id}
Обновить кандидата.

#### DELETE /hrm/recruiting/candidates/{id}
Удалить кандидата.

#### POST /hrm/recruiting/candidates/{id}/status
Изменить статус кандидата.

**Request Body:**
```json
{
    "status": "interview_scheduled",
    "notes": "Назначено собеседование на 25.01.2025"
}
```

### Interviews

#### GET /hrm/recruiting/candidates/{candidateId}/interviews
Получить собеседования кандидата.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "candidate_id": 1,
            "type": "video",
            "stage": "technical",
            "scheduled_at": "2025-01-25T14:00:00Z",
            "duration_minutes": 60,
            "location": "Google Meet",
            "interviewer_id": 5,
            "interviewer_name": "Каримов Б.Р.",
            "status": "scheduled",
            "feedback": null,
            "score": null,
            "recommendation": null
        }
    ]
}
```

#### POST /hrm/recruiting/candidates/{candidateId}/interviews
Создать собеседование.

**Request Body:**
```json
{
    "type": "video",
    "stage": "technical",
    "scheduled_at": "2025-01-25T14:00:00Z",
    "duration_minutes": 60,
    "location": "Google Meet",
    "interviewer_id": 5
}
```

#### GET /hrm/recruiting/stats
Получить статистику рекрутинга.

**Response:**
```json
{
    "total_vacancies": 10,
    "open_vacancies": 5,
    "total_candidates": 50,
    "candidates_by_status": {
        "new": 15,
        "screening": 10,
        "interview_scheduled": 8,
        "offer_sent": 3,
        "hired": 5,
        "rejected": 9
    },
    "avg_time_to_hire_days": 25,
    "conversion_rate": 10
}
```

---

## 5. Vacation Management

**Description:** Управление отпусками сотрудников.

**Base URL:** `/hrm/vacations`

### Endpoints

#### GET /hrm/vacations
Получить список отпусков.

**Query Parameters:**
- `status` (optional): Статус отпуска
- `year` (optional): Год

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "department_name": "IT-отдел",
            "type": "annual",
            "start_date": "2025-02-01",
            "end_date": "2025-02-15",
            "days_count": 15,
            "status": "approved",
            "reason": "Ежегодный отпуск",
            "substitute_id": 5,
            "substitute_name": "Петров П.П.",
            "approved_by": 10,
            "approved_by_name": "Руководитель",
            "approved_at": "2025-01-20T10:00:00Z"
        }
    ]
}
```

#### GET /hrm/vacations/{id}
Получить отпуск по ID.

#### POST /hrm/vacations
Создать заявку на отпуск.

**Request Body:**
```json
{
    "employee_id": 1,
    "type": "annual",
    "start_date": "2025-02-01",
    "end_date": "2025-02-15",
    "reason": "Ежегодный отпуск",
    "substitute_id": 5
}
```

#### PUT /hrm/vacations/{id}
Обновить заявку на отпуск.

#### DELETE /hrm/vacations/{id}
Удалить заявку на отпуск.

#### POST /hrm/vacations/{id}/approve
Одобрить заявку на отпуск.

**Response:**
```json
{
    "id": 1,
    "status": "approved",
    "approved_by": 10,
    "approved_at": "2025-01-20T10:00:00Z"
}
```

#### POST /hrm/vacations/{id}/reject
Отклонить заявку на отпуск.

**Request Body:**
```json
{
    "reason": "Причина отклонения"
}
```

#### POST /hrm/vacations/{id}/cancel
Отменить отпуск.

#### GET /hrm/vacations/balance/{employeeId}
Получить баланс отпусков сотрудника.

**Query Parameters:**
- `year` (optional): Год

**Response:**
```json
{
    "employee_id": 1,
    "year": 2025,
    "total_days": 24,
    "used_days": 10,
    "pending_days": 5,
    "remaining_days": 9,
    "carried_over_days": 3,
    "by_type": {
        "annual": { "total": 15, "used": 5, "remaining": 10 },
        "additional": { "total": 5, "used": 3, "remaining": 2 },
        "study": { "total": 3, "used": 2, "remaining": 1 }
    }
}
```

#### GET /hrm/vacations/balances
Получить балансы отпусков всех сотрудников.

#### GET /hrm/vacations/pending
Получить ожидающие одобрения заявки.

#### GET /hrm/vacations/calendar
Получить календарь отпусков.

**Query Parameters:**
- `month`: Месяц (1-12)
- `year`: Год

**Response:**
```json
{
    "month": 2,
    "year": 2025,
    "vacations": [
        {
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "department_name": "IT-отдел",
            "start_date": "2025-02-01",
            "end_date": "2025-02-15",
            "type": "annual",
            "status": "approved"
        }
    ]
}
```

---

## 6. Salary Management

**Description:** Управление заработной платой.

**Base URL:** `/hrm/salaries`

### Endpoints

#### GET /hrm/salaries
Получить список зарплат.

**Query Parameters:**
- `month` (optional): Месяц
- `year` (optional): Год
- `status` (optional): Статус

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "department_name": "IT-отдел",
            "position_name": "Инженер",
            "period_month": 1,
            "period_year": 2025,
            "base_salary": 5000000,
            "rank_allowance": 500000,
            "education_allowance": 300000,
            "seniority_allowance": 200000,
            "bonuses": 1000000,
            "gross_salary": 7000000,
            "income_tax": 840000,
            "pension_contribution": 560000,
            "other_deductions": 100000,
            "total_deductions": 1500000,
            "net_salary": 5500000,
            "status": "paid",
            "paid_at": "2025-02-05T10:00:00Z"
        }
    ]
}
```

#### GET /hrm/salaries/{id}
Получить зарплату по ID.

#### POST /hrm/salaries
Создать запись о зарплате.

**Request Body:**
```json
{
    "employee_id": 1,
    "period_month": 1,
    "period_year": 2025,
    "base_salary": 5000000,
    "rank_allowance": 500000,
    "education_allowance": 300000,
    "seniority_allowance": 200000
}
```

#### PUT /hrm/salaries/{id}
Обновить запись о зарплате.

#### DELETE /hrm/salaries/{id}
Удалить запись о зарплате.

#### POST /hrm/salaries/{id}/calculate
Рассчитать зарплату.

**Response:**
```json
{
    "id": 1,
    "gross_salary": 7000000,
    "income_tax": 840000,
    "pension_contribution": 560000,
    "other_deductions": 100000,
    "total_deductions": 1500000,
    "net_salary": 5500000,
    "status": "calculated"
}
```

#### POST /hrm/salaries/{id}/approve
Утвердить зарплату.

#### POST /hrm/salaries/{id}/pay
Отметить зарплату как выплаченную.

#### POST /hrm/salaries/bulk-calculate
Массовый расчёт зарплат.

**Request Body:**
```json
{
    "month": 1,
    "year": 2025
}
```

**Response:**
```json
{
    "calculated_count": 150,
    "total_gross": 1050000000,
    "total_net": 825000000
}
```

#### GET /hrm/salaries/structure/{employeeId}
Получить структуру зарплаты сотрудника.

**Response:**
```json
{
    "employee_id": 1,
    "base_salary": 5000000,
    "rank_allowance": 500000,
    "education_allowance": 300000,
    "seniority_allowance": 200000,
    "total_allowances": 1000000,
    "effective_date": "2024-01-01",
    "history": [
        {
            "effective_date": "2024-01-01",
            "base_salary": 5000000,
            "reason": "Повышение"
        }
    ]
}
```

#### PUT /hrm/salaries/structure/{employeeId}
Обновить структуру зарплаты.

#### GET /hrm/salaries/{salaryId}/deductions
Получить удержания из зарплаты.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "type": "loan",
            "description": "Погашение кредита",
            "amount": 500000,
            "start_date": "2024-01-01",
            "end_date": "2024-12-31"
        }
    ]
}
```

#### POST /hrm/salaries/{salaryId}/deductions
Добавить удержание.

#### GET /hrm/salaries/{salaryId}/bonuses
Получить бонусы.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "type": "performance",
            "description": "Премия за выполнение KPI",
            "amount": 1000000,
            "period_month": 1,
            "period_year": 2025
        }
    ]
}
```

#### POST /hrm/salaries/{salaryId}/bonuses
Добавить бонус.

#### GET /hrm/salaries/export
Экспорт зарплатной ведомости.

**Query Parameters:**
- `month`: Месяц
- `year`: Год

**Response:** Excel file (`application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`)

---

## 7. Timesheet

**Description:** Табель учёта рабочего времени.

**Base URL:** `/hrm/timesheets`

### Endpoints

#### GET /hrm/timesheets
Получить табели рабочего времени.

**Query Parameters:**
- `month`: Месяц
- `year`: Год
- `department_id` (optional): ID отдела

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "department_name": "IT-отдел",
            "month": 1,
            "year": 2025,
            "work_days_total": 22,
            "days_present": 20,
            "days_absent": 0,
            "days_vacation": 2,
            "days_sick_leave": 0,
            "days_business_trip": 0,
            "total_hours": 160,
            "overtime_hours": 5,
            "status": "approved"
        }
    ]
}
```

#### GET /hrm/timesheets/{id}
Получить табель по ID.

#### GET /hrm/timesheets/employee/{employeeId}
Получить табель сотрудника.

**Query Parameters:**
- `month`: Месяц
- `year`: Год

**Response:**
```json
{
    "employee_id": 1,
    "employee_name": "Иванов И.И.",
    "month": 1,
    "year": 2025,
    "entries": [
        {
            "date": "2025-01-02",
            "day_type": "work_day",
            "status": "present",
            "check_in": "09:00",
            "check_out": "18:00",
            "worked_hours": 8,
            "overtime_hours": 0,
            "notes": ""
        },
        {
            "date": "2025-01-03",
            "day_type": "work_day",
            "status": "vacation",
            "check_in": null,
            "check_out": null,
            "worked_hours": 0,
            "overtime_hours": 0,
            "notes": "Отпуск"
        }
    ],
    "summary": {
        "work_days_total": 22,
        "days_present": 20,
        "days_absent": 0,
        "days_vacation": 2,
        "total_hours": 160,
        "overtime_hours": 5
    }
}
```

#### POST /hrm/timesheets
Создать табель.

#### PUT /hrm/timesheets/{id}
Обновить табель.

#### POST /hrm/timesheets/{id}/approve
Утвердить табель.

#### GET /hrm/timesheets/corrections
Получить заявки на корректировку.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "date": "2025-01-15",
            "original_check_in": "09:30",
            "original_check_out": "18:00",
            "requested_check_in": "09:00",
            "requested_check_out": "18:00",
            "reason": "Забыл отметить вход",
            "status": "pending",
            "requested_at": "2025-01-16T10:00:00Z"
        }
    ]
}
```

#### POST /hrm/timesheets/corrections
Создать заявку на корректировку.

#### POST /hrm/timesheets/corrections/{id}/approve
Утвердить корректировку.

#### POST /hrm/timesheets/corrections/{id}/reject
Отклонить корректировку.

#### GET /hrm/timesheets/holidays
Получить список праздничных дней.

**Query Parameters:**
- `year`: Год

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "date": "2025-01-01",
            "name": "Новый год",
            "type": "national",
            "is_working": false
        }
    ]
}
```

#### GET /hrm/timesheets/work-schedule
Получить рабочий график.

**Response:**
```json
{
    "work_days": ["monday", "tuesday", "wednesday", "thursday", "friday"],
    "start_time": "09:00",
    "end_time": "18:00",
    "lunch_start": "13:00",
    "lunch_end": "14:00",
    "break_duration_minutes": 60
}
```

---

## 8. Training

**Description:** Обучение и развитие сотрудников.

**Base URL:** `/hrm/training`

### Trainings

#### GET /hrm/training/trainings
Получить список обучений.

**Query Parameters:**
- `status` (optional): Статус
- `type` (optional): Тип обучения

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "title": "Управление проектами",
            "type": "course",
            "description": "Курс по управлению проектами...",
            "provider": "Внутренний тренинг",
            "start_date": "2025-02-01",
            "end_date": "2025-02-15",
            "duration_hours": 40,
            "location": "Офис, ауд. 301",
            "max_participants": 20,
            "current_participants": 15,
            "status": "planned",
            "cost": 5000000,
            "trainer_name": "Тренер И.И."
        }
    ]
}
```

#### GET /hrm/training/trainings/{id}
Получить обучение по ID.

#### POST /hrm/training/trainings
Создать обучение.

**Request Body:**
```json
{
    "title": "Управление проектами",
    "type": "course",
    "description": "Курс по управлению проектами...",
    "provider": "Внутренний тренинг",
    "start_date": "2025-02-01",
    "end_date": "2025-02-15",
    "duration_hours": 40,
    "location": "Офис, ауд. 301",
    "max_participants": 20,
    "cost": 5000000
}
```

#### PUT /hrm/training/trainings/{id}
Обновить обучение.

#### DELETE /hrm/training/trainings/{id}
Удалить обучение.

### Participants

#### GET /hrm/training/trainings/{trainingId}/participants
Получить участников обучения.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "department_name": "IT-отдел",
            "enrollment_date": "2025-01-20",
            "status": "enrolled",
            "progress_percent": 0,
            "score": null,
            "completed_at": null,
            "certificate_url": null
        }
    ]
}
```

#### POST /hrm/training/trainings/{trainingId}/participants
Добавить участника.

**Request Body:**
```json
{
    "employee_id": 1
}
```

#### DELETE /hrm/training/trainings/{trainingId}/participants/{participantId}
Удалить участника.

#### POST /hrm/training/trainings/{trainingId}/participants/{participantId}/complete
Отметить завершение обучения.

**Request Body:**
```json
{
    "score": 92,
    "certificate_number": "CERT-2025-001"
}
```

#### GET /hrm/training/employees/{employeeId}/trainings
Получить обучения сотрудника.

#### GET /hrm/training/employees/{employeeId}/certificates
Получить сертификаты сотрудника.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "certificate_name": "Project Management Professional",
            "issuing_organization": "PMI",
            "issue_date": "2024-06-15",
            "expiry_date": "2027-06-15",
            "certificate_number": "PMP-123456",
            "file_url": "/certificates/pmp-123456.pdf"
        }
    ]
}
```

#### POST /hrm/training/employees/{employeeId}/certificates
Добавить сертификат.

**Request:** `multipart/form-data`

#### DELETE /hrm/training/employees/{employeeId}/certificates/{certificateId}
Удалить сертификат.

### Development Plans

#### GET /hrm/training/development-plans
Получить планы развития.

**Query Parameters:**
- `employee_id` (optional): ID сотрудника

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "title": "План развития на 2025 год",
            "start_date": "2025-01-01",
            "end_date": "2025-12-31",
            "status": "in_progress",
            "goals": [
                {
                    "id": 1,
                    "title": "Повысить уровень английского",
                    "description": "Достичь уровня B2",
                    "target_date": "2025-06-30",
                    "status": "in_progress",
                    "progress_percent": 30
                }
            ]
        }
    ]
}
```

#### GET /hrm/training/development-plans/{id}
Получить план развития по ID.

#### POST /hrm/training/development-plans
Создать план развития.

#### PUT /hrm/training/development-plans/{id}
Обновить план развития.

#### DELETE /hrm/training/development-plans/{id}
Удалить план развития.

#### POST /hrm/training/development-plans/{planId}/goals
Добавить цель в план.

#### PUT /hrm/training/development-plans/{planId}/goals/{goalId}
Обновить цель.

#### DELETE /hrm/training/development-plans/{planId}/goals/{goalId}
Удалить цель.

---

## 9. Competency Assessment

**Description:** Оценка компетенций сотрудников.

**Base URL:** `/hrm/competencies`

### Competencies

#### GET /hrm/competencies
Получить список компетенций.

**Query Parameters:**
- `category` (optional): Категория

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "name": "Лидерство",
            "description": "Способность вести за собой команду...",
            "category": "leadership",
            "level_definitions": [
                { "level": 1, "description": "Начальный уровень..." },
                { "level": 2, "description": "Базовый уровень..." },
                { "level": 3, "description": "Средний уровень..." },
                { "level": 4, "description": "Продвинутый уровень..." },
                { "level": 5, "description": "Экспертный уровень..." }
            ]
        }
    ]
}
```

#### GET /hrm/competencies/{id}
Получить компетенцию по ID.

#### POST /hrm/competencies
Создать компетенцию.

**Request Body:**
```json
{
    "name": "Лидерство",
    "description": "Способность вести за собой команду...",
    "category": "leadership",
    "level_definitions": [
        { "level": 1, "description": "Начальный уровень..." }
    ]
}
```

#### PUT /hrm/competencies/{id}
Обновить компетенцию.

#### DELETE /hrm/competencies/{id}
Удалить компетенцию.

### Assessments

#### GET /hrm/competencies/assessments
Получить оценки компетенций.

**Query Parameters:**
- `employee_id` (optional): ID сотрудника
- `status` (optional): Статус

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "assessor_id": 5,
            "assessor_name": "Каримов Б.Р.",
            "assessment_type": "manager",
            "assessment_date": "2025-01-15",
            "status": "completed",
            "scores": [
                {
                    "competency_id": 1,
                    "competency_name": "Лидерство",
                    "expected_level": 4,
                    "actual_level": 3,
                    "gap": -1,
                    "notes": "Требуется развитие..."
                }
            ],
            "overall_feedback": "Общий отзыв...",
            "recommendations": "Рекомендации..."
        }
    ]
}
```

#### GET /hrm/competencies/assessments/{id}
Получить оценку по ID.

#### POST /hrm/competencies/assessments
Создать оценку.

**Request Body:**
```json
{
    "employee_id": 1,
    "assessment_type": "manager",
    "scores": [
        {
            "competency_id": 1,
            "actual_level": 3,
            "notes": "Комментарий..."
        }
    ],
    "overall_feedback": "Общий отзыв...",
    "recommendations": "Рекомендации..."
}
```

#### PUT /hrm/competencies/assessments/{id}
Обновить оценку.

#### DELETE /hrm/competencies/assessments/{id}
Удалить оценку.

#### POST /hrm/competencies/assessments/{id}/complete
Завершить оценку.

#### GET /hrm/competencies/employees/{employeeId}/assessments
Получить все оценки сотрудника.

#### GET /hrm/competencies/employees/{employeeId}/gap-analysis
Получить анализ пробелов в компетенциях.

**Response:**
```json
{
    "employee_id": 1,
    "gaps": [
        {
            "competency_id": 1,
            "competency_name": "Лидерство",
            "expected_level": 4,
            "current_level": 3,
            "gap": -1,
            "priority": "high",
            "suggested_trainings": [
                { "id": 1, "title": "Курс лидерства" }
            ]
        }
    ]
}
```

### Competency Matrices

#### GET /hrm/competencies/matrices
Получить матрицы компетенций.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "position_id": 15,
            "position_name": "Инженер",
            "competencies": [
                {
                    "competency_id": 1,
                    "competency_name": "Техническая экспертиза",
                    "required_level": 4,
                    "is_critical": true
                }
            ]
        }
    ]
}
```

#### GET /hrm/competencies/matrices/position/{positionId}
Получить матрицу для должности.

#### POST /hrm/competencies/matrices
Создать матрицу компетенций.

#### PUT /hrm/competencies/matrices/{id}
Обновить матрицу.

#### GET /hrm/competencies/reports
Получить отчёт по компетенциям.

**Query Parameters:**
- `department_id` (optional): ID отдела
- `competency_id` (optional): ID компетенции

---

## 10. Performance Management

**Description:** Управление эффективностью сотрудников.

**Base URL:** `/hrm/performance`

### Reviews

#### GET /hrm/performance/reviews
Получить обзоры эффективности.

**Query Parameters:**
- `employee_id` (optional): ID сотрудника
- `status` (optional): Статус
- `year` (optional): Год

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "reviewer_id": 5,
            "reviewer_name": "Каримов Б.Р.",
            "review_period_start": "2024-01-01",
            "review_period_end": "2024-12-31",
            "review_type": "annual",
            "status": "completed",
            "self_review_comments": "Самооценка...",
            "manager_comments": "Комментарии руководителя...",
            "overall_rating": 4,
            "rating_label": "Превосходит ожидания",
            "goals_achieved": 5,
            "goals_total": 6,
            "strengths": "Сильные стороны...",
            "areas_for_improvement": "Области для улучшения...",
            "completed_at": "2025-01-15T10:00:00Z"
        }
    ]
}
```

#### GET /hrm/performance/reviews/{id}
Получить обзор по ID.

#### POST /hrm/performance/reviews
Создать обзор эффективности.

**Request Body:**
```json
{
    "employee_id": 1,
    "review_period_start": "2024-01-01",
    "review_period_end": "2024-12-31",
    "review_type": "annual",
    "goals": [
        {
            "title": "Завершить проект X",
            "description": "Описание...",
            "weight": 30,
            "target_value": 100
        }
    ]
}
```

#### PUT /hrm/performance/reviews/{id}
Обновить обзор.

#### DELETE /hrm/performance/reviews/{id}
Удалить обзор.

#### POST /hrm/performance/reviews/{id}/self-review
Заполнить самооценку.

**Request Body:**
```json
{
    "comments": "Моя самооценка...",
    "achievements": "Достижения...",
    "challenges": "Трудности..."
}
```

#### POST /hrm/performance/reviews/{id}/manager-review
Заполнить оценку руководителя.

**Request Body:**
```json
{
    "rating": 4,
    "comments": "Комментарии руководителя...",
    "strengths": "Сильные стороны...",
    "areas_for_improvement": "Области для улучшения..."
}
```

#### POST /hrm/performance/reviews/{id}/complete
Завершить обзор.

### Goals

#### GET /hrm/performance/goals
Получить цели.

**Query Parameters:**
- `employee_id` (optional): ID сотрудника
- `status` (optional): Статус

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "title": "Завершить проект X",
            "description": "Описание цели...",
            "metric": "Процент завершения",
            "target_value": 100,
            "actual_value": 75,
            "weight": 30,
            "progress_percent": 75,
            "start_date": "2024-01-01",
            "end_date": "2024-12-31",
            "status": "in_progress",
            "rating": null
        }
    ]
}
```

#### GET /hrm/performance/goals/{id}
Получить цель по ID.

#### POST /hrm/performance/goals
Создать цель.

#### PUT /hrm/performance/goals/{id}
Обновить цель.

#### DELETE /hrm/performance/goals/{id}
Удалить цель.

#### POST /hrm/performance/goals/{id}/progress
Обновить прогресс.

**Request Body:**
```json
{
    "actual_value": 85,
    "progress_percent": 85,
    "notes": "Комментарий к прогрессу..."
}
```

### KPIs

#### GET /hrm/performance/kpis
Получить KPI.

**Query Parameters:**
- `employee_id` (optional): ID сотрудника
- `month` (optional): Месяц
- `year` (optional): Год

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "name": "Объём продаж",
            "description": "Месячный объём продаж",
            "target_value": 100000000,
            "actual_value": 95000000,
            "unit": "UZS",
            "period_month": 1,
            "period_year": 2025,
            "achievement_percent": 95,
            "status": "on_track"
        }
    ]
}
```

#### POST /hrm/performance/kpis
Создать KPI.

#### PUT /hrm/performance/kpis/{id}
Обновить KPI.

#### DELETE /hrm/performance/kpis/{id}
Удалить KPI.

### Ratings

#### GET /hrm/performance/ratings
Получить рейтинги эффективности.

**Query Parameters:**
- `year` (optional): Год

#### GET /hrm/performance/ratings/employee/{employeeId}
Получить рейтинг сотрудника.

**Response:**
```json
{
    "employee_id": 1,
    "employee_name": "Иванов И.И.",
    "year": 2024,
    "overall_rating": 4.2,
    "percentile_rank": 85,
    "rating_history": [
        { "year": 2024, "rating": 4.2 },
        { "year": 2023, "rating": 3.8 }
    ]
}
```

#### GET /hrm/performance/dashboard
Получить дашборд эффективности.

---

## 11. HR Documents

**Description:** Документооборот HR.

**Base URL:** `/hrm/documents`

### Endpoints

#### GET /hrm/documents
Получить список документов.

**Query Parameters:**
- `type` (optional): Тип документа
- `category` (optional): Категория
- `status` (optional): Статус
- `employee_id` (optional): ID сотрудника

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "type": "employment_contract",
            "category": "personnel",
            "title": "Трудовой договор №123",
            "description": "Трудовой договор с сотрудником",
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "department_id": 1,
            "file_url": "/documents/contract-123.pdf",
            "file_size": 245000,
            "file_type": "application/pdf",
            "status": "signed",
            "created_at": "2025-01-15T10:00:00Z",
            "created_by": 5,
            "created_by_name": "Каримов Б.Р.",
            "requires_signature": true,
            "signatures": [
                {
                    "signer_id": 1,
                    "signer_name": "Иванов И.И.",
                    "status": "signed",
                    "signed_at": "2025-01-16T14:00:00Z"
                }
            ],
            "versions": [
                {
                    "version": 1,
                    "file_url": "/documents/contract-123-v1.pdf",
                    "created_at": "2025-01-15T10:00:00Z"
                }
            ]
        }
    ]
}
```

#### GET /hrm/documents/{id}
Получить документ по ID.

#### POST /hrm/documents
Создать документ.

**Request:** `multipart/form-data`
```
type: "employment_contract"
category: "personnel"
title: "Трудовой договор №123"
employee_id: 1
requires_signature: true
signers: [1, 5]
file: [file]
```

#### PUT /hrm/documents/{id}
Обновить документ.

#### DELETE /hrm/documents/{id}
Удалить документ.

#### GET /hrm/documents/{id}/download
Скачать документ.

**Response:** File (Content-Type based on file type)

#### POST /hrm/documents/{id}/sign
Подписать документ.

**Request Body:**
```json
{
    "comment": "Согласовано"
}
```

#### POST /hrm/documents/{id}/reject-signature
Отклонить подпись.

**Request Body:**
```json
{
    "reason": "Требуется доработка"
}
```

### Templates

#### GET /hrm/documents/templates
Получить шаблоны документов.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "name": "Трудовой договор",
            "type": "employment_contract",
            "content": "Шаблон с плейсхолдерами...",
            "placeholders": [
                {
                    "key": "employee_name",
                    "label": "ФИО сотрудника",
                    "type": "text",
                    "required": true
                },
                {
                    "key": "position",
                    "label": "Должность",
                    "type": "text",
                    "required": true
                },
                {
                    "key": "salary",
                    "label": "Оклад",
                    "type": "number",
                    "required": true
                }
            ]
        }
    ]
}
```

#### GET /hrm/documents/templates/{id}
Получить шаблон по ID.

#### POST /hrm/documents/templates
Создать шаблон.

#### PUT /hrm/documents/templates/{id}
Обновить шаблон.

#### DELETE /hrm/documents/templates/{id}
Удалить шаблон.

#### POST /hrm/documents/generate
Сгенерировать документ из шаблона.

**Request Body:**
```json
{
    "template_id": 1,
    "data": {
        "employee_name": "Иванов Иван Иванович",
        "position": "Инженер",
        "salary": 5000000
    }
}
```

### Document Requests

#### GET /hrm/documents/requests
Получить заявки на документы.

#### POST /hrm/documents/requests
Создать заявку на документ.

**Request Body:**
```json
{
    "type": "certificate",
    "purpose": "Для банка",
    "employee_id": 1
}
```

#### POST /hrm/documents/requests/{id}/process
Обработать заявку.

#### GET /hrm/documents/stats
Получить статистику по документам.

**Response:**
```json
{
    "total_documents": 500,
    "pending_signatures": 15,
    "expiring_soon": 10,
    "by_type": {
        "employment_contract": 100,
        "order": 200,
        "certificate": 50
    },
    "by_status": {
        "draft": 20,
        "pending_signature": 15,
        "signed": 450,
        "expired": 15
    }
}
```

---

## 12. Analytics

**Description:** HR-аналитика и отчётность.

**Base URL:** `/hrm/analytics`

### Endpoints

#### GET /hrm/analytics/dashboard
Получить дашборд HR-аналитики.

**Response:**
```json
{
    "headcount": {
        "total": 150,
        "by_department": [
            { "department_id": 1, "department_name": "IT-отдел", "count": 25 }
        ],
        "trend": [
            { "month": "2024-12", "count": 145 },
            { "month": "2025-01", "count": 150 }
        ]
    },
    "gender_distribution": {
        "male": 90,
        "female": 60
    },
    "age_distribution": {
        "18-25": 20,
        "26-35": 60,
        "36-45": 45,
        "46-55": 20,
        "55+": 5
    },
    "turnover_rate": 5.2,
    "avg_tenure_years": 3.5,
    "open_positions": 10
}
```

#### GET /hrm/analytics/reports/headcount
Получить отчёт по численности.

**Query Parameters:**
- `department_id` (optional): ID отдела
- `date` (optional): Дата

#### GET /hrm/analytics/reports/headcount-trend
Получить тренд численности.

**Query Parameters:**
- `months`: Количество месяцев

#### GET /hrm/analytics/reports/turnover
Получить отчёт по текучести.

**Query Parameters:**
- `start_date`: Начальная дата
- `end_date`: Конечная дата
- `department_id` (optional): ID отдела

**Response:**
```json
{
    "period": {
        "start_date": "2024-01-01",
        "end_date": "2024-12-31"
    },
    "total_hired": 30,
    "total_terminated": 15,
    "turnover_rate": 10.0,
    "voluntary_turnover_rate": 7.0,
    "involuntary_turnover_rate": 3.0,
    "by_department": [
        {
            "department_id": 1,
            "department_name": "IT-отдел",
            "hired": 10,
            "terminated": 3,
            "turnover_rate": 12.0
        }
    ],
    "by_reason": {
        "resignation": 10,
        "retirement": 2,
        "layoff": 2,
        "termination": 1
    }
}
```

#### GET /hrm/analytics/reports/turnover-trend
Получить тренд текучести.

#### GET /hrm/analytics/reports/attendance
Получить отчёт по посещаемости.

**Query Parameters:**
- `start_date`: Начальная дата
- `end_date`: Конечная дата
- `department_id` (optional): ID отдела

**Response:**
```json
{
    "period": {
        "start_date": "2025-01-01",
        "end_date": "2025-01-31"
    },
    "total_work_days": 22,
    "avg_attendance_rate": 95.5,
    "by_type": {
        "present": 2850,
        "vacation": 100,
        "sick_leave": 50,
        "absent": 20,
        "remote": 200
    },
    "by_department": [
        {
            "department_id": 1,
            "department_name": "IT-отдел",
            "attendance_rate": 97.0,
            "present_days": 500,
            "absent_days": 5
        }
    ]
}
```

#### GET /hrm/analytics/reports/salary
Получить отчёт по зарплатам.

**Query Parameters:**
- `month`: Месяц
- `year`: Год
- `department_id` (optional): ID отдела

**Response:**
```json
{
    "period": {
        "month": 1,
        "year": 2025
    },
    "total_gross": 1050000000,
    "total_net": 825000000,
    "total_deductions": 225000000,
    "avg_salary": 7000000,
    "median_salary": 6500000,
    "by_department": [
        {
            "department_id": 1,
            "department_name": "IT-отдел",
            "total_gross": 175000000,
            "employee_count": 25,
            "avg_salary": 7000000
        }
    ],
    "by_position": [
        {
            "position_id": 15,
            "position_name": "Инженер",
            "avg_salary": 6500000,
            "min_salary": 5000000,
            "max_salary": 8000000
        }
    ]
}
```

#### GET /hrm/analytics/reports/salary-trend
Получить тренд зарплат.

#### GET /hrm/analytics/reports/performance
Получить отчёт по эффективности.

**Query Parameters:**
- `year`: Год
- `department_id` (optional): ID отдела

#### GET /hrm/analytics/reports/training
Получить отчёт по обучению.

**Query Parameters:**
- `start_date`: Начальная дата
- `end_date`: Конечная дата

#### GET /hrm/analytics/reports/demographics
Получить демографическую статистику.

**Response:**
```json
{
    "total_employees": 150,
    "gender": {
        "male": 90,
        "female": 60,
        "male_percent": 60,
        "female_percent": 40
    },
    "age": {
        "avg_age": 32,
        "distribution": {
            "18-25": 20,
            "26-35": 60,
            "36-45": 45,
            "46-55": 20,
            "55+": 5
        }
    },
    "education": {
        "higher": 120,
        "secondary_special": 25,
        "secondary": 5
    },
    "tenure": {
        "avg_years": 3.5,
        "distribution": {
            "0-1": 30,
            "1-3": 50,
            "3-5": 40,
            "5-10": 25,
            "10+": 5
        }
    }
}
```

#### GET /hrm/analytics/reports/diversity
Получить отчёт по разнообразию.

#### POST /hrm/analytics/reports/custom
Создать пользовательский отчёт.

**Request Body:**
```json
{
    "report_type": "custom",
    "start_date": "2024-01-01",
    "end_date": "2024-12-31",
    "department_id": 1,
    "position_id": null,
    "metrics": ["headcount", "turnover", "salary"],
    "group_by": "department"
}
```

#### POST /hrm/analytics/export
Экспорт отчёта (по умолчанию).

**Request Body:** `ReportFilter`

**Response:** File

#### POST /hrm/analytics/export/pdf
Экспорт отчёта в PDF.

#### POST /hrm/analytics/export/excel
Экспорт отчёта в Excel.

---

## 13. Access Control

**Description:** Контроль доступа и пропускная система.

**Base URL:** `/hrm/access-control`

### Access Cards

#### GET /hrm/access-control/cards
Получить список карт доступа.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "card_number": "AC-001234",
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "status": "active",
            "valid_from": "2024-01-01",
            "valid_until": "2025-12-31",
            "access_zones": [
                { "zone_id": 1, "zone_name": "Главный вход" },
                { "zone_id": 2, "zone_name": "Офис IT" }
            ],
            "issued_at": "2024-01-01T10:00:00Z",
            "issued_by": "Служба безопасности",
            "last_used_at": "2025-01-28T09:05:00Z"
        }
    ]
}
```

#### GET /hrm/access-control/cards/{id}
Получить карту по ID.

#### POST /hrm/access-control/cards
Выдать карту доступа.

**Request Body:**
```json
{
    "card_number": "AC-001234",
    "employee_id": 1,
    "valid_from": "2024-01-01",
    "valid_until": "2025-12-31",
    "access_zone_ids": [1, 2, 3]
}
```

#### PUT /hrm/access-control/cards/{id}
Обновить карту доступа.

#### POST /hrm/access-control/cards/{id}/block
Заблокировать карту.

**Request Body:**
```json
{
    "reason": "Утеряна"
}
```

#### POST /hrm/access-control/cards/{id}/unblock
Разблокировать карту.

### Access Zones

#### GET /hrm/access-control/zones
Получить список зон доступа.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "name": "Главный вход",
            "code": "MAIN",
            "security_level": "standard",
            "description": "Главный вход в здание",
            "parent_zone_id": null,
            "max_occupancy": 500,
            "current_occupancy": 125,
            "readers": [
                {
                    "id": 1,
                    "type": "card",
                    "direction": "entry",
                    "ip_address": "192.168.1.10",
                    "is_online": true
                }
            ]
        }
    ]
}
```

#### GET /hrm/access-control/zones/{id}
Получить зону по ID.

#### POST /hrm/access-control/zones
Создать зону доступа.

#### PUT /hrm/access-control/zones/{id}
Обновить зону.

#### DELETE /hrm/access-control/zones/{id}
Удалить зону.

### Access Logs

#### GET /hrm/access-control/logs
Получить журнал доступа.

**Query Parameters:**
- `employee_id` (optional): ID сотрудника
- `zone_id` (optional): ID зоны
- `start_date` (optional): Начальная дата
- `end_date` (optional): Конечная дата
- `status` (optional): Статус (granted, denied)

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "timestamp": "2025-01-28T09:05:00Z",
            "card_id": 1,
            "card_number": "AC-001234",
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "zone_id": 1,
            "zone_name": "Главный вход",
            "reader_id": 1,
            "direction": "entry",
            "status": "granted",
            "denial_reason": null
        }
    ]
}
```

### Access Permissions

#### GET /hrm/access-control/permissions
Получить разрешения доступа.

#### POST /hrm/access-control/permissions
Создать разрешение.

**Request Body:**
```json
{
    "employee_id": 1,
    "zone_ids": [1, 2, 3],
    "valid_from": "2025-01-01",
    "valid_until": "2025-12-31",
    "is_temporary": false,
    "reason": "Основной доступ"
}
```

#### DELETE /hrm/access-control/permissions/{id}
Удалить разрешение.

### Access Requests

#### GET /hrm/access-control/requests
Получить заявки на доступ.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "employee_name": "Иванов И.И.",
            "requested_zones": [
                { "zone_id": 5, "zone_name": "Серверная" }
            ],
            "reason": "Необходим доступ для обслуживания серверов",
            "valid_from": "2025-02-01",
            "valid_until": "2025-02-28",
            "status": "pending",
            "requested_at": "2025-01-28T10:00:00Z"
        }
    ]
}
```

#### POST /hrm/access-control/requests
Создать заявку на доступ.

#### POST /hrm/access-control/requests/{id}/approve
Одобрить заявку.

#### POST /hrm/access-control/requests/{id}/reject
Отклонить заявку.

#### GET /hrm/access-control/stats
Получить статистику доступа.

**Response:**
```json
{
    "total_cards": 200,
    "active_cards": 150,
    "blocked_cards": 10,
    "expired_cards": 40,
    "zones_count": 20,
    "today_entries": 145,
    "today_exits": 130,
    "current_onsite": 125,
    "zone_occupancy": [
        {
            "zone_id": 1,
            "zone_name": "Офис",
            "current": 125,
            "max": 500,
            "percent": 25
        }
    ]
}
```

---

## 14. Org Structure

**Description:** Организационная структура.

**Base URL:** `/hrm/org-structure`

### Endpoints

#### GET /hrm/org-structure/units
Получить организационные единицы.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "name": "Головной офис",
            "code": "HQ",
            "type": "company",
            "parent_id": null,
            "head_id": 1,
            "head_name": "Директор И.И.",
            "employee_count": 150,
            "budget": 5000000000,
            "location": "г. Ташкент",
            "status": "active",
            "children": [
                {
                    "id": 2,
                    "name": "IT Департамент",
                    "type": "department",
                    "employee_count": 25
                }
            ]
        }
    ]
}
```

#### GET /hrm/org-structure/units/{id}
Получить организационную единицу по ID.

#### POST /hrm/org-structure/units
Создать организационную единицу.

**Request Body:**
```json
{
    "name": "Новый отдел",
    "code": "NEW",
    "type": "department",
    "parent_id": 1,
    "head_id": 5,
    "location": "г. Ташкент"
}
```

#### PUT /hrm/org-structure/units/{id}
Обновить организационную единицу.

#### DELETE /hrm/org-structure/units/{id}
Удалить организационную единицу.

#### GET /hrm/org-structure/chart
Получить организационную диаграмму.

**Response:**
```json
{
    "root": {
        "id": 1,
        "name": "Директор",
        "position": "Генеральный директор",
        "photo": "/photos/director.jpg",
        "email": "director@company.uz",
        "phone": "+998901234567",
        "children": [
            {
                "id": 2,
                "name": "Заместитель директора",
                "position": "Заместитель",
                "children": []
            }
        ]
    }
}
```

#### GET /hrm/org-structure/employees
Получить сотрудников в организационной структуре.

**Query Parameters:**
- `unit_id` (optional): ID организационной единицы

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_id": 1,
            "name": "Иванов И.И.",
            "position_name": "Инженер",
            "department_id": 2,
            "department_name": "IT Департамент",
            "manager_id": 5,
            "manager_name": "Каримов Б.Р.",
            "hire_date": "2019-02-15",
            "subordinates_count": 0,
            "photo": "/photos/emp-1.jpg"
        }
    ]
}
```

#### GET /hrm/org-structure/stats
Получить статистику организационной структуры.

**Response:**
```json
{
    "total_departments": 15,
    "total_employees": 150,
    "total_managers": 20,
    "avg_team_size": 7.5,
    "max_hierarchy_depth": 5,
    "vacant_positions": 10,
    "departments_by_type": {
        "department": 10,
        "division": 3,
        "team": 2
    }
}
```

#### GET /hrm/org-structure/departments/{id}/details
Получить детали отдела.

**Response:**
```json
{
    "id": 2,
    "name": "IT Департамент",
    "code": "IT",
    "head_id": 5,
    "head_name": "Каримов Б.Р.",
    "parent_id": 1,
    "parent_name": "Головной офис",
    "employees": [
        {
            "id": 1,
            "name": "Иванов И.И.",
            "position_name": "Инженер"
        }
    ],
    "sub_departments": [],
    "budget": 500000000,
    "stats": {
        "employee_count": 25,
        "avg_tenure": 3.2,
        "avg_age": 30
    }
}
```

---

## 15. Employee

**Description:** Управление сотрудниками.

**Base URL:** `/employees` or `/hrm/employees`

### Endpoints

#### GET /employees
Получить список сотрудников.

**Query Parameters:**
- `department_id` (optional): ID отдела
- `position_id` (optional): ID должности
- `status` (optional): Статус
- `search` (optional): Поиск по имени
- `page` (optional): Номер страницы
- `limit` (optional): Количество на странице

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "employee_code": "EMP-001",
            "first_name": "Иван",
            "last_name": "Иванов",
            "middle_name": "Иванович",
            "full_name": "Иванов Иван Иванович",
            "email": "ivanov@company.uz",
            "phone": "+998901234567",
            "internal_phone": "1234",
            "department_id": 1,
            "department_name": "IT-отдел",
            "position_id": 15,
            "position_name": "Инженер",
            "manager_id": 5,
            "manager_name": "Каримов Б.Р.",
            "hire_date": "2019-02-15",
            "birth_date": "1990-05-20",
            "gender": "male",
            "status": "active",
            "photo": "/photos/emp-1.jpg"
        }
    ],
    "meta": {
        "total": 150,
        "page": 1,
        "limit": 20,
        "total_pages": 8
    }
}
```

#### GET /employees/{id}
Получить сотрудника по ID.

**Response:**
```json
{
    "id": 1,
    "employee_code": "EMP-001",
    "first_name": "Иван",
    "last_name": "Иванов",
    "middle_name": "Иванович",
    "full_name": "Иванов Иван Иванович",
    "email": "ivanov@company.uz",
    "phone": "+998901234567",
    "internal_phone": "1234",
    "department_id": 1,
    "department_name": "IT-отдел",
    "position_id": 15,
    "position_name": "Инженер",
    "manager_id": 5,
    "manager_name": "Каримов Б.Р.",
    "hire_date": "2019-02-15",
    "birth_date": "1990-05-20",
    "gender": "male",
    "status": "active",
    "contract_type": "permanent",
    "address": "г. Ташкент, ул. Примерная, д. 1",
    "passport_series": "AA",
    "passport_number": "1234567",
    "inn": "123456789",
    "photo": "/photos/emp-1.jpg",
    "created_at": "2019-02-15T10:00:00Z",
    "updated_at": "2025-01-15T10:00:00Z"
}
```

#### POST /employees
Создать сотрудника.

**Request:** `multipart/form-data` or `application/json`
```json
{
    "first_name": "Иван",
    "last_name": "Иванов",
    "middle_name": "Иванович",
    "email": "ivanov@company.uz",
    "phone": "+998901234567",
    "department_id": 1,
    "position_id": 15,
    "manager_id": 5,
    "hire_date": "2025-02-01",
    "birth_date": "1990-05-20",
    "gender": "male",
    "contract_type": "permanent"
}
```

#### PUT /employees/{id}
Обновить сотрудника.

#### PATCH /employees/{id}
Частично обновить сотрудника.

#### DELETE /employees/{id}
Удалить сотрудника.

#### POST /employees/{id}/terminate
Уволить сотрудника.

**Request Body:**
```json
{
    "termination_date": "2025-02-28",
    "termination_reason": "resignation",
    "notes": "По собственному желанию"
}
```

#### GET /employees/{id}/subordinates
Получить подчинённых сотрудника.

#### GET /employees/search
Поиск сотрудников.

**Query Parameters:**
- `q`: Поисковый запрос

---

## 16. Department

**Description:** Управление отделами.

**Base URL:** `/department`

### Endpoints

#### GET /department
Получить список отделов.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "name": "IT-отдел",
            "description": "Отдел информационных технологий",
            "organization_id": 1,
            "organization": {
                "id": 1,
                "name": "Головной офис"
            }
        }
    ]
}
```

#### POST /department
Создать отдел.

**Request Body:**
```json
{
    "name": "Новый отдел",
    "description": "Описание отдела",
    "organization_id": 1
}
```

#### PATCH /department/{id}
Обновить отдел.

**Request Body:**
```json
{
    "name": "Обновлённое название",
    "description": "Обновлённое описание"
}
```

#### DELETE /department/{id}
Удалить отдел.

---

## 17. Position

**Description:** Управление должностями.

**Base URL:** `/positions`

### Endpoints

#### GET /positions
Получить список должностей.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "name": "Инженер",
            "description": "Должность инженера"
        },
        {
            "id": 2,
            "name": "Старший инженер",
            "description": "Должность старшего инженера"
        }
    ]
}
```

#### POST /positions
Создать должность.

**Request Body:**
```json
{
    "name": "Новая должность",
    "description": "Описание должности"
}
```

#### PATCH /positions/{id}
Обновить должность.

#### DELETE /positions/{id}
Удалить должность.

---

## 18. Users

**Description:** Управление пользователями системы.

**Base URL:** `/users`

### Endpoints

#### GET /users
Получить список пользователей.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "name": "Иванов И.И.",
            "login": "ivanov",
            "roles": [
                { "id": 1, "name": "admin" }
            ],
            "role_ids": [1],
            "contact": {
                "id": 1,
                "name": "Иванов Иван Иванович",
                "email": "ivanov@company.uz"
            }
        }
    ]
}
```

#### GET /users/{id}
Получить пользователя по ID.

#### POST /users
Создать пользователя.

**Request:** `multipart/form-data`
```
login: "newuser"
password: "securepassword"
role_ids: [1, 2]
contact_id: 1
```

или с созданием контакта:
```
login: "newuser"
password: "securepassword"
role_ids: [1, 2]
contact[name]: "Новый пользователь"
contact[email]: "new@company.uz"
contact[phone]: "+998901234567"
contact[organization_id]: 1
contact[department_id]: 1
contact[position_id]: 15
```

#### PATCH /users/{id}
Обновить пользователя.

**Request:** `multipart/form-data`
```
login: "updatedlogin"
password: "newpassword"
role_ids: [1, 2, 3]
```

#### DELETE /users/{id}
Удалить пользователя.

---

## 19. Roles

**Description:** Управление ролями.

**Base URL:** `/roles`

### Endpoints

#### GET /roles
Получить список ролей.

**Response:**
```json
{
    "data": [
        {
            "id": 1,
            "name": "admin",
            "description": "Администратор системы"
        },
        {
            "id": 2,
            "name": "hr_manager",
            "description": "HR менеджер"
        },
        {
            "id": 3,
            "name": "employee",
            "description": "Сотрудник"
        }
    ]
}
```

#### GET /roles/{id}
Получить роль по ID.

#### POST /roles
Создать роль.

**Request Body:**
```json
{
    "name": "new_role",
    "description": "Описание новой роли"
}
```

#### PUT /roles/{id}
Обновить роль.

#### DELETE /roles/{id}
Удалить роль.

---

## Data Models

### Common Types

#### Employment Status
```typescript
type EmploymentStatus = 'active' | 'on_leave' | 'on_sick_leave' | 'suspended' | 'terminated';
```

#### Contract Type
```typescript
type ContractType = 'permanent' | 'fixed_term' | 'probation' | 'contractor';
```

#### Gender
```typescript
type Gender = 'male' | 'female';
```

### Vacation Types
```typescript
type VacationType =
  | 'annual'       // Основной отпуск
  | 'additional'   // Дополнительный отпуск
  | 'study'        // Учебный отпуск
  | 'unpaid'       // Отпуск без сохранения з/п
  | 'sick'         // Больничный
  | 'maternity'    // Декретный отпуск
  | 'paternity'    // Отпуск по уходу за ребенком
  | 'comp_day'     // Отгул
  | 'other';       // Прочее
```

### Vacation Status
```typescript
type VacationStatus = 'draft' | 'pending' | 'approved' | 'rejected' | 'cancelled' | 'completed';
```

### Salary Status
```typescript
type SalaryStatus = 'draft' | 'calculated' | 'pending_approval' | 'approved' | 'rejected' | 'paid';
```

### Timesheet Status
```typescript
type TimesheetDayStatus =
  | 'present'        // Присутствует
  | 'absent'         // Отсутствует
  | 'vacation'       // Отпуск
  | 'sick_leave'     // Больничный
  | 'business_trip'  // Командировка
  | 'remote'         // Удалённая работа
  | 'day_off'        // Выходной
  | 'holiday'        // Праздник
  | 'unpaid_leave'   // Отпуск без содержания
  | 'late'           // Опоздание
  | 'left_early';    // Ранний уход
```

### Training Types
```typescript
type TrainingType = 'course' | 'workshop' | 'seminar' | 'certification' | 'mentoring' | 'self_study';
```

### Training Status
```typescript
type TrainingStatus = 'planned' | 'in_progress' | 'completed' | 'cancelled';
```

### Competency Categories
```typescript
type CompetencyCategory = 'technical' | 'soft' | 'leadership' | 'functional' | 'core';
```

### Assessment Types
```typescript
type AssessmentType = 'self' | 'manager' | '360' | 'technical_test' | 'interview' | 'assessment_center';
```

### Performance Review Types
```typescript
type ReviewType = 'annual' | 'semi_annual' | 'quarterly' | 'probation' | 'project';
```

### Performance Review Status
```typescript
type ReviewStatus = 'draft' | 'self_review' | 'manager_review' | 'calibration' | 'completed';
```

### Goal Status
```typescript
type GoalStatus = 'not_started' | 'in_progress' | 'completed' | 'exceeded' | 'not_achieved';
```

### KPI Status
```typescript
type KPIStatus = 'on_track' | 'at_risk' | 'behind' | 'achieved' | 'exceeded';
```

### Document Types
```typescript
type DocumentType =
  | 'employment_contract'   // Трудовой договор
  | 'contract_amendment'    // Дополнительное соглашение
  | 'order'                 // Приказ
  | 'statement'             // Заявление
  | 'certificate'           // Справка
  | 'memo'                  // Служебная записка
  | 'act'                   // Акт
  | 'protocol'              // Протокол
  | 'regulation'            // Положение
  | 'instruction'           // Инструкция
  | 'report'                // Отчёт
  | 'other';                // Прочее
```

### Document Categories
```typescript
type DocumentCategory = 'personnel' | 'financial' | 'administrative' | 'regulatory' | 'organizational';
```

### Document Status
```typescript
type DocumentStatus = 'draft' | 'pending_signature' | 'signed' | 'rejected' | 'expired' | 'archived';
```

### Access Card Status
```typescript
type CardStatus = 'active' | 'blocked' | 'expired' | 'lost' | 'returned';
```

### Security Levels
```typescript
type SecurityLevel = 'public' | 'standard' | 'restricted' | 'high' | 'critical';
```

### Access Reader Types
```typescript
type ReaderType = 'card' | 'biometric' | 'pin' | 'combined';
```

### Access Event Status
```typescript
type AccessEventStatus = 'granted' | 'denied' | 'error';
```

### Org Unit Types
```typescript
type OrgUnitType = 'company' | 'branch' | 'division' | 'department' | 'section' | 'group' | 'team';
```

### Notification Types
```typescript
type NotificationType =
  | 'vacation_approved'      // Отпуск одобрен
  | 'vacation_rejected'      // Отпуск отклонен
  | 'salary_paid'            // Зарплата выплачена
  | 'training_assigned'      // Назначено обучение
  | 'assessment_scheduled'   // Запланирована оценка
  | 'task_assigned'          // Назначена задача
  | 'document_ready'         // Документ готов
  | 'system'                 // Системное уведомление
  | 'other';                 // Прочее
```

### Notification Severity
```typescript
type NotificationSeverity = 'info' | 'success' | 'warn' | 'danger';
```

### Task Types
```typescript
type TaskType = 'training' | 'assessment' | 'document' | 'approval' | 'meeting' | 'other';
```

### Task Priority
```typescript
type TaskPriority = 'low' | 'medium' | 'high' | 'urgent';
```

### Task Status
```typescript
type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'overdue';
```

### Candidate Status
```typescript
type CandidateStatus =
  | 'new'                   // Новый
  | 'screening'             // Скрининг
  | 'phone_interview'       // Телефонное интервью
  | 'interview_scheduled'   // Собеседование назначено
  | 'interviewed'           // Прошёл собеседование
  | 'offer_sent'            // Оффер отправлен
  | 'offer_accepted'        // Оффер принят
  | 'hired'                 // Принят на работу
  | 'rejected'              // Отклонён
  | 'withdrawn';            // Отозвал кандидатуру
```

### Interview Types
```typescript
type InterviewType = 'phone' | 'video' | 'in_person';
```

### Interview Stages
```typescript
type InterviewStage = 'hr' | 'technical' | 'final' | 'offer';
```

---

## Error Handling

### Error Response Format

All endpoints return errors in a consistent format:

```json
{
    "error": {
        "code": "ERROR_CODE",
        "message": "Human readable error message",
        "details": [
            {
                "field": "field_name",
                "message": "Field-specific error message"
            }
        ]
    }
}
```

### Common Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `BAD_REQUEST` | Неверный формат запроса |
| 401 | `UNAUTHORIZED` | Требуется аутентификация |
| 403 | `FORBIDDEN` | Доступ запрещён |
| 404 | `NOT_FOUND` | Ресурс не найден |
| 409 | `CONFLICT` | Конфликт данных |
| 422 | `VALIDATION_ERROR` | Ошибка валидации |
| 429 | `TOO_MANY_REQUESTS` | Превышен лимит запросов |
| 500 | `INTERNAL_ERROR` | Внутренняя ошибка сервера |

### Validation Error Example

```json
{
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "Validation failed",
        "details": [
            {
                "field": "email",
                "message": "Invalid email format"
            },
            {
                "field": "start_date",
                "message": "Start date must be in the future"
            }
        ]
    }
}
```

---

## Authentication

### JWT Authentication

All API requests require JWT authentication via the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

### Obtaining a Token

```http
POST /auth/login
Content-Type: application/json

{
    "login": "username",
    "password": "password"
}
```

**Response:**
```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 3600
}
```

### Refreshing a Token

```http
POST /auth/refresh
Content-Type: application/json

{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Token Expiration

- Access token expires in 1 hour (3600 seconds)
- Refresh token expires in 7 days
- Use the refresh token to obtain a new access token before expiration

---

## Pagination

### Request Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | number | 1 | Page number |
| `limit` | number | 20 | Items per page (max: 100) |

### Response Format

```json
{
    "data": [...],
    "meta": {
        "total": 150,
        "page": 1,
        "limit": 20,
        "total_pages": 8
    }
}
```

---

## Date and Time Format

- **Dates:** ISO 8601 format `YYYY-MM-DD` (e.g., `2025-01-28`)
- **Timestamps:** ISO 8601 format with timezone `YYYY-MM-DDTHH:mm:ssZ` (e.g., `2025-01-28T10:30:00Z`)
- **Time:** 24-hour format `HH:mm` (e.g., `09:00`, `18:30`)

---

## Currency

All monetary values are in UZS (Uzbekistan Som) as integers. For example:
- `5000000` = 5,000,000 UZS

---

## File Uploads

For endpoints that accept file uploads, use `multipart/form-data` content type:

```http
POST /hrm/documents
Content-Type: multipart/form-data

------WebKitFormBoundary
Content-Disposition: form-data; name="file"; filename="document.pdf"
Content-Type: application/pdf

[file content]
------WebKitFormBoundary
Content-Disposition: form-data; name="type"

employment_contract
------WebKitFormBoundary--
```

---

## Rate Limiting

API requests are rate-limited:
- **Authenticated users:** 1000 requests per hour
- **Anonymous users:** 100 requests per hour

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1706437200
```

---

## Versioning

The API uses URL versioning. Current version: `v1`

All endpoints are prefixed with `/api/v1/`

---

## Notes for Backend Developers

1. **User Context:** Endpoints prefixed with `/my-*` use the JWT token to identify the current user. No `employee_id` parameter is needed.

2. **Workflow States:** Many entities have workflow states (draft → pending → approved/rejected). Ensure proper state transitions.

3. **Cascading Deletes:** When deleting parent entities, consider child records. Use soft deletes where appropriate.

4. **Audit Logging:** Log all create, update, and delete operations with user ID and timestamp.

5. **Real-time Updates:** Consider implementing WebSocket for notifications and real-time updates.

6. **File Storage:** Store uploaded files in a secure location and serve via signed URLs.

7. **Data Validation:** Validate all input data server-side. Don't rely on client-side validation.

8. **Performance:** Use pagination for list endpoints. Consider caching for frequently accessed data.

9. **Security:**
   - Validate user permissions for each request
   - Sanitize all input to prevent XSS and SQL injection
   - Use parameterized queries
   - Encrypt sensitive data at rest

10. **Internationalization:** Support multiple languages for error messages and notifications.
