-- ================================================
-- Migration UP: Complete Initial Schema with NULL Handling
-- File: 000000_initial_schema_with_null_fixes.up.sql
-- ================================================

-- Create department heads table
CREATE TABLE department_heads (
                                  id INT AUTO_INCREMENT PRIMARY KEY,
                                  email VARCHAR(255) NOT NULL UNIQUE,
                                  name VARCHAR(255) NULL DEFAULT '',
                                  sure_name VARCHAR(255) NULL DEFAULT '',
                                  department TEXT NOT NULL,
                                  department_en TEXT NOT NULL,
                                  job_title VARCHAR(255) NULL DEFAULT '',
                                  role INT NOT NULL DEFAULT 0,
                                  is_active BOOLEAN NOT NULL DEFAULT TRUE,
                                  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                                  INDEX idx_email (email),
                                  INDEX idx_role (role),
                                  INDEX idx_department (department(100)),
                                  INDEX idx_is_active (is_active)
);

-- Create commission members table with all enhancements
CREATE TABLE commission_members (
                                    id INT AUTO_INCREMENT PRIMARY KEY,
                                    access_code VARCHAR(64) NOT NULL UNIQUE,
                                    department TEXT NOT NULL,
                                    study_program VARCHAR(255) NULL,
                                    year INT NULL,
                                    description TEXT NULL,
                                    is_active BOOLEAN DEFAULT TRUE,
                                    expires_at BIGINT NOT NULL,
                                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                    last_accessed_at BIGINT NULL,
                                    created_by VARCHAR(255) NULL DEFAULT '',
                                    access_count INT DEFAULT 0,
                                    max_access INT DEFAULT 0,
                                    allowed_student_groups TEXT NULL COMMENT 'Comma-separated list of allowed groups',
                                    allowed_study_programs TEXT NULL COMMENT 'Comma-separated list of allowed programs',
                                    access_level ENUM('view_only', 'evaluate', 'full') DEFAULT 'view_only',
                                    commission_type VARCHAR(50) DEFAULT 'defense' COMMENT 'defense, review, evaluation',

                                    INDEX idx_access_code (access_code),
                                    INDEX idx_department (department(100)),
                                    INDEX idx_study_program (study_program),
                                    INDEX idx_year (year),
                                    INDEX idx_created_by (created_by),
                                    INDEX idx_expires_at (expires_at),
                                    INDEX idx_is_active (is_active),
                                    INDEX idx_commission_type (commission_type),
                                    INDEX idx_access_level (access_level)
);

-- Create student records table with proper NULL handling
CREATE TABLE student_records (
                                 id INT AUTO_INCREMENT PRIMARY KEY,
                                 student_group VARCHAR(50) NOT NULL,
                                 final_project_title TEXT NOT NULL,
                                 final_project_title_en TEXT NULL DEFAULT '',
                                 student_email VARCHAR(255) NOT NULL,
                                 student_name VARCHAR(255) NOT NULL,
                                 student_lastname VARCHAR(255) NOT NULL,
                                 student_number VARCHAR(50) NOT NULL,
                                 supervisor_email VARCHAR(255) NOT NULL,
                                 study_program VARCHAR(255) NOT NULL,
                                 department TEXT NOT NULL,
                                 program_code VARCHAR(50) NOT NULL,
                                 current_year INT NOT NULL,
                                 reviewer_email VARCHAR(255) NULL DEFAULT '',
                                 reviewer_name VARCHAR(255) NULL DEFAULT '',
                                 is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
                                 is_public_defense BOOLEAN DEFAULT FALSE,
                                 defense_date TIMESTAMP NULL,
                                 defense_location VARCHAR(255) NULL DEFAULT '',
                                 created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                 updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                 INDEX idx_student_email (student_email),
                                 INDEX idx_supervisor_email (supervisor_email),
                                 INDEX idx_reviewer_email (reviewer_email),
                                 INDEX idx_study_program (study_program),
                                 INDEX idx_department (department(100)),
                                 INDEX idx_student_group (student_group),
                                 INDEX idx_current_year (current_year),
                                 INDEX idx_student_number (student_number),
                                 INDEX idx_defense_date (defense_date),
                                 INDEX idx_is_public_defense (is_public_defense)
);

