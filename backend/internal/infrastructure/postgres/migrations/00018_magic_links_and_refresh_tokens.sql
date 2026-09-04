-- +goose Up
-- Tek kullanımlık giriş bağlantıları.
--
-- Token'ın kendisi saklanmaz, yalnızca hash'i. Kod hattındaki mantığın aynısı:
-- veritabanı sızarsa sızan şey kimlik bilgisi olmamalı. Kod'dan farkı, token
-- yüksek entropili olduğu için pepper gerekmemesi.
CREATE TABLE IF NOT EXISTS magic_links (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    email       VARCHAR(255) NOT NULL,
    token_hash  VARCHAR(64)  NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ  NOT NULL,
    used_at     TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ,
    request_ip  VARCHAR(45),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_magic_links_email_created
    ON magic_links(email, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_magic_links_expires ON magic_links(expires_at);

-- Rotasyonlu refresh token'lar.
--
-- family_id, aynı giriş oturumundan türeyen tüm token'ları bağlar. Kullanılmış
-- bir token tekrar geldiğinde ailenin tamamı iptal edilir: ya token çalınmış ve
-- saldırgan kullanıyor, ya da meşru kullanıcı tekrar oynatıyor. İkisini ayırt
-- edemediğimiz için güvenli taraf ikisini de kesmektir.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id    UUID NOT NULL,
    token_hash   VARCHAR(64) NOT NULL UNIQUE,
    device_id    UUID REFERENCES user_devices(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL,
    parent_id    UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    used_at      TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at);

-- Cihaz imza challenge'ları. Kısa ömürlüdür ve tek kullanımlıktır: bir
-- challenge yeniden oynatılabilir bir kimlik bilgisine dönüşmemeli.
CREATE TABLE IF NOT EXISTS device_challenges (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email            VARCHAR(255) NOT NULL,
    fingerprint_hash VARCHAR(64)  NOT NULL,
    challenge        VARCHAR(128) NOT NULL,
    expires_at       TIMESTAMPTZ  NOT NULL,
    consumed_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_challenges_lookup
    ON device_challenges(email, fingerprint_hash, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_challenges_expires ON device_challenges(expires_at);

-- +goose Down
DROP TABLE IF EXISTS device_challenges;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS magic_links;
