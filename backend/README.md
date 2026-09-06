# Prova — Backend

Kurumsal iletişim eğitimi ve sertifikasyonu. Çalışan, yapay zekâ tarafından
canlandırılan bir karakterle rol yapar; oturum bittiğinde rubriğe göre
puanlanır ve puan, transkriptten alınmış doğrulanabilir alıntılara dayanır.

Bu depo backend'i içerir. İstemci API'sinin tamamı GraphQL'dir.

---

## Hızlı başlangıç

```bash
cd backend

# 1. Altyapı (PostgreSQL, MongoDB, Redis, Mailpit)
docker compose -f deployments/docker-compose.yml up -d postgres mongo redis mailpit

# 2. Şema
./scripts/migrate.sh up

# 3. Yapılandırma
cp .env.example .env        # JWT_SECRET, AUTH_CODE_PEPPER ve EMAIL_VERIFICATION_OTP_PEPPER üretin
#   openssl rand -base64 48   → JWT_SECRET
#   openssl rand -base64 32   → EMAIL_VERIFICATION_OTP_PEPPER
#   openssl rand -base64 32   → AUTH_CODE_PEPPER

# 4. Tohum verisi (demo organizasyonu, kullanıcılar, karakterler, senaryolar, rubrik, LLM profilleri)
go run ./cmd/seed

# 5. Sunucu
go run ./cmd/server
```

Sunucu `:8080`'de açılır. Geliştirmede GraphQL playground `/playground`,
yakalanan e-postalar `http://localhost:8025` (Mailpit).

Demo giriş bilgileri:

- Yönetici: `yonetici@prova.local` / `SecurePass123!`
- Çalışan: `calisan@prova.local` / `DevPass456!`

Bu yerel/test hesapları `login` mutation'ıyla şifreli giriş yapar. Şifresi
olmayan hesaplar için `requestLoginCode` ve `verifyLoginCode` OTP akışı devam
eder.

**Gerçek bir LLM olmadan denemek için** sahte sağlayıcıyı kullanın:

```bash
go run ./cmd/mockllm -addr :8099 &
docker exec prova-mongo mongosh --quiet prova --eval \
  "db.llm_profiles.updateMany({}, {\$set: {base_url: 'http://localhost:8099/v1'}})"
```

**Gerçek bir sağlayıcı için** `LLM_API_KEY` verin ve profillerdeki `baseUrl`
ile `model` alanlarını `updateLLMProfile` mutation'ıyla güncelleyin. Model adı
ve sağlayıcı koda gömülü değildir; çalışma anında değişir.

---

## Doğrulama

```bash
./scripts/verify.sh      # 12 gereksinimi mekanik olarak kanıtlar (12/12 PASS bekleniyor)
./scripts/smoke.sh       # tek kullanıcının yolculuğunu baştan sona yürür
go build ./... && go vet ./... && go test ./...
```

`verify.sh` altyapıyı başlatır, migration'ları uygular, tohum verisini yazar,
sahte sağlayıcıyı ve sunucuyu ayağa kaldırır, sonra her gereksinimi tek tek
PASS/FAIL basar. Herhangi biri FAIL ise çıkış kodu 1'dir.

Gereksinim → dosya → doğrulama komutu eşlemesi için
[docs/TRACEABILITY.md](docs/TRACEABILITY.md).

---

## Mimari

Hexagonal katmanlama korunuyor:

```
cmd/
  server/     HTTP sunucusu ve bağımlılık grafiği
  seed/       tohum verisi
  verify/     12 gereksinimin mekanik doğrulaması
  smoke/      uçtan uca duman testi
  schema/     GraphQL şemasının SDL olarak dışa aktarımı
  mockllm/    OpenAI-uyumlu sahte sağlayıcı (test altyapısı)
  configdump/ güvenlikle ilgili varsayılanların makine okunur çıktısı

graph/                     GraphQL şeması, üretilmiş kod ve resolver'lar
internal/
  domain/prova/            Prova alan modeli: belgeler, sürümleme, prompt,
                           alıntı doğrulama, LLM portu
  domain/iam/              kimlik: kullanıcı, cihaz, giriş kodu, magic link,
                           refresh token
  domain/audit/            denetim kaydı portu
  application/prova/       oturum akışı, LLM geçidi, kural tabanlı yönlendirici
  application/iam/         giriş, cihaz, hesap yaşam döngüsü
  infrastructure/          PostgreSQL, MongoDB, LLM istemcisi, GraphQL sunucusu,
                           e-posta adaptörleri, zamanlanmış işler
  shared/                  yapılandırma, hata, ara katman, önbellek, kayıt
```

### İki veritabanı, iki soru

