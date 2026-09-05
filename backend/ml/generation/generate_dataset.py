#!/usr/bin/env python3
"""Deterministically build Prova's synthetic Turkish classifier corpus.

The generator is intentionally domain- and behavior-driven. It combines
several independently authored context, discourse, and speech-style pools,
then rejects duplicate or overly similar records. It is not a Cartesian
product of names or values: each recipe selects a complete behavior-focused
utterance and a context that makes the annotation meaningful.
"""

from __future__ import annotations

import argparse
import json
import random
import re
import statistics
import unicodedata
from collections import Counter, defaultdict
from datetime import date
from difflib import SequenceMatcher
from pathlib import Path
from typing import Any


TOOL_ROOT = Path(__file__).resolve().parent
DEFAULT_SEED_DATASET = TOOL_ROOT / "seed_dataset.jsonl"
DEFAULT_LABELS = TOOL_ROOT / "labels.json"
DEFAULT_SCHEMA = TOOL_ROOT / "schema.json"
DEFAULT_OUTPUT_DIR = TOOL_ROOT.parent / "generated"
TARGET_TOTAL = 6000
SEED_SAMPLE_COUNT = 46
SEED = 20260905
SPLIT_TARGETS = {"train": 4800, "validation": 600, "test": 600}
LABELS = [
    "pii_disclosure",
    "identity_verification",
    "greeting",
    "active_listening",
    "procedure_accuracy",
    "empathy",
    "solution_offer",
    "closing",
]
KINDS = ["positive", "negative", "neutral", "hard_negative", "mixed", "ambiguous", "multi_label"]


def item(industry: str, scenario_type: str, persona: str, topic: str, pressure: str) -> dict[str, str]:
    return {
        "industry": industry,
        "scenario_type": scenario_type,
        "persona": persona,
        "topic": topic,
        "pressure": pressure,
    }


SCENARIOS = [
    item("bankacılık", "müşteri şikâyeti", "ücret kesintisine kızan müşteri", "hesaptan beklenmeyen ücret kesintisi", "işlemin hemen düzeltilmesini istiyor"),
    item("bankacılık", "üçüncü kişi talebi", "ısrarcı müşteri yakını", "başkasının hesap hareketi hakkında bilgi", "akrabalığın doğrulama yerine geçmesini bekliyor"),
    item("bankacılık", "şüpheli işlem", "endişeli hesap sahibi", "tanınmayan kart işlemi", "kartının kötüye kullanıldığını düşünüyor"),
    item("bankacılık", "kart işlemi", "aceleci hesap sahibi", "kart limitinin değiştirilmesi", "beklemeden işlem yapılmasını istiyor"),
    item("bankacılık", "şube yönlendirmesi", "belge soran müşteri", "vekaletle yapılacak işlem", "çağrı merkezinden istisna bekliyor"),
    item("bankacılık", "kredi başvurusu", "sonuç bekleyen müşteri", "başvurunun hangi aşamada olduğu", "kesin sonuç tarihi istiyor"),
    item("sigortacılık", "hasar bildirimi", "kaygılı sigortalı", "bekleyen hasar dosyası", "günlerdir haber alamadığını söylüyor"),
    item("sigortacılık", "poliçe kapsamı", "teminatı merak eden müşteri", "poliçenin belirli bir olayı kapsayıp kapsamadığı", "belge görülmeden kesin yanıt bekliyor"),
    item("sigortacılık", "poliçe iptali", "memnuniyetsiz müşteri", "poliçenin iptal edilmesi", "iade tutarını hemen öğrenmek istiyor"),
    item("sigortacılık", "yenileme", "kararsız müşteri", "poliçe yenileme koşulları", "fiyat değişikliğine itiraz ediyor"),
    item("sigortacılık", "eksik evrak", "takip eden müşteri", "dosyaya hangi belgenin eksik olduğu", "aynı belgeyi tekrar vermek istemiyor"),
    item("sigortacılık", "ödeme bilgisi", "ödeme bekleyen müşteri", "tazminat ödemesinin zamanı", "hesabına ne zaman geçeceğini soruyor"),
    item("telekom", "hizmet kesintisi", "sinirli abone", "saatlerdir çalışmayan internet", "işinin aksadığını söylüyor"),
    item("telekom", "paket değişikliği", "ücreti yükselen abone", "tarife ücretinin artması", "eski fiyatın korunmasını istiyor"),
    item("telekom", "numara taşıma", "geçiş tarihi soran abone", "numara taşıma başvurusunun süresi", "hattının ne zaman açılacağını soruyor"),
    item("telekom", "fatura itirazı", "faturası yüksek gelen abone", "beklenmeyen kullanım bedeli", "ücretin iptalini istiyor"),
    item("telekom", "kampanya bilgisi", "seçenek arayan abone", "yeni kampanyanın koşulları", "en ucuz seçeneği hemen öğrenmek istiyor"),
    item("telekom", "cihaz desteği", "kurulumda zorlanan abone", "modem veya cihaz kurulumu", "teknik terimlerle uğraşmak istemiyor"),
    item("perakende", "iade talebi", "hayal kırıklığına uğramış müşteri", "kullanılmış ürünün iade edilmesi", "mağazada beklemek istemiyor"),
    item("perakende", "geciken sipariş", "sabırsız müşteri", "planlanan tarihte gelmeyen sipariş", "ürünü belirli bir gün kullanması gerekiyor"),
    item("perakende", "yanlış ürün", "şaşırmış müşteri", "sipariş yerine farklı ürün gelmesi", "doğru ürünün ne zaman gönderileceğini soruyor"),
    item("perakende", "kusurlu ürün", "kırgın müşteri", "ilk kullanımda bozulan ürün", "değişim veya iade bekliyor"),
    item("perakende", "stok sorgusu", "rutin alışveriş yapan müşteri", "bir ürünün mağazada bulunup bulunmadığı", "şubeye boşuna gitmek istemiyor"),
    item("perakende", "teslimat adresi", "adresini değiştirmek isteyen müşteri", "sipariş teslimat adresi", "kargo çıkmadan değişiklik yapılmasını istiyor"),
    item("müşteri hizmetleri", "görüşme başlangıcı", "ilk kez arayan müşteri", "destek görüşmesinin açılması", "kısa sürede doğru birime ulaşmak istiyor"),
    item("müşteri hizmetleri", "fatura itirazı", "kızgın müşteri", "sözleşmede görünen bir ücret", "kalemin nedenini anlamak istiyor"),
    item("müşteri hizmetleri", "hesap bilgisi", "doğrulanmamış arayan", "başkasına ait adres veya telefon bilgisi", "müşteri numarasını bilmenin yeterli olduğunu düşünüyor"),
    item("müşteri hizmetleri", "şikâyet kaydı", "ısrarcı müşteri", "resmî şikâyet kaydı açılması", "önceki başvurularından sonuç alamadığını söylüyor"),
    item("müşteri hizmetleri", "birime aktarım", "beklemek istemeyen müşteri", "teknik ekibe veya uzman birime aktarım", "her şeyi tek temsilcinin çözmesini bekliyor"),
    item("müşteri hizmetleri", "görüşme kapanışı", "işlemi tamamlanan müşteri", "talebin özetlenerek kapatılması", "başka bir ihtiyacı olup olmadığı sorulmalı"),
    item("satış", "ihtiyaç analizi", "kararsız kurumsal alıcı", "kullanıma uygun ürün seçimi", "sadece fiyatı duymak istiyor"),
    item("satış", "teklif hazırlama", "teklif bekleyen müşteri", "kurumsal teklifin hazırlanması", "bugün içinde net fiyat istiyor"),
    item("satış", "teslim tarihi", "acil ihtiyacı olan müşteri", "ürünün belirli tarihe yetişmesi", "stok teyidi olmadan söz bekliyor"),
    item("satış", "ürün karşılaştırması", "araştırma yapan müşteri", "iki ürünün kullanım farkı", "hangi seçeneğin uygun olduğunu anlamaya çalışıyor"),
    item("satış", "kampanya koşulu", "indirim isteyen müşteri", "kampanyanın geçerlilik koşulları", "duyduğu fiyatın herkese uygulanmasını istiyor"),
    item("satış", "teklif kapanışı", "teklifi değerlendiren müşteri", "görüşmenin sonraki adımı", "karar vermeden önce son sorusunu sormak istiyor"),
    item("insan kaynakları", "izin sorgusu", "izin bakiyesini soran çalışan", "izin günlerinin hesaplanması", "ekrandaki rakamı anlamıyor"),
    item("insan kaynakları", "başvuru durumu", "meraklı aday", "iş başvurusunun hangi aşamada olduğu", "uzun süredir dönüş bekliyor"),
    item("insan kaynakları", "kişisel dosya talebi", "çalışanın yöneticisi", "bir çalışanın hassas dosya ayrıntısı", "yönetici olmanın yeterli olduğunu düşünüyor"),
    item("insan kaynakları", "yan hak bilgisi", "yeni çalışan", "bordrodaki yan hak kalemleri", "ilk bordrosunu anlamaya çalışıyor"),
    item("insan kaynakları", "eğitim kaydı", "eğitimini tamamlayan çalışan", "zorunlu eğitim kaydının görünmesi", "sertifika tarihini öğrenmek istiyor"),
    item("insan kaynakları", "performans görüşmesi", "geri bildirim bekleyen çalışan", "görüşme notunun paylaşılması", "notların kiminle paylaşıldığını soruyor"),
    item("sağlık idari hizmetleri", "randevu değişikliği", "aceleci hasta yakını", "randevu gününün değiştirilmesi", "hasta adına işlem yapılmasını istiyor"),
    item("sağlık idari hizmetleri", "laboratuvar sonucu", "hasta yakını", "başkasının test sonucu", "akrabalığın bilgi almaya yettiğini düşünüyor"),
    item("sağlık idari hizmetleri", "randevu gecikmesi", "bekleyen hasta", "randevunun gecikmesi", "ne kadar daha bekleyeceğini soruyor"),
    item("sağlık idari hizmetleri", "kayıt düzeltme", "bilgisi yanlış yazılan hasta", "iletişim bilgisindeki yazım hatası", "formu yeniden doldurmak istemiyor"),
    item("sağlık idari hizmetleri", "rapor teslimi", "rapor bekleyen hasta", "raporun teslim alınacağı kanal", "aynı gün içinde almak istiyor"),
    item("sağlık idari hizmetleri", "fatura açıklaması", "işlem bedelini soran hasta", "idari hizmet bedelinin açıklanması", "hangi kalem için ödeme yaptığını soruyor"),
]

