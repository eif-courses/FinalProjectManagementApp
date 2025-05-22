-- migrations/000001_initial_schema.up.sql

-- Department Heads Table
CREATE TABLE department_heads (
                                  id INTEGER PRIMARY KEY,
                                  email TEXT NOT NULL UNIQUE,
                                  name TEXT NOT NULL DEFAULT '',
                                  sure_name TEXT NOT NULL DEFAULT '',
                                  department TEXT NOT NULL DEFAULT '',
                                  department_en TEXT NOT NULL DEFAULT '',
                                  job_title TEXT NOT NULL DEFAULT '',
                                  role INTEGER NOT NULL DEFAULT 0,
                                  is_active INTEGER NOT NULL DEFAULT 1,
                                  created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX department_heads_email_idx ON department_heads(email);
CREATE INDEX department_heads_role_idx ON department_heads(role);

-- Commission Members Table
CREATE TABLE commission_members (
                                    id INTEGER PRIMARY KEY,
                                    access_code TEXT NOT NULL UNIQUE,
                                    department TEXT NOT NULL,
                                    is_active INTEGER DEFAULT 1,
                                    expires_at INTEGER NOT NULL,
                                    created_at INTEGER DEFAULT (strftime('%s', 'now')),
                                    last_accessed_at INTEGER
);

-- Student Records Table
CREATE TABLE student_records (
                                 id INTEGER PRIMARY KEY,
                                 student_group TEXT NOT NULL,
                                 final_project_title TEXT NOT NULL DEFAULT '',
                                 final_project_title_en TEXT NOT NULL DEFAULT '',
                                 student_email TEXT NOT NULL,
                                 student_name TEXT NOT NULL,
                                 student_lastname TEXT NOT NULL,
                                 student_number TEXT NOT NULL,
                                 supervisor_email TEXT NOT NULL,
                                 study_program TEXT NOT NULL,
                                 department TEXT NOT NULL DEFAULT '',
                                 program_code TEXT NOT NULL,
                                 current_year INTEGER NOT NULL,
                                 reviewer_email TEXT NOT NULL DEFAULT '',
                                 reviewer_name TEXT NOT NULL DEFAULT '',
                                 is_favorite INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX student_email_idx ON student_records(student_email);
CREATE INDEX supervisor_email_idx ON student_records(supervisor_email);
CREATE INDEX reviewer_email_idx ON student_records(reviewer_email);
CREATE INDEX study_program_idx ON student_records(study_program);
CREATE INDEX department_idx ON student_records(department);

-- Documents Table
CREATE TABLE documents (
                           id INTEGER PRIMARY KEY,
                           document_type TEXT NOT NULL,
                           file_path TEXT NOT NULL,
                           uploaded_date INTEGER DEFAULT (strftime('%s', 'now')),
                           student_record_id INTEGER REFERENCES student_records(id) ON DELETE CASCADE
);

CREATE INDEX documents_student_record_idx ON documents(student_record_id);

-- Supervisor Reports Table
CREATE TABLE supervisor_reports (
                                    id INTEGER PRIMARY KEY,
                                    student_record_id INTEGER REFERENCES student_records(id) ON DELETE CASCADE,
                                    supervisor_comments TEXT NOT NULL DEFAULT '',
                                    supervisor_name TEXT NOT NULL DEFAULT '',
                                    supervisor_position TEXT NOT NULL DEFAULT '',
                                    supervisor_workplace TEXT NOT NULL DEFAULT '',
                                    is_pass_or_failed INTEGER DEFAULT 0,
                                    is_signed INTEGER NOT NULL DEFAULT 0,
                                    other_match REAL NOT NULL DEFAULT 0,
                                    one_match REAL NOT NULL DEFAULT 0,
                                    own_match REAL NOT NULL DEFAULT 0,
                                    join_match REAL NOT NULL DEFAULT 0,
                                    created_date INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX supervisor_reports_student_record_idx ON supervisor_reports(student_record_id);

-- Reviewer Reports Table
CREATE TABLE reviewer_reports (
                                  id INTEGER PRIMARY KEY,
                                  student_record_id INTEGER REFERENCES student_records(id) ON DELETE CASCADE,
                                  reviewer_personal_details TEXT NOT NULL DEFAULT '',
                                  grade REAL NOT NULL DEFAULT 0,
                                  review_goals TEXT NOT NULL DEFAULT '',
                                  review_theory TEXT NOT NULL DEFAULT '',
                                  review_practical TEXT NOT NULL DEFAULT '',
                                  review_theory_practical_link TEXT NOT NULL DEFAULT '',
                                  review_results TEXT NOT NULL DEFAULT '',
                                  review_practical_significance TEXT,
                                  review_language TEXT NOT NULL DEFAULT '',
                                  review_pros TEXT NOT NULL DEFAULT '',
                                  review_cons TEXT NOT NULL DEFAULT '',
                                  review_questions TEXT NOT NULL DEFAULT '',
                                  is_signed INTEGER NOT NULL DEFAULT 0,
                                  created_date INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX reviewer_reports_student_record_idx ON reviewer_reports(student_record_id);

-- Videos Table
CREATE TABLE videos (
                        id INTEGER PRIMARY KEY,
                        student_record_id INTEGER NOT NULL REFERENCES student_records(id) ON DELETE CASCADE,
                        key TEXT NOT NULL,
                        filename TEXT NOT NULL,
                        content_type TEXT NOT NULL,
                        size INTEGER,
                        url TEXT,
                        created_at TEXT DEFAULT (CURRENT_TIMESTAMP) NOT NULL
);

CREATE INDEX videos_student_record_idx ON videos(student_record_id);

-- Project Topic Registrations Table
CREATE TABLE project_topic_registrations (
                                             id INTEGER PRIMARY KEY,
                                             student_record_id INTEGER NOT NULL REFERENCES student_records(id) ON DELETE CASCADE,
                                             title TEXT NOT NULL,
                                             title_en TEXT NOT NULL,
                                             problem TEXT NOT NULL,
                                             objective TEXT NOT NULL,
                                             tasks TEXT NOT NULL,
                                             completion_date TEXT,
                                             supervisor TEXT NOT NULL,
                                             status TEXT NOT NULL DEFAULT 'draft',
                                             created_at INTEGER DEFAULT (strftime('%s', 'now')),
                                             updated_at INTEGER DEFAULT (strftime('%s', 'now')),
                                             submitted_at INTEGER,
                                             current_version INTEGER DEFAULT 1
);

-- Topic Registration Comments Table
CREATE TABLE topic_registration_comments (
                                             id INTEGER PRIMARY KEY,
                                             topic_registration_id INTEGER NOT NULL REFERENCES project_topic_registrations(id) ON DELETE CASCADE,
                                             field_name TEXT,
                                             comment_text TEXT NOT NULL,
                                             author_role TEXT NOT NULL,
                                             author_name TEXT NOT NULL,
                                             created_at INTEGER DEFAULT (strftime('%s', 'now')),
                                             parent_comment_id INTEGER,
                                             is_read INTEGER DEFAULT 1
);

-- Project Topic Registration Versions Table
CREATE TABLE project_topic_registration_versions (
                                                     id INTEGER PRIMARY KEY,
                                                     topic_registration_id INTEGER NOT NULL REFERENCES project_topic_registrations(id) ON DELETE CASCADE,
                                                     version_data TEXT NOT NULL,
                                                     created_by TEXT NOT NULL,
                                                     created_at INTEGER DEFAULT (strftime('%s', 'now'))
);