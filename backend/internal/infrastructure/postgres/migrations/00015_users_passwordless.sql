-- +goose Up
-- Move the users table onto the passwordless model.
--
-- The address is the credential, so the system records when it was proven
-- reachable. password_hash is dropped outright: an unused nullable column is
-- still an attack surface and still invites code that reintroduces passwords.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;

-- Addresses are compared case-insensitively everywhere, so uniqueness must be
-- too. The plain UNIQUE constraint from 00002 would happily accept both
-- user@corp.com and User@corp.com as separate accounts.
UPDATE users SET email = lower(email) WHERE email <> lower(email);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email_lower ON users (lower(email));

-- +goose Down
DROP INDEX IF EXISTS uq_users_email_lower;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
