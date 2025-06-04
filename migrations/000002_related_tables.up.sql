-- ================================================
-- Migration UP: Related Tables
-- File: 000002_related_tables.up.sql
-- ================================================

SET sql_mode = '';
SET foreign_key_checks = 0;

-- Create documents table
CREATE TABLE IF NOT EXISTS documents (
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

-- Create supervisor reports table
CREATE TABLE IF NOT EXISTS supervisor_reports (
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

-- Create reviewer reports table
CREATE TABLE IF NOT EXISTS reviewer_reports (
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

-- Create videos table
CREATE TABLE IF NOT EXISTS videos (
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

SET foreign_key_checks = 1;