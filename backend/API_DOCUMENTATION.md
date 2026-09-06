# Prova Backend API Dokümantasyonu

Bu belge, Prova backend'inin tekil kullanım ve işletim dokümanıdır. API sözleşmesi,
kimlik doğrulama, yetkilendirme, oturum akışı, içerik sürümleme, puanlama,
gerçek zamanlı olaylar, hata sözleşmesi, güvenlik ve yerel çalıştırma bilgileri
aynı yerde tutulur.

> Durum: Güncel backend API yüzeyi
>
> Son doğrulama: `go test ./...` ve `npm run build --workspace web -- --webpack`

## 1. Hızlı başvuru

| Amaç | Adres | Kimlik doğrulama |
|---|---|---|
| GraphQL API | `POST http://localhost:8080/graphql` | Alan bazında; çoğu alan Bearer token ister |
| GraphQL sorgularını GET ile çalıştırma | `GET http://localhost:8080/graphql` | Sorguya göre |
| GraphQL subscription | `ws://localhost:8080/graphql` | `connection_init` içindeki token |
| Geliştirme GraphiQL arayüzü | `http://localhost:8080/playground` | Yalnızca development |
| Liveness | `GET http://localhost:8080/health/live` | Gerekmez |
| Readiness | `GET http://localhost:8080/health/ready` | Gerekmez |
| Prometheus metrikleri | `GET http://localhost:8080/metrics` | Gerekmez |

Prova'nın istemci API'si GraphQL'dir. REST katmanında yalnızca sağlık ve metrik
uçları bulunur. `backend/graph/schema.graphqls` kaynak şemadır; bu dosya ise
şemanın nasıl kullanılacağını ve backend davranışını açıklar.

## 2. Backend'i yerelde çalıştırma

### Gereksinimler

- Go `1.26.4` veya `backend/go.mod` ile uyumlu daha yeni bir sürüm
- Docker ve Docker Compose
- E-posta akışını test etmek için Mailpit
- Web arayüzü build'i için Node.js ve npm

### Kurulum

```bash
cd backend

# PostgreSQL, MongoDB, Redis ve Mailpit
docker compose -f deployments/docker-compose.yml up -d postgres mongo redis mailpit

# Veritabanı migration'ları
./scripts/migrate.sh up

# Yapılandırma
cp .env.example .env

# .env içindeki en az şu sırları gerçek rastgele değerlerle doldurun:
# JWT_SECRET=$(openssl rand -base64 48)
# EMAIL_VERIFICATION_OTP_PEPPER=$(openssl rand -base64 32)
# AUTH_CODE_PEPPER=$(openssl rand -base64 32)

# Demo organizasyonu, içerikler ve LLM profilleri
go run ./cmd/seed

# API sunucusu
go run ./cmd/server
```

Sunucu varsayılan olarak `0.0.0.0:8080` üzerinde dinler. Mailpit arayüzü
`http://localhost:8025` adresindedir.

### Sağlık kontrolü

```bash
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
curl http://localhost:8080/metrics
```

Beklenen liveness yanıtı:

```json
{"status":"alive"}
```

Readiness, PostgreSQL veya Redis erişilemiyorsa `503` döner:

```json
{
  "status": "ready",
  "services": {
    "postgres": "healthy",
    "redis": "healthy"
  }
}
```

## 3. HTTP ve GraphQL istek biçimi

GraphQL HTTP isteği JSON gövdesiyle gönderilir:

```http
POST /graphql HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <access-token>

{
  "query": "query { me { id email firstName lastName } }",
  "variables": {}
}
```

Başarılı yanıt örneği:

```json
{
  "data": {
    "me": {
      "id": "00000000-0000-0000-0000-000000000000",
      "email": "calisan@example.com",
      "firstName": "Ayşe",
      "lastName": "Yılmaz"
    }
  }
}
```

Komut satırından örnek:

```bash
curl -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  --data-binary '{"query":"query { me { id email permissions } }"}'
```

### Genel kurallar

- UUID alanları standart UUID biçimindedir.
- Zaman alanları GraphQL `Time` scalar'ı ile taşınır.
- Serbest yapıdaki alanlar `JSON` scalar'ını kullanır.
- Kimlik doğrulama HTTP middleware'inde zorunlu kapı olarak değil, GraphQL
  alanlarındaki `@auth` directive'i ile uygulanır. Böylece login kodu isteme
  gibi public mutation'lar çalışabilir.