**PostgreSQL** "bu kim?" sorusunu yanıtlar: kullanıcılar, cihazlar, roller,
token'lar, denetim kaydı.

**MongoDB** "ne oynandı ve nasıl puanlandı?" sorusunu yanıtlar: karakterler,
senaryolar, rubrikler, LLM profilleri, oturumlar, transkriptler, puanlar,
yönlendirme kayıtları.

Ayrım keskin tutuluyor. İkisi karıştığında ortaya, ne kimliği ne içeriği
doğru modelleyen tek bir şema çıkar.

### Sürümleme

İçerik belgeleri (karakter, senaryo, rubrik, LLM profili) sürümlenir ve
yayınlandıktan sonra **değişmez**. Düzenleme yeni sürüm yaratır; soy kimliği
sabit kalır. Kural depo seviyesinde zorlanır: `UpdateDraft` sorgusu
`status: draft` koşulunu taşır, dolayısıyla yayınlanmış bir belgeye güncelleme
hiç eşleşmez.

Oturum başlarken senaryo, karakter ve rubrik **sürüm numaraları** oturum
belgesine yazılır. İçerik yarın değişse bile oturum kendi sürümüyle puanlanır.
Sertifikasyon iddiası tam olarak buna dayanır.

### Prova model stack'i

Prova'da modeller görevlerine göre ayrıdır; tek bir model bütün hattı
çalıştırmaz:

| Görev | Model / runtime | Konum |
|---|---|---|
| Canlı role-play / persona | `Qwen/Qwen3-4B-Instruct-2507` | vLLM, düşük gecikmeli hızlı kademe |
| Oturum sonu değerlendirme | `Qwen/Qwen3-30B-A3B-Instruct-2507` | vLLM/OpenAI-compatible güçlü kademe |
| Yerel Speech-to-Text | Whisper Large V3 Turbo | Electron + `whisper.cpp`; ham ses backend'e gitmez |
| Behavioral classifier | `dbmdz/bert-base-turkish-cased` (BERTurk) | Kendi multi-label dataset'imiz, sonra ONNX/Electron |

Canlı akışta yerel Whisper metin üretir; Qwen 4B persona yanıtını üretir ve
BERTurk davranış sinyallerini yerel olarak çıkarır. Oturum bitince transcript,
rubrik ve oturum bağlamı Qwen 30B-A3B evaluator'a gider. BERTurk, güçlü
evaluator'ın yerine geçmez; canlı göstergeler içindir.

### İki LLM kademesi

- **Hızlı** kademe her konuşma sırasında karakteri canlandırır. Gecikme burada
  kullanıcı deneyiminin kendisidir.
- **Güçlü** kademe oturum sonunda rubriğe göre puanlar. Burada gecikme
  önemsiz, doğruluk her şeydir.

Yönlendirme kuralları sırayla değerlendirilir, ilk eşleşen kazanır:

| # | Kural | Kademe | Neden |
|---|-------|--------|-------|
| 1 | `scoring` | güçlü | Puan bir sertifikanın dayanağı. |
| 2 | `mandatory_signal` | güçlü | Kararın sonucu doğrudan KALDI olabilir. |
| 3 | `failover` | güçlü | Hızlı kademenin devresi açık; oturum kesilmemeli. |
| 4 | `long_input` | güçlü | Hızlı modeller uzun bağlamda bozulur. |
| 5 | `default` | hızlı | Olağan yol; tasarrufun tamamı buradan gelir. |

Her karar kaydedilir — başarısızlar dâhil. `routingStats` sorgusu kademe
başına çağrı, gecikme, maliyet ve "hepsi güçlüye gitseydi" karşılaştırmasını
döndürür.

### Guardrail

Karakter prompt'unun en başındaki sabit metin koda gömülüdür ve yöneticinin
değiştirdiği hiçbir metinle ezilemez. İçeriği: karakter rolden çıkmaz, yapay
zekâ olduğunu söylemez, gerçek kişisel veri üretmez, çalışana doğru prosedürü
söylemez, hakaret üretmez.

Konumu tek başına yeterli bir savunma değildir; metnin kendisi de kendisini
ezmeye çalışan talimatları yok saymayı emreder. İkisi birlikte çalışır.

---

## GraphQL

Tek uç: `POST /graphql`. Abonelikler aynı yolda WebSocket üzerinden
(`graphql-transport-ws`). REST yalnızca `/health/live`, `/health/ready` ve
`/metrics` için kalmıştır.

Şema: [`graph/schema.graphqls`](graph/schema.graphqls).
İstemci kod üretimi için SDL: [`../schema.graphql`](../schema.graphql)
(`go run ./cmd/schema -out ../schema.graphql`).

