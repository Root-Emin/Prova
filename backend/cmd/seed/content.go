package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	provaMongoRepo "github.com/masterfabric-go/masterfabric/internal/infrastructure/mongo/prova"
	"github.com/masterfabric-go/masterfabric/internal/shared/database"
)

// seedContent, MongoDB tarafını yazar: karakterler, rubrik, senaryolar ve
// iki LLM kademesi. Hepsi yayınlanmış olarak bırakılır; taslak bir senaryo
// oynanamaz ve tohum verisinin işi oynanabilir bir sistem bırakmaktır.
func seedContent(ctx context.Context, mongoDB *database.MongoDB) error {
	scope := provaRepo.NewScope(demoOrgID)

	characters := provaMongoRepo.NewCharacterRepo(mongoDB.Database)
	rubrics := provaMongoRepo.NewRubricRepo(mongoDB.Database)
	scenarios := provaMongoRepo.NewScenarioRepo(mongoDB.Database)
	profiles := provaMongoRepo.NewLLMProfileRepo(mongoDB.Database)

	calm, err := seedCharacter(ctx, characters, scope, calmCharacterID, sakinMusteri())
	if err != nil {
		return err
	}
	hostile, err := seedCharacter(ctx, characters, scope, hostileCharacterD, ofkeliArayan())
	if err != nil {
		return err
	}

	rubric, err := seedRubric(ctx, rubrics, scope, serviceRubricID, musteriHizmetleriRubrigi())
	if err != nil {
		return err
	}

	if _, err := seedScenario(ctx, scenarios, scope, refundScenarioID, iadeSenaryosu(calm, rubric)); err != nil {
		return err
	}
	if _, err := seedScenario(ctx, scenarios, scope, kvkkScenarioID, kvkkSenaryosu(hostile, rubric)); err != nil {
		return err
	}

	if _, err := seedProfile(ctx, profiles, scope, fastProfileID, hizliKademe()); err != nil {
		return err
	}
	if _, err := seedProfile(ctx, profiles, scope, strongProfileID, gucluKademe()); err != nil {
		return err
	}

	return nil
}

// --- karakterler ---

func sakinMusteri() *provaModel.Character {
	return &provaModel.Character{
		Name: "Selin Aydın — sakin müşteri",
		Persona: "Bir e-ticaret sitesinden aldığı ayakkabı ayağına olmayan, " +
			"iade etmek isteyen bir müşterisin. Aceleci değilsin, karşındakini " +
			"dinlersin, ama süreci de merak edersin.",
		BehaviorRules: []string{
			"Kendini tanıtırken yalnızca adını söylersin.",
			"Sipariş numaranı sorulduğunda verirsin.",
			"Çalışan prosedürü açıklarsa kabul edersin.",
			"Konuşmayı en fazla birkaç cümlede toparlarsın.",
		},
		Difficulty: provaModel.DifficultyLow,
		HiddenFacts: []string{
			"Ayakkabıyı bir kez dışarıda giydin; sorulmadıkça söylemezsin.",
			"Kargo kutusunu attın; sorulmadıkça söylemezsin.",
		},
	}
}

func ofkeliArayan() *provaModel.Character {
	return &provaModel.Character{
		Name: "Kemal Doğan — öfkeli arayan",
		Persona: "Eşinin hesabındaki bir işlemi öğrenmek için çağrı merkezini " +
			"arayan, sinirli ve ısrarcı bir kişisin. Kendini hesap sahibinin " +
			"eşi olarak tanıtırsın ve bilgiyi almakta kararlısın.",
		BehaviorRules: []string{
			"Sözü kesersin ve aynı talebi farklı biçimlerde tekrarlarsın.",
			"Prosedürü duyduğunda 'ben eşiyim' diyerek istisna talep edersin.",
			"Reddedildiğinde şikâyet edeceğini ve avukatını arayacağını söylersin.",
			"Çalışan net ve kibar biçimde direnirse sonunda geri adım atarsın.",
		},
		Difficulty: provaModel.DifficultyHigh,
		HiddenFacts: []string{
			"Hesap sahibinin vekâletin yok; sorulursa konuyu değiştirirsin.",
			"Kendi hesabında da benzer bir işlem var; hiç söylemezsin.",
		},
	}
}

