CREATE TABLE prompt_blocks (
    code           TEXT PRIMARY KEY,
    title          TEXT NOT NULL,
    body           TEXT NOT NULL,
    categories_json TEXT NOT NULL DEFAULT '[]',
    is_custom      INTEGER NOT NULL DEFAULT 0,
    updated_at     TEXT NOT NULL
);

CREATE TABLE game_modes (
    code        TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    genre       TEXT NOT NULL,
    block_code  TEXT NOT NULL REFERENCES prompt_blocks (code),
    terms_json  TEXT NOT NULL DEFAULT '{}',
    sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE projects (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    game_code       TEXT NOT NULL REFERENCES game_modes (code),
    youtube_url     TEXT NOT NULL,
    video_title     TEXT NOT NULL DEFAULT '',
    team_a          TEXT NOT NULL DEFAULT '',
    team_b          TEXT NOT NULL DEFAULT '',
    target_minutes  INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL,
    error_message   TEXT NOT NULL DEFAULT '',
    duration_sec    REAL NOT NULL DEFAULT 0,
    width           INTEGER NOT NULL DEFAULT 0,
    height          INTEGER NOT NULL DEFAULT 0,
    fps             REAL NOT NULL DEFAULT 0,
    video_codec     TEXT NOT NULL DEFAULT '',
    video_file      TEXT NOT NULL DEFAULT '',
    transcript_lang TEXT NOT NULL DEFAULT '',
    has_highlight   INTEGER NOT NULL DEFAULT 0,
    size_bytes      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE TABLE jobs (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    status      TEXT NOT NULL,
    progress    REAL NOT NULL DEFAULT 0,
    message     TEXT NOT NULL DEFAULT '',
    error       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    started_at  TEXT,
    finished_at TEXT
);

CREATE INDEX idx_jobs_queue ON jobs (status, created_at);
CREATE INDEX idx_jobs_project ON jobs (project_id);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
