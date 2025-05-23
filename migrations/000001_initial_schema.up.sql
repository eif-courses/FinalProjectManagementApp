-- migrations/mysql/000001_complete_initial_schema.up.sql

-- Set SQL mode for better MySQL compatibility
SET sql_mode = 'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';

-- Department Heads Table
CREATE TABLE department_heads (
                                  id INT AUTO_INCREMENT PRIMARY KEY,
                                  email VARCHAR(255) NOT NULL UNIQUE,
                                  name VARCHAR(255) NOT NULL DEFAULT '',
                                  sure_name VARCHAR(255) NOT NULL DEFAULT '',
                                  department TEXT NOT NULL,
                                  department_en TEXT NOT NULL,
                                  job_title VARCHAR(255) NOT NULL DEFAULT '',
                                  role INT NOT NULL DEFAULT 0,
                                  is_active TINYINT(1) NOT NULL DEFAULT 1,
                                  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                                  INDEX idx_email (email),
                                  INDEX idx_role (role),
                                  INDEX idx_department (department(255)),
                                  INDEX idx_is_active (is_active)
);

-- Enhanced Commission Members Table
CREATE TABLE commission_members (
                                    id INT AUTO_INCREMENT PRIMARY KEY,
                                    access_code VARCHAR(64) NOT NULL UNIQUE,
                                    department TEXT NOT NULL,
                                    study_program VARCHAR(255) NULL,
                                    year INT NULL,
                                    description TEXT DEFAULT '',
                                    is_active TINYINT(1) DEFAULT 1,
                                    expires_at BIGINT NOT NULL,
                                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                    last_accessed_at BIGINT NULL,
                                    created_by VARCHAR(255) DEFAULT '',
                                    access_count INT DEFAULT 0,
                                    max_access INT DEFAULT 0,

                                    INDEX idx_access_code (access_code),
                                    INDEX idx_department (department(255)),
                                    INDEX idx_study_program (study_program),
                                    INDEX idx_year (year),
                                    INDEX idx_created_by (created_by),
                                    INDEX idx_expires_at (expires_at),
                                    INDEX idx_is_active (is_active)
);

-- Student Records Table
CREATE TABLE student_records (
                                 id INT AUTO_INCREMENT PRIMARY KEY,
                                 student_group VARCHAR(50) NOT NULL,
                                 final_project_title TEXT NOT NULL,
                                 final_project_title_en TEXT NOT NULL,
                                 student_email VARCHAR(255) NOT NULL,
                                 student_name VARCHAR(255) NOT NULL,
                                 student_lastname VARCHAR(255) NOT NULL,
                                 student_number VARCHAR(50) NOT NULL,
                                 supervisor_email VARCHAR(255) NOT NULL,
                                 study_program VARCHAR(255) NOT NULL,
                                 department TEXT NOT NULL,
                                 program_code VARCHAR(50) NOT NULL,
                                 current_year INT NOT NULL,
                                 reviewer_email VARCHAR(255) NOT NULL DEFAULT '',
                                 reviewer_name VARCHAR(255) NOT NULL DEFAULT '',
                                 is_favorite TINYINT(1) NOT NULL DEFAULT 0,
                                 created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                 updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                 INDEX idx_student_email (student_email),
                                 INDEX idx_supervisor_email (supervisor_email),
                                 INDEX idx_reviewer_email (reviewer_email),
                                 INDEX idx_study_program (study_program),
                                 INDEX idx_department (department(255)),
                                 INDEX idx_student_group (student_group),
                                 INDEX idx_current_year (current_year),
                                 INDEX idx_student_number (student_number)
);

-- Documents Table
CREATE TABLE documents (
                           id INT AUTO_INCREMENT PRIMARY KEY,
                           document_type VARCHAR(100) NOT NULL,
                           file_path TEXT NOT NULL,
                           uploaded_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                           student_record_id INT NOT NULL,
                           file_size BIGINT NULL,
                           mime_type VARCHAR(255) NULL,
                           original_filename TEXT NULL,

                           FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                           INDEX idx_student_record (student_record_id),
                           INDEX idx_document_type (document_type),
                           INDEX idx_uploaded_date (uploaded_date)
);