- Aynı HTTP isteğinde batch operasyonları kullanılabilir; varsayılan üst sınır
  `GRAPHQL_MAX_BATCH=5` değeridir.
- Sorgu derinliği varsayılan olarak `12`, karmaşıklığı `500` ile sınırlıdır.

## 4. Kimlik doğrulama ve token yaşam döngüsü

### 4.1 Standart giriş: e-posta ve tek kullanımlık kod

Production akışı e-posta ile gönderilen kısa ömürlü tek kullanımlık koddur.
Kayıtlı olmayan adresler için de yanıt gövdesi aynıdır; böylece hesap varlığı
kolayca keşfedilemez.

1. `requestLoginCode` ile kod istenir.
2. Kullanıcı e-postadaki kodu veya mevcut hesabı için magic link'i açar.
3. `verifyLoginCode` veya `verifyMagicLink` access ve refresh token döndürür.
4. API isteklerinde access token Bearer header'ında taşınır.
5. Access token süresi dolunca `refreshToken` ile döndürme yapılır.

```graphql
mutation RequestCode($input: RequestLoginCodeInput!) {
  requestLoginCode(input: $input) {
    sent
    expiresInSeconds
    resendAfterSeconds
  }
}
```

```json
{
  "input": { "email": "calisan@example.com" }
}
```

```graphql
mutation VerifyCode($input: VerifyLoginCodeInput!) {
  verifyLoginCode(input: $input) {
    accessToken
    refreshToken
    expiresAt
    organizationId
    user { id email status emailVerified permissions }
    device { id name platform isNew }
  }
}
```

```json
{
  "input": {
    "email": "calisan@example.com",
    "code": "482913",
    "device": {
      "fingerprint": "desktop-fingerprint",
      "name": "Şube PC",
      "platform": "win32",
      "publicKey": "base64-ed25519-public-key",
      "macAddress": "00:11:22:33:44:55"
    }
  }
}
```

İlk kez giriş yapan davetli kullanıcıda cihaz bilgisi zorunludur. Kayıtlı açık
anahtarı bulunan cihazlar için sunucunun verdiği challenge, cihaz özel
anahtarıyla imzalanarak `deviceSignature` alanında gönderilir.

### 4.2 Magic link

`requestLoginCode`, daha önce e-postası doğrulanmış hesaplar için aynı iletide
magic link üretebilir. Link web arayüzünün `/auth/magic?token=...` yoluna gider.
Link tek kullanımlıktır ve doğrulama şu mutation ile yapılır:

```graphql
mutation {
  verifyMagicLink(input: {
    token: "<magic-link-token>"
    device: {
      fingerprint: "desktop-fingerprint"
      name: "Ofis Mac"
      platform: "darwin"
    }
  }) {
    accessToken
    refreshToken
    expiresAt
    user { id email }
  }
}
```

### 4.3 Lokal/test şifre girişi

Şemada bulunan `login` mutation'ı, `password_hash` atanmış lokal/test
hesaplarını destekler. Standart ürün akışı passwordless'tır; şifreli login
yalnızca hesapta kayıtlı bir bcrypt password hash varsa çalışır.

```graphql
mutation {
  login(input: {
    email: "admin@example.com"
    password: "local-test-password"
  }) {
    accessToken
    refreshToken
    expiresAt
    user { id email }
  }
}
```

### 4.4 Cihaz challenge'ı

```graphql
mutation {
  requestDeviceChallenge(input: {
    email: "calisan@example.com"
    fingerprint: "desktop-fingerprint"
  }) {
    challenge
    expiresAt
  }
}
```

### 4.5 Token yenileme ve çıkış

```graphql
mutation {
  refreshToken(input: { refreshToken: "<refresh-token>" }) {
    accessToken
    refreshToken
    expiresAt
    organizationId
    user { id email }
  }
}
```

Refresh token döndürülür ve eski token tekrar kullanılırsa refresh ailesi iptal
edilir. Çıkış yapmak için access token ile:

```graphql
mutation { logout }
```

## 5. GraphQL operasyonları

Bu bölümdeki `auth` ve `permission` değerleri şemadaki gerçek kurallardır.
İzin yoksa nullable alanlarda `null`, nullable olmayan alanlarda GraphQL hatası
döner.

### 5.1 Query'ler

