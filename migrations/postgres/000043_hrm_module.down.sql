BEGIN;

DROP TABLE IF EXISTS department_blocked_periods CASCADE;
DROP TABLE IF EXISTS hrm_notifications CASCADE;
DROP TABLE IF EXISTS vacation_balances CASCADE;
DROP TABLE IF EXISTS vacations CASCADE;
DROP TABLE IF EXISTS personnel_transfers CASCADE;
DROP TABLE IF EXISTS personnel_documents CASCADE;
DROP TABLE IF EXISTS personnel_records CASCADE;

DELETE FROM roles WHERE name IN ('hrm_admin', 'hrm_manager', 'hrm_employee');

COMMIT;