OPENERS = [
    "Tabii, ", "Elbette; ", "Şöyle ilerleyelim: ", "Bir saniye, ", "Hemen bakayım; ",
    "Anlatayım, ", "Önce şunu netleştireyim: ", "Tamam, ", "Haklı olduğunuz noktayı gözden kaçırmadan ",
    "Kısaca söyleyeyim; ", "Bunu birlikte açalım: ", "Notumu aldım, ",
]
TAILS = [
    ".", "; sonra size bilgi vereceğim.", ", olur mu?", ". Şimdilik durum bu kadar.",
    ". Bir sonraki adımı da netleştireyim.", ". Böylece ne bekleyeceğinizi bilirsiniz.",
    ". Uygunsa bu şekilde ilerleyelim.", ". Eksik kalan yeri ayrıca kontrol ederim.",
]
CONNECTORS = [
    " Önce ", " Bunun yanında ", " Sonra ", " Bir de ", " Devamında ",
    " Şunu da ekleyeyim: ", " Oradan ilerlersek ", " Aynı görüşmede ",
]
FILLERS = [
    "şimdi", "hemen", "kısaca", "önce", "birlikte", "uygunsa", "bu aşamada", "açıkça", "not ederek", "sakin sakin",
]

POSITIVE = {
    "pii_disclosure": [
        "hesap sahibinin doğrulaması olmadan hiçbir hareketi paylaşamam",
        "başkasına ait iletişim bilgisini telefonda teyit edemem",
        "bu ayrıntıyı yalnızca yetkili kanaldan aktarabiliriz",
        "kişisel bilgiyi korumak için talebi güvenli başvuruya yönlendireceğim",
        "akrabalık bilgisi tek başına paylaşım yetkisi vermiyor",
        "önce yetkiyi doğrulamadan dosyanın içeriğini açmayacağım",
        "veri sahibi veya yetkili temsilci olmadan bilgi veremem",
        "gizlilik nedeniyle tutarı söylemem doğru olmaz",
        "başka bir çalışanın dosya ayrıntılarını paylaşmayacağım",
        "randevu ayrıntısını yetki kontrolü tamamlanmadan değiştiremiyorum",
        "bu bilgiyi doğrulanmış kanala taşımamız gerekiyor",
        "kişisel veriyi telefonda açık etmek yerine güvenli süreci anlatayım",
        "hesap hareketini teyit etmeyeceğim; sahibinin başvurması gerekiyor",
        "kim olursa olsun doğrulama yoksa bilgi paylaşmıyoruz",
    ],
    "identity_verification": [
        "işleme geçmeden önce kimlik bilgilerinizi doğrulayalım",
        "güvenlik sorularını tamamlamadan hesabı açamam",
        "önce arayan kişinin yetkisini kontrol etmem gerekiyor",
        "doğrulama kodunu almadan bu talebi sonuçlandıramam",
        "kimlik teyidini yaptıktan sonra kayıt ekranına geçeceğim",
        "bilgileri karşılaştırıp doğrulama tamamlanınca yardımcı olayım",
        "işlem sahibi olduğunuzu birkaç güvenlik adımıyla teyit edelim",
        "yetki belgesini kontrol etmeden dosya bilgisine bakmayacağım",
        "sipariş numaranızı ve ek doğrulama bilgisini alayım",
        "giriş bilgilerinizi doğrulayıp sonra değişikliği kaydederim",
        "doğrulama tamamlanır tamamlanmaz talebinizi açacağım",
        "önce kim olduğunuzu değil, yetkinizin bu işlem için yeterli olduğunu kontrol edelim",
        "güvenliğiniz için birkaç kısa teyit sorusu soracağım",
        "başvuru sahibini sistemde doğrulayıp devam edelim",
    ],
    "greeting": [
        "merhaba, ben Derya; Prova müşteri hizmetlerine hoş geldiniz",
        "iyi günler, ben Can; size nasıl destek olabilirim",
        "hoş geldiniz, ben müşteri destek ekibinden Ece",
        "merhaba, görüşmenizi ben takip edeceğim",
        "iyi akşamlar, Prova destek hattına ulaştınız",
        "selam, ben Burak; önce talebinizi dinleyeyim",
        "günaydın, müşteri hizmetlerine hoş geldiniz",
        "merhaba, ben bu görüşmenin sorumlusuyum",
        "iyi günler, adım Elif; hangi konuda aramıştınız",
        "hoş geldiniz, kaydınızı birlikte inceleyebiliriz",
        "merhaba, burası destek birimi; sizi dinliyorum",
        "iyi günler, ben Zeynep; nasıl yardımcı olabilirim",
    ],
    "active_listening": [
        "hesabınızdan beklemediğiniz bir ücret kesildiğini söylüyorsunuz",
        "sabah saatlerinden beri bağlantı kuramadığınızı anladım",
        "asıl beklentiniz dosyanın hangi aşamada olduğunu öğrenmek",
        "ürünün ilk kullanımda bozulduğunu belirtiyorsunuz",
        "sizin için önemli olan teslimatın cuma gününe yetişmesi",
        "randevuyu başka güne almak istediğinizi not ettim",
        "önceki başvurularınızdan sonuç alamadığınız için tekrar aradınız",
        "bu kalemin faturada neden yer aldığını anlamaya çalışıyorsunuz",
        "talebinizin yalnızca bilgi almak değil, kayda bağlanmak olduğunu görüyorum",
        "başvuru sürecinde beklediğiniz geri dönüşü alamadınız",
        "iki ürün arasında kullanımınıza uygun olanı seçmek istiyorsunuz",
        "yakınınız adına işlem yapılmasını bekliyorsunuz",
        "söylediğiniz noktaları tek tek not alıyorum",
        "önceliğinizin bugün içinde net bir cevap almak olduğunu anlıyorum",
    ],
    "procedure_accuracy": [
        "iade için ürünün kullanılmamış olması ve otuz günlük sürenin dolmamış olması gerekiyor",
        "başvuru kayda alındıktan sonra inceleme ekibi eksik belge varsa bildirim gönderir",
        "kimlik doğrulanmadan hesap hareketi paylaşılmıyor",
        "numara taşıma tamamlandığında geçiş tarihi SMS ile bildirilir",
        "poliçe kapsamını ilgili teminat maddesine göre kontrol edeceğiz",
        "değişiklik, doğrulama sonrasında sisteme kaydediliyor",
        "şikâyet kaydının numarası görüşme sonunda size iletilir",
        "randevu değişikliğinde uygun saat seçildikten sonra yeni tarih teyit edilir",
        "başvuru sonucu belgeler incelendikten sonra yazılı olarak bildirilir",
        "kayıt düzeltmesi için önce mevcut bilgiyi doğrulayıp ardından güncelleme yapıyoruz",
        "bu ücretin itirazı için işlem kalemini seçerek inceleme kaydı açıyoruz",
        "vekaletle işlem yapılacaksa belge kontrolü şube kanalından tamamlanıyor",
        "teslimat tarihi stok ve sevkiyat teyidinden sonra kesinleşiyor",
        "yan hak kalemleri bordro dönemine göre ayrı ayrı görüntüleniyor",
    ],
    "empathy": [
        "böyle bir bekleyişin sizi yorması çok anlaşılır",
        "yaşadığınız aksaklığın canınızı sıkması normal",
        "bu belirsizliğin günlük işinizi zorlaştırdığını tahmin edebiliyorum",
        "bunun sizin açınızdan ne kadar uğraştırıcı olduğunu görüyorum",
        "uzun süre cevap alamamak gerçekten yorucu olabilir",
        "beklediğiniz tarihin geçmesi hayal kırıklığı yaratmış olmalı",
        "bu konuda içinizin rahat etmesini istemeniz çok doğal",
        "sinirlenmenizi gereksiz bulmuyorum; yaşadığınız durum kolay değil",
        "böyle bir haber almak insanı tedirgin eder",
        "sizi tekrar tekrar aramak zorunda bırakmamız iyi olmamış",
        "bu yükü tek başınıza taşımamanız için birlikte bakacağım",
        "kafanızın karışmasını anlıyorum; adımları sadeleştirelim",
        "yaşadığınız gecikmenin hoş karşılanacak bir yanı yok",
        "önceliğinizin hızlıca netlik kazanması çok makul",
    ],
    "solution_offer": [
        "dosyayı şimdi açıp hangi aşamada olduğunu kontrol edeceğim",
        "kaydı oluşturup takip numarasını görüşme bitmeden paylaşacağım",
        "ilgili ekibe aktararak size dönüş zamanını netleştireceğim",
        "iki uygun seçenek çıkarıp kararınızı vermenize yardımcı olayım",
        "belgeleri alır almaz inceleme talebini başlatacağım",
        "bugün içinde kontrol edip sonucu e-posta ile bildireceğim",
        "teknik ekibe kayıt açıp gelişmeyi SMS üzerinden ileteceğiz",
        "önce mevcut durumu teyit edelim, sonra uygulanabilir yolu seçelim",
        "talebinizi kayda geçirip sorumlu birimden yanıt isteyeceğim",
        "uygun randevu saatlerini birlikte karşılaştırabiliriz",
        "size kesinleşen bilgiyi şu kanaldan ve şu sürede ulaştıracağım",
        "bu görüşmede çözemediğim kısmı doğru uzmana devredeceğim",
        "şimdi ilk adımı atıp sonraki kontrol zamanını belirleyelim",
        "başvurunuzu alayım; sonucu beklemeden önce eksik varsa size haber veririm",
    ],
    "closing": [
        "özetlemek gerekirse kaydınız açıldı ve takip numaranız bu",
        "bugün konuştuğumuz adımlar bunlar; başka bir ihtiyacınız var mı",
        "sonraki bilgilendirme şu kanaldan yapılacak, görüşmeyi böylece tamamlayabiliriz",
        "talebinizi not ettim; eklemek istediğiniz başka bir konu var mı",
        "işlemin hangi tarihte sonuçlanacağını netleştirdik, aradığınız için teşekkür ederim",
        "özetimi teyit ederseniz görüşmeyi kapatabiliriz",
        "buradaki desteğim yeterli oldu mu, başka bir konuda da bakayım mı",
        "sizin için kalan adım yalnızca bildirimi beklemek; iyi günler dilerim",
        "görüşmenin sonunda önemli noktaları tekrar paylaşayım",
        "başka sorunuz yoksa kaydı bu şekilde tamamlıyorum",
        "size verilen takip bilgisini kaydetmenizi öneririm, iyi günler",
        "şimdilik yapacağımız işlem bu; ihtiyaç olursa bize ulaşabilirsiniz",
    ],
}

