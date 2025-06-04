-- ================================================
-- Migration UP: System Tables
-- File: 000005_system_tables.up.sql
-- ================================================

SET sql_mode = '';
SET foreign_key_checks = 0;

-- Create academic audit logs table
CREATE TABLE IF NOT EXISTS academic_audit_logs (
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

-- Create user sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
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

-- Create audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
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
CREATE TABLE IF NOT EXISTS role_permissions (
                                                id INT AUTO_INCREMENT PRIMARY KEY,
                                                role_name VARCHAR(50) NOT NULL,
                                                permission VARCHAR(100) NOT NULL,
                                                resource_type VARCHAR(50) NULL,
                                                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                                                UNIQUE KEY unique_permission (role_name, permission, resource_type),
                                                INDEX idx_role_name (role_name),
                                                INDEX idx_permission (permission)
);

-- Create user preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
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

-- Create oauth states table
CREATE TABLE IF NOT EXISTS oauth_states (
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

SET foreign_key_checks = 1;