### İki directive

```graphql
directive @auth on FIELD_DEFINITION | OBJECT
directive @permission(requires: String!) on FIELD_DEFINITION
```

`@auth` alan bazındadır, middleware değil: kod isteme ve kod doğrulama
mutation'ları token'sız çağrılmak zorundadır.

`@permission` yetki yoksa **nullable alanda null**, nullable olmayan alanda
hata döndürür. Çalışan kendi puanını görebilmeli ama ezilip ezilmediğini
görmemeli; ezme alanını hata yapmak tüm skor sorgusunu patlatırdı.

### Örnekler

**Kod iste** (token gerekmez):

```graphql
mutation {
  requestLoginCode(input: { email: "calisan@prova.local" }) {
    sent
    expiresInSeconds
    resendAfterSeconds
  }
}
```

Yanıt, adresin kayıtlı olup olmadığına dair hiçbir sinyal taşımaz.

**Kodu doğrula ve cihaz kaydet**:

```graphql
mutation {
  verifyLoginCode(input: {
    email: "calisan@prova.local"
    code: "482913"
    device: {
      fingerprint: "makine-parmak-izi"
      name: "Şube PC"
      platform: "win32"          # Electron process.platform değeri
      publicKey: "base64-ed25519-acik-anahtar"
    }
  }) {
    accessToken
    refreshToken
    organizationId
    user { id email emailVerified }
    device { id platform isNew }
  }
}
```

Kayıtlı anahtarı olan cihazlarda sonraki girişlerde imza zorunludur:

```graphql
mutation { requestDeviceChallenge(input: {
  email: "calisan@prova.local", fingerprint: "makine-parmak-izi"
}) { challenge expiresAt } }
```

Cihaz `challenge` metnini özel anahtarıyla imzalar ve `deviceSignature`
alanında gönderir.

**Oturum oyna**:

```graphql
mutation { startSession(input: { scenarioLineageId: "…" }) { id status } }

mutation {
  submitTurn(input: { sessionId: "…", text: "Merhaba, nasıl yardımcı olabilirim?" }) {
    index role text signals maskedFieldCount
  }
}

mutation { endSession(sessionId: "…") { id status } }
```

**Puanı oku** (alıntılar ve doğrulama bayrağıyla):

```graphql
query {
  session(id: "…") {
    status
    score {
      total maxTotal passed failedMandatoryKeys model
      criteria {
        criterionKey title points maxPoints mandatory
        rationale quote turnIndex quoteVerified
      }
      override { reason previousTotal }   # yetkisiz kullanıcıda null
    }
  }
}
```

**Canlı oturum olayları** (WebSocket):

```graphql
subscription {
  sessionEvents(sessionId: "…") {
    type          # TRANSCRIPT_CHUNK | RUBRIC_SIGNAL | CHARACTER_REPLY |
                  # SCORING_PROGRESS | SCORE_READY
    text progress
    turn { index role text }
    score { total passed }
  }
}
```

Bağlantı `connectionParams` içinde `{"Authorization": "Bearer …"}` gönderir;
token orada doğrulanır ve oturum sahipliği bağlantı anında kontrol edilir.

**Yönlendirme istatistiği**:

```graphql
query {
  routingStats {
    perTier { tier calls avgLatencyMs costUsd inputTokens outputTokens }
    failoverCount totalCostUsd allStrongCostUsd savingsPercent
  }
}
```

**LLM profilini çalışma anında değiştir ve canlıya almadan dene**:

```graphql
mutation {
  updateLLMProfile(lineageId: "…", input: {
    tier: FAST, provider: "openai-compatible"
    baseUrl: "http://localhost:8000/v1", model: "Qwen/Qwen3-4B-Instruct-2507"
    temperature: 0.85, topP: 0.95, maxTokens: 600
    systemPromptSuffix: "Yanıtlarını kısa tut."
    inputCostPer1K: 0.00015, outputCostPer1K: 0.0006
  }) { version status }
}

mutation {
  testLLMProfile(input: { profileLineageId: "…", prompt: "Merhaba, kimsin?" }) {
    model output latencyMs inputTokens costUsd error
  }
}
```

**Hesap yaşam döngüsü**:

```graphql
mutation { exportMyData { document } }          # KVKK erişim hakkı
mutation { requestAccountDeletion { scheduledAt cancellable } }
mutation { cancelAccountDeletion { requestedAt } }
query    { deletionStatus { requestedAt scheduledAt cancellable } }
```

---

## Ortam değişkenleri

Tam liste ve gerekçeleri için [`.env.example`](.env.example). Yükte olanlar:

