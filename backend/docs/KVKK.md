# KVKK: hangi veri nerede, ne kadar saklanıyor, nasıl siliniyor

Bu belge Prova backend'inin işlediği kişisel verileri, saklandıkları yeri,
saklama süresini ve silme davranışını tanımlar.

---

## 1. İşlenen kişisel veriler

### PostgreSQL — "bu kim?"

| Tablo | Alan | Veri türü | Neden gerekli |
|---|---|---|---|
| `users` | `email` | Kimlik | Parolasız girişin tek kimlik bilgisi |
| `users` | `first_name`, `last_name` | Kimlik | Arayüzde hitap; zorunlu değil |
| `users` | `email_verified_at` | İşlem kaydı | Adresin kanıtlandığı an |
| `users` | `failed_attempts`, `locked_until` | Güvenlik | Kaba kuvvet koruması |
| `user_devices` | `fingerprint_hash` | Cihaz kimliği | Sınav bütünlüğü; **hash'lenmiş saklanır** |
| `user_devices` | `public_key` | Cihaz kimliği | İmza doğrulaması; özel anahtar sunucuya hiç gelmez |
| `user_devices` | `name`, `platform` | Cihaz kimliği | Kullanıcının kendi cihazını tanıması |
| `login_codes` | `email`, `code_digest` | Kimlik bilgisi | Giriş; **kod düz metin saklanmaz** |
| `magic_links` | `email`, `token_hash` | Kimlik bilgisi | Giriş; **token düz metin saklanmaz** |
| `refresh_tokens` | `token_hash`, `family_id` | Oturum | Rotasyon ve yeniden kullanım tespiti |
| `device_challenges` | `email`, `fingerprint_hash` | Güvenlik | İmza doğrulaması |
| `audit_logs` | `user_id`, `ip_address`, `user_agent` | İşlem kaydı | KVKK hesap verebilirlik ilkesi |
| `organization_users` | `user_id` | İlişki | Kiracı üyeliği |

### MongoDB — "ne oynandı ve nasıl puanlandı?"

| Koleksiyon | Alan | Veri türü | Neden gerekli |
|---|---|---|---|
| `sessions` | `employee_id` | Kimlik bağı | Puanın kime ait olduğu |
| `sessions` | `device_id` | Cihaz bağı | Sertifikanın makineye bağlanması |
| `turns` | `text` (rol=employee) | **Serbest metin** | Puanlamanın dayanağı; kullanıcının yazdığı her şey |
| `turns` | `text` (rol=character) | Üretilmiş metin | Kişisel veri taşımaz |
| `scores` | `criteria[].quote` | Transkript alıntısı | Puanın gerekçesi |
| `scores` | `override.overridden_by` | Kimlik bağı | Ezmeyi yapan yönetici |
| `routing_records` | `session_id` | Dolaylı bağ | Maliyet ve dağılım istatistiği |

**Çalışanın yazdığı metin en hassas alandır.** Rol yapma sırasında kullanıcı,
senaryonun gerektirmediği kişisel verileri de yazabilir. Bu yüzden:

- Metin LLM'e giderken maskelenir (aşağıya bakın).
- Metin, kalıcı silmede temizlenir.

### Saklanmayanlar

| Veri | Durum |
|---|---|
| **Parola** | Sistemde parola yoktur ve olmayacaktır |
| **Ham ses kaydı** | Backend'e **hiç gelmez**. İstemci konuşmayı kendi cihazında metne çevirir ve yalnızca metin gönderir. `LLM_STORE_RAW_AUDIO` varsayılan olarak kapalıdır. Ses kaydı biyometrik veridir; KVKK'da özel nitelikli sayılır ve ayrı açık rıza gerektirir. |
| **Giriş kodunun düz metni** | Yalnızca peppered digest saklanır |
| **Magic link token'ının düz metni** | Yalnızca SHA-256 özeti saklanır |
| **Cihaz parmak izinin düz metni** | Yalnızca SHA-256 özeti saklanır |
| **Cihazın özel anahtarı** | İşletim sisteminin güvenli deposunda kalır |
| **LLM API anahtarları** | Ortam değişkeninde; veritabanına yazılmaz |

---

## 2. LLM'e giden veri

Prova, konuşma metnini üçüncü taraf bir model sağlayıcısına gönderir. Bu,
işlemenin en riskli noktasıdır ve iki kuralla sınırlandırılmıştır.

### Maskeleme

LLM'e giden **kopyada** şu desenler maskelenir:

| Desen | Etiket | Doğrulama |
|---|---|---|
| TC kimlik numarası | `[TCKN]` | Resmî checksum algoritması |
| IBAN | `[IBAN]` | TR + 24 hane biçimi |
| Kredi kartı | `[KART]` | Luhn |
| Telefon | `[TELEFON]` | TR mobil biçimleri |
| E-posta | `[EPOSTA]` | — |

Checksum doğrulaması, sipariş ve fatura numaralarının yanlışlıkla
maskelenmesini engeller: modelin göremediği bilgi puanlamayı bozar.

### Transkript maskelenmez

Maskeleme **yalnızca giden kopyada** yapılır. Transkriptin kendisi orijinal
metni tutar.

Bunun nedeni, gizliliği zayıflatmak değil, ölçmeyi mümkün kılmaktır:
rubriğin zorunlu kriteri "çalışan kişisel veri ifşa etti mi" sorusunu sorar,
ve maskelenmiş bir transkript üzerinden bu soru cevaplanamaz. İfşayı
görmeyen bir sistem, ifşayı önlemeyi de öğretemez.

Maskelenen alan sayısı hem konuşma sırası belgesine hem yönlendirme kaydına
yazılır; "hangi veriyi işlediniz" sorusunun ölçülebilir cevabı budur.

### Model sağlayıcısı

Sağlayıcı ve model, `llm_profiles` koleksiyonundan okunur ve yönetici
tarafından değiştirilebilir. Bu, veri işleyen üçüncü tarafın değişebileceği
anlamına gelir; **hangi sağlayıcının kullanıldığı, aydınlatma metninde
güncel tutulmalıdır.** Sağlayıcı değişikliği `llm.profile.changed` olarak
denetime düşer.

---

## 3. Saklama süreleri

| Veri | Süre | Silme mekanizması |
|---|---|---|
| Giriş kodları | TTL (varsayılan 10 dk) + temizlik | `DeleteExpiredBefore`; kalıcı silmede `DeleteByEmail` |
| Magic link'ler | TTL (varsayılan 15 dk) | Kalıcı silmede `DeleteByUser` |
| Cihaz challenge'ları | TTL (varsayılan 2 dk) | Tek kullanımlık; süresi dolanlar geçersiz |
| Refresh token'lar | `TOKEN_REFRESH_TTL_SECONDS` (varsayılan 30 gün) | Çıkış, cihaz iptali, kalıcı silme |
| Cihaz kayıtları | Hesap ömrü | Kullanıcı iptali; kalıcı silmede satır düşer |
| Kullanıcı kaydı | Hesap ömrü | Kalıcı silmede kişisel alanlar null'a çekilir |
| Oturumlar ve transkriptler | **Süresiz** | Kalıcı silmede kimliksizleştirilir, silinmez |
| Puanlar | **Süresiz** | Silinmez (kurum istatistiği) |
| Denetim kaydı | **Süresiz** | **Silinmez** (hesap verebilirlik ilkesi) |

Oturumların ve puanların süresiz saklanması bilinçlidir: bir sertifika yıllar
sonra savunulabilir olmalıdır. Kişisel bağ ise silinir — aşağıya bakın.

---

## 4. Silme (unutulma hakkı)

### Süreç

1. Kullanıcı `requestAccountDeletion` çağırır.
2. Hesap `pending_deletion` durumuna geçer ve **çalışmaya devam eder**.
   Erişimi hemen kesmek, fikrini değiştiren kullanıcıyı geri alma düğmesine
   ulaşamaz hâle getirirdi.
3. Geri alma penceresi boyunca `cancelAccountDeletion` ile geri alınabilir.
   Pencere `LIFECYCLE_DELETION_GRACE_SECONDS` ile ayarlanır; üretimde en az
   24 saat olmak zorundadır.
4. Pencere dolunca zamanlanmış iş kalıcı silmeyi uygular.

### Kalıcı silme ne yapar

**PostgreSQL'de kişisel veri gerçekten silinir:**

| Alan | Sonuç |
|---|---|
| `users.email` | `NULL` |
| `users.first_name`, `last_name` | `NULL` |
| `users.email_verified_at` | `NULL` |
| `users.status` | `deleted` |
| `user_devices` | Satırlar silinir |
| `login_codes` | Satırlar silinir |
| `magic_links` | Satırlar silinir |
| `refresh_tokens` | Satırlar silinir |

Kullanıcı satırı **kalır** ve kimliksiz bir kabuğa dönüşür. Sebep: denetim
kaydı silinmez ve bir kullanıcı kimliğine bağlanabilmelidir.

**MongoDB'de oturumlar silinmez, kimliksizleştirilir:**

| Alan | Sonuç |
|---|---|
| `sessions.employee_id` | Sıfır UUID |
| `sessions.anonymized` | `true` |
| `sessions.device_id` | `null` |
| `turns.text` (rol=employee) | `[silindi]` |
| `turns.text` (rol=character) | Değişmez (kişisel veri taşımaz) |
| `scores` | Değişmez |