-- Create reviewer access tokens table
CREATE TABLE reviewer_access_tokens (
                                        id INT AUTO_INCREMENT PRIMARY KEY,
                                        reviewer_email VARCHAR(255) NOT NULL,
                                        reviewer_name VARCHAR(255) NOT NULL,
                                        access_token VARCHAR(255) UNIQUE NOT NULL,
                                        department VARCHAR(100) NULL,
                                        created_at BIGINT NOT NULL,
                                        expires_at BIGINT NOT NULL,
                                        max_access INT DEFAULT 0,
                                        access_count INT DEFAULT 0,
                                        last_accessed_at BIGINT NULL,
                                        is_active BOOLEAN DEFAULT true,
                                        created_by VARCHAR(255) NOT NULL,

                                        INDEX idx_reviewer_access_token (access_token),
                                        INDEX idx_reviewer_email (reviewer_email),
                                        INDEX idx_expires_at (expires_at),
                                        INDEX idx_is_active (is_active)
);

-- Create documents table with proper NULL handling
CREATE TABLE documents (
                           id INT AUTO_INCREMENT PRIMARY KEY,
                           document_type VARCHAR(100) NOT NULL,
                           file_path TEXT NOT NULL,
                           uploaded_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                           student_record_id INT NOT NULL,
                           file_size BIGINT NULL,
                           mime_type VARCHAR(255) NULL,
                           original_filename TEXT NULL,
                           is_confidential BOOLEAN DEFAULT TRUE,
                           access_level ENUM('public', 'commission', 'reviewer', 'supervisor') DEFAULT 'supervisor',
                           watermark_applied BOOLEAN DEFAULT FALSE,
                           repository_url TEXT NULL,
                           repository_id VARCHAR(255) NULL,
                           commit_id VARCHAR(255) NULL,
                           submission_id VARCHAR(36) NULL,
                           validation_status ENUM('pending', 'valid', 'invalid') DEFAULT 'pending',
                           validation_errors TEXT NULL,
                           upload_status ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending',

                           FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                           INDEX idx_student_record (student_record_id),
                           INDEX idx_document_type (document_type),
                           INDEX idx_uploaded_date (uploaded_date),
                           INDEX idx_access_level (access_level),
                           INDEX idx_is_confidential (is_confidential),
                           INDEX idx_repository_id (repository_id),
                           INDEX idx_submission_id (submission_id),
                           INDEX idx_upload_status (upload_status)
);

-- Create supervisor reports table with proper NULL handling
CREATE TABLE supervisor_reports (
                                    id INT AUTO_INCREMENT PRIMARY KEY,
                                    student_record_id INT NOT NULL,
                                    supervisor_comments TEXT NOT NULL,
                                    supervisor_name VARCHAR(255) NULL DEFAULT '',
                                    supervisor_position VARCHAR(255) NULL DEFAULT '',
                                    supervisor_workplace TEXT NOT NULL,
                                    is_pass_or_failed BOOLEAN DEFAULT FALSE,
                                    is_signed BOOLEAN NOT NULL DEFAULT FALSE,
                                    other_match DECIMAL(5,2) NOT NULL DEFAULT 0.00,
                                    one_match DECIMAL(5,2) NOT NULL DEFAULT 0.00,
                                    own_match DECIMAL(5,2) NOT NULL DEFAULT 0.00,
                                    join_match DECIMAL(5,2) NOT NULL DEFAULT 0.00,
                                    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                    updated_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                    grade INT NULL,
                                    final_comments TEXT NULL,

                                    FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                                    INDEX idx_student_record (student_record_id),
                                    INDEX idx_is_signed (is_signed),
                                    INDEX idx_created_date (created_date)
);

