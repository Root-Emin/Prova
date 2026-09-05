# Gereksinim izlenebilirlik tablosu

On iki gereksinimin her biri için: hangi dosyalar karşılıyor ve hangi komut
kanıtlıyor.

Tamamını çalıştırmak için:

```bash
cd backend && ./scripts/verify.sh
```

Tek bir gereksinimi çalıştırmak için `ONLY=<n> ./scripts/verify.sh`.

---

## 1. GraphQL

Tüm istemci API yüzeyi GraphQL; REST yalnızca sağlık uçları.

| Dosya | Rol |
|---|---|
| `graph/schema.graphqls` | Şemanın tamamı: sorgu, mutation, subscription, iki directive |
| `graph/schema.resolvers.go` | Resolver'lar; iş kuralı içermez, use case çağırır |
| `internal/infrastructure/graphql/server.go` | Taşıma, kimlik çözümü, hata sunumu, sertleştirme |
| `internal/infrastructure/graphql/directives.go` | `@auth` ve `@permission` uygulamaları |
| `internal/infrastructure/graphql/depth.go` | Derinlik limiti (fragment içi dâhil) |
| `internal/infrastructure/http/router/router.go` | `/graphql` mount'u, JSON 404, sağlık uçları |
| `internal/shared/authctx/viewer.go` | Doğrulanmış çağıran; org ID her zaman JWT claim'inden |

**Doğrulama:** `ONLY=1 ./scripts/verify.sh`
Sorgu, mutation ve subscription çalışıyor; izin korumalı alan yetkisiz
kullanıcıda null, yetkilide dolu; eski REST auth uçları 404.

**Birim testleri:** `go test ./internal/infrastructure/graphql/`

---

## 2. Object database

Sürümlenebilir belgeler, yayınlanmış belge asla değişmez.

| Dosya | Rol |
|---|---|
| `internal/domain/prova/model/document.go` | Sürüm zarfı, soy, referans |
| `internal/domain/prova/repository/repository.go` | Generic `Versioned[T]` sözleşmesi, `Scope` tipi |
| `internal/infrastructure/mongo/prova/versioned.go` | Generic depo; sürümleme kuralları burada zorlanır |
| `internal/infrastructure/mongo/collections.go` | Index'ler; org+lineage+version unique |
| `internal/infrastructure/mongo/registry.go` | UUID'lerin BSON binary olarak yazılması |
| `cmd/seed/` | Tohum verisi |

**Doğrulama:** `ONLY=2 ./scripts/verify.sh`
Düzenleme v2 üretiyor, v1 korunuyor, yayınlanmış belge değişmiyor, kiracı
sınırı uygulanıyor.