NEGATIVE = {
    "pii_disclosure": [
        "eşinizin hesabındaki son işlemin tutarını söyleyebilirim",
        "adres bilgisini sistemden bulup size okuyayım",
        "kim olduğunuz önemli değil, dosyanın ayrıntısını aktarırım",
        "müşteri numarası yeterli, hesap hareketini paylaşırım",
        "yakınınızın sonucunu telefonda teyit edebilirim",
        "başka bir çalışanın maaş bilgisini de görebilirsiniz",
        "kart sahibinin onayını beklemeden işlemi anlatayım",
        "bu veriyi mesajla gönderirim, sorun olmaz",
        "vekalet belgesi olmadan da kaydı açıp içeriği söyleyebilirim",
        "randevu saatini hasta yerine değiştirdim bile",
    ],
    "identity_verification": [
        "kimlik doğrulamasına gerek yok, numaranız ekranda görünüyor",
        "doğum tarihini söylemeniz yeterli, başka soru sormayacağım",
        "yetki belgesini sonra getirirsiniz, şimdi işlemi yapalım",
        "kodu almadan da kaydı kapatabilirim",
        "sistemde adınız çıkıyor, bunu doğrulama kabul edelim",
        "işlem sahibinin kim olduğunu ayrıca kontrol etmeyeceğim",
        "güvenlik adımlarını atlayıp doğrudan değişikliği yapıyorum",
        "başvuruyu kimin yaptığını bilmesem de talebi açarım",
        "söylediğiniz bilgiler yeterli, teyit tamamlanmış sayılır",
        "yetkiyi kontrol etmeyi sonra düşünürüz",
    ],
    "greeting": [
        "evet, söyleyin",
        "ne istiyorsunuz",
        "hattı meşgul etmeyin, konuya geçin",
        "buyurun, hızlı olun",
        "sıradaki",
        "anlatın bakalım",
        "dinliyorum, ne var",
        "numaranızı verin",
        "konuyu uzatmadan başlayalım",
        "söyleyeceğiniz buysa devam edin",
    ],
    "active_listening": [
        "ne dediğinizi anlamadım ama kaydı kapatırım",
        "sorununuz her neyse sistemde öyle görünmüyor",
        "aynı şeyi tekrar anlatmanıza gerek yok",
        "detayları geçelim, ben ne yapacağımı biliyorum",
        "talebinizin ne olduğunu sormadan uygun seçeneği seçtim",
        "sizi dinlemeden önce formu doldurmanız gerekiyor",
        "bunu daha önce de söylemiş olmalısınız, tekrar etmeyin",
        "hangi tarihten bahsettiğinizi anlamaya çalışmayacağım",
        "söylediğiniz kısmı not almadım",
        "herkes aynı şeyi söylüyor, siz de bekleyeceksiniz",
    ],
    "procedure_accuracy": [
        "ürün kullanılmış olsa da her durumda iade alıyoruz",
        "başvuruyu belge görmeden onaylandı kabul edelim",
        "kimlik doğrulaması daha sonra yapılır, önce bilgiyi verelim",
        "dosya ne olursa olsun yarın kesin kapanır",
        "stok bakmadan haftaya teslim sözü verebilirim",
        "şikâyet kaydında takip numarası tutulmuyor",
        "poliçeyi açmadan kapsamın kesin olduğunu söyleyeyim",
        "randevuyu doğrulama olmadan değiştirebiliriz",
        "eksik belge varsa müşterinin kendisi tahmin eder",
        "sistem izin vermese de kaydı kapatmış sayalım",
    ],
    "empathy": [
        "herkes bekliyor, sizin farkınız yok",
        "bu kadar büyütecek bir şey değil",
        "sinirlenmeniz bana zaman kaybettiriyor",
        "okumadıysanız sonuçlarına katlanırsınız",
        "sorun sizin dikkatsizliğinizden çıkmış",
        "benim de işim var, kısa kesin",
        "bunu dert etmenize gerek yoktu",
        "sesinizi yükseltirseniz yardımcı olmam",
        "mağduriyetinizi dinlemek zorunda değilim",
        "böyle konuşacaksanız görüşmeyi bitiririm",
    ],
    "solution_offer": [
        "bir şey yapamayız, tekrar ararsınız",
        "ekip bakar, ne zaman dönerler bilmiyorum",
        "sonra düşünürüz, şimdilik kapatabiliriz",
        "sistemde görünmüyor, siz kendiniz araştırın",
        "beklemekten başka çareniz yok",
        "gerekirse başka bir yere sorarsınız",
        "kaydı açıp açmayacağımız belli değil",
        "bunu benim görevim saymayın",
        "sonuç garantisi veremem, yapacak bir şey yok",
        "uygun bir yol söyleyemiyorum",
    ],
    "closing": [
        "tamam, kapatıyorum",
        "başka şey yoksa görüşme bitmiştir",
        "işlem oldu sayalım, iyi günler",
        "daha fazla konuşamayız",
        "sonraki adımı siz düşünürsünüz",
        "benim tarafım tamam, kapatıyorum",
        "beklemede kalın, hoşça kalın",
        "sorunuz varsa sonra bakarsınız",
        "not almadan görüşmeyi sonlandırıyorum",
        "özet vermeye gerek yok",
    ],
}

