-- HRM Module Performance Indexes

-- For salary status filtering
CREATE INDEX IF NOT EXISTS idx_hrm_salaries_status ON hrm_salaries(status);

-- For vacation sorting by requested_at
CREATE INDEX IF NOT EXISTS idx_hrm_vacations_requested_at ON hrm_vacations(requested_at DESC);

-- For vacation status filtering
CREATE INDEX IF NOT EXISTS idx_hrm_vacations_status ON hrm_vacations(status);

-- For vacation employee lookup
CREATE INDEX IF NOT EXISTS idx_hrm_vacations_employee_id ON hrm_vacations(employee_id);

-- For department filtering via contacts
CREATE INDEX IF NOT EXISTS idx_contacts_department_id ON contacts(department_id);

-- For timesheet status filtering
CREATE INDEX IF NOT EXISTS idx_hrm_timesheets_status ON hrm_timesheets(status);

-- For access logs time filtering
CREATE INDEX IF NOT EXISTS idx_hrm_access_logs_event_time ON hrm_access_logs(event_time DESC);

-- For access logs employee lookup
CREATE INDEX IF NOT EXISTS idx_hrm_access_logs_employee_id ON hrm_access_logs(employee_id);

-- For employee manager hierarchy queries
CREATE INDEX IF NOT EXISTS idx_hrm_employees_manager_id ON hrm_employees(manager_id);

-- For vacation balance lookups
CREATE INDEX IF NOT EXISTS idx_hrm_vacation_balances_employee_year ON hrm_vacation_balances(employee_id, year);

-- For salary period lookups
CREATE INDEX IF NOT EXISTS idx_hrm_salaries_employee_period ON hrm_salaries(employee_id, year, month);
