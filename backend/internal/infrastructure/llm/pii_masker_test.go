package llm

import (
	"strings"
	"testing"
)

// Geçerli bir TC kimlik numarası (checksum doğru), yalnızca test için.
const validTCKN = "10000000146"

func TestMasker_MasksTurkishPII(t *testing.T) {
	m := NewMasker()

	cases := map[string]struct {
		input string
		want  string
	}{
		"tc kimlik": {"Kimlik numaram " + validTCKN + " efendim.", maskTCKN},
		"iban":      {"IBAN: TR33 0006 1005 1978 6457 8413 26", maskIBAN},
		"telefon":   {"Beni 0532 123 45 67 numarasından arayın.", maskPhone},
		"eposta":    {"Adresim ayse.yilmaz@example.com olarak kayıtlı.", maskEmail},
		"kart":      {"Kart numaram 4111 1111 1111 1111", maskCard},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			masked, count := m.Mask(tc.input)
			if !strings.Contains(masked, tc.want) {
				t.Fatalf("%s maskelenmeliydi: %q", name, masked)
			}
			if count == 0 {
				t.Fatal("maskelenen alan sayısı sıfır olmamalı")
			}
		})
	}
}

// Model verinin YERİNİ görmeli ama değerini görmemeli: "[TCKN]" yazan bir
// metin çalışanın kimlik numarası paylaştığını gösterir ve puanlama bunu
// değerlendirebilir.
func TestMasker_RemovesTheValueButKeepsThePosition(t *testing.T) {
	m := NewMasker()

	masked, _ := m.Mask("Kimlik " + validTCKN + " ve IBAN TR330006100519786457841326")

	if strings.Contains(masked, validTCKN) {
		t.Fatal("kimlik numarasının kendisi metinde kalmamalı")
	}
	if strings.Contains(masked, "TR330006100519786457841326") {
		t.Fatal("IBAN'ın kendisi metinde kalmamalı")
	}
	if !strings.Contains(masked, maskTCKN) || !strings.Contains(masked, maskIBAN) {
		t.Fatalf("maskeleme etiketleri yerinde olmalı: %q", masked)
	}
}

// Doğrulamasız bir 11 haneli desen, sipariş ve fatura numaralarını da
// maskeler; modelin göremediği bilgi puanlamayı bozar.
func TestMasker_LeavesNonPIINumbersAlone(t *testing.T) {
	m := NewMasker()

	// Checksum'ı tutmayan 11 haneli bir sayı ve kısa bir sipariş numarası.
	input := "Sipariş numaram 12345678901, takip kodu 55512."
	masked, count := m.Mask(input)

	if masked != input {
		t.Fatalf("kişisel veri olmayan sayılar maskelenmemeli: %q", masked)
	}
	if count != 0 {
		t.Fatalf("maskelenen alan olmamalı, %d sayıldı", count)
	}
}

// Luhn tutmayan uzun bir rakam dizisi kart sayılmamalı.
func TestMasker_RejectsNumbersThatFailLuhn(t *testing.T) {
	m := NewMasker()

	masked, _ := m.Mask("Referans 4111111111111112 numarası")

	if strings.Contains(masked, maskCard) {
		t.Fatalf("Luhn tutmayan sayı kart sayılmamalı: %q", masked)
	}
}

// IBAN'dan önce telefon aransaydı, IBAN'ın içindeki rakam dizisi telefon
// sanılıp parça parça maskelenir ve geri kalanı sızardı.
func TestMasker_HandlesOverlappingPatternsWithoutLeaking(t *testing.T) {
	m := NewMasker()

	masked, _ := m.Mask("Hesabım TR33 0006 1005 1978 6457 8413 26, telefonum 05321234567")

	if !strings.Contains(masked, maskIBAN) {
		t.Fatalf("IBAN bütün olarak maskelenmeli: %q", masked)
	}
	if !strings.Contains(masked, maskPhone) {
		t.Fatalf("telefon maskelenmeli: %q", masked)
	}
	for _, fragment := range []string{"0006", "6457", "8413", "5321234567"} {
		if strings.Contains(masked, fragment) {
			t.Fatalf("parça sızdı (%s): %q", fragment, masked)
		}
	}
}

// Sayı kayda geçiyor: KVKK hesap verebilirliği "hangi veriyi işlediniz"
// sorusunu soruyor ve bu, ölçülebilir cevabın kendisi.
func TestMasker_CountsEveryMaskedField(t *testing.T) {
	m := NewMasker()

	_, count := m.Mask("Kimlik " + validTCKN + ", e-posta a@b.com, telefon 05321234567")

	if count != 3 {
		t.Fatalf("üç alan maskelenmeliydi, %d sayıldı", count)
	}
}

func TestMasker_LeavesCleanTextUntouched(t *testing.T) {
	m := NewMasker()

	input := "Merhaba, iade işleminiz için sipariş bilgilerinizi alabilir miyim?"
	masked, count := m.Mask(input)

	if masked != input || count != 0 {
		t.Fatalf("temiz metin değişmemeli: %q (%d)", masked, count)
	}
}

func TestIsValidTCKN(t *testing.T) {
	if !isValidTCKN(validTCKN) {
		t.Fatal("geçerli TC kimlik numarası kabul edilmeli")
	}
	for _, invalid := range []string{"00000000000", "12345678901", "1234567890", validTCKN + "1"} {
		if isValidTCKN(invalid) {
			t.Errorf("geçersiz numara reddedilmeli: %s", invalid)
		}
	}
}
