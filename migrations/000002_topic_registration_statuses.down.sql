-- ================================================
-- Migration DOWN: Remove Supervisor Approval Workflow
-- File: 000002_add_supervisor_approval_workflow.down.sql
-- ================================================

-- First, update any existing records with new status values back to compatible ones
UPDATE project_topic_registrations
SET status = 'submitted'
WHERE status = 'supervisor_approved';

UPDATE project_topic_registrations
SET status = 'rejected'
WHERE status = 'revision_requested';

-- Revert status enum to original values (remove new workflow states)
ALTER TABLE project_topic_registrations
    MODIFY COLUMN status ENUM('draft', 'submitted', 'approved', 'rejected') DEFAULT 'draft';

-- Remove the supervisor approval columns and index
ALTER TABLE project_topic_registrations
    DROP INDEX idx_supervisor_approved_by,
    DROP COLUMN supervisor_approved_at,
    DROP COLUMN supervisor_approved_by,
    DROP COLUMN supervisor_rejection_reason;