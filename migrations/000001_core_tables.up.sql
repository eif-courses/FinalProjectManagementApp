-- ================================================
-- Migration UP: Core Tables
-- File: 000001_core_tables.up.sql
-- ================================================

SET sql_mode = '';
SET foreign_key_checks = 0;

-- Create department heads table
CREATE TABLE IF NOT EXISTS department_heads (
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

-- Create commission members table
CREATE TABLE IF NOT EXISTS commission_members (
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

-- Create student records table
CREATE TABLE IF NOT EXISTS student_records (
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
CREATE TABLE IF NOT EXISTS reviewer_access_tokens (
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

SET foreign_key_checks = 1;