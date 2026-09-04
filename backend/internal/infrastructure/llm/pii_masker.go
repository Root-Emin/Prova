package llm

import (
	"regexp"
	"strings"
)

// Masker, LLM'e giden metinden kişisel veriyi temizler.
//
// Yalnızca giden kopyada çalışır. Transkriptin kendisi maskelenmez, çünkü
// "çalışan kişisel veri ifşa etti mi" sorusu ancak orijinal metinle
// cevaplanabilir — maskelenmiş bir transkript üzerinden puanlama, ölçmesi
// gereken şeyi göremez.
//
// Desenlerin sırası önemli: en uzun ve en belirli olan önce gelir. IBAN'dan
// önce telefon aransaydı, IBAN'ın içindeki rakam dizisi telefon sanılıp
// parça parça maskelenir ve geri kalanı sızardı.
type Masker struct {
	rules []maskRule
}

type maskRule struct {
	name        string
	pattern     *regexp.Regexp
	replacement string
	// validate, deseni tutan bir eşleşmenin gerçekten o tür olduğunu
	// doğrular. Boş bırakılabilir.
	validate func(string) bool
}

// Maskeleme etiketleri. Model, verinin YERİNİ görüyor ama değerini görmüyor:
// "[TCKN]" yazan bir metin, çalışanın bir kimlik numarası paylaştığını
// gösterir ve puanlama bunu değerlendirebilir.
const (
	maskTCKN  = "[TCKN]"
	maskIBAN  = "[IBAN]"
	maskCard  = "[KART]"
	maskPhone = "[TELEFON]"
	maskEmail = "[EPOSTA]"
)

// NewMasker builds the masker with the Turkish PII rule set.
func NewMasker() *Masker {
	return &Masker{rules: []maskRule{
		{
			// IBAN önce: TR + 24 hane. Kart ve telefon desenlerinden önce
			// aranmalı, yoksa içindeki rakam dizileri onlara yem olur.
			name:        "iban",
			pattern:     regexp.MustCompile(`(?i)\bTR\s?[0-9]{2}(?:\s?[0-9]{4}){5}\s?[0-9]{2}\b`),
			replacement: maskIBAN,
		},
		{
			name:        "email",
			pattern:     regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),
			replacement: maskEmail,
		},
		{
			// Kart numarası: 13-19 hane, boşluk ya da tire ile gruplanabilir.
			// Luhn doğrulaması yapılıyor çünkü bu desen aksi hâlde her uzun
			// rakam dizisini yakalar.
			name:        "card",
			pattern:     regexp.MustCompile(`\b(?:[0-9][ -]?){13,19}\b`),
			replacement: maskCard,
			validate:    isLuhnValid,
		},
		{
			// TC kimlik: 11 hane, ilk hane sıfır olamaz. Checksum
			// doğrulaması yapılıyor: doğrulamasız bir 11 haneli desen,
			// sipariş numarasını da kimlik numarası sanardı.
			name:        "tckn",
			pattern:     regexp.MustCompile(`\b[1-9][0-9]{10}\b`),
			replacement: maskTCKN,
			validate:    isValidTCKN,
		},
		{
			// Telefon: +90 ile ya da 0 ile başlayan Türkiye numaraları ve
			// on haneli çıplak numaralar.
			name:        "phone",
			pattern:     regexp.MustCompile(`(?:\+90|0)?[ .\-]?\(?5[0-9]{2}\)?[ .\-]?[0-9]{3}[ .\-]?[0-9]{2}[ .\-]?[0-9]{2}\b`),
			replacement: maskPhone,
		},
	}}
}

// Mask, metni maskeler ve kaç alanın maskelendiğini döndürür.
//
// Sayı kayda geçiyor: KVKK hesap verebilirliği "hangi veriyi işlediniz"
// sorusunu soruyor, ve "kaç alanı maskeledik" bunun ölçülebilir cevabı.
func (m *Masker) Mask(text string) (string, int) {
	if text == "" {
		return text, 0
	}

	masked := text
	var count int

	for _, rule := range m.rules {
		masked = rule.pattern.ReplaceAllStringFunc(masked, func(match string) string {
			if rule.validate != nil && !rule.validate(match) {
				return match
			}
			count++
			return rule.replacement
		})
	}

	return masked, count
}

// isValidTCKN, TC kimlik numarasının kontrol hanelerini doğrular.
//
// Algoritma resmî: ilk 9 hanenin tek ve çift konumlu toplamlarından 10.
// hane, ilk 10 hanenin toplamının mod 10'undan 11. hane üretilir.
// Doğrulamasız bir 11 haneli desen, sipariş ve fatura numaralarını da
// maskeler ve modelin göremediği bilgi puanlamayı bozardı.
func isValidTCKN(s string) bool {
	digits := onlyDigits(s)
	if len(digits) != 11 || digits[0] == '0' {
		return false
	}

	value := func(i int) int { return int(digits[i] - '0') }

	var odd, even int
	for i := 0; i < 9; i += 2 {
		odd += value(i)
	}
	for i := 1; i < 8; i += 2 {
		even += value(i)
	}

	tenth := ((odd * 7) - even) % 10
	if tenth < 0 {
		tenth += 10
	}
	if tenth != value(9) {
		return false
	}

	var sum int
	for i := 0; i < 10; i++ {
		sum += value(i)
	}
	return sum%10 == value(10)
}

// isLuhnValid, kart numarasının Luhn kontrolünü uygular.
func isLuhnValid(s string) bool {
	digits := onlyDigits(s)
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	var sum int
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

func onlyDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
