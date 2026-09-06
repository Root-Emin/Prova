-- +goose Up
-- The IP is observed by the API server during successful Desktop sign-in;
-- the MAC is supplied by the desktop's active adapter and validated in the
-- domain layer. Both fields are nullable so existing device records remain
-- valid and platforms without a physical adapter are supported.
ALTER TABLE user_devices ADD COLUMN IF NOT EXISTS ip_address  VARCHAR(45);
ALTER TABLE user_devices ADD COLUMN IF NOT EXISTS mac_address VARCHAR(17);

-- +goose Down
ALTER TABLE user_devices DROP COLUMN IF EXISTS mac_address;
ALTER TABLE user_devices DROP COLUMN IF EXISTS ip_address;
