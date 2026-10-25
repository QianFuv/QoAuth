CREATE TABLE accounts (
    user_id TEXT PRIMARY KEY REFERENCES users(id), username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL, admin INTEGER NOT NULL DEFAULT 0, owner INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1, require_mfa INTEGER NOT NULL DEFAULT 0,
    factor BLOB, pending_factor BLOB, pending_until INTEGER NOT NULL DEFAULT 0,
    last_step INTEGER NOT NULL DEFAULT -1
);
CREATE TABLE installation (id INTEGER PRIMARY KEY CHECK(id=1), key_id TEXT NOT NULL);
CREATE TABLE browser_sessions (
    digest TEXT PRIMARY KEY, user_id TEXT REFERENCES users(id), version INTEGER NOT NULL DEFAULT 0,
    csrf TEXT NOT NULL, assurance INTEGER NOT NULL DEFAULT 0, auth_time INTEGER NOT NULL DEFAULT 0,
    expires INTEGER NOT NULL, revoked INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX sessions_user ON browser_sessions(user_id);
CREATE TABLE client_config (client_id TEXT PRIMARY KEY REFERENCES clients(id), payload BLOB NOT NULL);
CREATE TABLE consents (user_id TEXT REFERENCES users(id), client_id TEXT REFERENCES clients(id), payload BLOB NOT NULL, PRIMARY KEY(user_id,client_id));
CREATE TABLE login_requests (
    id TEXT PRIMARY KEY, client_id TEXT REFERENCES clients(id), browser_digest TEXT NOT NULL,
    payload BLOB NOT NULL, expires INTEGER NOT NULL, completed INTEGER NOT NULL DEFAULT 0
);
ALTER TABLE auth_requests ADD COLUMN session_digest TEXT NOT NULL DEFAULT '';
