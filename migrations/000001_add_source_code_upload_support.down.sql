-- ================================================
-- Migration DOWN: Remove Source Code Upload Support
-- File: 000001_add_source_code_upload_support.down.sql
-- ================================================

-- Remove source code upload permissions
DELETE FROM role_permissions
WHERE permission IN (
                     'manage_source_uploads',
                     'view_all_repositories',
                     'view_source_uploads',
                     'download_source_code',
                     'view_student_source',
                     'download_student_source',
                     'view_thesis_source',
                     'view_thesis_source_code',
                     'upload_source_code',
                     'view_own_source_uploads'
    ) AND resource_type = 'documents';

-- Recreate original student_summary_view
DROP VIEW IF EXISTS student_summary_view;

CREATE VIEW student_summary_view AS
SELECT
    sr.*,
    CASE
        WHEN ptr.status = 'approved' THEN 1
        ELSE 0
        END as topic_approved,
    ptr.status as topic_status,
    ptr.approved_by,
    ptr.approved_at,
    CASE
        WHEN sup_rep.id IS NOT NULL THEN 1
        ELSE 0
        END as has_supervisor_report,
    CASE
        WHEN rev_rep.id IS NOT NULL THEN 1
        ELSE 0
        END as has_reviewer_report,
    CASE
        WHEN v.id IS NOT NULL THEN 1
        ELSE 0
        END as has_video,
    sup_rep.is_signed as supervisor_report_signed,
    rev_rep.is_signed as reviewer_report_signed,
    rev_rep.grade as reviewer_grade
FROM student_records sr
         LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
         LEFT JOIN supervisor_reports sup_rep ON sr.id = sup_rep.student_record_id
         LEFT JOIN reviewer_reports rev_rep ON sr.id = rev_rep.student_record_id
         LEFT JOIN videos v ON sr.id = v.student_record_id;

-- Remove indexes first
DROP INDEX idx_upload_status ON documents;
DROP INDEX idx_submission_id ON documents;
DROP INDEX idx_repository_id ON documents;

-- Remove columns from documents table
ALTER TABLE documents
    DROP COLUMN repository_url,
    DROP COLUMN repository_id,
    DROP COLUMN commit_id,
    DROP COLUMN submission_id,
    DROP COLUMN validation_status,
    DROP COLUMN validation_errors,
    DROP COLUMN upload_status;

-- Log rollback
INSERT INTO audit_logs (user_email, user_role, action, resource_type, details, created_at)
VALUES ('system', 'admin', 'migration_rolled_back', 'documents', 'Removed source code upload support from documents table', NOW());