-- Create reviewer reports table with proper NULL handling
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
                                  is_signed BOOLEAN NOT NULL DEFAULT FALSE,
                                  created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                  updated_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                  FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                                  INDEX idx_student_record (student_record_id),
                                  INDEX idx_is_signed (is_signed),
                                  INDEX idx_grade (grade),
                                  INDEX idx_created_date (created_date)
);

-- Create commission access logs table with proper NULL handling
CREATE TABLE commission_access_logs (
                                        id INT AUTO_INCREMENT PRIMARY KEY,
                                        commission_member_id INT NOT NULL,
                                        student_record_id INT NULL,
                                        action VARCHAR(100) NOT NULL,
                                        resource_accessed VARCHAR(255) NULL,
                                        ip_address VARCHAR(45) NULL,
                                        user_agent TEXT NULL,
                                        access_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                        session_duration INT NULL,

                                        FOREIGN KEY (commission_member_id) REFERENCES commission_members(id) ON DELETE CASCADE,
                                        FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE SET NULL,
                                        INDEX idx_commission_member (commission_member_id),
                                        INDEX idx_student_record (student_record_id),
                                        INDEX idx_access_timestamp (access_timestamp),
                                        INDEX idx_action (action),
                                        INDEX idx_commission_logs_compound (commission_member_id, access_timestamp)
);

-- Create commission evaluations table with proper NULL handling
CREATE TABLE commission_evaluations (
                                        id INT AUTO_INCREMENT PRIMARY KEY,
                                        commission_member_id INT NOT NULL,
                                        student_record_id INT NOT NULL,
                                        presentation_score DECIMAL(3,1) DEFAULT 0.0,
                                        defense_score DECIMAL(3,1) DEFAULT 0.0,
                                        answers_score DECIMAL(3,1) DEFAULT 0.0,
                                        overall_score DECIMAL(3,1) DEFAULT 0.0,
                                        comments TEXT NULL,
                                        questions_asked TEXT NULL,
                                        evaluation_status ENUM('pending', 'completed', 'approved') DEFAULT 'pending',
                                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                        FOREIGN KEY (commission_member_id) REFERENCES commission_members(id) ON DELETE CASCADE,
                                        FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                                        UNIQUE KEY unique_evaluation (commission_member_id, student_record_id),
                                        INDEX idx_commission_member (commission_member_id),
                                        INDEX idx_student_record (student_record_id),
                                        INDEX idx_evaluation_status (evaluation_status),
                                        INDEX idx_overall_score (overall_score),
                                        INDEX idx_commission_eval_compound (student_record_id, evaluation_status),

                                        CONSTRAINT chk_presentation_score CHECK (presentation_score >= 0 AND presentation_score <= 10),
                                        CONSTRAINT chk_defense_score CHECK (defense_score >= 0 AND defense_score <= 10),
                                        CONSTRAINT chk_answers_score CHECK (answers_score >= 0 AND answers_score <= 10),
                                        CONSTRAINT chk_overall_score CHECK (overall_score >= 0 AND overall_score <= 10)
);

-- Create academic audit logs table with proper NULL handling
CREATE TABLE academic_audit_logs (
                                     id INT AUTO_INCREMENT PRIMARY KEY,
                                     access_type ENUM('commission', 'reviewer', 'supervisor', 'admin', 'student') NOT NULL,
                                     access_identifier VARCHAR(191) NOT NULL,
                                     student_record_id INT NULL,
                                     action VARCHAR(100) NOT NULL,
                                     resource_type VARCHAR(50) NOT NULL,
                                     resource_id VARCHAR(100) NULL,
                                     ip_address VARCHAR(45) NOT NULL,
                                     user_agent TEXT NULL,
                                     session_id VARCHAR(128) NULL,
                                     success BOOLEAN DEFAULT TRUE,
                                     error_message TEXT NULL,
                                     metadata TEXT NULL,
                                     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                                     FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE SET NULL,
                                     INDEX idx_access_type (access_type),
                                     INDEX idx_access_identifier (access_identifier),
                                     INDEX idx_student_record (student_record_id),
                                     INDEX idx_action (action),
                                     INDEX idx_created_at (created_at),
                                     INDEX idx_success (success),
                                     INDEX idx_academic_audit_compound (access_type, created_at)
);

