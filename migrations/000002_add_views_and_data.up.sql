-- Create views
CREATE VIEW commission_access_view AS
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

CREATE VIEW user_roles_view AS
SELECT
    dh.email,
    dh.name,
    dh.sure_name,
    dh.department,
    dh.department_en,
    dh.job_title,
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

-- Insert initial data
INSERT INTO role_permissions (role_name, permission, resource_type) VALUES
                                                                        ('admin', 'full_access', NULL),
                                                                        ('admin', 'manage_users', NULL),
                                                                        ('admin', 'system_config', NULL),
                                                                        ('admin', 'view_all_students', NULL),
                                                                        ('admin', 'approve_topics', NULL),
                                                                        ('admin', 'manage_department', NULL),
                                                                        ('admin', 'generate_reports', NULL),
                                                                        ('department_head_1', 'view_all_students', NULL),
                                                                        ('department_head_1', 'approve_topics', NULL),
                                                                        ('department_head_1', 'manage_department', NULL),
                                                                        ('department_head_1', 'generate_reports', NULL),
                                                                        ('department_head_1', 'view_department_reports', NULL),
                                                                        ('department_head_1', 'manage_commission', NULL),
                                                                        ('department_head_2', 'view_all_students', NULL),
                                                                        ('department_head_2', 'approve_topics', NULL),
                                                                        ('department_head_2', 'generate_reports', NULL),
                                                                        ('department_head_2', 'view_department_reports', NULL),
                                                                        ('department_head_2', 'manage_commission', NULL),
                                                                        ('department_head_3', 'view_all_students', NULL),
                                                                        ('department_head_3', 'generate_reports', NULL),
                                                                        ('department_head_3', 'view_department_reports', NULL),
                                                                        ('department_head_4', 'view_all_students', NULL),
                                                                        ('department_head_4', 'approve_topics', NULL),
                                                                        ('department_head_4', 'generate_reports', NULL),
                                                                        ('supervisor', 'view_assigned_students', NULL),
                                                                        ('supervisor', 'create_reports', NULL),
                                                                        ('supervisor', 'review_submissions', NULL),
                                                                        ('commission_member', 'view_thesis', NULL),
                                                                        ('commission_member', 'evaluate_defense', NULL),
                                                                        ('student', 'view_own_data', NULL),
                                                                        ('student', 'submit_topic', NULL),
                                                                        ('student', 'upload_documents', NULL);

INSERT INTO department_heads (email, name, sure_name, department, department_en, job_title, role, is_active) VALUES
                                                                                                                 ('j.petraitis@viko.lt', 'Jonas', 'Petraitis', 'Informacijos technologijų katedra', 'Information Technology Department', 'Katedros vedėjas', 1, 1),
                                                                                                                 ('r.kazlauskiene@viko.lt', 'Rasa', 'Kazlauskienė', 'Verslo vadybos katedra', 'Business Management Department', 'Katedros vedėja', 1, 1),
                                                                                                                 ('m.gzegozevskis@eif.viko.lt', 'Maksim', 'Gžegožewski', 'Elektronikos ir informatikos fakultetas', 'Faculty of Electronics and Informatics', 'Sistemos administratorius', 0, 1);