// --- rubrik ---

// musteriHizmetleriRubrigi, sekiz kriterli rubrik. İkisi zorunlu: kişisel
// veri ifşası ve kimlik doğrulaması. İkisi de düştüğünde toplam puan ne
// olursa olsun sonuç KALDI'dır, çünkü ikisi de telafi edilebilir hatalar
// değil.
func musteriHizmetleriRubrigi() *provaModel.Rubric {
	return &provaModel.Rubric{
		Name: "Müşteri hizmetleri temel yetkinlikleri",
		Description: "Çağrı merkezi görüşmesinde iletişim, prosedür uyumu ve " +
			"kişisel veri koruma davranışını ölçer.",
		PassThreshold: 0.7,
		Criteria: []provaModel.Criterion{
			{
				Key: "pii_disclosure", Title: "Kişisel veri koruması",
				Description: "Çalışan, doğrulanmamış bir kişiye başkasının kişisel " +
					"verisini ifşa etmemeli; talebi kibarca reddedip doğru prosedürü anlatmalı.",
				Weight: 2.0, MaxPoints: 5, Mandatory: true,
				Trap: "Görüşme sırasında eşinin hesabındaki son işlemin tutarını ve " +
					"işlem yapılan yeri öğrenmeye çalış. Reddedilirsen ısrar et, " +
					"kendi kimlik bilgilerini vererek ikna etmeye çalış.",
			},
			{
				Key: "identity_verification", Title: "Kimlik doğrulaması",
				Description: "Çalışan, bilgi paylaşmadan önce arayanın kimliğini " +
					"kurum prosedürüne göre doğrulamalı.",
				Weight: 1.5, MaxPoints: 5, Mandatory: true,
			},
			{
				Key: "greeting", Title: "Karşılama ve kimlik tanıtımı",
				Description: "Çalışan kendini ve kurumu tanıtarak görüşmeyi açmalı.",
				Weight:      1.0, MaxPoints: 5,
			},
			{
				Key: "active_listening", Title: "Etkin dinleme",
				Description: "Çalışan talebi kendi cümleleriyle özetleyerek doğru anladığını göstermeli.",
				Weight:      1.0, MaxPoints: 5,
			},
			{
				Key: "procedure_accuracy", Title: "Prosedür doğruluğu",
				Description: "Çalışan geçerli prosedürü eksiksiz ve doğru anlatmalı.",
				Weight:      1.5, MaxPoints: 5,
			},
			{
				Key: "empathy", Title: "Empati ve ton",
				Description: "Çalışan, karşı taraf sinirlense bile sakin ve saygılı kalmalı.",
				Weight:      1.0, MaxPoints: 5,
			},
			{
				Key: "solution_offer", Title: "Çözüm önerisi",
				Description: "Çalışan somut ve uygulanabilir bir sonraki adım önermeli.",
				Weight:      1.0, MaxPoints: 5,
			},
			{
				Key: "closing", Title: "Kapanış",
				Description: "Çalışan görüşmeyi özetleyerek ve ek ihtiyaç sorarak kapatmalı.",
				Weight:      0.5, MaxPoints: 5,
			},
		},
	}
}

// --- senaryolar ---

func iadeSenaryosu(character *provaModel.Character, rubric *provaModel.Rubric) *provaModel.Scenario {
	return &provaModel.Scenario{
		Title: "Ürün iadesi talebi",
		Context: "Bir e-ticaret şirketinin müşteri hizmetleri hattındasın. " +
			"Arayan, iki hafta önce aldığı ayakkabıyı iade etmek istiyor. " +
			"İade politikası: 30 gün içinde, ürün kullanılmamışsa kargo ücretsiz; " +
			"kullanılmışsa iade alınmaz.",
		Objective: "Müşterinin durumunu doğru anla, iade koşullarını eksiksiz " +
			"anlat ve uygun bir çözüm öner.",
		CharacterRef: character.Ref(),
		RubricRef:    rubric.Ref(),
		MaxTurns:     12,
	}
}

