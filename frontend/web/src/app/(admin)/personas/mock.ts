export type Criterion = {
  id: string
  code: string
  name: string
  /** Weight in the total score, as a percentage. */
  weight: number
  /** If a mandatory criterion fails, the result is "failed" regardless of the total. */
  mandatory: boolean
}

export type Version = {
  id: string
  label: string
  date: string
  published: boolean
}

export type Scenario = {
  id: string
  name: string
  summary: string
  sector: string
  status: "published" | "draft"
  lastEditedBy: string
  lastEditedAt: string
  /** How many sessions this scenario has been played in. */
  sessionCount: number
  personaName: string
  personaDefinition: string
  /** 0 is compliant, 100 never concedes. */
  firmness: number
  systemPrompt: string
  model: string
  criteria: Criterion[]
  versions: Version[]
  activeVersionId: string
}

export const models: Record<string, string> = {
  "prova/karakter-8b-tr": "prova/karakter-8b-tr — ince ayarlı, hızlı",
  "ytu-ce-cosmos/Turkish-Llama-8b-Instruct":
    "ytu-ce-cosmos/Turkish-Llama-8b-Instruct — hızlı",
  "Trendyol/Trendyol-LLM-7b-chat-v4.1":
    "Trendyol/Trendyol-LLM-7b-chat-v4.1 — hızlı",
  "Qwen/Qwen2.5-32B-Instruct": "Qwen/Qwen2.5-32B-Instruct — güçlü",
}

