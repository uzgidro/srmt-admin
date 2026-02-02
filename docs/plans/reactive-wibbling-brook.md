# HRM Module Analysis - Detailed Issues & Proposed Fixes

## Summary

| Category | Issues Found | Priority |
|----------|--------------|----------|
| Security | 5 critical | CRITICAL |
| Data Integrity | 6 high | HIGH |
| Business Logic | 5 medium | MEDIUM |
| Code Quality | 4 low | LOW |

---

## 1. SECURITY ISSUES

### 1.1 Missing Row-Level Access Control for Salary

**Location:** `internal/http-server/handlers/hrm/salary/salary.go:66-97`

**Current code:**
```go
func GetStructures(log *slog.Logger, repo SalaryStructureRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... validation ...
        if empIDStr := q.Get("employee_id"); empIDStr != "" {
            val, _ := strconv.ParseInt(empIDStr, 10, 64)
            filter.EmployeeID = &val  // <-- NO PERMISSION CHECK!
        }
        structures, err := repo.GetSalaryStructures(r.Context(), filter)
        // Returns ANY employee's salary
    }
}
```

**Proposed fix:**
```go
func GetStructures(log *slog.Logger, repo SalaryStructureRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        claims, _ := mwauth.ClaimsFromContext(r.Context())

        if empIDStr := q.Get("employee_id"); empIDStr != "" {
            val, _ := strconv.ParseInt(empIDStr, 10, 64)

            // Check permission: user can only see own salary or subordinates
            if !canAccessEmployeeSalary(claims, val, repo) {
                render.Status(r, http.StatusForbidden)
                render.JSON(w, r, resp.Forbidden("Access denied to this employee's salary"))
                return
            }
            filter.EmployeeID = &val
        } else {
            // No employee_id - restrict to accessible employees only
            filter.AccessibleByUserID = &claims.UserID
        }
        // ...
    }
}

func canAccessEmployeeSalary(claims *auth.Claims, employeeID int64, repo interface{}) bool {
    // 1. Check if user is the employee themselves
    // 2. Check if user is manager of this employee
    // 3. Check if user has HR role with access to department
    return false // implement
}
```

**Affected files:**
- `internal/http-server/handlers/hrm/salary/salary.go` (GetStructures, GetSalaries)
- `internal/http-server/handlers/hrm/vacation/vacation.go` (GetVacations)
- `internal/http-server/handlers/hrm/access/access.go` (GetAccessLogs)

---

### 1.2 Missing Authorization Middleware

**Location:** `internal/http-server/router/router.go`

**Current state:** HRM routes only check for authenticated user, not specific roles.

**Proposed fix:** Add role checks for each endpoint group:
```go
r.Route("/hrm", func(r chi.Router) {
    // Salary - only HR and managers
    r.Route("/salaries", func(r chi.Router) {
        r.Use(mwauth.RequireAnyRole("hr", "manager", "admin"))
        r.Get("/", salary.GetSalaries(log, repo))
        // ...
    })

    // Access logs - only security and admin
    r.Route("/access", func(r chi.Router) {
        r.Use(mwauth.RequireAnyRole("security", "admin"))
        r.Get("/logs", access.GetAccessLogs(log, repo))
    })
})
```

---

### 1.3 Ignored Parsing Errors

**Location:** `internal/http-server/handlers/hrm/access/access.go:286-292`

**Current code:**
```go
if limitStr := q.Get("limit"); limitStr != "" {
    val, _ := strconv.Atoi(limitStr)  // ERROR IGNORED!
    filter.Limit = val
}
```

**Proposed fix:**
```go
if limitStr := q.Get("limit"); limitStr != "" {
    val, err := strconv.Atoi(limitStr)
    if err != nil {
        log.Warn("invalid 'limit' parameter", sl.Err(err))
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, resp.BadRequest("Invalid 'limit' parameter"))
        return
    }
    if val < 1 || val > 1000 {
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, resp.BadRequest("Limit must be between 1 and 1000"))
        return
    }
    filter.Limit = val
}
```