-- Supervisor Reports Table
CREATE TABLE supervisor_reports (
                                    id INT AUTO_INCREMENT PRIMARY KEY,
                                    student_record_id INT NOT NULL,
                                    supervisor_comments TEXT NOT NULL,
                                    supervisor_name VARCHAR(255) NOT NULL DEFAULT '',
                                    supervisor_position VARCHAR(255) NOT NULL DEFAULT '',
                                    supervisor_workplace TEXT NOT NULL,
                                    is_pass_or_failed TINYINT(1) DEFAULT 0,
                                    is_signed TINYINT(1) NOT NULL DEFAULT 0,
                                    other_match DECIMAL(5,2) NOT NULL DEFAULT 0.00,
                                    one_match DECIMAL(5,2) NOT NULL DEFAULT 0.00,
                                    own_match DECIMAL(5,2) NOT NULL DEFAULT 0.00,
                                    join_match DECIMAL(5,2) NOT NULL DEFAULT 0.00,
                                    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                    updated_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                    grade INT NULL,
                                    final_comments TEXT DEFAULT '',

                                    FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                                    INDEX idx_student_record (student_record_id),
                                    INDEX idx_is_signed (is_signed),
                                    INDEX idx_created_date (created_date)
);

-- Reviewer Reports Table
CREATE TABLE reviewer_reports (
                                  id INT AUTO_INCREMENT PRIMARY KEY,
                                  student_record_id INT NOT NULL,
                                  reviewer_personal_details TEXT NOT NULL,
                                  grade DECIMAL(3,1) NOT NULL DEFAULT 0.0,
                                  review_goals TEXT NOT NULL,
                                  review_theory TEXT NOT NULL,
                                  review_practical TEXT NOT NULL,
                                  review_theory_practical_link TEXT NOT NULL,
                                  review_results TEXT NOT NULL,
                                  review_practical_significance TEXT NULL,
                                  review_language TEXT NOT NULL,
                                  review_pros TEXT NOT NULL,
                                  review_cons TEXT NOT NULL,
                                  review_questions TEXT NOT NULL,
                                  is_signed TINYINT(1) NOT NULL DEFAULT 0,
                                  created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                  updated_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                  FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                                  INDEX idx_student_record (student_record_id),
                                  INDEX idx_is_signed (is_signed),
                                  INDEX idx_grade (grade),
                                  INDEX idx_created_date (created_date)
);

-- Videos Table
CREATE TABLE videos (
                        id INT AUTO_INCREMENT PRIMARY KEY,
                        student_record_id INT NOT NULL,
                        `key` VARCHAR(255) NOT NULL,
                        filename VARCHAR(255) NOT NULL,
                        content_type VARCHAR(100) NOT NULL,
                        size BIGINT NULL,
                        url TEXT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        duration INT NULL COMMENT 'Video duration in seconds',
                        status VARCHAR(20) DEFAULT 'pending' COMMENT 'pending, processing, ready, error',

                        FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                        INDEX idx_student_record (student_record_id),
                        INDEX idx_status (status),
                        INDEX idx_created_at (created_at)
);

-- Project Topic Registrations Table
CREATE TABLE project_topic_registrations (
                                             id INT AUTO_INCREMENT PRIMARY KEY,
                                             student_record_id INT NOT NULL,
                                             title TEXT NOT NULL,
                                             title_en TEXT NOT NULL,
                                             problem TEXT NOT NULL,
                                             objective TEXT NOT NULL,
                                             tasks TEXT NOT NULL,
                                             completion_date VARCHAR(100) NULL,
                                             supervisor VARCHAR(255) NOT NULL,
                                             status VARCHAR(20) NOT NULL DEFAULT 'draft',
                                             created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                             updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                             submitted_at BIGINT NULL,
                                             current_version INT DEFAULT 1,
                                             approved_by VARCHAR(255) NULL,
                                             approved_at BIGINT NULL,
                                             rejection_reason TEXT NULL,

                                             FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                                             INDEX idx_student_record (student_record_id),
                                             INDEX idx_status (status),
                                             INDEX idx_supervisor (supervisor),
                                             INDEX idx_submitted_at (submitted_at),
                                             INDEX idx_approved_by (approved_by)
);

