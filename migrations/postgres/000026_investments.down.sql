-- Drop tables in reverse order (respect foreign keys)
DROP TABLE IF EXISTS investment_file_links;
DROP TRIGGER IF EXISTS set_timestamp ON investments;
DROP TABLE IF EXISTS investments;
DROP TABLE IF EXISTS investment_status;