Puanların korunması bilinçlidir: kurumun eğitim etkinliği ölçümü kişisel veri
değildir ve silinen bir çalışan yüzünden geriye dönük bozulmamalıdır.
Kimliksizleştirilmiş bir oturum artık hiçbir kişiye bağlanamaz.

**Denetim kaydı silinmez** ve silme olayının kendisi (`account.purged`)
denetime yazılır. KVKK hesap verebilirlik ilkesi bunu gerektirir: silindiğini
kanıtlayamayan bir silme, yapılmamış bir silmeden ayırt edilemez.

### Silme sırası

Önce nesne veritabanı kimliksizleştirilir, sonra ilişkili kimlik satırları
düşürülür, en son kullanıcı silinmiş işaretlenir. Ters sırada bir hata,
kullanıcıyı "silindi" gösterirken oturumlarını hâlâ kimliğine bağlı
bırakırdı.

---

## 5. Erişim hakkı (veri dışa aktarma)

`exportMyData` mutation'ı tek bir JSON belgesi döndürür:

- **profil**: kimlik, durum, doğrulama ve silme zaman damgaları
- **cihazlar**: ad, platform, son görülme, iptal tarihi
- **oturumlar**: durum, oynanan sürüm numaraları, tarihler
- **transkriptler**: her konuşma sırası, sıra numarası ve rolüyle
- **puanlar**: kriter kriter puan, gerekçe, alıntı, doğrulama bayrağı
- **denetim kayıtları**: eylem, kaynak, IP, zaman

Kullanıcı birden fazla organizasyonda oynamış olabilir; dışa aktarma hepsini
kapsar.

**Dışa aktarmaya girmeyenler:** cihaz parmak izi hash'i ve açık anahtarı.
İkisi de kullanıcının verisi değil, cihazın kimlik bilgisidir; paylaşılabilir
bir dosyaya çalışan bir kimlik bilgisi koymak dosyayı bir anahtarlığa
çevirirdi.

Dışa aktarma işlemi `account.data.exported` olarak denetime düşer.

---

## 6. KVKK içerik katmanı

Ürün yalnızca KVKK'ya uymakla kalmaz, KVKK davranışını **öğretir ve ölçer**.

- Rubrikteki `trap` alanı karaktere talimat olarak verilir: karakter,
  görüşme sırasında çalışandan başkasının kişisel verisini istemeye çalışır.
- "Kişisel veri koruması" kriteri **zorunlu** işaretlidir.
- Zorunlu kriter düşerse toplam puan ne olursa olsun sonuç **KALDI**'dır.
  Diğer maddelerden toplanan puanla satın alınamaz.
- Tohum verisinde bunu içeren bir senaryo bulunur: "Üçüncü kişi adına bilgi
  talebi (KVKK)".

---

## 7. Denetim kaydına yazılanlar

Denetim kaydı silinmeyen tek koleksiyondur; bu yüzden içine yazılan her şey
kalıcıdır ve **serbest metin ile kişisel veri bilerek dışarıda bırakılır.**

| Yazılan | Yazılmayan |
|---|---|
| Eylem adı (sabit liste) | E-posta adresi |
| Kaynak türü ve kimliği | Konuşma metni |
| Kullanıcı kimliği | Prompt eki metni (yalnızca uzunluğu) |
| IP adresi ve tarayıcı | Giriş kodu, token |
| Olaya özgü sayısal alanlar | |

Kayda geçen olaylar: giriş talebi/başarı/başarısızlık/kilit, magic link
gönderimi ve kullanımı, çıkış, refresh rotasyonu ve yeniden kullanım tespiti,
cihaz eşleşme/challenge/iptal, profil güncelleme, veri dışa aktarma, silme
talebi/iptal/kalıcı silme, LLM profil değişikliği ve testi, skor ezme, içerik
sürümleme ve yayınlama, oturum başlangıcı ve bitişi.

---

## 8. Doğrulama

```bash
cd backend && ONLY=12 ./scripts/verify.sh
```

Kanıtladıkları: LLM isteğinde TC ve IBAN maskelenmiş, transkript orijinal,
maskelenen alan sayısı kayıtta, zorunlu kriter ihlali KALDI üretiyor, KVKK
tuzağı rubrikte zorunlu kriterde tanımlı, dışa aktarma profil/oturum/
transkript/denetim içeriyor, ham ses saklama varsayılanı kapalı.

Silme zinciri için `ONLY=7 ./scripts/verify.sh` ve `./scripts/smoke.sh`.