-- Create videos table with proper NULL handling
CREATE TABLE videos (
                        id INT AUTO_INCREMENT PRIMARY KEY,
                        student_record_id INT NOT NULL,
                        `key` VARCHAR(255) NOT NULL,
                        filename VARCHAR(255) NOT NULL,
                        content_type VARCHAR(100) NOT NULL,
                        size BIGINT NULL,
                        url TEXT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        duration INT NULL,
                        status VARCHAR(20) DEFAULT 'pending',

                        FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                        INDEX idx_student_record (student_record_id),
                        INDEX idx_status (status),
                        INDEX idx_created_at (created_at)
);

-- Create project topic registrations table with proper NULL handling
CREATE TABLE project_topic_registrations (
                                             id INT AUTO_INCREMENT PRIMARY KEY,
                                             student_record_id INT NOT NULL,
                                             title TEXT NOT NULL,
                                             title_en TEXT NOT NULL,
                                             problem TEXT NOT NULL,
                                             objective TEXT NOT NULL,
                                             tasks TEXT NOT NULL,
                                             completion_date VARCHAR(100) NULL DEFAULT '',
                                             supervisor VARCHAR(255) NOT NULL,
                                             status ENUM('draft', 'submitted', 'supervisor_approved', 'approved', 'rejected', 'revision_requested') DEFAULT 'draft',
                                             created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                             updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                             submitted_at BIGINT NULL,
                                             current_version INT DEFAULT 1,
                                             approved_by VARCHAR(255) NULL,
                                             approved_at BIGINT NULL,
                                             rejection_reason TEXT NULL,
                                             supervisor_approved_at BIGINT NULL,
                                             supervisor_approved_by VARCHAR(255) NULL,
                                             supervisor_rejection_reason TEXT NULL,

                                             FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                                             INDEX idx_student_record (student_record_id),
                                             INDEX idx_status (status),
                                             INDEX idx_supervisor (supervisor),
                                             INDEX idx_submitted_at (submitted_at),
                                             INDEX idx_approved_by (approved_by),
                                             INDEX idx_supervisor_approved_by (supervisor_approved_by)
);

-- Create topic registration comments table with proper NULL handling
CREATE TABLE topic_registration_comments (
                                             id INT AUTO_INCREMENT PRIMARY KEY,
                                             topic_registration_id INT NOT NULL,
                                             field_name VARCHAR(100) NULL,
                                             comment_text TEXT NOT NULL,
                                             author_role VARCHAR(50) NOT NULL,
                                             author_name VARCHAR(255) NOT NULL,
                                             author_email VARCHAR(255) NULL DEFAULT '',
                                             created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                             parent_comment_id INT NULL,
                                             is_read BOOLEAN DEFAULT TRUE,
                                             comment_type VARCHAR(20) DEFAULT 'comment',

                                             FOREIGN KEY (topic_registration_id) REFERENCES project_topic_registrations(id) ON DELETE CASCADE,
                                             FOREIGN KEY (parent_comment_id) REFERENCES topic_registration_comments(id) ON DELETE SET NULL,
                                             INDEX idx_topic_registration (topic_registration_id),
                                             INDEX idx_author_email (author_email),
                                             INDEX idx_created_at (created_at),
                                             INDEX idx_parent_comment (parent_comment_id)
);

-- Create project topic registration versions table with proper NULL handling
CREATE TABLE project_topic_registration_versions (
                                                     id INT AUTO_INCREMENT PRIMARY KEY,
                                                     topic_registration_id INT NOT NULL,
                                                     version_data LONGTEXT NOT NULL,
                                                     created_by VARCHAR(255) NOT NULL,
                                                     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                                     version_number INT NOT NULL,
                                                     change_summary TEXT NULL,

                                                     FOREIGN KEY (topic_registration_id) REFERENCES project_topic_registrations(id) ON DELETE CASCADE,
                                                     INDEX idx_topic_registration (topic_registration_id),
                                                     INDEX idx_created_by (created_by),
                                                     INDEX idx_created_at (created_at),
                                                     INDEX idx_version_number (version_number)
);

