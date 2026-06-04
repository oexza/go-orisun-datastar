CREATE TABLE IF NOT EXISTS projector_checkpoint (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    commit_position INTEGER NOT NULL,
    prepare_position INTEGER NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS auth_user (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    email_verified INTEGER NOT NULL DEFAULT 0,
    image TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    username TEXT UNIQUE,
    display_username TEXT,
    role TEXT,
    banned INTEGER DEFAULT 0,
    ban_reason TEXT,
    ban_expires TEXT,
    user_registered_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS auth_user_email_idx ON auth_user (email);
CREATE INDEX IF NOT EXISTS auth_user_username_idx ON auth_user (username);
CREATE INDEX IF NOT EXISTS auth_user_user_registered_id_idx ON auth_user (user_registered_id);

CREATE TABLE IF NOT EXISTS auth_account (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES auth_user(id) ON DELETE CASCADE,
    access_token TEXT,
    refresh_token TEXT,
    id_token TEXT,
    access_token_expires_at TEXT,
    refresh_token_expires_at TEXT,
    scope TEXT,
    password TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS account_userId_idx ON auth_account (user_id);

CREATE TABLE IF NOT EXISTS auth_session (
    id TEXT PRIMARY KEY,
    expires_at TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ip_address TEXT,
    user_agent TEXT,
    user_id TEXT NOT NULL REFERENCES auth_user(id) ON DELETE CASCADE,
    impersonated_by TEXT
);

CREATE INDEX IF NOT EXISTS session_userId_idx ON auth_session (user_id);

CREATE TABLE IF NOT EXISTS auth_verification (
    id TEXT PRIMARY KEY,
    identifier TEXT NOT NULL,
    value TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS verification_identifier_idx ON auth_verification (identifier);

CREATE TABLE IF NOT EXISTS profile_stats (
    user_id TEXT PRIMARY KEY,
    name TEXT,
    username TEXT,
    email TEXT,
    image TEXT,
    bio TEXT,
    header_image_url TEXT,
    last_event_commit_position INTEGER NOT NULL,
    last_event_prepare_position INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS profile_stats_user_id_idx ON profile_stats (user_id);
CREATE INDEX IF NOT EXISTS profile_stats_username_idx ON profile_stats (username);

CREATE TABLE IF NOT EXISTS todo_items (
    todo_id TEXT PRIMARY KEY,
    user_registered_id TEXT NOT NULL,
    title TEXT NOT NULL,
    completed INTEGER NOT NULL DEFAULT 0,
    completed_at TEXT,
    deleted_at TEXT,
    last_event_commit_position INTEGER NOT NULL,
    last_event_prepare_position INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS todo_items_user_active_created_idx
    ON todo_items (user_registered_id, created_at, todo_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS todo_items_user_completed_idx
    ON todo_items (user_registered_id, completed);
