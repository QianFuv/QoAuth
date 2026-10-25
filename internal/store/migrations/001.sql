CREATE TABLE users (id TEXT PRIMARY KEY, enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)));
CREATE TABLE clients (id TEXT PRIMARY KEY, enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)));
CREATE TABLE assignments (user_id TEXT REFERENCES users(id), client_id TEXT REFERENCES clients(id), PRIMARY KEY(user_id,client_id));
CREATE TABLE auth_requests (
    id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id), client_id TEXT NOT NULL REFERENCES clients(id),
    payload BLOB NOT NULL, expires INTEGER NOT NULL, ready INTEGER NOT NULL DEFAULT 0,
    code_hash TEXT UNIQUE, consumed INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE grants (
    id TEXT PRIMARY KEY, request_id TEXT NOT NULL UNIQUE REFERENCES auth_requests(id),
    user_id TEXT NOT NULL REFERENCES users(id), client_id TEXT NOT NULL REFERENCES clients(id),
    expires INTEGER NOT NULL, revoked INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE tokens (
    digest TEXT PRIMARY KEY, grant_id TEXT NOT NULL REFERENCES grants(id),
    kind TEXT NOT NULL CHECK(kind IN ('access','refresh')), payload BLOB NOT NULL,
    expires INTEGER NOT NULL, consumed INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX tokens_grant ON tokens(grant_id);
CREATE TABLE recovery_codes (digest TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id), consumed INTEGER NOT NULL DEFAULT 0);
CREATE TABLE activity (id INTEGER PRIMARY KEY, action TEXT NOT NULL, subject TEXT NOT NULL, recorded_at INTEGER NOT NULL);