| Query | Parametreler | Auth / permission | Açıklama |
|---|---|---|---|
| `me` | — | auth | Aktif kullanıcının güncel profilini ve RBAC izinlerini döndürür. |
| `myDevices` | — | auth | Kullanıcının eşleştirilmiş cihazları. |
| `organizationUsers` | — | auth + `users:read` | Aktif kurum üyeleri ve cihazları. |
| `mySessions` | `limit`, `offset` | auth | Kullanıcının oturumları; varsayılan `20/0`. |
| `session` | `id` | auth | Tek oturum ve varsa puanı. |
| `scenarios` | `publishedOnly` | auth | Senaryo listesi; varsayılan yalnızca yayınlanmışlar. |
| `scenario` | `lineageId`, `version` | auth | Senaryonun belirli veya güncel yayınlanmış sürümü. |
| `characters` | `publishedOnly` | auth | Karakter listesi. |
| `character` | `lineageId`, `version` | auth | Karakter sürümü. |
| `rubrics` | `publishedOnly` | auth | Rubrik listesi. |
| `rubric` | `lineageId`, `version` | auth | Rubrik sürümü. |
| `llmProfiles` | — | auth + `llm:read` | LLM profil sürümleri. |
| `routingStats` | `from`, `to` | auth + `routing:read` | Kademe, maliyet, gecikme ve tasarruf istatistikleri. |
| `routingRecords` | `sessionId`, `limit` | auth + `routing:read` | Tek tek LLM yönlendirme kayıtları. Varsayılan limit `50`. |
| `auditLog` | `limit`, `offset` | auth + `audit:read` | Kurum kapsamındaki denetim kayıtları. |
| `deletionStatus` | — | auth | Kullanıcının hesap silme durumunu döndürür. |

Örnek profil ve oturum sorgusu:

```graphql
query Dashboard {
  me { id email firstName lastName status permissions }
  mySessions(limit: 20, offset: 0) {
    id status startedAt endedAt
    scenario { lineageId version }
    score { total maxTotal passed failedMandatoryKeys }
  }
}
```

Puanın içindeki `score` alanı ayrıca `score:read` izniyle korunur. Çalışan
oturumunu görebilir fakat puan görme izni yoksa sonuç alanı gizlenebilir.

### 5.2 Kimlik ve kullanıcı mutation'ları

| Mutation | Girdi | Auth / permission |
|---|---|---|
| `inviteUser` | `InviteUserInput` | auth + `users:write` |
| `login` | `PasswordLoginInput` | public; lokal/test password hash |
| `requestLoginCode` | `RequestLoginCodeInput` | public |
| `verifyLoginCode` | `VerifyLoginCodeInput` | public |
| `verifyMagicLink` | `VerifyMagicLinkInput` | public |
| `requestDeviceChallenge` | `DeviceChallengeInput` | public |
| `refreshToken` | `RefreshTokenInput` | public |
| `logout` | — | auth |
| `revokeDevice` | `deviceId` | auth |
| `updateProfile` | `UpdateProfileInput` | auth |

Kullanıcı daveti örneği:

```graphql
mutation {
  inviteUser(input: {
    email: "yeni.kullanici@example.com"
    firstName: "Yeni"
    lastName: "Kullanıcı"
    role: EMPLOYEE
  }) {
    invited
    email
  }
}
```

`role` değerleri `EMPLOYEE` ve `ORG_ADMIN`'dır. Davet edilen hesap ilk login
kodunu doğruladığında e-posta doğrulanır ve inactive üyelik active hale gelir.

### 5.3 İçerik mutation'ları

Karakter, senaryo ve rubrikler sürümlüdür. Önce draft oluşturulur/güncellenir,
sonra belirli sürüm yayınlanır.

| Mutation | Girdi | Auth / permission |
|---|---|---|
| `createCharacter` | `CharacterInput` | auth + `content:write` |
| `updateCharacter` | `lineageId`, `CharacterInput` | auth + `content:write` |
| `publishCharacter` | `lineageId`, `version` | auth + `content:publish` |
| `createScenario` | `ScenarioInput` | auth + `content:write` |
| `updateScenario` | `lineageId`, `ScenarioInput` | auth + `content:write` |
| `publishScenario` | `lineageId`, `version` | auth + `content:publish` |
| `createRubric` | `RubricInput` | auth + `content:write` |
| `updateRubric` | `lineageId`, `RubricInput` | auth + `content:write` |
| `publishRubric` | `lineageId`, `version` | auth + `content:publish` |

