CREATE TABLE department_heads (
                                  id INT AUTO_INCREMENT PRIMARY KEY,
                                  email VARCHAR(255) NOT NULL UNIQUE,
                                  name VARCHAR(255) NOT NULL DEFAULT '',
                                  sure_name VARCHAR(255) NOT NULL DEFAULT '',
                                  department TEXT NOT NULL,
                                  department_en TEXT NOT NULL,
                                  job_title VARCHAR(255) NOT NULL DEFAULT '',
                                  role INT NOT NULL DEFAULT 0,
                                  is_active BOOLEAN NOT NULL DEFAULT TRUE,
                                  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                                  INDEX idx_email (email),
                                  INDEX idx_role (role),
                                  INDEX idx_department (department(100)),
                                  INDEX idx_is_active (is_active)
);

CREATE TABLE commission_members (
                                    id INT AUTO_INCREMENT PRIMARY KEY,
                                    access_code VARCHAR(64) NOT NULL UNIQUE,
                                    department TEXT NOT NULL,
                                    study_program VARCHAR(255) NULL,
                                    year INT NULL,
                                    description TEXT DEFAULT '',
                                    is_active BOOLEAN DEFAULT TRUE,
                                    expires_at BIGINT NOT NULL,
                                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                    last_accessed_at BIGINT NULL,
                                    created_by VARCHAR(255) DEFAULT '',
                                    access_count INT DEFAULT 0,
                                    max_access INT DEFAULT 0,

                                    INDEX idx_access_code (access_code),
                                    INDEX idx_department (department(100)),
                                    INDEX idx_study_program (study_program),
                                    INDEX idx_year (year),
                                    INDEX idx_created_by (created_by),
                                    INDEX idx_expires_at (expires_at),
                                    INDEX idx_is_active (is_active)
);

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
                                 is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
                                 created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                 updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                 INDEX idx_student_email (student_email),
                                 INDEX idx_supervisor_email (supervisor_email),
                                 INDEX idx_reviewer_email (reviewer_email),
                                 INDEX idx_study_program (study_program),
                                 INDEX idx_department (department(100)),
                                 INDEX idx_student_group (student_group),
                                 INDEX idx_current_year (current_year),
                                 INDEX idx_student_number (student_number)
);

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

CREATE TABLE supervisor_reports (
                                    id INT AUTO_INCREMENT PRIMARY KEY,
                                    student_record_id INT NOT NULL,
                                    supervisor_comments TEXT NOT NULL,
                                    supervisor_name VARCHAR(255) NOT NULL DEFAULT '',
                                    supervisor_position VARCHAR(255) NOT NULL DEFAULT '',
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
                                    final_comments TEXT DEFAULT '',

                                    FOREIGN KEY (student_record_id) REFERENCES student_records(id) ON DELETE CASCADE,
                                    INDEX idx_student_record (student_record_id),
                                    INDEX idx_is_signed (is_signed),
                                    INDEX idx_created_date (created_date)
);

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
                                             is_read BOOLEAN DEFAULT TRUE,
                                             comment_type VARCHAR(20) DEFAULT 'comment' COMMENT 'comment, suggestion, approval, rejection',

                                             FOREIGN KEY (topic_registration_id) REFERENCES project_topic_registrations(id) ON DELETE CASCADE,
                                             FOREIGN KEY (parent_comment_id) REFERENCES topic_registration_comments(id) ON DELETE SET NULL,
                                             INDEX idx_topic_registration (topic_registration_id),
                                             INDEX idx_author_email (author_email),
                                             INDEX idx_created_at (created_at),
                                             INDEX idx_parent_comment (parent_comment_id)
);

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
                               is_active BOOLEAN DEFAULT TRUE,

                               INDEX idx_session_id (session_id),
                               INDEX idx_user_email (user_email),
                               INDEX idx_expires_at (expires_at),
                               INDEX idx_is_active (is_active)
);

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
                            success BOOLEAN DEFAULT TRUE,

                            INDEX idx_user_email (user_email),
                            INDEX idx_action (action),
                            INDEX idx_resource_type (resource_type),
                            INDEX idx_created_at (created_at),
                            INDEX idx_success (success)
);

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

CREATE TABLE user_preferences (
                                  id INT AUTO_INCREMENT PRIMARY KEY,
                                  user_email VARCHAR(255) NOT NULL UNIQUE,
                                  language VARCHAR(5) DEFAULT 'lt',
                                  theme VARCHAR(20) DEFAULT 'light',
                                  notifications_enabled BOOLEAN DEFAULT TRUE,
                                  email_notifications BOOLEAN DEFAULT TRUE,
                                  timezone VARCHAR(50) DEFAULT 'Europe/Vilnius',
                                  preferences_json JSON NULL COMMENT 'Additional JSON preferences',
                                  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                  INDEX idx_user_email (user_email)
);

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