# Prova Desktop

## Prova AI stack'i

Masaüstü uygulamasının hedef ses hattı şudur:

```text
mikrofon → Whisper Large V3 Turbo / whisper.cpp → metin
                                              ↓
                                    Qwen3-4B persona
                                              ↓
                                    canlı transcript
```

`dbmdz/bert-base-turkish-cased` tabanlı BERTurk classifier, konuşma sırasında
yerel davranış sinyalleri üretmek için ayrı bir hattır; Qwen3-4B'nin yerine
geçmez. Oturum sonu değerlendirme için transcript backend üzerinden
`Qwen/Qwen3-30B-A3B-Instruct-2507` güçlü evaluator'a gönderilir. Ham ses
backend'e gönderilmez veya varsayılan olarak saklanmaz.

Bu checkout'ta Electron güvenlik kabuğu, mikrofon izin kontrolü ve bas-konuş
UI'sı hazırdır; gerçek `whisper.cpp` native binding/model paketlemesi henüz
tamamlanmış değildir. Yazılı önizleme bu nedenle bilinçli fallback olarak
korunur.

The desktop application keeps its own Next.js renderer and wraps its static
export with Electron. Development loads only `http://127.0.0.1:3100`;
production loads `app://prova/` from the packaged `out` directory. It never
loads `frontend/web` remotely.

## Commands

Run these from `frontend`:

```text
npm run dev:desktop
npm run build:desktop
npm run test:protocol --workspace desktop
npm run test:security --workspace desktop
npm run dist --workspace desktop
npm run dist:mac --workspace desktop
npm run dist:win --workspace desktop
npm run dist:linux --workspace desktop
```

Native installers are written to `frontend/desktop/release`.

## İlk açılış tanıtımı

Uygulama ilk açılışta girişten önce üç tanıtım ekranı gösterir (`/onboarding`);
dördüncü ekran giriş kartıdır. Tanıtım giriş ekranıyla aynı kabuğu kullanır,
bu yüzden dördüncü ekrana geçerken yalnızca kartın içi değişir. Kurallar
`.notes/DESIGN.md` içindeki "İlk açılış tanıtımı" bölümündedir.

Ana süreç pencereyi açmadan önce durumu okur ve ilk rotayı buna göre seçer:
görülmemişse `/onboarding`, görülmüşse `/login`. "Görüldü" bilgisi
`userData/onboarding.json` dosyasında tutulur; sır olmadığı için safe storage'a
yazılmaz — keyring'i olmayan Linux'ta safe storage fail-closed davranır ve
pencerenin açılmasını engellerdi. Dosya bozuksa tanıtım yeniden gösterilir.
Renderer bu durumu `window.prova.onboarding` üzerinden okur ve tamamlar.

Tanıtımı yeniden görmek için o dosya silinir:

```text
macOS   ~/Library/Application Support/Prova/onboarding.json
Windows %APPDATA%\Prova\onboarding.json
Linux   ~/.config/Prova/onboarding.json
```

## E-posta ile giriş

Login ekranı artık doğrudan ana süreçteki güvenli auth IPC katmanını kullanır:
yerel/test hesapları `login` mutation'ıyla kullanıcı adı ve şifre üzerinden
giriş yapar. Şifresiz hesaplar için `requestLoginCode` gerçek 6 haneli kodu
backend’e ister, doğrulama ekranı `verifyLoginCode` ile kodu tüketir. Access ve refresh token’lar renderer’a
verilmeden Electron safe storage içine yazılır. Backend’in gönderici ayarı
`EMAIL_PROVIDER=resend` veya `EMAIL_PROVIDER=smtp` olmalı; Resend için
`RESEND_API_KEY` ve `RESEND_FROM_EMAIL`, SMTP için ilgili `SMTP_*` değişkenleri
dolu olmalıdır. Masaüstü backend endpoint’i varsayılan olarak
`http://127.0.0.1:8080/graphql` adresini kullanır; farklı dağıtımlarda
`PROVA_GRAPHQL_URL` ile değiştirilir.

## Process boundary

The sandboxed renderer receives only `window.prova`. It can read normalized
system/device information, read microphone permission status, obtain the
generated device registration material, and read or set the onboarding flag. It cannot access Electron, Node.js, the
filesystem, the shell, raw IPC, refresh tokens, signing operations, or the
private device key; challenge signing remains a main-process-only operation.

The main process creates a random per-install device identifier and an Ed25519
key pair. The backend already accepts the resulting fingerprint/public key via
`DeviceInput`, issues `requestDeviceChallenge`, and accepts
`deviceSignature` during login. The desktop auth IPC client performs this
challenge/signature exchange and stores the resulting session in safe storage;
the renderer never receives the private key or bearer tokens.

## Secure storage

`ElectronSafeStorage` is the main-process boundary for refresh tokens, device
secrets, and the device private key. Encrypted blobs are stored below
Electron's per-user `userData` directory and never exposed to the renderer.

On Linux, Electron can fall back to `basic_text` when no Secret Service or
KWallet backend exists. Prova fails closed in that case: it will not persist a
private key or future token until a supported keyring is available. It never
opts into plaintext encryption. Signed builds are still required before
production distribution so macOS Keychain identity remains stable across
updates.

## Packaging security

The builder configuration contains no signing or notarization credentials.
Platform icons, Apple signing/notarization, Windows code signing, and publishing
credentials belong in the release pipeline later. The checked-in macOS config
sets `identity: null` so local/CI builds remain predictably unsigned; a release
pipeline must override it with the managed signing identity and notarization
credentials. macOS packaging already contains the microphone usage description.
