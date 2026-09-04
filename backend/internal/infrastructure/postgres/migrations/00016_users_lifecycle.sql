-- +goose Up
-- Hesap yaşam döngüsü ve kaba kuvvet koruması alanları.
--
-- Silme, satırı düşürmekle yapılmaz: denetim kaydı silinmez ve bir kullanıcıya
-- bağlanabilmelidir. Bu yüzden hesap "deleted" durumuna geçer, kişisel veri
-- alanları null'a çekilir, ve satır kimliksiz bir kabuk olarak kalır.
ALTER TABLE users ADD COLUMN IF NOT EXISTS deletion_requested_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deletion_scheduled_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at            TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until          TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_attempts       INTEGER NOT NULL DEFAULT 0;

-- E-posta silinen hesapta null'a çekilir, bu yüzden NOT NULL kısıtı kalkmalı.
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;

-- Kalıcı silme işi bu index üzerinden çalışır: geri alma penceresi dolmuş
-- hesapları tarama olmadan bulur.
CREATE INDEX IF NOT EXISTS idx_users_deletion_scheduled
    ON users(deletion_scheduled_at)
    WHERE deletion_scheduled_at IS NOT NULL AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_users_deletion_scheduled;
ALTER TABLE users DROP COLUMN IF EXISTS failed_attempts;
ALTER TABLE users DROP COLUMN IF EXISTS locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deletion_scheduled_at;
ALTER TABLE users DROP COLUMN IF EXISTS deletion_requested_at;