-- Topic Registration Comments Table
CREATE TABLE topic_registration_comments (
                                             id INT AUTO_INCREMENT PRIMARY KEY,
                                             topic_registration_id INT NOT NULL,
                                             field_name VARCHAR(100) NULL,
                                             comment_text TEXT NOT NULL,
                                             author_role VARCHAR(50) NOT NULL,
                                             author_name VARCHAR(255) NOT NULL,
                                             author_email VARCHAR(255) NOT NULL DEFAULT '',
                                             created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                             parent_comment_id INT NULL,
                                             is_read TINYINT(1) DEFAULT 1,
                                             comment_type VARCHAR(20) DEFAULT 'comment' COMMENT 'comment, suggestion, approval, rejection',

                                             FOREIGN KEY (topic_registration_id) REFERENCES project_topic_registrations(id) ON DELETE CASCADE,
                                             FOREIGN KEY (parent_comment_id) REFERENCES topic_registration_comments(id) ON DELETE SET NULL,
                                             INDEX idx_topic_registration (topic_registration_id),
                                             INDEX idx_author_email (author_email),
                                             INDEX idx_created_at (created_at),
                                             INDEX idx_parent_comment (parent_comment_id)
);

-- Project Topic Registration Versions Table
CREATE TABLE project_topic_registration_versions (
                                                     id INT AUTO_INCREMENT PRIMARY KEY,
                                                     topic_registration_id INT NOT NULL,
                                                     version_data LONGTEXT NOT NULL,
                                                     created_by VARCHAR(255) NOT NULL,
                                                     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                                     version_number INT NOT NULL,
                                                     change_summary TEXT DEFAULT '',

                                                     FOREIGN KEY (topic_registration_id) REFERENCES project_topic_registrations(id) ON DELETE CASCADE,
                                                     INDEX idx_topic_registration (topic_registration_id),
                                                     INDEX idx_created_by (created_by),
                                                     INDEX idx_created_at (created_at),
                                                     INDEX idx_version_number (version_number)
);

-- User Sessions Table
CREATE TABLE user_sessions (
                               id INT AUTO_INCREMENT PRIMARY KEY,
                               session_id VARCHAR(128) NOT NULL UNIQUE,
                               user_email VARCHAR(255) NOT NULL,
                               user_data LONGTEXT NOT NULL COMMENT 'JSON blob of user data',
                               created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                               last_accessed TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                               expires_at BIGINT NOT NULL,
                               ip_address VARCHAR(45) NULL,
                               user_agent TEXT NULL,
                               is_active TINYINT(1) DEFAULT 1,

                               INDEX idx_session_id (session_id),
                               INDEX idx_user_email (user_email),
                               INDEX idx_expires_at (expires_at),
                               INDEX idx_is_active (is_active)
);

-- Audit Logs Table
CREATE TABLE audit_logs (
                            id INT AUTO_INCREMENT PRIMARY KEY,
                            user_email VARCHAR(255) NOT NULL,
                            user_role VARCHAR(50) NOT NULL,
                            action VARCHAR(100) NOT NULL,
                            resource_type VARCHAR(50) NOT NULL,
                            resource_id VARCHAR(100) NULL,
                            details JSON NULL COMMENT 'JSON blob for additional details',
                            ip_address VARCHAR(45) NULL,
                            user_agent TEXT NULL,
                            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                            success TINYINT(1) DEFAULT 1,

                            INDEX idx_user_email (user_email),
                            INDEX idx_action (action),
                            INDEX idx_resource_type (resource_type),
                            INDEX idx_created_at (created_at),
                            INDEX idx_success (success)
);

-- Role Permissions Table
CREATE TABLE role_permissions (
                                  id INT AUTO_INCREMENT PRIMARY KEY,
                                  role_name VARCHAR(50) NOT NULL,
                                  permission VARCHAR(100) NOT NULL,
                                  resource_type VARCHAR(50) NULL,
                                  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                                  UNIQUE KEY unique_permission (role_name, permission, resource_type),
                                  INDEX idx_role_name (role_name),
                                  INDEX idx_permission (permission)
);

