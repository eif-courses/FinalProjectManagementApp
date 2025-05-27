-- ================================================
-- Migration DOWN: Drop Complete Initial Schema
-- File: 000001_initial_schema.down.sql
-- ================================================

-- Disable foreign key checks to avoid dependency issues
SET FOREIGN_KEY_CHECKS = 0;

-- Drop views first
DROP VIEW IF EXISTS student_summary_view;
DROP VIEW IF EXISTS user_roles_view;
DROP VIEW IF EXISTS commission_access_view;

-- Drop all tables in reverse order of creation to handle dependencies
DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS audit_logs;
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
DROP TABLE IF EXISTS reviewer_invitations;
DROP TABLE IF EXISTS student_records;
DROP TABLE IF EXISTS commission_members;
DROP TABLE IF EXISTS department_heads;

-- Re-enable foreign key checks
SET FOREIGN_KEY_CHECKS = 1;

-- Migration rollback completed successfully