Karakter oluşturma:

```graphql
mutation {
  createCharacter(input: {
    name: "Zor Müşteri"
    persona: "Sabırsız ama gerçekçi bir müşteri."
    behaviorRules: ["Kısa cevap ver", "Doğru sorulursa gizli bilgiyi paylaş"]
    difficulty: HIGH
    hiddenFacts: ["Sipariş numarası 12345"]
  }) {
    id lineageId version status name difficulty
  }
}
```

Senaryo oluşturma:

```graphql
mutation {
  createScenario(input: {
    title: "İade talebi"
    context: "Müşteri aldığı ürünü iade etmek istiyor."
    objective: "Çalışan, prosedüre uygun şekilde talebi yönetmeli."
    characterLineageId: "<character-lineage-id>"
    rubricLineageId: "<rubric-lineage-id>"
    maxTurns: 20
  }) {
    id lineageId version status title characterRef rubricRef maxTurns
  }
}
```

Yayınlama örneği:

```graphql
mutation {
  publishScenario(lineageId: "<scenario-lineage-id>", version: 1) {
    id lineageId version status publishedAt
  }
}
```

### 5.4 Eğitim oturumu ve puanlama

| Mutation | Girdi | Auth / permission |
|---|---|---|
| `startSession` | `StartSessionInput` | auth |
| `submitTurn` | `SubmitTurnInput` | auth |
| `endSession` | `sessionId` | auth |
| `overrideScore` | `OverrideScoreInput` | auth + `score:override` |

Oturum başlatma:

```graphql
mutation {
  startSession(input: { scenarioLineageId: "<published-scenario-lineage-id>" }) {
    id
    status
    scenario { lineageId version }
    character { lineageId version }
    rubric { lineageId version }
    startedAt
  }
}
```

Çalışan mesajı gönderme:

```graphql
mutation {
  submitTurn(input: {
    sessionId: "<session-id>"
    requestId: "<stable-client-request-id>"
    text: "Merhaba, size nasıl yardımcı olabilirim?"
  }) {
    index
    role
    text
    signals
    maskedFieldCount
    createdAt
  }
}
```

`requestId`, ağ tekrar denemelerinde aynı çalışan/karakter turn çiftinin iki
kez yazılmasını önler. Aynı `requestId` farklı metinle tekrar kullanılırsa
`CONFLICT` hatası döner.

Oturumu bitirme ve sonucu okuma:

```graphql
mutation {
  endSession(sessionId: "<session-id>") {
    id
    status
    endedAt
    score {
      total
      maxTotal
      passed
      failedMandatoryKeys
      model
      criteria {
        criterionKey
        title
        points
        maxPoints
        mandatory
        rationale
        quote
        turnIndex
        quoteVerified
      }
    }
  }
}
```

Puan formülü:

```text
total   = Σ(kriter puanı × kriter ağırlığı)
maximum = Σ(kriter maksimumu × kriter ağırlığı)
ratio   = total / maximum
passed  = ratio >= passThreshold VE başarısız zorunlu kriter yok
```

Model alıntısı belirtilen turn içinde bulunamazsa `quoteVerified=false` olarak
kalır. Zorunlu kriterin başarısız olması toplam puan yeterli olsa bile sonucu
başarısız yapar.

Yönetici puan ezme örneği:

```graphql
mutation {
  overrideScore(input: {
    scoreId: "<score-id>"
    total: 82.5
    passed: true
    reason: "Manuel incelemede kriter kanıtı doğrulandı."
  }) {
    id total maxTotal passed
    override { overriddenBy reason previousTotal previousPassed overriddenAt }
  }
}
```

Gerekçe zorunludur ve yeni puan `0..maxTotal` aralığında olmalıdır. Eski puan
ve karar silinmez; override kaydıyla birlikte saklanır.

### 5.5 LLM profil yönetimi ve yönlendirme

| Mutation / Query | Girdi | Auth / permission |
|---|---|---|
| `llmProfiles` | — | auth + `llm:read` |
| `createLLMProfile` | `LLMProfileInput` | auth + `llm:write` |
| `updateLLMProfile` | `lineageId`, `LLMProfileInput` | auth + `llm:write` |
| `publishLLMProfile` | `lineageId`, `version` | auth + `llm:write` |
| `testLLMProfile` | `TestLLMProfileInput` | auth + `llm:write` |
| `routingStats` | `from`, `to` | auth + `routing:read` |
| `routingRecords` | `sessionId`, `limit` | auth + `routing:read` |

