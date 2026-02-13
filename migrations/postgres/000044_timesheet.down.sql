DROP TRIGGER IF EXISTS set_timesheet_corrections_updated_at ON timesheet_corrections;
DROP TRIGGER IF EXISTS set_timesheet_entries_updated_at ON timesheet_entries;

DROP TABLE IF EXISTS timesheet_corrections;
DROP TABLE IF EXISTS holidays;
DROP TABLE IF EXISTS timesheet_entries;
