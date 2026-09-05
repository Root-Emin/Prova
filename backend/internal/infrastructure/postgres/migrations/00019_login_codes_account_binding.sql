-- +goose Up
-- Account-bound verification challenges. Existing rows remain nullable so a
-- rolling deployment can expire legacy e-mail-only challenges safely; the
-- running application refuses to redeem them once the bound verifier is on.
ALTER TABLE login_codes
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_login_codes_user_purpose_created
    ON login_codes(user_id, purpose, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_login_codes_user_purpose_created;
ALTER TABLE login_codes DROP COLUMN IF EXISTS user_id;