LLM profili modeli ve sağlayıcıyı çalışma anında değiştirir; API anahtarları
profilde tutulmaz, environment üzerinden sağlanır.

```graphql
mutation {
  testLLMProfile(input: {
    profileLineageId: "<profile-lineage-id>"
    prompt: "Müşteriyi kibarca karşıla."
  }) {
    profileId
    model
    output
    latencyMs
    inputTokens
    outputTokens
    costUsd
    error
  }
}
```

İki kademe vardır:

- `FAST`: canlı persona cevapları için.
- `STRONG`: oturum sonu değerlendirme ve gerektiğinde failover için.

### 5.6 KVKK ve hesap yaşam döngüsü

```graphql
query { deletionStatus { requestedAt scheduledAt cancellable } }

mutation { exportMyData { exportedAt document } }

mutation {
  requestAccountDeletion {
    requestedAt scheduledAt cancellable
  }
}

mutation {
  cancelAccountDeletion {
    requestedAt scheduledAt cancellable
  }
}
```

`exportMyData`, kullanıcının profilini, cihazlarını, oturumlarını ve kendi
denetim kayıtlarını tek JSON belgede döndürür. Silme talebi geri alma penceresi
boyunca iptal edilebilir; kalıcı silme çalıştıktan sonra `cancellable=false`
olur.

## 6. GraphQL Subscription ve canlı oturum olayları

Subscription, HTTP ile aynı `/graphql` adresinde `graphql-transport-ws`
protokolünü kullanır. Eski gateway dokümanındaki `/api/v1/ws` adresi Prova'nın
güncel GraphQL transport'u değildir.

Bağlantı akışı:

```text
1. WebSocket ws://localhost:8080/graphql
2. Sec-WebSocket-Protocol: graphql-transport-ws
3. connection_init + Authorization payload
4. subscribe + GraphQL subscription
```

Örnek mesajlar:

```json
{
  "type": "connection_init",
  "payload": { "Authorization": "Bearer <access-token>" }
}
```

```json
{
  "id": "session-events-1",
  "type": "subscribe",
  "payload": {
    "query": "subscription($sessionId: UUID!) { sessionEvents(sessionId: $sessionId) { sessionId type text progress turn { index role text } score { total passed } occurredAt } }",
    "variables": { "sessionId": "<session-id>" }
  }
}
```

Subscription şeması:

```graphql
subscription($sessionId: UUID!) {
  sessionEvents(sessionId: $sessionId) {
    sessionId
    type
    text
    signals
    progress
    turn { index role text }
    score { total maxTotal passed }
    occurredAt
  }
}
```

Olay türleri:

| Tür | Dolu alanlar | Anlam |
|---|---|---|
| `TRANSCRIPT_CHUNK` | `text`, gerekirse `turn` | Transkript parçası |
| `RUBRIC_SIGNAL` | `signals` | Tetiklenen kriter anahtarları |
| `CHARACTER_REPLY` | `text`, `turn` | Persona yanıtı |
| `SCORING_PROGRESS` | `progress` | `0..100` arası puanlama ilerlemesi |
| `SCORE_READY` | `score` | Tamamlanmış puan |

Subscription bağlantısında token ve oturum sahipliği bağlantı/subscribe
aşamasında doğrulanır. `score` alanı ayrıca `score:read` iznine tabidir.

## 7. Temel veri modelleri

### Kullanıcı ve cihaz

`User` alanları: `id`, `email`, `firstName`, `lastName`, `status`,
`emailVerified`, `organizationId`, `permissions`, silme zamanları ve
`createdAt`.

`Device` alanları: `id`, `name`, `platform`, son gözlenen `ipAddress`,
`macAddress`, `lastSeenAt`, `revokedAt`, `hasPublicKey`.

Kullanıcı/rol kimliği PostgreSQL'de, oturum ve eğitim belgeleri MongoDB'de
tutulur. Kurum kapsamı resolver ve repository katmanına kadar taşınır.

### Sürümlü içerik

Karakter, senaryo, rubrik ve LLM profilleri şu yaşam döngüsünü kullanır:

```text
DRAFT -> PUBLISHED -> ARCHIVED
```

