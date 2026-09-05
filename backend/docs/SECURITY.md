# Güvenlik: saldırı → savunma

Her satır bir saldırıyı, ona karşı konan savunmayı ve savunmanın kodda nerede
durduğunu gösterir. Uygulanmayan şeyler en sonda, açıkça listelenmiştir.

---

## Kimlik ve oturum

| Saldırı | Savunma | Nerede |
|---|---|---|
| **Hesap sayımı** — "bu adres kayıtlı mı?" | Kod isteme ucu kayıtlı, bilinmeyen ve limitlenmiş adres için aynı gövdeyi döndürür; her yanıt sabit bir süre tabanına yastıklanır | `request_login_code.go` (`padResponseTime`, `response`) |
| **Aynı sorunun cihaz ucundan sorulması** | Cihaz challenge'ı kullanıcının var olup olmadığından bağımsız üretilir | `device_challenge.go` |
| **Aynı sorunun kilit üzerinden sorulması** | Kilitli hesap DOĞRU kodla da geçersiz kod yanıtını alır | `verify_login_code.go` (`user.IsLocked`) |
| **Giriş kodunun kaba kuvvetle bulunması** | Kod başına deneme bütçesi; bütçe dolunca kod yakılır | `verify_login_code.go` (`recordFailedAttempt`) |
| **Bütçenin her seferinde yeni kod isteyerek sıfırlanması** | Hesap seviyesinde ardışık başarısızlık sayacı; eşik aşılınca geçici kilit | `chargeAccountAttempt`, `user_repository.go` (`RecordFailedAttempt`) |
| **Sayaç yarışı** (eşzamanlı denemeler sayacı aynı değerde okur) | Artırma ve kilitleme tek SQL ifadesinde | `user_repository.go` |
| **Adres başına kod bombardımanı** | Adres ve IP başına hız limiti + yeniden gönderim cooldown'ı | `request_login_code.go` |
| **Aynı anda birden fazla geçerli kod** | Yeni kod üretilince eskiler yakılır | `codes.InvalidateActive` |
| **Digest kolonunun sızması** | Kod digest'i sunucu tarafı pepper ile HMAC'lenir; 10^6'lık uzay pepper olmadan düz metne eşdeğerdir | `login_code_service.go`, `AUTH_CODE_PEPPER` |
| **Magic link token'ının sızması** | DB'de yalnız SHA-256 özeti; token 256 bit rastgele | `magic_link_service.go` |
| **Magic link'in yeniden oynatılması** | Tek kullanımlık; yakma koşulu `UPDATE … WHERE used_at IS NULL` sorgusunun içinde | `magic_link_repository.go` (`Consume`) |
| **Eski bağlantıların birikmesi** | Yeni bağlantı üretilince adresin kullanılmamış bağlantıları iptal edilir | `magic_link.go` (`Issue`) |

---

## Token

| Saldırı | Savunma | Nerede |
|---|---|---|
| **Çalınmış access token'ın uzun ömrü** | Kısa TTL (varsayılan 15 dk); oturum rotasyonlu refresh ile sürer | `TOKEN_ACCESS_TTL_SECONDS` |
| **Çalınmış refresh token'ın sınırsız kullanımı** | Her kullanımda rotasyon: eski halka harcanır, yenisi verilir | `refresh_token.go` (`Execute`, `rotate`) |
| **Çalınmış halkanın yeniden oynatılması** | Harcanmış halka tekrar gelirse ailenin TAMAMI iptal edilir ve olay denetime düşer | `refresh_token.go` (`killFamily`) |
| **Yarış: iki istek aynı halkayı harcar** | `MarkUsed` koşulu sorgunun içinde; yarışı kaybeden istek de aileyi öldürür | `refresh_token_repository.go` |
| **Yenilemenin oturumu sonsuza uzatması** | Yeni halka öncekinin son kullanma tarihini devralır | `refresh_token.go` (`rotate`) |
| **Çıkıştan sonra token'ın yaşaması** | Çıkış aktif token'ı denylist'e alır ve aileyi iptal eder | `logout.go` |
| **İptal edilen cihazın token'ının yaşaması** | Cihaz kimliği JWT'ye yazılır; denylist cihaz ekseninde de çalışır | `denylist.go`, `jwt_service.go` |
| **İmzalı token'ın iptal edilememesi** | Redis tabanlı denylist (jti, cihaz, aile); Redis yoksa süreç içi yedek | `denylist.go` |
| **İstemcinin kiracı seçmesi** | `org_id` yalnızca JWT claim'inden; başlık tabanlı kiracı çözümlemesi yok | `authctx/viewer.go`, `graph/helpers.go` |

---

## Cihaz