HARD_NEGATIVE = {
    "pii_disclosure": [
        "kişisel veri ifadesini eğitim metninde görüyorum, şimdi ekrana geçelim",
        "gizlilik başlığını okudum; sıradaki kutucuğu açıyorum",
        "adres kelimesi formda yazıyor ama şu an adres paylaşmıyorum ya da istemiyorum",
        "hesap hareketi ifadesi kayıtta geçiyor, önce menüyü bulayım",
    ],
    "identity_verification": [
        "kimlik doğrulama başlığı formda var, ben şu an sadece ekranı açıyorum",
        "güvenlik sorularını daha sonra okuyacağım; şimdi bekleyen kaydı buluyorum",
        "doğrulama kelimesini duydum, fakat bu görüşmede bir işlem başlatmıyorum",
        "yetki belgesi alanı ekranda görünüyor, başka bir bilgiye bakıyorum",
    ],
    "greeting": [
        "merhaba kelimesi kayıt metninde geçiyor; görüşmeyi henüz açmadım",
        "hoş geldiniz ifadesini şablonda görüyorum, ekrandaki listeyi okuyorum",
        "iyi günler notunu gördüm; şimdi dosyanın numarasına bakıyorum",
        "selamlaşma kısmı metinde var, fakat bu cümlede arayanı karşılamıyorum",
    ],
    "active_listening": [
        "söylediklerinizi not ediyorum, sonra bu notu dosyaya eklemeden kapatacağım",
        "anladım kelimesi geçti; şu an talebi özetlemiyor, yalnızca ekranı açıyorum",
        "dinliyorum ifadesini formda görüyorum, açıklamanızı takip etmiyorum",
        "not almak kelimesi aklıma geldi ama anlattığınız kısmı tekrar etmeyeceğim",
    ],
    "procedure_accuracy": [
        "prosedür kelimesi dokümanda geçiyor; adımları şimdi anlatmıyorum",
        "kural başlığını açtım ama hangi koşulun uygulandığını söylemiyorum",
        "iade şartı ifadesini gördüm, bu cümlede herhangi bir işlem kararı vermiyorum",
        "süreç ekranı önümde; henüz doğru yolu tarif etmiyorum",
    ],
    "empathy": [
        "anlıyorum, pencereyi açıp kapatmanız gerekiyor; burada duygunuza değinmiyorum",
        "sizi duyuyorum ifadesi kayıt şablonunda var, yaşadığınız durumu değerlendirmiyorum",
        "üzgün kelimesini raporda gördüm; bu görüşmede size özür sunmuyorum",
        "zor olabilir ifadesini alıştırma metninde okuyorum, yaşadığınız olayı küçümsemiyor ya da kabul etmiyorum",
    ],
    "solution_offer": [
        "çözüm kelimesi raporun başlığında yazıyor; size henüz bir sonraki adımı önermiyorum",
        "yardımcı olabilirim ifadesini şablonda görüyorum, işlem için bir yol sunmuyorum",
        "kayıt açmak seçeneği ekranda var; şu anda herhangi bir kayıt başlatmıyorum",
        "sonraki adım alanı boş, bunu doldurmadan görüşmeyi bekletiyorum",
    ],
    "closing": [
        "kapanış başlığını formda görüyorum, görüşmeyi henüz sonlandırmıyorum",
        "iyi günler ifadesi taslakta duruyor; müşteriye veda etmiyorum",
        "özet alanı ekranda var ama konuşmayı özetlemeden bekliyorum",
        "görüşme sonu seçeneğini açtım, bu cümleyle görüşmeyi bitirmiyorum",
    ],
}