Her sürüm `lineageId`, `version`, `id` ve `status` taşır. Senaryo; karakter ve
rubrik sürümlerini `DocumentRef` ile tam olarak bağlar. Oturum başladığında bu
referanslar snapshot olarak saklanır; içerik sonradan değişse bile puanlama aynı
sürümler üzerinden yapılır.

### Oturum

`Session.status` değerleri: `ACTIVE`, `SCORING`, `COMPLETED`, `ABANDONED`.
Oturum; çalışan, senaryo, karakter, rubrik ve turn listesini taşır. `maxTurns`
aşılırsa yeni turn kabul edilmez. `submitTurn` başarılı olduğunda çalışan ve
karakter turn'ü birlikte kalıcılaştırılır.

### Puan

`Score` toplam puanı, maksimum puanı, geçme kararını, başarısız zorunlu kriter
anahtarlarını, kriter bazlı gerekçeleri, alıntı doğrulama bayrağını, model
kimliğini ve varsa manuel override'ı taşır.

## 8. Yetki matrisi

| Permission | Kullanıldığı alanlar |
|---|---|
| `users:read` | `organizationUsers` |
| `users:write` | `inviteUser` |
| `content:write` | Karakter/senaryo/rubrik create ve update |
| `content:publish` | Karakter/senaryo/rubrik publish |
| `score:read` | `Session.score`, `SessionEvent.score` |
| `score:override` | `overrideScore`, `Score.override` |
| `llm:read` | `llmProfiles` |
| `llm:write` | LLM profil create/update/publish/test |
| `routing:read` | `routingStats`, `routingRecords` |
| `audit:read` | `auditLog` |

`@auth` token'ın geçerli olmasını, `@permission` ise kullanıcının etkin kurum
ve rol kapsamındaki iznini kontrol eder. Geçersiz veya süresi dolmuş Bearer
token, kimliksiz istekten farklı olarak istemciye yeniden giriş gerektiğini
bildiren `401` yanıtı üretir.

## 9. Hata sözleşmesi

GraphQL hataları standart `errors` dizisinde, makine tarafından okunabilir
`extensions.code` alanıyla döner:

```json
{
  "errors": [
    {
      "message": "oturum açmanız gerekiyor",
      "extensions": { "code": "UNAUTHENTICATED" }
    }
  ],
  "data": null
}
```

Desteklenen kodlar:

| Kod | Anlam |
|---|---|
| `UNAUTHENTICATED` | Token yok, geçersiz veya süresi dolmuş |
| `FORBIDDEN` | Kullanıcının gerekli izni yok veya kaynak kapsam dışı |
| `NOT_FOUND` | Kaynak bulunamadı |
| `BAD_REQUEST` | Validation veya bozuk istek |
| `CONFLICT` | Çakışan veya tekrar kullanılan kaynak/anahtar |
| `RATE_LIMITED` | Rate limit aşıldı |
| `NOT_IMPLEMENTED` | Özellik henüz uygulanmamış |
| `INTERNAL` | Sunucu iç hatası |

REST özel durumları:

```json
{"error":"not found","code":404}
```

GraphQL endpoint'i middleware'de token'ı çözemediğinde HTTP `401` ve GraphQL
şekline benzer `errors` gövdesi döndürür. Uygulama hatalarının çoğu GraphQL
protokolü gereği HTTP `200` içinde `errors` alanında taşınabilir; istemci her
iki katmanı da kontrol etmelidir.

## 10. Güvenlik ve kişisel veri kuralları

- Üretimde varsayılan JWT secret, boş OTP pepper, localhost MongoDB ve
  `EMAIL_PROVIDER=none` ile boot engellenir.
- Access token kısa ömürlüdür; refresh token rotasyonludur ve replay durumunda
  aile iptal edilir.
- E-posta kodları Redis'te TTL ile tutulur, denemeler ve IP/e-posta rate limit'i
  uygulanır.
- Cihaz parmak izi düz metin saklanmaz; cihaz public key'i Ed25519 olarak
  kaydedilebilir, private key backend'e gönderilmez.
- LLM'e gönderilen kopyada kişisel veri maskelemesi yapılır. Orijinal turn metni
  puan alıntısı doğrulanabilsin diye saklanan transkriptte korunur.
- Ham ses backend'e gelmez ve `LLM_STORE_RAW_AUDIO` varsayılan olarak `false`'tur.
- Prompt'a giren persona, senaryo ve kullanıcı metni güvenilmeyen veri sınırları
  içinde ele alınır; kullanıcı metni sistem guardrail'lerini değiştiremez.