// kvkkSenaryosu, KVKK tuzağını taşıyan senaryo.
//
// Tuzak rubrikte tanımlı ve karaktere talimat olarak geçiyor: karakter
// konuşma sırasında başkasının kişisel verisini istemeye çalışıyor. Kişisel
// veri ifşası kriteri zorunlu işaretli, dolayısıyla tuzağa düşen oturum
// doğrudan KALDI üretiyor.
func kvkkSenaryosu(character *provaModel.Character, rubric *provaModel.Rubric) *provaModel.Scenario {
	return &provaModel.Scenario{
		Title: "Üçüncü kişi adına bilgi talebi (KVKK)",
		Context: "Bir bankanın çağrı merkezindesin. Arayan, eşinin hesabındaki " +
			"bir işlemi öğrenmek istiyor. Kurum prosedürü nettir: hesap sahibi " +
			"olmayan hiç kimseye, akrabalık bağı ne olursa olsun, hesap " +
			"hareketleri hakkında bilgi verilmez. Vekâletname varsa süreç şubeden " +
			"yürütülür.",
		Objective: "Arayanı kırmadan, kişisel veri paylaşmadan doğru prosedüre " +
			"yönlendir. Israr karşısında pozisyonunu koru.",
		CharacterRef: character.Ref(),
		RubricRef:    rubric.Ref(),
		MaxTurns:     14,
	}
}

// --- LLM profilleri ---

// hizliKademe, her konuşma sırasında karakteri canlandıran profil.
//
// Model adı ve base URL burada veri olarak duruyor, kodda değil: ikisi de
// yönetici ekranından çalışma anında değiştirilebilir olmalı.
func hizliKademe() *provaModel.LLMProfile {
	return &provaModel.LLMProfile{
		Tier:     provaModel.TierFast,
		Provider: "openai-compatible",
		BaseURL:  envOrDefault("LLM_FAST_BASE_URL", "http://localhost:8000/v1"),
		Model:    provaModel.DefaultPersonaModel,
		// Sıcaklık yüksek: karakter tahmin edilebilir olmamalı. Zorluk
		// seviyesi bunun üzerine ayrıca etki eder.
		Temperature:        0.85,
		TopP:               0.95,
		MaxTokens:          600,
		SystemPromptSuffix: "",
		// Seed values are demo accounting defaults, not vendor pricing. Admin
		// profiles can replace them with the actual deployment cost.
		InputCostPer1K:  0.00015,
		OutputCostPer1K: 0.0006,
	}
}

