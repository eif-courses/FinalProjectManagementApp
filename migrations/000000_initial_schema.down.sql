-- ================================================
-- Migration DOWN: Drop all tables and views
-- File: 000000_initial_schema_with_null_fixes.down.sql
-- ================================================

-- Drop triggers first
DROP TRIGGER IF EXISTS before_student_records_insert;
DROP TRIGGER IF EXISTS before_student_records_update;

-- Drop views
DROP VIEW IF EXISTS student_summary_view;
DROP VIEW IF EXISTS user_roles_view;
DROP VIEW IF EXISTS commission_access_view;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS project_topic_registration_versions;
DROP TABLE IF EXISTS topic_registration_comments;
DROP TABLE IF EXISTS project_topic_registrations;
DROP TABLE IF EXISTS videos;
DROP TABLE IF EXISTS academic_audit_logs;
DROP TABLE IF EXISTS commission_evaluations;
DROP TABLE IF EXISTS commission_access_logs;
DROP TABLE IF EXISTS reviewer_reports;
DROP TABLE IF EXISTS supervisor_reports;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS reviewer_access_tokens;
DROP TABLE IF EXISTS student_records;
DROP TABLE IF EXISTS commission_members;
DROP TABLE IF EXISTS department_heads;