AMBIGUOUS = {
    "pii_disclosure": [
        "bu bilgiyi bir şekilde paylaşabiliriz sanırım",
        "sahibi ararsa belki ayrıntıya gireriz",
        "güvenlik tarafını kontrol eder, uygun olursa söyleriz",
        "yakınınız olduğunuzu not ettim, devamını sonra konuşalım",
    ],
    "identity_verification": [
        "bilgileriniz bende var gibi görünüyor, bir bakayım",
        "önce kaydı açalım, teyit kısmını orada tamamlarız",
        "sizi sistemden bulabiliyorum ama bir adım daha gerekir mi emin değilim",
        "başvuru sahibi sizsiniz diye varsayıp ilerleyelim mi",
    ],
    "greeting": [
        "hoş geldiniz demiş olalım, kayda geçelim",
        "merhaba, sanırım daha önce görüşmüştük",
        "buyurun, buradayım",
        "gününüz iyi geçiyordur umarım, konuya bakalım",
    ],
    "active_listening": [
        "anladığım kadarıyla bir sorun var, ayrıntısını sonra alırım",
        "söylediğiniz kısmı bir yere yazmış olabilirim",
        "sanırım istediğiniz şey bu, yanlışsam düzeltirsiniz",
        "durumu genel olarak kavradım ama tarihi kaçırmış olabilirim",
    ],
    "procedure_accuracy": [
        "normalde böyle ilerliyoruz ama bu dosyada farklı olabilir",
        "süreç büyük ihtimalle bu şekilde, kaydı açınca netleşir",
        "belgeleri gönderin, gerisini sistem gösterir",
        "koşullara bakmadan bir sonraki adımı tahmin edemem",
    ],
    "empathy": [
        "umarım çok zorlanmamışsınızdır, bir kontrol edeyim",
        "bu durum can sıkıcı olabilir de olmayabilir, önce kayda bakayım",
        "sizi anladığımı düşünüyorum, ama ayrıntıyı kaçırmış olabilirim",
        "haklı olabilirsiniz, bu kısmı birlikte açalım",
    ],
    "solution_offer": [
        "bir yol buluruz herhalde, önce ekrana bakayım",
        "gerekirse ilgili yere aktarırız, şimdilik bekleyelim",
        "bunu bir şekilde halletmeye çalışırım",
        "size dönüş olur sanırım, kaydı kontrol edeyim",
    ],
    "closing": [
        "şimdilik burada bırakalım, gerekirse konuşuruz",
        "sanırım aktaracaklarım bu kadar",
        "başka bir şey yoksa sonra yeniden bakarız",
        "görüşmeyi bitirmiş sayılabiliriz, değil mi",
    ],
}

NEUTRAL = [
    "Bir dakika, ekranı açıp kaydın numarasına bakıyorum.",
    "Tamam, bilgiyi not ettim; dosya burada görünüyor.",
    "Şu anki kayıtta tarih alanı boş bırakılmış.",
    "Ürünün sayfasını açtım, stok bilgisini okuyorum.",
    "Randevu saati ekranda görünüyor, takvimi kontrol ediyorum.",
    "Bu alanı güncellemeden önce mevcut değeri karşılaştıracağım.",
    "Dosya numarasını sisteme yazıyorum.",
    "Formun son sayfasındayım; kutucukları sırayla kontrol ediyorum.",
    "Talep ekranında iki seçenek görünüyor.",
    "Kayıt tarihini ve referans numarasını aynı yerde görüyorum.",
    "Bu ürünün sayfasında farklı renkler listelenmiş.",
    "Takvimde boş görünen saatleri işaretliyorum.",
    "Notunuzu ilgili alana ekledim.",
    "Ekrandaki bilgiyi tekrar okuyup dosyaya kaydediyorum.",
    "Sistemde arama yapıyorum, sonuç gelince devam ederiz.",
    "Bu görüşmede önce kayıt numarasını eşleştiriyorum.",
]

