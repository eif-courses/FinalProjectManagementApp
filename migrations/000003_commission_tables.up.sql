-- ================================================
-- Migration UP: Commission & Evaluation Tables
-- File: 000003_commission_tables.up.sql
-- ================================================

SET sql_mode = '';
SET foreign_key_checks = 0;

-- Create commission access logs table
CREATE TABLE IF NOT EXISTS commission_access_logs (
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

-- Create commission evaluations table (without CHECK constraints for compatibility)
CREATE TABLE IF NOT EXISTS commission_evaluations (
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
                                                      INDEX idx_commission_eval_compound (student_record_id, evaluation_status)
);

SET foreign_key_checks = 1;