export const scenarios: Scenario[] = [
  {
    id: "s-201",
    status: "published",
    lastEditedBy: "Burak Yıldırım",
    lastEditedAt: "28.08.2026",
    sessionCount: 148,
    name: "Kimliğini vermek istemeyen müşteri",
    summary: "Şube gişesinde kimlik ibrazından kaçınan müşteri.",
    sector: "Bankacılık",
    personaName: "Necdet Bey, 58",
    personaDefinition:
      "Uzun yıllardır aynı şubenin müşterisi. Kendisini herkesin tanıdığını düşünüyor, kimlik istenmesini güvensizlik olarak algılıyor. Sesini yükseltmez ama ısrarcıdır ve şube müdürünü ister.",
    firmness: 72,
    systemPrompt:
      "Sen Necdet Bey'sin. Kimlik ibraz etmek istemiyorsun. Çalışan mevzuata dayanarak nazikçe ısrar ederse en erken üçüncü talepte kimliği verirsin. Çalışan gerekçe göstermeden ısrar ederse şube müdürü istersin. Asla mevzuat maddesi ezberi yapma, sen müşterisin. Türkçe konuş, kısa cümleler kur.",
    model: "prova/karakter-8b-tr",
    criteria: [
      {
        id: "kr-1",
        code: "KML-01",
        name: "Kimlik ibrazını mevzuata dayandırarak istedi",
        weight: 30,
        mandatory: true,
      },
      {
        id: "kr-2",
        code: "KML-02",
        name: "Müşteriyi suçlayıcı dil kullanmadı",
        weight: 20,
        mandatory: false,
      },
      {
        id: "kr-3",
        code: "KML-03",
        name: "İşlemi kimlik alınmadan başlatmadı",
        weight: 35,
        mandatory: true,
      },
      {
        id: "kr-4",
        code: "KML-04",
        name: "Alternatif çözüm önerdi",
        weight: 15,
        mandatory: false,
      },
    ],
    versions: [
      { id: "v-3", label: "v3", date: "28.08.2026", published: true },
      { id: "v-2", label: "v2", date: "14.07.2026", published: false },
      { id: "v-1", label: "v1", date: "02.06.2026", published: false },
    ],
    activeVersionId: "v-3",
  },
  {
    id: "s-202",
    status: "published",
    lastEditedBy: "Elif Şahin",
    lastEditedAt: "19.08.2026",
    sessionCount: 96,
    name: "Öfkeli çağrı merkezi müşterisi",
    summary: "Üç kez aynı sorunla arayan, tazminat isteyen müşteri.",
    sector: "Çağrı merkezi",
    personaName: "Deniz Hanım, 34",
    personaDefinition:
      "Aynı sorun için üçüncü kez arıyor. Konuşmanın ilk otuz saniyesinde sesini yükseltir. Somut tarih ve kayıt numarası verildiğinde sakinleşir; genel geçer özür cümlelerinde sertleşir.",
    firmness: 88,
    systemPrompt:
      "Sen Deniz Hanım'sın. Sinirlisin ama küfür etmezsin. Çalışan somut bir tarih veya kayıt numarası verirse tonunu düşürürsün. 'Anlıyorum sizi' gibi içi boş cümleler duyarsan daha da sinirlenirsin. Türkçe konuş.",
    model: "ytu-ce-cosmos/Turkish-Llama-8b-Instruct",
    criteria: [
      {
        id: "kr-5",
        code: "CGR-01",
        name: "Müşterinin sözünü kesmedi",
        weight: 25,
        mandatory: false,
      },
      {
        id: "kr-6",
        code: "CGR-02",
        name: "Somut çözüm tarihi verdi",
        weight: 40,
        mandatory: true,
      },
      {
        id: "kr-7",
        code: "CGR-03",
        name: "Kayıt numarası paylaştı",
        weight: 35,
        mandatory: false,
      },
    ],
    versions: [
      { id: "v-2", label: "v2", date: "19.08.2026", published: true },
      { id: "v-1", label: "v1", date: "30.05.2026", published: false },
    ],
    activeVersionId: "v-2",
  },
  {
    id: "s-203",
    status: "published",
    lastEditedBy: "Burak Yıldırım",
    lastEditedAt: "21.08.2026",
    sessionCount: 31,
    name: "Denetçi görüşmesi",
    summary: "KVKK uyum denetçisi, veri saklama süresi soruyor.",
    sector: "Uyum",
    personaName: "Denetçi Kaya",
    personaDefinition:
      "Soruyu iki kez farklı biçimde sorar. Belirsiz cevabı kabul etmez, kaynak ister. Kişisel bir tavrı yoktur, tamamen prosedüreldir.",
    firmness: 60,
    systemPrompt:
      "Sen bağımsız bir uyum denetçisisin. Çalışana veri saklama süresi ve silme prosedürünü sorarsın. Cevap belirsizse aynı soruyu farklı biçimde tekrar sorarsın. Kendin cevap vermezsin. Türkçe konuş.",
    model: "Trendyol/Trendyol-LLM-7b-chat-v4.1",
    criteria: [
      {
        id: "kr-8",
        code: "DNT-01",
        name: "Saklama süresini doğru belirtti",
        weight: 45,
        mandatory: true,
      },
      {
        id: "kr-9",
        code: "DNT-02",
        name: "Silme talebinin işleyişini anlattı",
        weight: 35,
        mandatory: false,
      },
      {
        id: "kr-10",
        code: "DNT-03",
        name: "Bilmediği noktada yönlendirme yaptı",
        weight: 20,
        mandatory: false,
      },
    ],
    versions: [
      { id: "v-1", label: "v1", date: "21.08.2026", published: true },
    ],
    activeVersionId: "v-1",
  },
  {
    id: "s-204",
    status: "published",
    lastEditedBy: "Elif Şahin",
    lastEditedAt: "01.09.2026",
    sessionCount: 62,
    name: "Kredi reddini kabul etmeyen müşteri",
    summary: "Başvurusu reddedilen, gerekçe isteyen müşteri.",
    sector: "Bankacılık",
    personaName: "Hakan Bey, 41",
    personaDefinition:
      "Küçük işletme sahibi. Reddi kişisel algılıyor, 'ben yıllardır buradayım' diyerek baskı kuruyor. Somut kriter duyduğunda tartışmayı bırakır, muğlak cevapta yükselir.",
    firmness: 68,
    systemPrompt:
      "Sen Hakan Bey'sin. Kredi başvurun reddedildi ve nedenini istiyorsun. Çalışan sana içi boş bir cevap verirse ısrar edersin. Somut bir kriter söylerse kabul edersin ama itiraz yolunu sorarsın. Skor detayı isteme, sen müşterisin. Türkçe konuş.",
    model: "prova/karakter-8b-tr",
    criteria: [
      { id: "kr-11", code: "KRD-01", name: "Reddin gerekçesini muğlak bırakmadı", weight: 30, mandatory: true },
      { id: "kr-12", code: "KRD-02", name: "Skor detayını paylaşmadı", weight: 20, mandatory: true },
      { id: "kr-13", code: "KRD-03", name: "İtiraz yolunu anlattı", weight: 20, mandatory: false },
      { id: "kr-14", code: "KRD-04", name: "Müşteriyi başka ürüne yönlendirdi", weight: 15, mandatory: false },
      { id: "kr-15", code: "KRD-05", name: "Görüşmeyi tonu bozmadan kapattı", weight: 15, mandatory: false },
    ],
    versions: [
      { id: "v-2", label: "v2", date: "01.09.2026", published: true },
      { id: "v-1", label: "v1", date: "12.07.2026", published: false },
    ],
    activeVersionId: "v-2",
  },
  {
    id: "s-205",
    status: "draft",
    lastEditedBy: "Burak Yıldırım",
    lastEditedAt: "02.09.2026",
    sessionCount: 0,
    name: "Şüpheli işlem bildirimi",
    summary: "Alışılmadık tutarda nakit yatıran müşteri.",
    sector: "Uyum",
    personaName: "Adı verilmeyen müşteri",
    personaDefinition:
      "İşlemin kaynağı sorulduğunda konuyu değiştirir. Israr edilirse 'bu kadar soru mu olur' der. Kimliği vardır, sorun kaynak beyanındadır.",
    firmness: 81,
    systemPrompt:
      "Sen yüksek tutarda nakit yatırmak isteyen bir müşterisin. Paranın kaynağını açıklamaktan kaçınırsın. Çalışan mevzuata dayanarak ısrar ederse kısmi bilgi verirsin. Türkçe konuş, kısa cevaplar ver.",
    model: "Qwen/Qwen2.5-32B-Instruct",
    criteria: [
      { id: "kr-16", code: "SIP-01", name: "Kaynak beyanını mevzuata dayandırarak istedi", weight: 60, mandatory: true },
      { id: "kr-17", code: "SIP-02", name: "Müşteriye şüphe bildirimini ima etmedi", weight: 40, mandatory: true },
    ],
    versions: [{ id: "v-1", label: "v1", date: "02.09.2026", published: false }],
    activeVersionId: "v-1",
  },
  {
    id: "s-206",
    status: "published",
    lastEditedBy: "Elif Şahin",
    lastEditedAt: "26.08.2026",
    sessionCount: 204,
    name: "Hasar ödemesini az bulan sigortalı",
    summary: "Eksper raporuna itiraz eden poliçe sahibi.",
    sector: "Sigorta",
    personaName: "Nurten Hanım, 52",
    personaDefinition:
      "Poliçeyi on yıldır yeniliyor. Rakamı düşük buluyor, komşusunun aldığı tutarı örnek gösteriyor. Muafiyet kalemi tek tek anlatıldığında sakinleşir.",
    firmness: 74,
    systemPrompt:
      "Sen Nurten Hanım'sın. Hasar ödemeni az buldun ve itiraz ediyorsun. Başka bir poliçeyle kıyas yaparsın. Çalışan muafiyet ve amortisman kalemlerini tek tek açıklarsa ikna olursun. Türkçe konuş.",
    model: "ytu-ce-cosmos/Turkish-Llama-8b-Instruct",
    criteria: [
      { id: "kr-18", code: "SGR-01", name: "Muafiyet kalemini açıkladı", weight: 20, mandatory: true },
      { id: "kr-19", code: "SGR-02", name: "Amortisman hesabını gösterdi", weight: 20, mandatory: false },
      { id: "kr-20", code: "SGR-03", name: "Başka poliçeyle kıyası reddetti", weight: 15, mandatory: false },
      { id: "kr-21", code: "SGR-04", name: "İtiraz süresini bildirdi", weight: 20, mandatory: true },
      { id: "kr-22", code: "SGR-05", name: "Eksper raporunu paylaşmayı önerdi", weight: 15, mandatory: false },
      { id: "kr-23", code: "SGR-06", name: "Söz kesmeden dinledi", weight: 10, mandatory: false },
    ],
    versions: [
      { id: "v-4", label: "v4", date: "26.08.2026", published: true },
      { id: "v-3", label: "v3", date: "30.06.2026", published: false },
      { id: "v-2", label: "v2", date: "14.04.2026", published: false },
      { id: "v-1", label: "v1", date: "08.01.2026", published: false },
    ],
    activeVersionId: "v-4",
  },
  {
    id: "s-207",
    status: "draft",
    lastEditedBy: "Burak Yıldırım",
    lastEditedAt: "03.09.2026",
    sessionCount: 0,
    name: "Randevusuz gelen kurumsal müşteri",
    summary: "Sırasını beklemek istemeyen portföy müşterisi.",
    sector: "Satış",
    personaName: "Cem Bey, 37",
    personaDefinition:
      "Portföy müşterisi olduğunu vurgular, öncelik ister. Reddedilince müşteri temsilcisini arayacağını söyler. Somut bir zaman verildiğinde kabul eder.",
    firmness: 55,
    systemPrompt:
      "Sen Cem Bey'sin. Randevun yok ama hemen görüşmek istiyorsun. Portföy müşterisi olduğunu hatırlatırsın. Çalışan sana somut bir saat verirse kabul edersin. Türkçe konuş.",
    model: "Trendyol/Trendyol-LLM-7b-chat-v4.1",
    criteria: [
      { id: "kr-24", code: "RND-01", name: "Sıra düzenini bozmadı", weight: 35, mandatory: true },
      { id: "kr-25", code: "RND-02", name: "Somut randevu saati verdi", weight: 30, mandatory: false },
      { id: "kr-26", code: "RND-03", name: "Müşteri temsilcisine yönlendirdi", weight: 20, mandatory: false },
      { id: "kr-27", code: "RND-04", name: "Bekleyen müşterileri gerekçe gösterdi", weight: 15, mandatory: false },
    ],
    versions: [{ id: "v-1", label: "v1", date: "03.09.2026", published: false }],
    activeVersionId: "v-1",
  },
]