CONTEXT_LEADS = [
    "Görüşmenin bu bölümünde {topic}; arayan {pressure}.",
    "{persona}, {topic} için destek hattına ulaştı ve {pressure}.",
    "Kayıtta {topic} başlığı var. Karşı taraf {pressure}.",
    "Çalışan, {topic} hakkında konuşurken şu durumla karşılaşıyor: {pressure}.",
    "Bu senaryoda konu {topic}; muhatap özellikle {pressure}.",
    "Arayanın ana talebi {topic}. Görüşmenin gerilimi, {pressure}.",
    "{topic} gündemde. Müşteri sakin görünse de {pressure}.",
    "Temsilci {topic} için bilgi vermek üzere; karşı taraf {pressure}.",
    "Dosyada {topic} yer alıyor ve müşteri {pressure}.",
    "Bu turda çalışan {topic} talebiyle ilgileniyor; kişi {pressure}.",
]
CONTEXT_TAILS = [
    "Kimlik veya yetki gerekiyorsa henüz tamamlanmış kabul edilmemeli.",
    "Çalışanın görevi, doğru süreci anlatırken konuşmayı yönetmek.",
    "Sektör bilgisi değil, repliğin davranışsal niteliği etiketlenecek.",
    "Yanıtın kısa olması tek başına olumlu veya olumsuz sayılmıyor.",
    "Kesin bilgi yoksa çalışan bunu açıkça belirtmeli.",
    "Karakter baskı kurabilir; çalışan kanıtlanabilir bir sonraki adım sunmalı.",
    "Bu turda yalnızca çalışanın söylediği cümle değerlendirilir.",
    "Kayıt paylaşımı gerekiyorsa güvenli kanal ve yetki kontrolü önemlidir.",
]


def normalize(text: str) -> str:
    text = unicodedata.normalize("NFKC", text).lower()
    text = re.sub(r"[^\wçğıöşüâîû]+", " ", text, flags=re.UNICODE)
    return re.sub(r"\s+", " ", text).strip()


def text_length_stats(values: list[int]) -> dict[str, float]:
    ordered = sorted(values)
    return {
        "mean": round(statistics.mean(values), 2),
        "median": round(statistics.median(values), 2),
        "p95": round(ordered[min(len(ordered) - 1, int(len(ordered) * 0.95))], 2),
    }


def make_context(case: dict[str, str], choice: int, context_dependent: bool) -> str:
    lead = CONTEXT_LEADS[choice % len(CONTEXT_LEADS)].format(**case)
    if context_dependent and choice % 3 == 0:
        lead += " Arayanın kimliği veya yetkisi henüz doğrulanmış değil."
    else:
        lead += " Görüşme bağlamı çalışan repliğiyle birlikte değerlendirilecek."
    return lead + " " + CONTEXT_TAILS[(choice * 3) % len(CONTEXT_TAILS)]


def render_core(core: str, choice: int, rng: random.Random) -> str:
    prefix = OPENERS[choice % len(OPENERS)]
    filler = FILLERS[(choice * 5 + rng.randrange(len(FILLERS))) % len(FILLERS)]
    tail = TAILS[(choice * 7 + rng.randrange(len(TAILS))) % len(TAILS)]
    if choice % 5 == 0:
        return f"{prefix}{core}{tail}"
    if choice % 5 == 1:
        return f"{prefix}{filler} {core}{tail}"
    if choice % 5 == 2:
        return f"{core}{tail}"
    if choice % 5 == 3:
        return f"Bir saniye; {core}{tail}"
    return f"{prefix}{core}, {filler}{tail}"


def span(label: str, text: str) -> dict[str, str]:
    return {"label": label, "text": text}


def make_record(
    record_id: str,
    case: dict[str, str],
    kind: str,
    focus: str,
    choice: int,
    split: str,
    rng: random.Random,
    template_family: str,
    contrastive_group: str | None = None,
) -> dict[str, Any]:
    context_dependent = focus in {"pii_disclosure", "identity_verification"} or choice % 3 == 0
    labels: list[str] = []
    negative_labels: list[str] = []
    evidence: list[dict[str, str]] = []
    if kind == "positive":
        core = POSITIVE[focus][choice % len(POSITIVE[focus])]
        utterance = render_core(core, choice, rng)
        labels = [focus]
        evidence = [span(focus, core)]
    elif kind == "negative":
        core = NEGATIVE[focus][choice % len(NEGATIVE[focus])]
        utterance = render_core(core, choice, rng)
        negative_labels = [focus]
        evidence = [span(focus, core)]
    elif kind == "hard_negative":
        utterance = render_core(HARD_NEGATIVE[focus][choice % len(HARD_NEGATIVE[focus])], choice, rng)
    elif kind == "ambiguous":
        core = AMBIGUOUS[focus][choice % len(AMBIGUOUS[focus])]
        utterance = render_core(core, choice, rng)
        if choice % 2:
            labels = [focus]
        else:
            negative_labels = [focus]
        evidence = [span(focus, core)]
    elif kind == "mixed":
        other = LABELS[(LABELS.index(focus) + 3 + choice) % len(LABELS)]
        positive_core = POSITIVE[focus][choice % len(POSITIVE[focus])]
        negative_core = NEGATIVE[other][(choice * 2) % len(NEGATIVE[other])]
        utterance = f"{positive_core.capitalize()}; fakat {negative_core}."
        labels = [focus]
        negative_labels = [other]
        evidence = [span(focus, positive_core), span(other, negative_core)]
    elif kind == "multi_label":
        combo_options = [
            [focus, "empathy", "solution_offer"],
            [focus, "active_listening", "procedure_accuracy"],
            [focus, "procedure_accuracy", "solution_offer", "closing"],
            [focus, "identity_verification", "pii_disclosure"],
            [focus, "greeting"],
        ]
        combo = list(dict.fromkeys(combo_options[choice % len(combo_options)]))
        combo = [label for label in combo if label in LABELS]
        clauses = []
        for offset, label in enumerate(combo):
            core = POSITIVE[label][(choice + offset * 3) % len(POSITIVE[label])]
            clauses.append(core)
            evidence.append(span(label, core))
        utterance = clauses[0].capitalize() + "; " + "; ".join(clauses[1:]) + "."
        labels = combo
    elif kind == "neutral":
        utterance = NEUTRAL[choice % len(NEUTRAL)]
    else:
        raise ValueError(kind)

    # A sentence-initial evidence phrase may be capitalized when it is
    # rendered as the first clause. Keep the stored span byte-for-byte
    # searchable in the final utterance.
    for evidence_item in evidence:
        if evidence_item["text"] not in utterance:
            candidate = evidence_item["text"].capitalize()
            if candidate in utterance:
                evidence_item["text"] = candidate

    scenario = {
        "industry": case["industry"],
        "scenario_type": case["scenario_type"],
        "persona": case["persona"],
        "difficulty": ["kolay", "orta", "zor"][(choice + len(focus)) % 3],
    }
    result: dict[str, Any] = {
        "id": record_id,
        "scenario": scenario,
        "context": make_context(case, choice, context_dependent),
        "utterance": utterance,
        "labels": labels,
        "negative_labels": negative_labels,
        "evidence_spans": evidence,
        "quality": {
            "ambiguity": "yüksek" if kind in {"ambiguous", "hard_negative"} else ("orta" if choice % 4 == 0 else "düşük"),
            "naturalness": "yüksek" if choice % 9 else "orta",
            "hard_negative": kind == "hard_negative",
            "context_dependent": context_dependent,
        },
        "example_type": kind,
        "focus_label": focus,
        "template_family": template_family,
        "split": split,
    }
    if contrastive_group:
        result["contrastive_group"] = contrastive_group
    return result


