-- ================================================
-- Migration UP: Topic Registration Tables
-- File: 000004_topic_registration.up.sql
-- ================================================

SET sql_mode = '';
SET foreign_key_checks = 0;

-- Create project topic registrations table
CREATE TABLE IF NOT EXISTS project_topic_registrations (
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

-- Create topic registration comments table
CREATE TABLE IF NOT EXISTS topic_registration_comments (
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

-- Create project topic registration versions table
CREATE TABLE IF NOT EXISTS project_topic_registration_versions (
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

SET foreign_key_checks = 1;