| Saldırı | Savunma | Nerede |
|---|---|---|
| **Parmak izinin kopyalanıp gönderilmesi** | Anahtar çifti: her girişte rastgele challenge imzalanır, sunucu genel anahtarla doğrular | `device_signature.go`, `verify_login_code.go` |
| **İmzanın yeniden oynatılması** | Challenge tek kullanımlık ve kısa ömürlü | `device_challenge_repository.go` (`Consume`) |
| **İmzanın başka challenge'a taşınması** | İmza challenge metnine bağlı (Ed25519) | `device_signature.go` |
| **Veritabanından parmak izinin okunup gönderilmesi** | Parmak izi yalnızca hash'lenmiş saklanır; düz metin kolonu migration ile düşürüldü | `00017_user_devices_keypair.sql` |
| **Çalınmış token'la cihazın devralınması** | Açık anahtar yalnızca ilk kayıtta yazılır (`COALESCE`); sonradan değiştirilemez | `device_repository.go` (`Pair`) |
| **İptal edilmiş cihazdan sınav başlatılması** | Oturum başlatma cihazın yetkisini depodan kontrol eder (denylist'e ek olarak) | `session.go` (`ensureDeviceUsable`) |

---

## GraphQL

| Saldırı | Savunma | Nerede |
|---|---|---|
| **Derin iç içe sorguyla kaynak tüketimi** | Derinlik limiti; fragment içindeki derinlik de sayılır | `graphql/depth.go` |
| **`__typename` ekleyerek limitin atlatılması** | Muafiyet yalnızca SALT introspection sorgularına | `depth.go` (`isIntrospectionOnly`) |
| **Geniş alan seçimiyle kaynak tüketimi** | Karmaşıklık limiti | `graphql/server.go` |
| **Batch ile limitlerin operasyon başına dağıtılması** | Batch limiti; gövde dizi ise sayılır ve reddedilir | `graphql/server.go` (`batchGuard`) |
| **Şemanın haritasının çıkarılması** | Üretimde introspection ve playground zorla kapalı | `config/validate.go` (`Harden`) |
| **Yetkisiz alan okuma** | `@permission` RBAC'a sorar; nullable alanda null, değilse hata | `graphql/directives.go` |
| **Yetki servisi yokken açık kalma** | RBAC bağlı değilse kapalı taraf seçilir | `directives.go` |
| **Başkasının transkriptini okuma** | Oturum sahipliği kontrolü; yönetici için ayrı `session:read:all` izni | `schema.resolvers.go`, `session.go` |
| **Abonelikten başkasının oturumunu dinleme** | Sahiplik bağlantı anında doğrulanır | `schema.resolvers.go` (`SessionEvents`) |
| **Kimliksiz WebSocket aboneliği** | `connectionParams` içindeki token doğrulanır | `graphql/server.go` (`websocketInit`) |
| **Sunucu hata metninden bilgi sızması** | Domain hataları koda çevrilir; sarmalanan neden yalnızca loga gider | `graphql/server.go` (`errorPresenter`) |
| **Yabancı origin'den istek** | CORS izin listesi; WebSocket'te de origin deseni | `middleware/cors.go`, `server.go` |

---

## Kiracı ve veri

| Saldırı | Savunma | Nerede |
|---|---|---|
| **Kiracı sınırının unutulması** | `Scope` tipi; boş scope ile sorgu kurulamaz (`ErrOrgRequired`) | `prova/repository/repository.go` |
| **Filtreye kendi `org_id`'sini yazarak sınırı genişletme** | `org_id` her filtreye EN SON yazılır | `mongo/prova/versioned.go` (`filter`) |
| **Gövdedeki `org_id` ile başka kiracıya yazma** | Kiracı her zaman scope'tan alınır, belgeden değil | `versioned.go` (`Create`) |
| **Yayınlanmış içeriğin sonradan değiştirilmesi** | `UpdateDraft` sorgusu `status: draft` koşulunu taşır | `versioned.go` (`UpdateDraft`) |
| **Aynı sürüm numarasının iki belgeye verilmesi** | `(org_id, lineage_id, version)` unique index | `mongo/collections.go` |
| **Oturumun sonradan başka içerikle puanlanması** | Sürüm referansları oturum başlarken donar | `prova/model/session.go` |

---

## LLM

| Saldırı | Savunma | Nerede |
|---|---|---|
| **Prompt injection ile karakterin rolden çıkarılması** | Guardrail koda gömülü, promptun en başında, kendisini ezmeye çalışan talimatları yok saymayı emreder | `prova/service/guardrail.go` |
| **Yönetici prompt'uyla guardrail'in ezilmesi** | Yönetici eki her zaman guardrail'den SONRA gelir | `guardrail.go` (`CompileCharacterPrompt`) |
| **Karakterin gerçek kişisel veri üretmesi** | Guardrail 3. madde | `guardrail.go` |
| **Karakterin çalışana doğru cevabı söylemesi** | Guardrail 4. madde; senaryonun hedefi karakter promptuna hiç girmez | `guardrail.go`, `character.go` |
| **Kişisel verinin sağlayıcıya gitmesi** | PII maskeleme (TC kimlik, IBAN, telefon, e-posta, kart) yalnızca giden kopyada | `llm/pii_masker.go`, `prova/service/gateway.go` |
| **Yanlış pozitif maskeleme (sipariş no → TCKN)** | TC kimlik ve kart için checksum doğrulaması | `pii_masker.go` |
| **Uydurma alıntıyla puanın savunulamaz hâle gelmesi** | Alıntı iddia edilen konuşma sırasında aranır; bulunamazsa işaretlenir | `prova/service/quote.go` |
| **Zorunlu kriterin diğer puanlarla telafi edilmesi** | Zorunlu kriter düşerse toplam ne olursa olsun KALDI | `prova/model/score.go` (`Evaluate`) |
| **Modelin bir kriteri atlayarak cezadan kaçması** | Rubrik referans alınır; atlanan kriter sıfır puanla görünür | `prova/service/scoring.go` (`ParseScoring`) |
| **Uydurma sinyalle pahalı kademeye yönlendirme** | Sinyaller rubrik anahtarlarına göre süzülür | `prova/service/character.go` |
| **Aralık dışı puanla eşiğin anlamsızlaşması** | Puan `[0, maxPoints]` aralığına sıkıştırılır | `scoring.go` (`clampPoints`) |
| **Sağlayıcı çöktüğünde oturumun kesilmesi** | Devre kesici + güçlü kademeye failover | `application/prova/service/breaker.go`, `gateway.go` |
| **API anahtarının veritabanı yedeğinde bulunması** | Anahtarlar profilde değil env'de | `llm/keys.go` |

---

## Yapılandırma ve işletim

| Saldırı | Savunma | Nerede |
|---|---|---|
| **Varsayılan JWT secret ile üretime çıkma** | Fail-fast: boot durur | `config/validate.go` |
| **Pepper'sız üretime çıkma** | Fail-fast | `config/validate.go` |
| **E-posta sağlayıcısı yokken "çalışıyor" görünme** | Üretimde `EMAIL_PROVIDER=none` ile boot durur | `config/validate.go` |
| **Yerel Mongo URI ile üretime çıkma** | Fail-fast | `config/validate.go` |
| **Sıfır bekleme süresiyle anında silme** | Üretimde en az 24 saat zorunlu | `config/validate.go` |
| **Kodun log akışına düşmesi** | Kod hiçbir zaman loglanmaz; adres de log satırlarında yok | `request_login_code.go` |
| **Denetim kaydının kaybolması** | Yazımlar eşzamansız ama takip edilir; kapanışta beklenir | `infrastructure/audit/recorder.go` |
| **Denetim kaydına kişisel veri yazılması** | Metadata'ya adres ve serbest metin yazılmaz; prompt eki yerine uzunluğu | `llm_profile.go`, `request_login_code.go` |

---

## Bilinen sınırlar

Dürüstlük gereği: aşağıdakiler **uygulanmadı**.

- **Ters vekil arkasında istemci IP'si.** `X-Forwarded-For` bilerek
  okunmuyor; vekil olmadan bu başlık istemci tarafından uydurulabilir ve
  adrese bağlı her limit anlamsızlaşır. Üretimde vekil konulacaksa bu
  başlığın güvenilir biçimde işlenmesi eklenmelidir.
- **Çok örnekli dağıtımda abonelikler.** Oturum olayı dağıtıcısı süreç
  içidir. Birden fazla sunucu örneği çalıştırılacaksa Redis pub/sub'a
  taşınmalıdır; arayüz aynı kalır.
- **Denylist Redis'e bağlıdır.** Redis yokken süreç içi yedek kullanılır ve
  iptal yalnızca isteği alan örnekte geçerli olur. Üretimde Redis şarttır.
- **Rol yönetimi API'si yok.** Roller ve izinler tohum verisiyle ve
  doğrudan veritabanı üzerinden yönetiliyor; GraphQL'de rol atama mutation'ı
  bulunmuyor.
- **İçerik silme yok.** Belgeler arşivlenebilir ama silinemez. Sertifikasyon
  iddiası bunu gerektiriyor; bir "kalıcı içerik silme" ihtiyacı doğarsa
  verilmiş sertifikaların ne olacağı ayrıca kararlaştırılmalıdır.