**Entegrasyon testleri:** `go test ./internal/infrastructure/mongo/...`
(gerçek MongoDB'ye karşı; yoksa atlanır)

---

## 3. Web arayüzü

Şema istemci kod üretimine hazır, CORS yapılandırılmış.

| Dosya | Rol |
|---|---|
| `cmd/schema/main.go` | SDL dışa aktarımı (belirlenimci sıralama) |
| `schema.graphql` (depo kökü) | Üretilmiş SDL |
| `internal/shared/middleware/cors.go` | İzinli origin listesi |

**Doğrulama:** `ONLY=3 ./scripts/verify.sh`
SDL dosyası var ve ana tipleri içeriyor; izinli origin kabul, izinsiz origin
reddediliyor.

---

## 4. Electron

Cihaz uçları ve platform normalizasyonu.

| Dosya | Rol |
|---|---|
| `internal/domain/iam/model/device.go` | `NormalizePlatform`, parmak izi hash'i |
| `internal/application/iam/usecase/manage_devices.go` | Listeleme ve iptal |
| `graph/schema.graphqls` | `myDevices`, `revokeDevice` |

**Doğrulama:** `ONLY=4 ./scripts/verify.sh`
`win32`, `darwin`, `linux` üçü de doğru normalize ediliyor; listeleme ve
iptal çalışıyor.

**Birim testleri:** `go test ./internal/domain/iam/model/`

---

## 5. İki LLM kademesi

| Dosya | Rol |
|---|---|
| `internal/domain/prova/service/llm.go` | Sağlayıcı portu: tamamlama, akış, sağlık |
| `internal/infrastructure/llm/openai.go` | OpenAI-uyumlu tek istemci |
| `internal/domain/prova/model/llm_profile.go` | Kademe, model, sağlayıcı, fiyat |
| `internal/domain/prova/service/guardrail.go` | Sabit guardrail ve prompt derleme |
| `internal/domain/prova/service/character.go` | Hızlı kademe: replik + rubrik sinyalleri |
| `internal/domain/prova/service/scoring.go` | Güçlü kademe: JSON modda alıntılı puanlama |
| `internal/domain/prova/service/quote.go` | Alıntı doğrulaması |
| `internal/domain/prova/model/score.go` | Zorunlu kriter kuralı (`Evaluate`) |

**Doğrulama:** `ONLY=5 ./scripts/verify.sh`
İki kademe kayıtlı ve yayınlanmış; konuşma hızlıyı, puanlama güçlüyü
kullanıyor; alıntılar transkriptte gerçekten bulunuyor.

**Birim testleri:** `go test ./internal/domain/prova/...`

---

## 6. Otomatik dağılım

| Dosya | Rol |
|---|---|
| `internal/application/prova/service/router.go` | Kural motoru (sırayla, ilk eşleşen kazanır) |
| `internal/application/prova/service/breaker.go` | Üç durumlu devre kesici |
| `internal/application/prova/service/gateway.go` | Tek kapı: profil, maskeleme, çağrı, kayıt |
| `internal/domain/prova/model/routing.go` | Yönlendirme kaydı ve istatistik modeli |
| `internal/infrastructure/mongo/prova/routing_repository.go` | Veritabanında toplama |
| `internal/application/prova/usecase/routing_stats.go` | Tasarruf hesabı |

**Doğrulama:** `ONLY=6 ./scripts/verify.sh`
Hızlı sağlayıcı bilerek bozulduğunda oturum kesintisiz devam ediyor,
`rule=failover` kaydı oluşuyor, istatistik tasarruf yüzdesi döndürüyor.

**Birim testleri:** `go test ./internal/application/prova/service/`

---

## 7. Hesap yaşam döngüsü

| Dosya | Rol |
|---|---|
| `internal/domain/iam/model/user.go` | Silme ve kilit alanları, durum listesi |
| `internal/application/iam/usecase/account.go` | Profil, dışa aktarma, silme, geri alma, purge |
| `internal/application/prova/usecase/export.go` | Oturum ve transkript dışa aktarımı |
| `internal/infrastructure/jobs/purge.go` | Zamanlanmış kalıcı silme işi |
| `internal/infrastructure/postgres/iam/user_repository.go` | `Purge`, `ListDuePurge` |
| `internal/infrastructure/mongo/prova/session_repository.go` | `AnonymizeByEmployee` |
| `internal/infrastructure/postgres/migrations/00016_users_lifecycle.sql` | Şema |

**Doğrulama:** `ONLY=7 ./scripts/verify.sh`
Zincirin tamamı; silme sonrası e-posta null, durum `deleted`, oturum
kimliksizleştirilmiş, puan ve denetim korunmuş.

---

## 8. Güvenlik

| Dosya | Rol |
|---|---|
| `internal/application/iam/usecase/refresh_token.go` | Rotasyon, aile, yeniden kullanım tespiti |
| `internal/infrastructure/auth/denylist.go` | Token, cihaz ve aile iptali |
| `internal/application/iam/usecase/verify_login_code.go` | Kilit kontrolü ve sayaç |
| `internal/infrastructure/postgres/iam/user_repository.go` | `RecordFailedAttempt` (tek ifade) |
| `internal/infrastructure/graphql/depth.go` | Derinlik limiti |
| `internal/infrastructure/graphql/server.go` | Karmaşıklık, batch, introspection |
| `internal/shared/config/validate.go` | Fail-fast doğrulama ve `Harden` |

**Doğrulama:** `ONLY=8 ./scripts/verify.sh`
Kilit devreye giriyor ve yanıt ayırt edilemiyor; refresh reuse aileyi iptal
ediyor; derinlik ve batch limitleri reddediyor; production sertleştirmesi
çalışıyor.

Ayrıntı: [SECURITY.md](SECURITY.md)

---

## 9. E-posta doğrulama

| Dosya | Rol |
|---|---|
| `internal/application/iam/usecase/request_login_code.go` | Kod + link aynı iletide |
| `internal/application/iam/usecase/magic_link.go` | Bağlantı üretimi ve doğrulaması |
| `internal/infrastructure/auth/magic_link_service.go` | Token üretimi ve hash'leme |
| `internal/domain/notification/template/login_code.go` | İleti şablonu |
| `internal/infrastructure/email/smtp/` | Mailpit / SMTP adaptörü |
| `internal/infrastructure/email/resend/` | Üretim adaptörü |
| `deployments/docker-compose.yml` | Mailpit servisi |

**Doğrulama:** `ONLY=9 ./scripts/verify.sh`
Tek mailde hem kod hem link var, ikisi de ayrı ayrı doğruluyor, link tek
kullanımlık, bağlantı web arayüzüne işaret ediyor.

---

## 10. Kayıtlı cihaz

| Dosya | Rol |
|---|---|
| `internal/infrastructure/auth/device_signature.go` | Ed25519 challenge ve doğrulama |
| `internal/application/iam/usecase/device_challenge.go` | Challenge üretimi |
| `internal/application/iam/usecase/verify_login_code.go` | `verifyDeviceSignature` |
| `internal/infrastructure/postgres/iam/device_repository.go` | Hash'lenmiş parmak izi, açık anahtar |
| `internal/infrastructure/postgres/migrations/00017_user_devices_keypair.sql` | Taşıma |

**Doğrulama:** `ONLY=10 ./scripts/verify.sh`
Kayıt, imzasız girişin reddi, challenge imzalayarak giriş, sahte imzanın
reddi, iptal ve iptal sonrası token reddi.

**Birim testleri:** `go test ./internal/infrastructure/auth/`

---

## 11. LLM manipülasyonu

| Dosya | Rol |
|---|---|
| `internal/application/prova/usecase/llm_profile.go` | Profil yönetimi ve tek seferlik test |
| `internal/application/prova/usecase/content.go` | Generic içerik akışı (oluştur/düzenle/yayınla) |
| `internal/domain/prova/model/character.go` | Zorluk → sıcaklık ve davranış talimatı |

**Doğrulama:** `ONLY=11 ./scripts/verify.sh`
Karakter değişimi yeni sürüm üretiyor, profil değişimi denetime düşüyor,
test mutation'ı yanıt veriyor.

---

## 12. KVKK

| Dosya | Rol |
|---|---|
| `internal/infrastructure/llm/pii_masker.go` | TC kimlik, IBAN, telefon, e-posta, kart |
| `internal/application/prova/service/gateway.go` | Maskeleme yalnızca giden kopyada |
| `internal/domain/prova/model/rubric.go` | Tuzak alanı, zorunlu kriter |
| `internal/domain/prova/model/score.go` | Zorunlu kriter düşerse KALDI |
| `cmd/seed/content.go` | KVKK tuzaklı senaryo |
| `internal/application/iam/usecase/account.go` | Veri dışa aktarma |

**Doğrulama:** `ONLY=12 ./scripts/verify.sh`
LLM isteğinde TC ve IBAN maskelenmiş, transkript orijinal; zorunlu kriter
ihlali KALDI üretiyor; dışa aktarma tüm kişisel veriyi döndürüyor; ham ses
varsayılanı kapalı.

Ayrıntı: [KVKK.md](KVKK.md)

---

## Uçtan uca

```bash
./scripts/smoke.sh
```

Tek kullanıcının yolculuğu: kod iste → doğrula → cihaz kaydet → challenge
imzala → oturum başlat → konuşma sıraları → oturumu bitir → skoru al →
alıntıları bağımsız doğrula → skoru ez → token yenile → veriyi dışa aktar →
hesabı sil → kalıcı silmeyi tetikle → oturumun kimliksizleştiğini doğrula.
