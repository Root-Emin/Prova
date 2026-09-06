-- +goose Up
-- Optional password hashes support local/test sign-in. Accounts may remain
-- passwordless when this column is NULL and continue using mailbox OTP.
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