| Değişken | Varsayılan | Not |
|---|---|---|
| `APP_ENV` | `development` | `production` katı doğrulamayı açar ve introspection'ı kapatır |
| `JWT_SECRET` | *(dev varsayılanı)* | Üretimde zorunlu; varsayılanla boot durur |
| `EMAIL_VERIFICATION_OTP_PEPPER` | *(boş)* | Üretimde zorunlu; kullanıcıya bağlı kod digest'lerinin tek koruması |
| `EMAIL_VERIFICATION_OTP_TTL_SECONDS` | `300` | Doğrulama challenge'ı Redis'te tam 5 dakika yaşar |
| `EMAIL_VERIFICATION_MAX_ATTEMPTS` | `5` | Challenge başına yanlış deneme sınırı |
| `EMAIL_VERIFICATION_RESEND_COOLDOWN_SECONDS` | `60` | Aynı adrese yeniden gönderim aralığı |
| `AUTH_CODE_PEPPER` | *(boş)* | Parolasız giriş kodunun ayrı pepper'ı |
| `MONGO_URI` | `mongodb://localhost:27017` | Üretimde varsayılanla boot durur |
| `EMAIL_PROVIDER` | `resend` | `resend`, `smtp`, `none`. Üretimde `none` ile boot durur |
| `RESEND_API_KEY` | *(boş)* | Resend API anahtarı; kaynak koda/loglara yazılmaz |
| `RESEND_FROM_EMAIL` | *(boş)* | Örn. `'Prova <onboarding@resend.dev>'`; özel domain doğrulanınca yalnızca bu değer değişir |
| `WEB_BASE_URL` | `http://localhost:3000` | Magic link'in açılacağı arayüz |
| `TOKEN_ACCESS_TTL_SECONDS` | `900` | Kısa; oturum refresh rotasyonuyla sürer |
| `TOKEN_REFRESH_TTL_SECONDS` | `2592000` | Ailenin toplam ömrü; rotasyon uzatmaz |
| `TOKEN_MAX_FAILED_ATTEMPTS` | `10` | Eşik aşılınca hesap geçici kilitlenir |
| `LIFECYCLE_DELETION_GRACE_SECONDS` | `2592000` | Geri alma penceresi; üretimde en az 24 saat |
| `LLM_API_KEY` / `LLM_API_KEYS` | *(boş)* | Anahtarlar profilde değil env'de |
| `LLM_FAST_BASE_URL` | `http://localhost:8000/v1` | Seed'deki Qwen 4B persona/vLLM endpoint'i |
| `LLM_STRONG_BASE_URL` | `http://localhost:8001/v1` | Seed'deki Qwen 30B-A3B evaluator/vLLM endpoint'i |
| `LLM_STORE_RAW_AUDIO` | `false` | Kapalı kalmalı; aşağıya bakın |
| `GRAPHQL_MAX_DEPTH` | `12` | Fragment içindeki derinlik de sayılır |
| `GRAPHQL_MAX_COMPLEXITY` | `500` | |
| `GRAPHQL_MAX_BATCH` | `5` | |

### Ham ses

**Ham ses backend'e hiç gelmez.** İstemci konuşmayı kendi cihazında metne
çevirir ve sunucuya yalnızca metin gönderir. `LLM_STORE_RAW_AUDIO` varsayılan
olarak kapalıdır ve kapalı kalmalıdır: ses kaydı biyometrik veridir, KVKK'da
özel nitelikli kişisel veri sayılır ve saklanması ayrı bir açık rıza
gerektirir.

---

## Belgeler

- [docs/TRACEABILITY.md](docs/TRACEABILITY.md) — 12 gereksinim → dosyalar → doğrulama komutu
- [docs/SECURITY.md](docs/SECURITY.md) — saldırı → savunma
- [docs/KVKK.md](docs/KVKK.md) — hangi veri nerede, ne kadar saklanıyor, nasıl siliniyor
- [docs/WEBSOCKET.md](docs/WEBSOCKET.md) — gerçek zamanlı taşıma notları

---

## Devre dışı bırakılan modüller

Bu depo masterfabric-go'dan türedi. Prova'ya ait olmayan üç modül **koddan
silinmedi ama hiçbir yerden bağlanmıyor**:

- `internal/gateway/`, `internal/infrastructure/gateway/` — API ağ geçidi
- `internal/{domain,application}/apimanagement/` — dinamik uç yönetimi
- `internal/infrastructure/kafka/` — olay veri yolu artık süreç içi

Ağ geçidi interceptor'larındaki PII maskeleyici kurtarıldı ve LLM hattına
taşındı (`internal/infrastructure/llm/pii_masker.go`).

Kafka servisleri compose'da `kafka` profilindedir ve varsayılan stack'te
başlamaz.
