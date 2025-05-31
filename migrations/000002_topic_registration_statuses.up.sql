-- Migration: Add supervisor approval workflow
ALTER TABLE project_topic_registrations
    ADD COLUMN supervisor_approved_at BIGINT NULL,
    ADD COLUMN supervisor_approved_by VARCHAR(255) NULL,
    ADD COLUMN supervisor_rejection_reason TEXT NULL,
    ADD INDEX idx_supervisor_approved_by (supervisor_approved_by);

-- Update status enum to include new workflow states
ALTER TABLE project_topic_registrations
    MODIFY COLUMN status ENUM('draft', 'submitted', 'supervisor_approved', 'approved', 'rejected', 'revision_requested') DEFAULT 'draft';