**Similar issues in:**
- `access.go:294-297` (offset parameter)
- Multiple other handlers with limit/offset

---

## 2. DATA INTEGRITY ISSUES

### 2.1 Race Condition in Vacation Balance Update

**Location:** `internal/storage/repo/hrm_vacation.go:480-500`

**Current code:**
```go
func (r *Repo) UpdateVacationBalanceUsedDays(ctx context.Context,
    employeeID int64, vacationTypeID int, year int, daysToAdd float64) error {

    const query = `
        UPDATE hrm_vacation_balances
        SET used_days = used_days + $1
        WHERE employee_id = $2 AND vacation_type_id = $3 AND year = $4`

    res, err := r.db.ExecContext(ctx, query, daysToAdd, employeeID, vacationTypeID, year)
    // ...
}
```

**Problem:** Two concurrent vacation approvals can both succeed, consuming days twice.

**Proposed fix:**
```go
func (r *Repo) UpdateVacationBalanceUsedDays(ctx context.Context,
    employeeID int64, vacationTypeID int, year int, daysToAdd float64) error {

    // Start transaction
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Lock the row for update
    var currentUsed, entitled, carriedOver float64
    const selectQuery = `
        SELECT used_days, entitled_days, carried_over_days
        FROM hrm_vacation_balances
        WHERE employee_id = $1 AND vacation_type_id = $2 AND year = $3
        FOR UPDATE`

    err = tx.QueryRowContext(ctx, selectQuery, employeeID, vacationTypeID, year).
        Scan(&currentUsed, &entitled, &carriedOver)
    if err != nil {
        return fmt.Errorf("failed to get balance: %w", err)
    }

    // Validate sufficient balance
    availableDays := entitled + carriedOver - currentUsed
    if daysToAdd > availableDays {
        return storage.ErrInsufficientBalance
    }

    // Update with lock held
    const updateQuery = `
        UPDATE hrm_vacation_balances
        SET used_days = used_days + $1
        WHERE employee_id = $2 AND vacation_type_id = $3 AND year = $4`

    _, err = tx.ExecContext(ctx, updateQuery, daysToAdd, employeeID, vacationTypeID, year)
    if err != nil {
        return fmt.Errorf("failed to update: %w", err)
    }

    return tx.Commit()
}
```

---

### 2.2 Missing Transaction for Vacation Approval

**Location:** `internal/storage/repo/hrm_vacation.go` (ApproveVacation - needs to be added)

**Current state:** Vacation approval and balance update are separate operations.

**Proposed fix:** Create transactional approval:
```go
func (r *Repo) ApproveVacationWithBalanceUpdate(ctx context.Context,
    vacationID int64, approvedBy int64) error {

    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 1. Get vacation details
    var employeeID int64
    var vacationTypeID, year int
    var daysCount float64
    // ... query vacation ...

    // 2. Update vacation status
    _, err = tx.ExecContext(ctx, `
        UPDATE hrm_vacations
        SET status = 'approved', approved_by = $1, approved_at = NOW()
        WHERE id = $2 AND status = 'pending'`, approvedBy, vacationID)
    // ...

    // 3. Update balance (with FOR UPDATE lock)
    // ... same as above ...

    return tx.Commit()
}
```

---

### 2.3 Missing Vacation Overlap Check

**Location:** `internal/storage/repo/hrm_vacation.go:504-519` (AddVacation)

**Current code:** No overlap validation before insert.

**Proposed fix:**
```go
func (r *Repo) AddVacation(ctx context.Context, req hrm.AddVacationRequest) (int64, error) {
    // Check for overlapping vacations first
    const overlapQuery = `
        SELECT COUNT(*) FROM hrm_vacations
        WHERE employee_id = $1
        AND status NOT IN ('rejected', 'cancelled')
        AND (
            (start_date <= $2 AND end_date >= $2) OR  -- new start within existing
            (start_date <= $3 AND end_date >= $3) OR  -- new end within existing
            (start_date >= $2 AND end_date <= $3)     -- existing within new
        )`

    var count int
    err := r.db.QueryRowContext(ctx, overlapQuery,
        req.EmployeeID, req.StartDate, req.EndDate).Scan(&count)
    if err != nil {
        return 0, err
    }
    if count > 0 {
        return 0, storage.ErrVacationOverlap
    }

    // Proceed with insert...
}
```