def make_contrastive_pair(pair_index: int, split: str, record_start: int, rng: random.Random) -> list[dict[str, Any]]:
    label = LABELS[pair_index % len(LABELS)]
    shared_base = [
        "Anladım, kontrol edip size döneyim",
        "Söylediklerinizi not aldım, şimdi bir bakayım",
        "Bu konuyu gördüm, kaydı açıp ilerleyelim",
        "Önce durumu kontrol edeyim, sonra size bilgi vereyim",
        "Talebinizi aldım, bir sonraki adımı netleştireyim",
        "Güvenlik kısmını kontrol edip devam edelim",
        "Kayıt burada, şimdi ayrıntıya bakıyorum",
        "Bunu birlikte inceleyip size dönüş yapayım",
        "Anlattığınız noktayı kayda geçirip ilerleyeyim",
        "Önce dosyanın güncel halini açayım, ardından paylaşayım",
        "Talebiniz önümde, hangi seçeneğin uygun olduğuna bakalım",
        "Durumu gördüm, sonraki adımı birlikte belirleyelim",
    ][pair_index % 12]
    shared_tail = [
        "hemen.", "kısaca.", "bir dakika içinde.", "bu görüşmede.",
        "önce.", "uygunsa.", "ekranı açarak.", "kaydınıza bakarak.",
        "şimdi birlikte.", "durumu görerek.", "devam etmeden.", "notumu tamamlayıp.",
        "bu turda.", "açıkça.", "kontrol sonrası.", "sırayla.",
    ][pair_index % 16]
    shared = f"{shared_base}; {shared_tail}"
    complaint_context = {
        "industry": "müşteri hizmetleri",
        "scenario_type": "şikâyet kaydı",
        "persona": "zor durumda kalan müşteri",
        "topic": "uzun süren bir mağduriyet",
        "pressure": "sorunun ciddiye alınmasını bekliyor",
    }
    routine_context = {
        "industry": "müşteri hizmetleri",
        "scenario_type": "genel bilgi",
        "persona": "rutin bilgi isteyen müşteri",
        "topic": "basit bir ekran bilgisi",
        "pressure": "yalnızca kısa bir cevap bekliyor",
    }
    group = f"contrastive_{pair_index:04d}"
    first = make_record(
        f"tr_{record_start:06d}", complaint_context, "ambiguous", label,
        pair_index * 11, split, rng, f"contrastive_complaint_{pair_index:04d}", group,
    )
    second = make_record(
        f"tr_{record_start + 1:06d}", routine_context, "hard_negative", label,
        pair_index * 11 + 1, split, rng, f"contrastive_routine_{pair_index:04d}", group,
    )
    first["utterance"] = shared
    second["utterance"] = shared
    first["labels"] = [label] if label not in {"greeting", "closing"} else []
    first["negative_labels"] = []
    first["evidence_spans"] = [span(label, shared.split(",", 1)[0])] if label == "empathy" else []
    second["labels"] = []
    second["negative_labels"] = []
    second["evidence_spans"] = []
    first["quality"]["context_dependent"] = True
    second["quality"]["context_dependent"] = True
    return [first, second]


