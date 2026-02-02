-- Rollback HRM Module Performance Indexes

DROP INDEX IF EXISTS idx_hrm_salaries_status;
DROP INDEX IF EXISTS idx_hrm_vacations_requested_at;
DROP INDEX IF EXISTS idx_hrm_vacations_status;
DROP INDEX IF EXISTS idx_hrm_vacations_employee_id;
DROP INDEX IF EXISTS idx_contacts_department_id;
DROP INDEX IF EXISTS idx_hrm_timesheets_status;
DROP INDEX IF EXISTS idx_hrm_access_logs_event_time;
DROP INDEX IF EXISTS idx_hrm_access_logs_employee_id;
DROP INDEX IF EXISTS idx_hrm_employees_manager_id;
DROP INDEX IF EXISTS idx_hrm_vacation_balances_employee_year;
DROP INDEX IF EXISTS idx_hrm_salaries_employee_period;