---

### 2.4 Floating Point for Money

**Location:** `internal/storage/repo/hrm_salary.go` (multiple functions)

**Current code:**
```go
grossAmount := baseAmount + allowancesAmount + bonusesAmount
netAmount := grossAmount - deductionsAmount - taxAmount
```

**Proposed fix options:**

**Option A: Use decimal library**
```go
import "github.com/shopspring/decimal"

grossAmount := decimal.NewFromFloat(baseAmount).
    Add(decimal.NewFromFloat(allowancesAmount)).
    Add(decimal.NewFromFloat(bonusesAmount))
netAmount := grossAmount.
    Sub(decimal.NewFromFloat(deductionsAmount)).
    Sub(decimal.NewFromFloat(taxAmount))
```

**Option B: Store as cents (int64)**
```go
// In database: amount_cents BIGINT
// In code:
grossAmountCents := baseAmountCents + allowancesAmountCents + bonusesAmountCents
netAmountCents := grossAmountCents - deductionsAmountCents - taxAmountCents
// Display: float64(netAmountCents) / 100
```

---

### 2.5 Net Amount Can Be Negative

**Location:** `internal/storage/repo/hrm_salary.go`

**Proposed fix:**
```go
netAmount := grossAmount - deductionsAmount - taxAmount
if netAmount < 0 {
    return 0, storage.ErrNegativeNetAmount
}
```

---

## 3. BUSINESS LOGIC ISSUES

### 3.1 Vacation Days Exceed Balance

**Location:** `internal/storage/repo/hrm_vacation.go:504` (AddVacation)

**Proposed fix:**
```go
func (r *Repo) AddVacation(ctx context.Context, req hrm.AddVacationRequest) (int64, error) {
    // Get current balance
    balance, err := r.GetVacationBalance(ctx, req.EmployeeID, req.VacationTypeID, time.Now().Year())
    if err != nil {
        return 0, err
    }

    availableDays := balance.EntitledDays + balance.CarriedOverDays - balance.UsedDays
    if req.DaysCount > availableDays {
        return 0, storage.ErrInsufficientVacationDays
    }
    // ...
}
```

---

### 3.2 Manager Hierarchy Cycle Detection

**Location:** `internal/storage/repo/hrm_employee.go` (AddEmployee, EditEmployee)

**Proposed fix:**
```go
func (r *Repo) validateManagerHierarchy(ctx context.Context, employeeID, newManagerID int64) error {
    // Walk up the tree to check for cycles
    visited := make(map[int64]bool)
    currentID := newManagerID

    for currentID != 0 {
        if currentID == employeeID {
            return storage.ErrCircularManagerHierarchy
        }
        if visited[currentID] {
            return storage.ErrCircularManagerHierarchy
        }
        visited[currentID] = true

        var managerID sql.NullInt64
        err := r.db.QueryRowContext(ctx,
            `SELECT manager_id FROM hrm_employees WHERE id = $1`, currentID).
            Scan(&managerID)
        if err != nil {
            return err
        }
        if !managerID.Valid {
            break
        }
        currentID = managerID.Int64
    }
    return nil
}
```

---

### 3.3 Substitute Employee Cannot Be Self

**Location:** `internal/storage/repo/hrm_vacation.go:504` (AddVacation)

**Proposed fix:**
```go
if req.SubstituteEmployeeID != nil && *req.SubstituteEmployeeID == req.EmployeeID {
    return 0, storage.ErrSubstituteCannotBeSelf
}
```

---

## 4. CODE QUALITY ISSUES

### 4.1 Duplicate Scan Functions

**Location:** `internal/storage/repo/hrm_employee.go:462-571` and `:574-683`

**Current code:**
```go
// 110 lines - identical except sql.Row vs sql.Rows
func (r *Repo) scanEmployee(row *sql.Row) (*hrmmodel.Employee, error) { ... }
func (r *Repo) scanEmployeeRow(rows *sql.Rows) (*hrmmodel.Employee, error) { ... }
```