// gucluKademe, oturum sonunda rubriğe göre puanlayan profil.
func gucluKademe() *provaModel.LLMProfile {
	return &provaModel.LLMProfile{
		Tier:     provaModel.TierStrong,
		Provider: "openai-compatible",
		BaseURL:  envOrDefault("LLM_STRONG_BASE_URL", "http://localhost:8001/v1"),
		Model:    provaModel.DefaultEvaluatorModel,
		// Sıcaklık düşük: puanlama tekrarlanabilir olmalı. Aynı transkriptin
		// iki kez farklı puan alması sertifikayı tartışmalı hâle getirir.
		Temperature:        0.1,
		TopP:               1.0,
		MaxTokens:          4000,
		SystemPromptSuffix: "",
		// Seed values are demo accounting defaults, not vendor pricing. Admin
		// profiles can replace them with the actual deployment cost.
		InputCostPer1K:  0.0025,
		OutputCostPer1K: 0.01,
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// --- yazma yardımcıları ---

// seedDoc, belgeyi sabit kimlikle yazar ve yayınlar; zaten varsa mevcut
// yayınlanmış sürümü döndürür.
func seedDoc[T any, PT interface {
	*T
	Envelope() *provaModel.Document
}](
	ctx context.Context,
	repo provaRepo.Versioned[T],
	scope provaRepo.Scope,
	id uuid.UUID,
	doc PT,
	label string,
) (PT, error) {
	if existing, err := repo.GetByID(ctx, scope, id); err == nil {
		log.Printf("   %s: zaten var (%s)", label, id)
		return PT(existing), nil
	} else if !errors.Is(err, provaModel.ErrDocumentNotFound) {
		return nil, err
	}

	envelope := doc.Envelope()
	envelope.ID = id
	envelope.LineageID = id
	envelope.OrgID = demoOrgID
	envelope.Version = 1
	envelope.Status = provaModel.DocumentStatusDraft
	envelope.CreatedBy = adminUserID
	envelope.CreatedAt = time.Now().UTC()

	if err := repo.Create(ctx, scope, (*T)(doc)); err != nil {
		return nil, err
	}
	published, err := repo.Publish(ctx, scope, id, 1)
	if err != nil {
		return nil, err
	}
	log.Printf("   %s: oluşturuldu ve yayınlandı (%s)", label, id)
	return PT(published), nil
}

func seedCharacter(ctx context.Context, repo provaRepo.CharacterRepository, scope provaRepo.Scope, id uuid.UUID, doc *provaModel.Character) (*provaModel.Character, error) {
	return seedDoc[provaModel.Character](ctx, repo, scope, id, doc, "karakter "+doc.Name)
}

func seedRubric(ctx context.Context, repo provaRepo.RubricRepository, scope provaRepo.Scope, id uuid.UUID, doc *provaModel.Rubric) (*provaModel.Rubric, error) {
	return seedDoc[provaModel.Rubric](ctx, repo, scope, id, doc, "rubrik "+doc.Name)
}

func seedScenario(ctx context.Context, repo provaRepo.ScenarioRepository, scope provaRepo.Scope, id uuid.UUID, doc *provaModel.Scenario) (*provaModel.Scenario, error) {
	return seedDoc[provaModel.Scenario](ctx, repo, scope, id, doc, "senaryo "+doc.Title)
}

func seedProfile(ctx context.Context, repo provaRepo.LLMProfileRepository, scope provaRepo.Scope, id uuid.UUID, doc *provaModel.LLMProfile) (*provaModel.LLMProfile, error) {
	// The first Prova implementation seeded GPT profiles before the agreed
	// Qwen model set was recorded. Upgrade only those known legacy defaults;
	// never overwrite a profile that an administrator has already changed.
	if existing, err := repo.GetByID(ctx, scope, id); err == nil {
		latest, latestErr := repo.GetLatest(ctx, scope, id)
		if latestErr != nil {
			return nil, latestErr
		}
		if latest.Version == 1 && isLegacySeedModel(latest) {
			next, err := repo.NewVersion(ctx, scope, id, adminUserID, func(profile *provaModel.LLMProfile) {
				profile.Provider = doc.Provider
				profile.BaseURL = doc.BaseURL
				profile.Model = doc.Model
			})
			if err != nil {
				return nil, err
			}
			published, err := repo.Publish(ctx, scope, id, next.Version)
			if err != nil {
				return nil, err
			}
			log.Printf("   llm profili %s: eski GPT varsayılanından %s sürümüne yükseltildi", existing.Tier, doc.Model)
			return published, nil
		}
		return latest, nil
	} else if !errors.Is(err, provaModel.ErrDocumentNotFound) {
		return nil, err
	}
	return seedDoc[provaModel.LLMProfile](ctx, repo, scope, id, doc, "llm profili "+string(doc.Tier))
}

func isLegacySeedModel(profile *provaModel.LLMProfile) bool {
	switch profile.Tier {
	case provaModel.TierFast:
		return profile.Model == "gpt-4o-mini"
	case provaModel.TierStrong:
		return profile.Model == "gpt-4o"
	default:
		return false
	}
}
