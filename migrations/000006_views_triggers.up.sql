-- ================================================
-- Migration UP: Views and Triggers
-- File: 000006_views_triggers.up.sql
-- ================================================

-- Create views with COALESCE for NULL handling
CREATE OR REPLACE VIEW commission_access_view AS
SELECT
    cm.*,
    CASE
        WHEN cm.expires_at <= UNIX_TIMESTAMP() THEN 1
        ELSE 0
        END as is_expired,
    CASE
        WHEN cm.max_access > 0 AND cm.access_count >= cm.max_access THEN 1
        ELSE 0
        END as is_access_limit_reached,
    FROM_UNIXTIME(cm.expires_at) as expires_at_formatted,
    cm.created_at as created_at_formatted,
    CASE
        WHEN cm.last_accessed_at IS NOT NULL THEN FROM_UNIXTIME(cm.last_accessed_at)
        ELSE NULL
        END as last_accessed_at_formatted
FROM commission_members cm;

CREATE OR REPLACE VIEW user_roles_view AS
SELECT
    dh.email,
    COALESCE(dh.name, '') as name,
    COALESCE(dh.sure_name, '') as sure_name,
    dh.department,
    dh.department_en,
    COALESCE(dh.job_title, '') as job_title,
    'department_head' as role_type,
    dh.role as role_id,
    dh.is_active,
    CASE
        WHEN dh.role = 0 THEN 'admin'
        WHEN dh.role = 1 THEN 'department_head'
        WHEN dh.role = 2 THEN 'deputy_head'
        WHEN dh.role = 3 THEN 'secretary'
        WHEN dh.role = 4 THEN 'coordinator'
        ELSE 'unknown'
        END as role_name,
    dh.created_at as created_at_formatted
FROM department_heads dh
WHERE dh.is_active = 1;

CREATE OR REPLACE VIEW student_summary_view AS
SELECT
    sr.id,
    sr.student_group,
    sr.student_name,
    sr.student_lastname,
    sr.student_email,
    sr.final_project_title,
    COALESCE(sr.final_project_title_en, '') as final_project_title_en,
    sr.supervisor_email,
    COALESCE(sr.reviewer_name, '') as reviewer_name,
    COALESCE(sr.reviewer_email, '') as reviewer_email,
    sr.study_program,
    sr.department,
    sr.current_year,
    sr.program_code,
    sr.student_number,
    sr.is_favorite,
    sr.is_public_defense,
    sr.defense_date,
    COALESCE(sr.defense_location, '') as defense_location,
    sr.created_at,
    sr.updated_at,
    CASE
        WHEN ptr.status = 'approved' THEN 1
        ELSE 0
        END as topic_approved,
    COALESCE(ptr.status, '') as topic_status,
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
    rev_rep.review_questions as reviewer_questions,
    source_doc.repository_url,
    source_doc.upload_status as source_upload_status,
    source_doc.validation_status as source_validation_status
FROM student_records sr
         LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
         LEFT JOIN supervisor_reports sup_rep ON sr.id = sup_rep.student_record_id
         LEFT JOIN reviewer_reports rev_rep ON sr.id = rev_rep.student_record_id
         LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
         LEFT JOIN documents source_doc ON sr.id = source_doc.student_record_id
    AND source_doc.document_type = 'thesis_source_code';