**Proposed fix:**
```go
// Define scanner interface
type scanner interface {
    Scan(dest ...interface{}) error
}

// Single implementation
func (r *Repo) scanEmployeeFromScanner(s scanner) (*hrmmodel.Employee, error) {
    var emp hrmmodel.Employee
    var userID, managerID sql.NullInt64
    // ... all the same fields ...

    err := s.Scan(
        &emp.ID, &emp.ContactID, &userID, /* ... */
    )
    if err != nil {
        return nil, err
    }
    // ... process nullable fields ...
    return &emp, nil
}

// Wrapper for sql.Row
func (r *Repo) scanEmployee(row *sql.Row) (*hrmmodel.Employee, error) {
    return r.scanEmployeeFromScanner(row)
}

// Wrapper for sql.Rows
func (r *Repo) scanEmployeeRow(rows *sql.Rows) (*hrmmodel.Employee, error) {
    return r.scanEmployeeFromScanner(rows)
}
```

**Similar duplication in:**
- `hrm_salary.go` (scanSalary/scanSalaryRow)
- `hrm_timesheet.go` (scanTimesheet/scanTimesheetRow)
- `hrm_vacation.go` (scanVacation/scanVacationRow)

---

### 4.2 Missing Database Indexes

**New migration file needed:** `migrations/postgres/000051_hrm_indexes.up.sql`

```sql
-- For salary status filtering
CREATE INDEX idx_hrm_salaries_status ON hrm_salaries(status);

-- For vacation sorting
CREATE INDEX idx_hrm_vacations_requested_at ON hrm_vacations(requested_at DESC);

-- For department filtering via contacts
CREATE INDEX idx_contacts_department_id ON contacts(department_id);

-- For timesheet status filtering
CREATE INDEX idx_hrm_timesheets_status ON hrm_timesheets(status);

-- For access logs time filtering
CREATE INDEX idx_hrm_access_logs_event_time ON hrm_access_logs(event_time DESC);
```

---

## 5. NEW ERROR TYPES NEEDED

**Location:** `internal/storage/errors.go`

```go
var (
    ErrInsufficientBalance       = errors.New("insufficient vacation balance")
    ErrVacationOverlap           = errors.New("vacation dates overlap with existing request")
    ErrCircularManagerHierarchy  = errors.New("circular manager hierarchy detected")
    ErrSubstituteCannotBeSelf    = errors.New("substitute employee cannot be the same as requesting employee")
    ErrNegativeNetAmount         = errors.New("net salary amount cannot be negative")
    ErrInsufficientVacationDays  = errors.New("requested days exceed available vacation balance")
)
```

---

## 6. FILES TO MODIFY (Summary)

| Priority | File | Changes |
|----------|------|---------|
| CRITICAL | `handlers/hrm/salary/salary.go` | Add permission checks |
| CRITICAL | `handlers/hrm/vacation/vacation.go` | Add permission checks |
| CRITICAL | `handlers/hrm/access/access.go` | Fix parsing errors, add auth |
| CRITICAL | `router/router.go` | Add role middleware |
| HIGH | `repo/hrm_vacation.go` | Transactions, overlap check, balance check |
| HIGH | `repo/hrm_salary.go` | Decimal for money, negative check |
| HIGH | `repo/hrm_employee.go` | Manager cycle check |
| MEDIUM | `storage/errors.go` | Add new error types |
| LOW | All `hrm_*.go` | Refactor duplicate scan functions |
| LOW | New migration | Add indexes |

---

## 7. VERIFICATION CHECKLIST

After fixes:
- [ ] Try GET /hrm/salaries?employee_id=OTHER_ID - should return 403
- [ ] Create two concurrent vacation requests for same period - second should fail
- [ ] Request vacation exceeding balance - should return error
- [ ] Set employee.manager_id to form cycle - should return error
- [ ] Calculate salary with large deductions - net should not be negative
- [ ] Run `EXPLAIN ANALYZE` on common queries - should use new indexes