-- Create user sessions table with proper NULL handling
CREATE TABLE user_sessions (
                               id INT AUTO_INCREMENT PRIMARY KEY,
                               session_id VARCHAR(128) NOT NULL UNIQUE,
                               user_email VARCHAR(255) NOT NULL,
                               user_data LONGTEXT NOT NULL,
                               created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                               last_accessed TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                               expires_at BIGINT NOT NULL,
                               ip_address VARCHAR(45) NULL,
                               user_agent TEXT NULL,
                               is_active BOOLEAN DEFAULT TRUE,

                               INDEX idx_session_id (session_id),
                               INDEX idx_user_email (user_email),
                               INDEX idx_expires_at (expires_at),
                               INDEX idx_is_active (is_active)
);

-- Create audit logs table with proper NULL handling
CREATE TABLE audit_logs (
                            id INT AUTO_INCREMENT PRIMARY KEY,
                            user_email VARCHAR(255) NOT NULL,
                            user_role VARCHAR(50) NOT NULL,
                            action VARCHAR(100) NOT NULL,
                            resource_type VARCHAR(50) NOT NULL,
                            resource_id VARCHAR(100) NULL,
                            details TEXT NULL,
                            ip_address VARCHAR(45) NULL,
                            user_agent TEXT NULL,
                            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                            success BOOLEAN DEFAULT TRUE,

                            INDEX idx_user_email (user_email),
                            INDEX idx_action (action),
                            INDEX idx_resource_type (resource_type),
                            INDEX idx_created_at (created_at),
                            INDEX idx_success (success)
);

-- Create role permissions table
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

-- Create user preferences table with proper NULL handling
CREATE TABLE user_preferences (
                                  id INT AUTO_INCREMENT PRIMARY KEY,
                                  user_email VARCHAR(255) NOT NULL UNIQUE,
                                  language VARCHAR(5) DEFAULT 'lt',
                                  theme VARCHAR(20) DEFAULT 'light',
                                  notifications_enabled BOOLEAN DEFAULT TRUE,
                                  email_notifications BOOLEAN DEFAULT TRUE,
                                  timezone VARCHAR(50) DEFAULT 'Europe/Vilnius',
                                  preferences_json TEXT NULL,
                                  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                  INDEX idx_user_email (user_email)
);

-- Create oauth states table with proper NULL handling
CREATE TABLE oauth_states (
                              id INT AUTO_INCREMENT PRIMARY KEY,
                              state_value VARCHAR(128) NOT NULL UNIQUE,
                              created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                              expires_at BIGINT NOT NULL,
                              used BOOLEAN DEFAULT FALSE,
                              ip_address VARCHAR(45) NULL,

                              INDEX idx_state_value (state_value),
                              INDEX idx_expires_at (expires_at),
                              INDEX idx_used (used)
);

-- Create views with COALESCE for NULL handling
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

CREATE VIEW student_summary_view AS
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

-- Create triggers to handle NULLs automatically
DELIMITER //

CREATE TRIGGER before_student_records_insert
    BEFORE INSERT ON student_records
    FOR EACH ROW
BEGIN
    IF NEW.defense_location IS NULL THEN
        SET NEW.defense_location = '';
    END IF;
    IF NEW.final_project_title_en IS NULL THEN
        SET NEW.final_project_title_en = '';
    END IF;
    IF NEW.reviewer_email IS NULL THEN
        SET NEW.reviewer_email = '';
    END IF;
    IF NEW.reviewer_name IS NULL THEN
        SET NEW.reviewer_name = '';
    END IF;
END//

CREATE TRIGGER before_student_records_update
    BEFORE UPDATE ON student_records
    FOR EACH ROW
