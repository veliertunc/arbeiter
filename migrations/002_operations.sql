BEGIN;

CREATE TABLE IF NOT EXISTS rate_limits (
    bucket CHAR(64) PRIMARY KEY,
    window_start TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL CHECK (attempts >= 0)
);
CREATE INDEX IF NOT EXISTS rate_limits_window ON rate_limits(window_start);

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    subject_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_events_org_recent ON audit_events(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_events_actor_recent ON audit_events(actor_user_id, created_at DESC);

COMMIT;