- Production'da GraphQL Playground ve introspection kapatılmalıdır.
- `.env` dosyası ve sırlar commit edilmemelidir. Sırlar yalnızca environment
  veya secret manager üzerinden sağlanmalıdır.

## 11. Ortam değişkenleri

Tam örnek yapılandırma [`.env.example`](.env.example) dosyasındadır. Aşağıdaki
tablo backend davranışını etkileyen değişkenlerin tekil özetidir.

| Grup | Değişkenler | Varsayılan / not |
|---|---|---|
| Ortam | `APP_ENV` | `development`; production katı doğrulama açar |
| Sunucu | `SERVER_HOST`, `SERVER_PORT` | `0.0.0.0`, `8080` |
| Sunucu | `CORS_ALLOWED_ORIGINS` | Frontend origin listesi |
| Sunucu | `SERVER_READ_TIMEOUT_SECONDS`, `SERVER_WRITE_TIMEOUT_SECONDS`, `SERVER_IDLE_TIMEOUT_SECONDS` | `15`, `15`, `60` |
| Sunucu | `MAX_BODY_BYTES` | `1048576` |
| PostgreSQL | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | Yerel Postgres varsayılanları |
| PostgreSQL | `DB_MAX_CONNS`, `DB_MIN_CONNS` | `25`, `5` |
| MongoDB | `MONGO_URI`, `MONGO_DATABASE` | `mongodb://localhost:27017`, `prova` |
| MongoDB | `MONGO_CONNECT_TIMEOUT_SECONDS`, `MONGO_MAX_POOL_SIZE`, `MONGO_MIN_POOL_SIZE` | `10`, `50`, `5` |
| Redis | `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | `localhost`, `6379`, boş, `0` |
| JWT | `JWT_SECRET`, `JWT_ISSUER` | Secret zorunlu; issuer `prova` |
| Login kodu | `AUTH_CODE_PEPPER`, `AUTH_CODE_LENGTH`, `AUTH_CODE_TTL_SECONDS` | `6`, `600` saniye |
| Login kodu | `AUTH_CODE_MAX_ATTEMPTS`, `AUTH_CODE_RESEND_COOLDOWN_SECONDS` | `5`, `60` saniye |
| Login rate limit | `AUTH_CODE_MAX_REQUESTS_PER_EMAIL`, `AUTH_CODE_MAX_REQUESTS_PER_IP`, `AUTH_CODE_RATE_LIMIT_WINDOW_SECONDS` | `5`, `20`, `3600` |
| E-posta doğrulama | `EMAIL_VERIFICATION_OTP_PEPPER`, `EMAIL_VERIFICATION_OTP_TTL_SECONDS`, `EMAIL_VERIFICATION_MAX_ATTEMPTS` | `300` saniye, `5` deneme |
| E-posta | `EMAIL_PROVIDER`, `RESEND_API_KEY`, `RESEND_FROM_EMAIL` | `resend`, secret, gönderen adresi |
| SMTP | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_USE_TLS` | Mailpit için `localhost:1025` |
| Persona servisi | `AI_SERVICE_URL`, `AI_SERVICE_TIMEOUT_SECONDS` | `http://localhost:8090`, `10` |
| LLM | `LLM_API_KEY`, `LLM_API_KEYS` | Anahtarlar profil yerine environment'ta |
| LLM | `LLM_FAST_BASE_URL`, `LLM_STRONG_BASE_URL` | `localhost:8000/v1`, `localhost:8001/v1` |
| LLM taşıma | `LLM_REQUEST_TIMEOUT_SECONDS`, `LLM_CIRCUIT_THRESHOLD`, `LLM_CIRCUIT_COOLDOWN_SECONDS` | `60`, `3`, `30` |
| LLM taşıma | `LLM_LONG_INPUT_THRESHOLD`, `LLM_STORE_RAW_AUDIO` | `4000`, `false` |
| GraphQL | `GRAPHQL_PLAYGROUND`, `GRAPHQL_INTROSPECTION` | Development'da `true`; production'da kapat |
| GraphQL | `GRAPHQL_MAX_DEPTH`, `GRAPHQL_MAX_COMPLEXITY`, `GRAPHQL_MAX_BATCH` | `12`, `500`, `5` |
| Token | `TOKEN_ACCESS_TTL_SECONDS`, `TOKEN_REFRESH_TTL_SECONDS` | `900`, `2592000` |
| Token | `TOKEN_MAGIC_LINK_TTL_SECONDS`, `TOKEN_DEVICE_CHALLENGE_TTL_SECONDS` | `900`, `120` |
| Token | `TOKEN_MAX_FAILED_ATTEMPTS`, `TOKEN_LOCK_DURATION_SECONDS` | `10`, `900` |
| Web | `WEB_BASE_URL` | `http://localhost:3000` |
| Yaşam döngüsü | `LIFECYCLE_DELETION_GRACE_SECONDS`, `LIFECYCLE_PURGE_INTERVAL_SECONDS`, `LIFECYCLE_PURGE_ENABLED` | `2592000`, `3600`, `true` |
| Log | `LOG_LEVEL`, `LOG_FORMAT` | `info`, `json` |