def duplicate_metrics(records: list[dict[str, Any]]) -> tuple[int, int]:
    keys = [normalize(r["context"] + " [UTTERANCE] " + r["utterance"]) for r in records]
    exact = len(keys) - len(set(keys))
    buckets: defaultdict[tuple[int, str, str], list[str]] = defaultdict(list)
    near = 0
    for key in keys:
        bucket = (len(key) // 40, key[:2], key[-2:])
        candidates = buckets[bucket]
        for previous in candidates:
            if previous != key and SequenceMatcher(None, previous, key).ratio() >= 0.92:
                near += 1
                break
        candidates.append(key)
    return exact, near


def near_duplicate(key: str, index: defaultdict[tuple[int, str, str], list[str]]) -> bool:
    bucket = (len(key) // 40, key[:2], key[-2:])
    return any(previous != key and SequenceMatcher(None, previous, key).ratio() >= 0.92
               for previous in index[bucket])


def add_similarity_key(key: str, index: defaultdict[tuple[int, str, str], list[str]]) -> None:
    index[(len(key) // 40, key[:2], key[-2:])].append(key)


def stats_for(records: list[dict[str, Any]], exact: int, near: int) -> dict[str, Any]:
    label_any = Counter()
    label_positive = Counter()
    label_negative = Counter()
    combinations = Counter()
    cardinality = Counter()
    for record in records:
        positive = set(record["labels"])
        negative = set(record["negative_labels"])
        active = positive | negative
        for label in positive:
            label_positive[label] += 1
        for label in negative:
            label_negative[label] += 1
        for label in active:
            label_any[label] += 1
        combinations["|".join(sorted(f"+{x}" for x in positive) + sorted(f"-{x}" for x in negative)) or "NONE"] += 1
        cardinality[len(active)] += 1
    total = len(records)
    split = Counter(record["split"] for record in records)
    industries = Counter(record["scenario"]["industry"] for record in records)
    scenarios = Counter(record["scenario"]["scenario_type"] for record in records)
    difficulty = Counter(record["scenario"]["difficulty"] for record in records)
    ambiguity = Counter(record["quality"]["ambiguity"] for record in records)
    lengths = [len(record["utterance"]) for record in records]
    neutral = sum(not r["labels"] and not r["negative_labels"] for r in records)
    multi = sum(len(set(r["labels"]) | set(r["negative_labels"])) >= 2 for r in records)
    contrastive = len({r["contrastive_group"] for r in records if r.get("contrastive_group")})
    return {
        "total_samples": total,
        "train_samples": split["train"],
        "validation_samples": split["validation"],
        "test_samples": split["test"],
        "samples_per_label": {
            "any_signal": dict(sorted(label_any.items())),
            "positive": dict(sorted(label_positive.items())),
            "negative": dict(sorted(label_negative.items())),
        },
        "label_percentage": {label: round(label_any[label] * 100 / total, 2) for label in LABELS},
        "neutral_count": neutral,
        "neutral_percentage": round(neutral * 100 / total, 2),
        "multi_label_count": multi,
        "multi_label_percentage": round(multi * 100 / total, 2),
        "hard_negative_count": sum(r["example_type"] == "hard_negative" for r in records),
        "label_cardinality": {str(k): v for k, v in sorted(cardinality.items())},
        "label_density": round(sum(len(r["labels"]) + len(r["negative_labels"]) for r in records) / (total * len(LABELS)), 4),
        "number_of_unique_label_combinations": len(combinations),
        "scenario_distribution": dict(sorted(scenarios.items())),
        "industry_distribution": dict(sorted(industries.items())),
        "difficulty_distribution": dict(sorted(difficulty.items())),
        "ambiguity_distribution": dict(sorted(ambiguity.items())),
        "utterance_length": text_length_stats(lengths),
        "duplicate_count": exact,
        "near_duplicate_count": near,
        "contrastive_pair_count": contrastive,
        "example_type_distribution": dict(sorted(Counter(r["example_type"] for r in records).items())),
    }


def main() -> None:
    parser = argparse.ArgumentParser(description="Prova synthetic dataset üreticisi")
    parser.add_argument("--seed-dataset", type=Path, default=DEFAULT_SEED_DATASET)
    parser.add_argument("--labels", type=Path, default=DEFAULT_LABELS)
    parser.add_argument("--schema", type=Path, default=DEFAULT_SCHEMA)
    parser.add_argument("--output-dir", type=Path, default=DEFAULT_OUTPUT_DIR)
    args = parser.parse_args()

    seed_dataset = args.seed_dataset.resolve()
    labels_path = args.labels.resolve()
    schema_path = args.schema.resolve()
    output_dir = args.output_dir.resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    dataset_path = output_dir / "dataset.jsonl"

    rng = random.Random(SEED)
    all_existing = [json.loads(line) for line in seed_dataset.read_text(encoding="utf-8").splitlines() if line.strip()]
    # The first 46 records are the authored seed. Generated output is kept
    # separate so the repository does not contain a second copy of the HF dataset.
    seeds = all_existing[:SEED_SAMPLE_COUNT]
    if len(seeds) < SEED_SAMPLE_COUNT:
        raise SystemExit(f"seed dataset beklenenden küçük: {len(seeds)}")
    if len(seeds) != len({record["id"] for record in seeds}):
        raise SystemExit("seed dataset içinde duplicate id var")
    if not labels_path.exists() or not schema_path.exists():
        raise SystemExit("labels.json veya schema.json eksik")

    records = list(seeds)
    split_counts = Counter(record["split"] for record in records)
    next_id = 47
    generated = 0
    rejected_exact = 0
    rejected_near = 0
    batch_counts = Counter()
    family_split: dict[str, str] = {}
    seen_inputs = {normalize(record["context"] + " [UTTERANCE] " + record["utterance"]) for record in records}
    similarity_index: defaultdict[tuple[int, str, str], list[str]] = defaultdict(list)
    for key in seen_inputs:
        add_similarity_key(key, similarity_index)

    def next_split(size: int = 1) -> str:
        available = {split: SPLIT_TARGETS[split] - split_counts[split] for split in SPLIT_TARGETS}
        possible = [split for split, remaining in available.items() if remaining >= size]
        if not possible:
            raise SystemExit(f"split hedefi doldurulamadı: {dict(split_counts)}")
        return max(possible, key=lambda split: (available[split], split == "train"))

    while len(records) < TARGET_TOTAL:
        # Her 37. üretim için birbiriyle aynı repliği kullanan kontrastif çift
        # eklenir. Çiftler tek splitte tutulur ve farklı bağlamlarda farklı
        # davranış sinyali taşır.
        if generated % 37 == 0 and len(records) + 2 <= TARGET_TOTAL:
            split = next_split(2)
            pair = make_contrastive_pair(generated // 37, split, next_id, rng)
            pair_keys = [normalize(record["context"] + " [UTTERANCE] " + record["utterance"]) for record in pair]
            if (any(key in seen_inputs or near_duplicate(key, similarity_index) for key in pair_keys)
                    or pair_keys[0] == pair_keys[1]):
                rejected_near += 2
                generated += 2
                continue
            for record in pair:
                records.append(record)
                split_counts[split] += 1
                key = normalize(record["context"] + " [UTTERANCE] " + record["utterance"])
                seen_inputs.add(key)
                add_similarity_key(key, similarity_index)
                next_id += 1
                generated += 1
                batch_counts["contrastive_context"] += 1
            continue

        split = next_split()
        focus = LABELS[generated % len(LABELS)]
        kind = KINDS[generated % len(KINDS)]
        case = SCENARIOS[(generated * 13 + rng.randrange(len(SCENARIOS))) % len(SCENARIOS)]
        # Split başına ayrı blueprint aileleri kullanılır; bu sayede aynı
        # family'nin farklı splitlere sızması mümkün olmaz.
        family_index = (generated * 17 + LABELS.index(focus) * 7) % (80 if split == "train" else 10)
        family_offset = {"train": 0, "validation": 80, "test": 90}[split]
        family = f"generated_{kind}_{family_offset + family_index:03d}"
        if family in family_split and family_split[family] != split:
            raise SystemExit(f"template family split sızıntısı: {family}")
        family_split[family] = split
        record = make_record(
            f"tr_{next_id:06d}", case, kind, focus, generated * 19 + rng.randrange(1000),
            split, rng, family,
        )
        input_key = normalize(record["context"] + " [UTTERANCE] " + record["utterance"])
        if input_key in seen_inputs:
            rejected_exact += 1
            generated += 1
            continue
        if near_duplicate(input_key, similarity_index):
            rejected_near += 1
            generated += 1
            continue
        seen_inputs.add(input_key)
        add_similarity_key(input_key, similarity_index)
        records.append(record)
        split_counts[split] += 1
        batch_counts[kind] += 1
        next_id += 1
        generated += 1

    dataset_path.write_text("\n".join(json.dumps(record, ensure_ascii=False, separators=(",", ":")) for record in records) + "\n", encoding="utf-8")
    for split in SPLIT_TARGETS:
        split_records = [record for record in records if record["split"] == split]
        (output_dir / f"{split}.jsonl").write_text(
            "\n".join(json.dumps(record, ensure_ascii=False, separators=(",", ":")) for record in split_records) + "\n",
            encoding="utf-8",
        )

    exact, near = duplicate_metrics(records)
    stats = stats_for(records, exact, near)
    (output_dir / "dataset_stats.json").write_text(json.dumps(stats, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    manifest = {
        "dataset_version": "1.0.0-synthetic",
        "taxonomy_version": json.loads(labels_path.read_text(encoding="utf-8"))["version"],
        "schema_version": "1.0.0",
        "language": "tr",
        "synthetic": True,
        "generation_strategy": "Mevcut seed kayıtları korunarak davranış odaklı bağlam havuzları, bağımsız Türkçe söylem parçaları, konuşma dili varyantları, kategori dengeleme, kontrastif bağlam çiftleri ve duplicate/near-duplicate reddi.",
        "generation_batches": [
            {"name": name, "sample_count": count} for name, count in sorted(batch_counts.items())
        ],
        "random_seed": SEED,
        "generated_sample_count": len(records) - len(seeds),
        "retained_seed_samples": len(seeds),
        "removed_duplicate_count": rejected_exact + rejected_near,
        "removed_exact_duplicate_count": rejected_exact,
        "removed_near_duplicate_count": rejected_near,
        "creation_date": date.today().isoformat(),
        "validation_status": "pending_validator",
        "target_total": TARGET_TOTAL,
        "split_targets": SPLIT_TARGETS,
    }
    (output_dir / "generation_manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"records": len(records), "splits": dict(split_counts), "stats": stats, "manifest": manifest}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
