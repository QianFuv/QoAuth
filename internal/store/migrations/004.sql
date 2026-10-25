CREATE INDEX login_expiry ON login_requests(expires);
CREATE INDEX login_browser ON login_requests(browser_digest);
CREATE INDEX sessions_anonymous ON browser_sessions(expires) WHERE user_id IS NULL;
