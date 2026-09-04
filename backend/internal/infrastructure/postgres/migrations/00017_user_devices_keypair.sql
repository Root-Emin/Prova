-- +goose Up
-- digest() pgcrypto'dan gelir; aşağıdaki taşıma ona bağlı.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Cihaz kaydını parmak izinden anahtar çiftine yükseltir.
--
-- Parmak izi tek başına bir sırdır ve düz metin saklanıyordu: veritabanına
-- erişen biri onu istemci gibi gönderebilirdi. Artık yalnızca hash'i
-- saklanıyor, ve gerçek doğrulama her girişte imzalanan bir challenge ile
-- yapılıyor.
ALTER TABLE user_devices ADD COLUMN IF NOT EXISTS fingerprint_hash VARCHAR(64);
ALTER TABLE user_devices ADD COLUMN IF NOT EXISTS public_key       TEXT;

-- Mevcut düz metin parmak izleri hash'e taşınır. sha256, parmak izi zaten
-- yüksek entropili olduğu için yeterli; pepper gerekmiyor çünkü tahmin edilecek
-- küçük bir uzay yok.
UPDATE user_devices
   SET fingerprint_hash = encode(digest(fingerprint, 'sha256'), 'hex')
 WHERE fingerprint_hash IS NULL AND fingerprint IS NOT NULL;

-- Benzersizlik hash'e taşınır: eski kısıt düz metin kolona bağlıydı ve o kolon
-- düşürülüyor.
ALTER TABLE user_devices DROP CONSTRAINT IF EXISTS uq_user_devices_user_fingerprint;
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_devices_user_fingerprint_hash
    ON user_devices(user_id, fingerprint_hash);

ALTER TABLE user_devices DROP COLUMN IF EXISTS fingerprint;

-- +goose Down
ALTER TABLE user_devices ADD COLUMN IF NOT EXISTS fingerprint VARCHAR(128);
DROP INDEX IF EXISTS uq_user_devices_user_fingerprint_hash;
ALTER TABLE user_devices DROP COLUMN IF EXISTS public_key;
ALTER TABLE user_devices DROP COLUMN IF EXISTS fingerprint_hash;