-- User Preferences Table
CREATE TABLE user_preferences (
                                  id INT AUTO_INCREMENT PRIMARY KEY,
                                  user_email VARCHAR(255) NOT NULL UNIQUE,
                                  language VARCHAR(5) DEFAULT 'lt',
                                  theme VARCHAR(20) DEFAULT 'light',
                                  notifications_enabled TINYINT(1) DEFAULT 1,
                                  email_notifications TINYINT(1) DEFAULT 1,
                                  timezone VARCHAR(50) DEFAULT 'Europe/Vilnius',
                                  preferences_json JSON NULL COMMENT 'Additional JSON preferences',
                                  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                  INDEX idx_user_email (user_email)
);

-- OAuth States Table
CREATE TABLE oauth_states (
                              id INT AUTO_INCREMENT PRIMARY KEY,
                              state_value VARCHAR(128) NOT NULL UNIQUE,
                              created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                              expires_at BIGINT NOT NULL,
                              used TINYINT(1) DEFAULT 0,
                              ip_address VARCHAR(45) NULL,

                              INDEX idx_state_value (state_value),
                              INDEX idx_expires_at (expires_at),
                              INDEX idx_used (used)
);

-- Views for easier data access

-- Commission Access View
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

-- User Roles View
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

-- Student Summary View
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

-- Insert default role permissions
INSERT INTO role_permissions (role_name, permission, resource_type) VALUES
-- Admin permissions (role 0)
('admin', 'full_access', NULL),
('admin', 'manage_users', NULL),
('admin', 'system_config', NULL),
('admin', 'view_all_students', NULL),
('admin', 'approve_topics', NULL),
('admin', 'manage_department', NULL),
('admin', 'generate_reports', NULL),

-- Department Head permissions (role 1)
('department_head_1', 'view_all_students', NULL),
('department_head_1', 'approve_topics', NULL),
('department_head_1', 'manage_department', NULL),
('department_head_1', 'generate_reports', NULL),
('department_head_1', 'view_department_reports', NULL),
('department_head_1', 'manage_commission', NULL),

-- Deputy Head permissions (role 2)
('department_head_2', 'view_all_students', NULL),
('department_head_2', 'approve_topics', NULL),
('department_head_2', 'generate_reports', NULL),
('department_head_2', 'view_department_reports', NULL),
('department_head_2', 'manage_commission', NULL),

-- Secretary permissions (role 3)
('department_head_3', 'view_all_students', NULL),
('department_head_3', 'generate_reports', NULL),
('department_head_3', 'view_department_reports', NULL),

-- Coordinator permissions (role 4)
('department_head_4', 'view_all_students', NULL),
('department_head_4', 'approve_topics', NULL),
('department_head_4', 'generate_reports', NULL),

-- Supervisor permissions
('supervisor', 'view_assigned_students', NULL),
('supervisor', 'create_reports', NULL),
('supervisor', 'review_submissions', NULL),

-- Commission Member permissions
('commission_member', 'view_thesis', NULL),
('commission_member', 'evaluate_defense', NULL),

-- Student permissions
('student', 'view_own_data', NULL),
('student', 'submit_topic', NULL),
('student', 'upload_documents', NULL);

-- Sample data for department heads (adjust emails as needed)
INSERT INTO department_heads (email, name, sure_name, department, department_en, job_title, role, is_active) VALUES
                                                                                                                 ('j.petraitis@viko.lt', 'Jonas', 'Petraitis', 'Informacijos technologijų katedra', 'Information Technology Department', 'Katedros vedėjas', 1, 1),
                                                                                                                 ('r.kazlauskiene@viko.lt', 'Rasa', 'Kazlauskienė', 'Verslo vadybos katedra', 'Business Management Department', 'Katedros vedėja', 1, 1),
                                                                                                                 ('m.gzegozevskis@eif.viko.lt', 'Maksim', 'Gžegožewski', 'Elektronikos ir informatikos fakultetas', 'Faculty of Electronics and Informatics', 'Sistemos administratorius', 0, 1);