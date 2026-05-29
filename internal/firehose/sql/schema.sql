CREATE TABLE IF NOT EXISTS records (
    uri         TEXT PRIMARY KEY,
    did         TEXT NOT NULL,
    collection  TEXT NOT NULL,
    rkey        TEXT NOT NULL,
    record      TEXT NOT NULL, -- Raw JSON record
    cid         TEXT NOT NULL DEFAULT '',
    indexed_at  TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_records_created ON records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_did ON records(did);
CREATE INDEX IF NOT EXISTS idx_records_coll_created ON records(collection, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_did_coll ON records(did, collection, created_at DESC);

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value BLOB
);

CREATE TABLE IF NOT EXISTS known_dids (did TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS registered_dids (
    did           TEXT PRIMARY KEY,
    registered_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS backfilled (did TEXT PRIMARY KEY, backfilled_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS profiles (
    did        TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS did_by_handle (
    handle     TEXT PRIMARY KEY,
    did        TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_did_by_handle_did ON did_by_handle(did);

CREATE TABLE IF NOT EXISTS likes (
    subject_uri TEXT NOT NULL,
    actor_did   TEXT NOT NULL,
    rkey        TEXT NOT NULL,
    PRIMARY KEY (subject_uri, actor_did)
);
CREATE INDEX IF NOT EXISTS idx_likes_actor ON likes(actor_did, subject_uri);

CREATE TABLE IF NOT EXISTS comments (
    actor_did   TEXT NOT NULL,
    rkey        TEXT NOT NULL,
    subject_uri TEXT NOT NULL,
    parent_uri  TEXT NOT NULL DEFAULT '',
    parent_rkey TEXT NOT NULL DEFAULT '',
    cid         TEXT NOT NULL DEFAULT '',
    text        TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    PRIMARY KEY (actor_did, rkey)
);
CREATE INDEX IF NOT EXISTS idx_comments_subject ON comments(subject_uri, created_at);

CREATE TABLE IF NOT EXISTS notifications (
    id          TEXT NOT NULL,
    target_did  TEXT NOT NULL,
    type        TEXT NOT NULL,
    actor_did   TEXT NOT NULL,
    subject_uri TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notif_target ON notifications(target_did, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_notif_dedup ON notifications(target_did, type, actor_did, subject_uri);

CREATE TABLE IF NOT EXISTS notifications_meta (
    target_did TEXT PRIMARY KEY,
    last_read  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS moderation_hidden_records (
    uri         TEXT PRIMARY KEY,
    hidden_at   TEXT NOT NULL,
    hidden_by   TEXT NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    auto_hidden INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS moderation_blacklist (
    did            TEXT PRIMARY KEY,
    blacklisted_at TEXT NOT NULL,
    blacklisted_by TEXT NOT NULL,
    reason         TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS moderation_reports (
    id           TEXT PRIMARY KEY,
    subject_uri  TEXT NOT NULL DEFAULT '',
    subject_did  TEXT NOT NULL DEFAULT '',
    reporter_did TEXT NOT NULL,
    reason       TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    resolved_by  TEXT NOT NULL DEFAULT '',
    resolved_at  TEXT
);
CREATE INDEX IF NOT EXISTS idx_modreports_uri      ON moderation_reports(subject_uri);
CREATE INDEX IF NOT EXISTS idx_modreports_did      ON moderation_reports(subject_did);
CREATE INDEX IF NOT EXISTS idx_modreports_reporter ON moderation_reports(reporter_did, created_at);
CREATE INDEX IF NOT EXISTS idx_modreports_status   ON moderation_reports(status);

CREATE TABLE IF NOT EXISTS moderation_audit_log (
    id         TEXT PRIMARY KEY,
    action     TEXT NOT NULL,
    actor_did  TEXT NOT NULL,
    target_uri TEXT NOT NULL DEFAULT '',
    reason     TEXT NOT NULL DEFAULT '',
    details    TEXT NOT NULL DEFAULT '{}',
    timestamp  TEXT NOT NULL,
    auto_mod   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_modaudit_ts ON moderation_audit_log(timestamp DESC);

CREATE TABLE IF NOT EXISTS moderation_autohide_resets (
    did      TEXT PRIMARY KEY,
    reset_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS moderation_labels (
    id          TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    label       TEXT NOT NULL,
    value       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    created_by  TEXT NOT NULL,
    expires_at  TEXT,
    UNIQUE(entity_type, entity_id, label)
);
CREATE INDEX IF NOT EXISTS idx_modlabels_entity ON moderation_labels(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_modlabels_expires ON moderation_labels(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_settings (
    did  TEXT PRIMARY KEY,
    profile_stats_visibility TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS oauth_sessions (
    did        TEXT NOT NULL,
    session_id TEXT NOT NULL,
    data       TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (did, session_id)
);
CREATE INDEX IF NOT EXISTS idx_oauth_sessions_did ON oauth_sessions(did);

CREATE TABLE IF NOT EXISTS oauth_auth_requests (
    state      TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    created_at TEXT NOT NULL
);
