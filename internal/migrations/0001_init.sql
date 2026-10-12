-- +goose Up

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Anyone who has signed in with Google. There are no passwords to store.
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub    text        NOT NULL UNIQUE,
    email         text        NOT NULL UNIQUE,
    name          text        NOT NULL DEFAULT '',
    picture_url   text        NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz NOT NULL DEFAULT now()
);

-- Opaque session tokens. Stored server-side so a sign-out revokes immediately.
CREATE TABLE sessions (
    token      text PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    user_agent text        NOT NULL DEFAULT '',
    ip_address text        NOT NULL DEFAULT ''
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- Short-lived rows guarding the Google sign-in round trip against forgery.
CREATE TABLE oauth_states (
    state         text PRIMARY KEY,
    code_verifier text        NOT NULL,
    redirect_to   text        NOT NULL DEFAULT '',
    expires_at    timestamptz NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX oauth_states_expires_at_idx ON oauth_states (expires_at);

-- A course, e.g. "Computer Science".
CREATE TABLE subjects (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       text        NOT NULL,
    colour     text        NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT subjects_name_not_blank CHECK (length(btrim(name)) > 0)
);

-- Case-insensitive, so "Databases" and "databases" cannot both exist for one user.
CREATE UNIQUE INDEX subjects_user_name_key ON subjects (user_id, lower(btrim(name)));
CREATE INDEX subjects_user_id_idx ON subjects (user_id);

-- A unit of a course, e.g. "Databases". Lives inside a subject.
CREATE TABLE modules (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id uuid        NOT NULL REFERENCES subjects (id) ON DELETE CASCADE,
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT modules_name_not_blank CHECK (length(btrim(name)) > 0)
);

CREATE UNIQUE INDEX modules_subject_name_key ON modules (subject_id, lower(btrim(name)));
CREATE INDEX modules_subject_id_idx ON modules (subject_id);

-- An uploaded lecture file. The bytes live on disk; the extracted text is cached
-- here so regenerating at a new difficulty never re-runs Tika.
CREATE TABLE files (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id      uuid        NOT NULL REFERENCES modules (id) ON DELETE CASCADE,
    user_id        uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    original_name  text        NOT NULL,
    sha256         text        NOT NULL,
    byte_size      bigint      NOT NULL,
    content_type   text        NOT NULL DEFAULT '',
    disk_path      text        NOT NULL,
    extracted_text text        NOT NULL DEFAULT '',
    uploaded_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT files_byte_size_positive CHECK (byte_size > 0)
);

-- The same slides cannot be uploaded to one module twice.
CREATE UNIQUE INDEX files_module_sha_key ON files (module_id, sha256);
CREATE INDEX files_module_id_idx ON files (module_id);
CREATE INDEX files_user_id_idx ON files (user_id);
CREATE INDEX files_user_sha_idx ON files (user_id, sha256);

-- The ledger of work already paid for. One row per file per kind of study
-- material per variant, which is what stops a regenerate duplicating anything.
CREATE TABLE generations (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id      uuid        NOT NULL REFERENCES files (id) ON DELETE CASCADE,
    kind         text        NOT NULL,
    variant      text        NOT NULL DEFAULT '',
    model        text        NOT NULL DEFAULT '',
    status       text        NOT NULL DEFAULT 'pending',
    item_count   int         NOT NULL DEFAULT 0,
    error        text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT generations_kind_valid CHECK (kind IN ('quiz', 'flashcard', 'teaching')),
    CONSTRAINT generations_status_valid CHECK (status IN ('pending', 'complete', 'failed'))
);

-- The no-duplicate rule, enforced by the database rather than by application code.
CREATE UNIQUE INDEX generations_file_kind_variant_key ON generations (file_id, kind, variant);
CREATE INDEX generations_file_id_idx ON generations (file_id);

CREATE TABLE quiz_questions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_id uuid        NOT NULL REFERENCES generations (id) ON DELETE CASCADE,
    file_id       uuid        NOT NULL REFERENCES files (id) ON DELETE CASCADE,
    question      text        NOT NULL,
    position      int         NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX quiz_questions_generation_id_idx ON quiz_questions (generation_id);
CREATE INDEX quiz_questions_file_id_idx ON quiz_questions (file_id);

CREATE TABLE quiz_answers (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id uuid NOT NULL REFERENCES quiz_questions (id) ON DELETE CASCADE,
    text        text NOT NULL,
    is_correct  boolean NOT NULL DEFAULT false,
    explanation text NOT NULL DEFAULT '',
    position    int  NOT NULL DEFAULT 0
);

CREATE INDEX quiz_answers_question_id_idx ON quiz_answers (question_id);

CREATE TABLE flashcards (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_id uuid        NOT NULL REFERENCES generations (id) ON DELETE CASCADE,
    file_id       uuid        NOT NULL REFERENCES files (id) ON DELETE CASCADE,
    term          text        NOT NULL,
    definition    text        NOT NULL,
    position      int         NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX flashcards_generation_id_idx ON flashcards (generation_id);
CREATE INDEX flashcards_file_id_idx ON flashcards (file_id);

CREATE TABLE teaching_cards (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_id uuid        NOT NULL REFERENCES generations (id) ON DELETE CASCADE,
    file_id       uuid        NOT NULL REFERENCES files (id) ON DELETE CASCADE,
    subtopic      text        NOT NULL,
    teaching      text        NOT NULL,
    position      int         NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX teaching_cards_generation_id_idx ON teaching_cards (generation_id);
CREATE INDEX teaching_cards_file_id_idx ON teaching_cards (file_id);

-- One row per generation request, so open signup cannot spend the whole Gemini quota.
CREATE TABLE generate_events (
    id         bigserial PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX generate_events_user_created_idx ON generate_events (user_id, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS generate_events;
DROP TABLE IF EXISTS teaching_cards;
DROP TABLE IF EXISTS flashcards;
DROP TABLE IF EXISTS quiz_answers;
DROP TABLE IF EXISTS quiz_questions;
DROP TABLE IF EXISTS generations;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS modules;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
