-- +goose Up
-- Devices paired with a user account at the moment an e-mail login completes.
--
-- The fingerprint is never an identity on its own; it is a second constraint on
-- top of a verified mailbox, and the record that makes a certificate traceable
-- to a known machine.
CREATE TABLE IF NOT EXISTS user_devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fingerprint     VARCHAR(128) NOT NULL,
    name            VARCHAR(255) NOT NULL DEFAULT '',
    platform        VARCHAR(32)  NOT NULL DEFAULT 'unknown',
    last_seen_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- One row per machine per account. The uniqueness is what makes "is this a
    -- new device?" — and therefore the notification e-mail — a decision the
    -- database makes rather than a race between two concurrent logins.
    CONSTRAINT uq_user_devices_user_fingerprint UNIQUE (user_id, fingerprint)
);

CREATE INDEX idx_user_devices_user_id ON user_devices(user_id);

-- +goose Down
DROP TABLE IF EXISTS user_devices;
