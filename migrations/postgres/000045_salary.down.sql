DROP TRIGGER IF EXISTS set_timestamp_salary_structures ON salary_structures;
DROP TRIGGER IF EXISTS set_timestamp_salaries ON salaries;

DROP INDEX IF EXISTS idx_salary_deductions_salary_id;
DROP INDEX IF EXISTS idx_salary_bonuses_salary_id;
DROP INDEX IF EXISTS idx_salary_structures_employee_id;
DROP INDEX IF EXISTS idx_salaries_status;
DROP INDEX IF EXISTS idx_salaries_period;
DROP INDEX IF EXISTS idx_salaries_employee_id;

DROP TABLE IF EXISTS salary_deductions;
DROP TABLE IF EXISTS salary_bonuses;
DROP TABLE IF EXISTS salaries;
DROP TABLE IF EXISTS salary_structures;