BEGIN
    IF NEW.defense_location IS NULL THEN
        SET NEW.defense_location = '';
    END IF;
    IF NEW.final_project_title_en IS NULL THEN
        SET NEW.final_project_title_en = '';
    END IF;
    IF NEW.reviewer_email IS NULL THEN
        SET NEW.reviewer_email = '';
    END IF;
    IF NEW.reviewer_name IS NULL THEN
        SET NEW.reviewer_name = '';
    END IF;
END//

DELIMITER ;

-- Insert comprehensive role permissions
INSERT INTO role_permissions (role_name, permission, resource_type) VALUES
-- Admin permissions
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

-- Department Head Level 1 permissions
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

-- Department Head Level 2 permissions
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

-- Department Head Level 3 permissions
('department_head_3', 'view_all_students', NULL),
('department_head_3', 'generate_reports', NULL),
('department_head_3', 'view_department_reports', NULL),

-- Department Head Level 4 permissions
('department_head_4', 'view_all_students', NULL),
('department_head_4', 'approve_topics', NULL),
('department_head_4', 'generate_reports', NULL),

-- Supervisor permissions
('supervisor', 'view_assigned_students', NULL),
('supervisor', 'create_reports', NULL),
('supervisor', 'review_submissions', NULL),
('supervisor', 'approve_student_topics', NULL),
('supervisor', 'view_student_source', 'documents'),
('supervisor', 'download_student_source', 'documents'),

-- Commission member permissions
('commission_member', 'view_thesis', NULL),
('commission_member', 'evaluate_defense', NULL),
('commission_member', 'view_defense_materials', 'documents'),
('commission_member', 'submit_evaluation', 'commission_evaluations'),
('commission_member', 'view_student_summary', 'student_records'),
('commission_member', 'view_thesis_source', 'documents'),

-- Reviewer permissions
('reviewer', 'view_thesis_materials', 'documents'),
('reviewer', 'submit_review', 'reviewer_reports'),
('reviewer', 'download_thesis', 'documents'),
('reviewer', 'view_thesis_source_code', 'documents'),

-- Student permissions
('student', 'view_own_data', NULL),
('student', 'submit_topic', NULL),
('student', 'upload_documents', NULL),
('student', 'upload_source_code', 'documents'),
('student', 'view_own_source_uploads', 'documents');

-- Insert initial department heads
INSERT INTO department_heads (id, email, name, sure_name, department, department_en, job_title, role, is_active, created_at) VALUES (1, 't.liogiene@eif.viko.lt', 'Tatjana', 'Liogienė', 'Informacinių sistemų', 'Information Systems', 'Katedros vedėja', 1, 1, '2025-06-04 01:40:59');
INSERT INTO department_heads (id, email, name, sure_name, department, department_en, job_title, role, is_active, created_at) VALUES (2, 'a.kirdeikiene@eif.viko.lt', 'Aliona', 'Kirdeikienė', 'Elektronikos ir kompiuterių inžinerijos', 'Electronics and Computer Engineering', 'Katedros vedėja', 1, 1, '2025-06-04 01:40:59');
INSERT INTO department_heads (id, email, name, sure_name, department, department_en, job_title, role, is_active, created_at) VALUES (3, 'm.gzegozevskis@eif.viko.lt', 'Marius', 'Gžegoževskis', 'Programinės įrangos', 'Software Engineering', 'Sistemos administratorius', 1, 1, '2025-06-04 04:40:59');
INSERT INTO department_heads (id, email, name, sure_name, department, department_en, job_title, role, is_active, created_at) VALUES (4, 'j.zailskas@eif.viko.lt', 'Justinas', 'Zailskas', 'Programinės įrangos', 'Software Engineering', 'Katedros vedėjas', 1, 1, '2025-06-04 14:10:46');

-- Create audit log entry for schema creation
INSERT INTO audit_logs (user_email, user_role, action, resource_type, details, created_at)
VALUES ('system', 'admin', 'schema_created', 'database', 'Complete initial schema with NULL handling created', NOW());