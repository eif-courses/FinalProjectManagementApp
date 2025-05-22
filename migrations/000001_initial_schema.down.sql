-- migrations/000001_initial_schema.down.sql

-- Drop tables in reverse order due to foreign key constraints
DROP TABLE IF EXISTS project_topic_registration_versions;
DROP TABLE IF EXISTS topic_registration_comments;
DROP TABLE IF EXISTS project_topic_registrations;
DROP TABLE IF EXISTS videos;
DROP TABLE IF EXISTS reviewer_reports;
DROP TABLE IF EXISTS supervisor_reports;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS student_records;
DROP TABLE IF EXISTS commission_members;
DROP TABLE IF EXISTS department_heads;

-- Drop indexes (SQLite automatically drops indexes when tables are dropped,
-- but it's good practice to be explicit)
DROP INDEX IF EXISTS department_heads_email_idx;
DROP INDEX IF EXISTS department_heads_role_idx;
DROP INDEX IF EXISTS student_email_idx;
DROP INDEX IF EXISTS supervisor_email_idx;
DROP INDEX IF EXISTS reviewer_email_idx;
DROP INDEX IF EXISTS study_program_idx;
DROP INDEX IF EXISTS department_idx;
DROP INDEX IF EXISTS documents_student_record_idx;
DROP INDEX IF EXISTS supervisor_reports_student_record_idx;
DROP INDEX IF EXISTS reviewer_reports_student_record_idx;
DROP INDEX IF EXISTS videos_student_record_idx;