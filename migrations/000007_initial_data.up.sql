-- ================================================
-- Migration UP: Initial Data
-- File: 000007_initial_data.up.sql
-- ================================================

-- Insert comprehensive role permissions
INSERT INTO role_permissions (role_name, permission, resource_type) VALUES
                                                                        ('admin', 'full_access', NULL),
                                                                        ('admin', 'manage_users', NULL),
                                                                        ('admin', 'system_config', NULL),
                                                                        ('admin', 'view_all_students', NULL),
                                                                        ('admin', 'approve_topics', NULL),
                                                                        ('admin', 'manage_department', NULL),
                                                                        ('admin', 'generate_reports', NULL),
                                                                        ('admin', 'manage_commission_access', 'commission_members'),
                                                                        ('admin', 'view_academic_audit_logs', 'academic_audit_logs'),
                                                                        ('admin', 'generate_commission_reports', 'commission_evaluations'),
                                                                        ('admin', 'manage_source_uploads', 'documents'),
                                                                        ('admin', 'view_all_repositories', 'documents'),
                                                                        ('admin', 'manage_reviewer_access_tokens', 'reviewer_access_tokens'),
                                                                        ('department_head_1', 'view_all_students', NULL),
                                                                        ('department_head_1', 'approve_topics', NULL),
                                                                        ('department_head_1', 'manage_department', NULL),
                                                                        ('department_head_1', 'generate_reports', NULL),
                                                                        ('department_head_1', 'view_department_reports', NULL),
                                                                        ('department_head_1', 'manage_commission', NULL),
                                                                        ('department_head_1', 'create_commission_access', 'commission_members'),
                                                                        ('department_head_1', 'view_commission_evaluations', 'commission_evaluations'),
                                                                        ('department_head_1', 'view_source_uploads', 'documents'),
                                                                        ('department_head_1', 'download_source_code', 'documents'),
                                                                        ('department_head_1', 'create_reviewer_access_tokens', 'reviewer_access_tokens'),
                                                                        ('department_head_2', 'view_all_students', NULL),
                                                                        ('department_head_2', 'approve_topics', NULL),
                                                                        ('department_head_2', 'generate_reports', NULL),
                                                                        ('department_head_2', 'view_department_reports', NULL),
                                                                        ('department_head_2', 'manage_commission', NULL),
                                                                        ('department_head_2', 'create_commission_access', 'commission_members'),
                                                                        ('department_head_2', 'view_commission_evaluations', 'commission_evaluations'),
                                                                        ('department_head_2', 'view_source_uploads', 'documents'),
                                                                        ('department_head_2', 'download_source_code', 'documents'),
                                                                        ('department_head_2', 'create_reviewer_access_tokens', 'reviewer_access_tokens'),
                                                                        ('department_head_3', 'view_all_students', NULL),
                                                                        ('department_head_3', 'generate_reports', NULL),
                                                                        ('department_head_3', 'view_department_reports', NULL),
                                                                        ('department_head_4', 'view_all_students', NULL),
                                                                        ('department_head_4', 'approve_topics', NULL),
                                                                        ('department_head_4', 'generate_reports', NULL),
                                                                        ('supervisor', 'view_assigned_students', NULL),
                                                                        ('supervisor', 'create_reports', NULL),
                                                                        ('supervisor', 'review_submissions', NULL),
                                                                        ('supervisor', 'approve_student_topics', NULL),
                                                                        ('supervisor', 'view_student_source', 'documents'),
                                                                        ('supervisor', 'download_student_source', 'documents'),
                                                                        ('commission_member', 'view_thesis', NULL),
                                                                        ('commission_member', 'evaluate_defense', NULL),
                                                                        ('commission_member', 'view_defense_materials', 'documents'),
                                                                        ('commission_member', 'submit_evaluation', 'commission_evaluations'),
                                                                        ('commission_member', 'view_student_summary', 'student_records'),
                                                                        ('commission_member', 'view_thesis_source', 'documents'),
                                                                        ('reviewer', 'view_thesis_materials', 'documents'),
                                                                        ('reviewer', 'submit_review', 'reviewer_reports'),
                                                                        ('reviewer', 'download_thesis', 'documents'),
                                                                        ('reviewer', 'view_thesis_source_code', 'documents'),
                                                                        ('student', 'view_own_data', NULL),
                                                                        ('student', 'submit_topic', NULL),
                                                                        ('student', 'upload_documents', NULL),
                                                                        ('student', 'upload_source_code', 'documents'),
                                                                        ('student', 'view_own_source_uploads', 'documents');

-- Insert initial department heads
INSERT INTO department_heads (email, name, sure_name, department, department_en, job_title, role, is_active) VALUES
                                                                                                                 ('t.liogiene@eif.viko.lt', 'Tatjana', 'Liogienė', 'Informacinių sistemų', 'Information Systems', 'Katedros vedėja', 1, 1),
                                                                                                                 ('a.kirdeikiene@eif.viko.lt', 'Aliona', 'Kirdeikienė', 'Elektronikos ir kompiuterių inžinerijos', 'Electronics and Computer Engineering', 'Katedros vedėja', 1, 1),
                                                                                                                 ('m.gzegozevskis@eif.viko.lt', 'Marius', 'Gžegoževskis', 'Programinės įrangos', 'Software Engineering', 'Sistemos administratorius', 1, 1),
                                                                                                                 ('j.zailskas@eif.viko.lt', 'Justinas', 'Zailskas', 'Programinės įrangos', 'Software Engineering', 'Katedros vedėjas', 1, 1);

-- Create audit log entry for schema creation
INSERT INTO audit_logs (user_email, user_role, action, resource_type, details, success)
VALUES ('system', 'admin', 'schema_created', 'database', 'Complete initial schema with NULL handling created', TRUE);