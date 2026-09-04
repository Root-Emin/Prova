package service

import (
	"strings"
	"unicode"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// QuoteVerifier, puanlama modelinin gösterdiği alıntıyı transkriptte arar.
//
// Uydurma alıntı ürünün güvenilirliğini bitirir: "şunu söylediniz" diyen bir
// sertifika, söylenmemiş bir cümleyi gösterirse yalnızca o madde değil tüm
// puanlama tartışmalı hâle gelir. Bu yüzden alıntı sessizce kabul edilmiyor,
// aranıyor ve bulunamazsa doğrulanmadı olarak işaretleniyor.
type QuoteVerifier struct {
	// turns, konuşma sırası numarasından normalize edilmiş metne.
	turns map[int]string
	// order, sıraların artan numarası. Sıra numarası verilmediğinde
	// belirlenimci bir tarama için gerekli.
	order []int
}

// NewQuoteVerifier builds a verifier over a session transcript.
func NewQuoteVerifier(turns []*model.Turn) *QuoteVerifier {
	v := &QuoteVerifier{turns: make(map[int]string, len(turns))}
	for _, turn := range turns {
		if turn == nil {
			continue
		}
		v.turns[turn.Index] = NormalizeForQuote(turn.Text)
		v.order = append(v.order, turn.Index)
	}
	for i := 1; i < len(v.order); i++ {
		for j := i; j > 0 && v.order[j] < v.order[j-1]; j-- {
			v.order[j], v.order[j-1] = v.order[j-1], v.order[j]
		}
	}
	return v
}

// Verify, alıntının transkriptte gerçekten bulunup bulunmadığını söyler.
//
// claimedTurn doluysa yalnızca o sıraya bakılır: model hem alıntıyı hem de
// nereden geldiğini iddia ediyor, ve iddianın ikinci yarısı da yanlış
// olabilir. Alıntı başka bir sırada bulunsa bile, iddia edilen sırada yoksa
// iddia yanlıştır.
//
// claimedTurn boşsa tüm transkript taranır ve bulunan sıra döndürülür; bu,
// sıra numarası vermeyen bir modelin cezalandırılmaması içindir.
func (v *QuoteVerifier) Verify(quote string, claimedTurn *int) (verified bool, foundTurn *int) {
	needle := NormalizeForQuote(quote)
	if needle == "" {
		// Boş alıntı doğrulanmış sayılmaz: gerekçesiz bir puan, gerekçesi
		// uydurulmuş bir puandan daha iyi değildir.
		return false, nil
	}

	if claimedTurn != nil {
		text, ok := v.turns[*claimedTurn]
		if ok && strings.Contains(text, needle) {
			return true, claimedTurn
		}
		return false, nil
	}

	for _, index := range v.order {
		if strings.Contains(v.turns[index], needle) {
			found := index
			return true, &found
		}
	}
	return false, nil
}

// VerifyAll, bir puanın tüm kriter alıntılarını doğrular ve işaretler.
func (v *QuoteVerifier) VerifyAll(criteria []model.CriterionScore) []model.CriterionScore {
	for i := range criteria {
		verified, found := v.Verify(criteria[i].Quote, criteria[i].TurnIndex)
		criteria[i].QuoteVerified = verified
		if verified && criteria[i].TurnIndex == nil {
			criteria[i].TurnIndex = found
		}
	}
	return criteria
}

// NormalizeForQuote, karşılaştırma için metni sadeleştirir.
//
// Küçük harfe çevirir, noktalama işaretlerini atar ve boşlukları tekilleştirir.
// Bunlar olmadan doğrulama, modelin cümle sonundaki noktayı düşürmesi ya da
// tırnak işaretini değiştirmesi gibi anlamsız nedenlerle başarısız olur — ve
// yanlış negatifler, doğrulamanın kendisine olan güveni yok eder.
//
// Türkçe'ye özel bir ayrıntı: "İ" harfinin küçüğü unicode.ToLower ile "i̇"
// (i + birleşen nokta) olur ve bu, kullanıcının yazdığı "i" ile eşleşmez.
// Birleşen işaretler bu yüzden atılıyor.
func NormalizeForQuote(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	lastWasSpace := true
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsSpace(r):
			if !lastWasSpace {
				b.WriteRune(' ')
				lastWasSpace = true
			}
		case unicode.Is(unicode.Mn, r):
			// Birleşen işaret: atlanır (bkz. yukarıdaki "İ" notu).
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastWasSpace = false
		default:
			// Noktalama ve semboller atılır.
		}
	}

	return strings.TrimSpace(b.String())
}
