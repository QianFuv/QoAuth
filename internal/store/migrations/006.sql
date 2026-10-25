CREATE INDEX grants_expiry ON grants(expires,id);
CREATE INDEX tokens_grant_expiry ON tokens(grant_id,expires);
CREATE INDEX requests_expiry ON auth_requests(expires,id);
CREATE INDEX requests_session ON auth_requests(session_digest);
CREATE INDEX sessions_expiry ON browser_sessions(expires,digest);
CREATE INDEX activity_expiry ON activity(recorded_at,id);