`KAFKA_*` ve eski bağımsız `WS_*` değişkenleri örnek env dosyasında geriye
dönük uyumluluk için bulunabilir; Prova'nın güncel istemci gerçek zamanlı akışı
GraphQL subscription üzerinden `/graphql` yolunu kullanır.

## 12. Mimari ve veri akışı

```text
Web / Electron
      |
      | GraphQL HTTP + graphql-transport-ws
      v
Go backend (chi + gqlgen)
      |
      +--> PostgreSQL : kullanıcı, kurum, rol, cihaz, token, audit
      +--> MongoDB    : karakter, senaryo, rubrik, oturum, turn, score, routing
      +--> Redis      : OTP, rate limit, token denylist
      +--> E-posta    : Resend veya SMTP/Mailpit
      +--> Persona    : dahili Python AI-01 servisi
      +--> LLM Gateway: güçlü evaluator ve profil testi
```

Katmanlar:

- `graph/`: GraphQL SDL, generated code, resolver'lar ve mapper'lar.
- `internal/application/`: kullanım senaryoları ve orkestrasyon.
- `internal/domain/`: iş modelleri, kurallar ve repository port'ları.
- `internal/infrastructure/`: HTTP, GraphQL, veritabanı, Redis, e-posta, LLM.
- `cmd/server/`: yapılandırma ve dependency wiring.
- `cmd/seed/`: demo içerik ve profil seed'i.
- `cmd/verify/`, `cmd/smoke/`: mekanik ve uçtan uca doğrulama.

### Canlı persona akışı

```text
submitTurn
  -> oturum sahipliği ve turn limiti kontrolü
  -> frozen character/scenario yükleme
  -> Python persona service
  -> çalışan + karakter turn'lerini birlikte kaydetme
  -> sessionEvents yayını
```

### Puanlama akışı

```text
endSession
  -> oturumu SCORING durumuna alma
  -> frozen transcript + scenario + rubric
  -> STRONG LLM evaluator
  -> JSON puan, rationale ve quote doğrulama
  -> zorunlu kriter kuralı
  -> COMPLETED + SCORE_READY
```

## 13. Doğrulama ve geliştirme komutları

Backend:

```bash
cd backend
go test ./...
go build ./...
go vet ./...
./scripts/lint.sh
./scripts/verify.sh
./scripts/smoke.sh
```

GraphQL şemasını yeniden üretme:

```bash
cd backend
go run ./cmd/schema -out ../schema.graphql
```

Frontend web build'i:

```bash
cd frontend
npm run build --workspace web -- --webpack
```

## 14. Kaynak dosyalar ve uyumluluk notu

- GraphQL kaynak şeması: [`graph/schema.graphqls`](graph/schema.graphqls)
- Üretilmiş SDL: [`../schema.graphql`](../schema.graphql)
- Yapılandırma örneği: [`.env.example`](.env.example)
- Güvenlik ayrıntıları: [`docs/SECURITY.md`](docs/SECURITY.md)
- KVKK ayrıntıları: [`docs/KVKK.md`](docs/KVKK.md)
- Gereksinim izlenebilirliği: [`docs/TRACEABILITY.md`](docs/TRACEABILITY.md)

Depoda bulunan `POSTMAN_COLLECTION_GUIDE.md` ve `postman/` koleksiyonu eski
MasterFabric REST gateway ürününden kalmıştır. `internal/gateway/`, API
management ve Kafka wiring'i Prova'nın güncel istemci yüzeyine bağlı değildir.
Bu nedenle güncel entegrasyon için bu belge ile `graph/schema.graphqls` esas
alınmalıdır.
