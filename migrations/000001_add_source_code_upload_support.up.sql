-- ================================================
-- Migration UP: Add Source Code Upload Support
-- File: 000001_add_source_code_upload_support.up.sql
-- ================================================

-- Add source code upload columns to documents table
ALTER TABLE documents
    ADD COLUMN repository_url TEXT NULL COMMENT 'Azure DevOps repository URL',
    ADD COLUMN repository_id VARCHAR(255) NULL COMMENT 'Azure DevOps repository ID',
    ADD COLUMN commit_id VARCHAR(255) NULL COMMENT 'Git commit ID',
    ADD COLUMN submission_id VARCHAR(36) NULL COMMENT 'Unique submission identifier',
    ADD COLUMN validation_status ENUM('pending', 'valid', 'invalid') DEFAULT 'pending',
    ADD COLUMN validation_errors TEXT NULL COMMENT 'Validation error messages',
    ADD COLUMN upload_status ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending',
    ADD INDEX idx_repository_id (repository_id),
    ADD INDEX idx_submission_id (submission_id),
    ADD INDEX idx_upload_status (upload_status);

-- Add new document type for source code uploads
INSERT INTO role_permissions (role_name, permission, resource_type) VALUES
                                                                        ('admin', 'manage_source_uploads', 'documents'),
                                                                        ('admin', 'view_all_repositories', 'documents'),
                                                                        ('department_head_1', 'view_source_uploads', 'documents'),
                                                                        ('department_head_1', 'download_source_code', 'documents'),
                                                                        ('department_head_2', 'view_source_uploads', 'documents'),
                                                                        ('department_head_2', 'download_source_code', 'documents'),
                                                                        ('supervisor', 'view_student_source', 'documents'),
                                                                        ('supervisor', 'download_student_source', 'documents'),
                                                                        ('commission_member', 'view_thesis_source', 'documents'),
                                                                        ('reviewer', 'view_thesis_source_code', 'documents'),
                                                                        ('student', 'upload_source_code', 'documents'),
                                                                        ('student', 'view_own_source_uploads', 'documents');

-- Update existing views to include source code information
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
    CASE
        WHEN source_doc.id IS NOT NULL THEN 1
        ELSE 0
        END as has_source_code,
    sup_rep.is_signed as supervisor_report_signed,
    rev_rep.is_signed as reviewer_report_signed,
    rev_rep.grade as reviewer_grade,
    source_doc.repository_url,
    source_doc.upload_status as source_upload_status,
    source_doc.validation_status as source_validation_status
FROM student_records sr
         LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
         LEFT JOIN supervisor_reports sup_rep ON sr.id = sup_rep.student_record_id
         LEFT JOIN reviewer_reports rev_rep ON sr.id = rev_rep.student_record_id
         LEFT JOIN videos v ON sr.id = v.student_record_id
         LEFT JOIN documents source_doc ON sr.id = source_doc.student_record_id
    AND source_doc.document_type = 'thesis_source_code';

-- Create source code upload audit log entries
INSERT INTO audit_logs (user_email, user_role, action, resource_type, details, created_at)
VALUES ('system', 'admin', 'migration_applied', 'documents', 'Added source code upload support to documents table', NOW());