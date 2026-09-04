-- +goose Up
-- One-time login codes for the passwordless sign-in flow.
--
-- Keyed by e-mail rather than user id: a code is issued before the system will
-- admit whether an account exists, which is what lets the request endpoint
-- answer identically for known and unknown addresses.
CREATE TABLE IF NOT EXISTS login_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL,
    code_digest     VARCHAR(128) NOT NULL,
    purpose         VARCHAR(32)  NOT NULL DEFAULT 'login',
    attempts        INTEGER      NOT NULL DEFAULT 0,
    max_attempts    INTEGER      NOT NULL DEFAULT 5,
    expires_at      TIMESTAMPTZ  NOT NULL,
    consumed_at     TIMESTAMPTZ,
    request_ip      INET,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Covers both the "latest code for this address" lookup on verify and the
-- "codes issued since" count that backs the per-address rate limit.
CREATE INDEX idx_login_codes_email_purpose_created
    ON login_codes(email, purpose, created_at DESC);

-- Supports the cleanup job that drops spent codes.
CREATE INDEX idx_login_codes_expires_at ON login_codes(expires_at);

-- +goose Down
DROP TABLE IF EXISTS login_codes;
