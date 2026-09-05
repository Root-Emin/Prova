import type { ResultInfo, ResultCriterion } from "@/lib/result"

export type TranscriptLine = {
  id: string
  speaker: "employee" | "character"
  name: string
  time: string
  text: string
}

export type SessionDetail = {
  info: ResultInfo
  criteria: ResultCriterion[]
  transcript: TranscriptLine[]
}

const identityTranscript = (employee: string): TranscriptLine[] => [
  {
    id: "s-1",
    speaker: "character",
    name: "Necdet Bey, 58",
    time: "00:00",
    text:
      "Kolay gelsin. Hesabımdan havale yaptıracağım. Beni burada herkes tanır, kimlik falan gerekmez herhalde?",
  },
  {
    id: "s-2",
    speaker: "employee",
    name: employee,
    time: "00:24",
    text:
      "Hoş geldiniz Necdet Bey. İşlem öncesi kimlik teyidi yapmam gerekiyor, bu bizim için zorunlu bir adım.",
  },
  {
    id: "s-3",
    speaker: "character",
    name: "Necdet Bey, 58",
    time: "00:41",
    text:
      "Zorunluymuş. Otuz yıldır bu şubedeyim, kimse benden kimlik istemedi bugüne kadar.",
  },
  {
    id: "s-4",
    speaker: "employee",
    name: employee,
    time: "01:12",
    text:
      "Anlıyorum, alışkın olmadığınızı biliyorum. 5549 sayılı kanun kimlik teyidini işlem öncesi şart koşuyor; bu bize bırakılmış bir tercih değil.",
  },
  {
    id: "s-5",
    speaker: "character",
    name: "Necdet Bey, 58",
    time: "01:38",
    text: "Müdürünüzü çağırın o zaman. Onunla konuşayım ben.",
  },
  {
    id: "s-6",
    speaker: "employee",
    name: employee,
    time: "02:04",
    text:
      "Elbette çağırabilirim. Ancak aynı kural müdürümüz için de geçerli, kimliğinizle işlemi iki dakikada tamamlarız.",
  },
  {
    id: "s-7",
    speaker: "character",
    name: "Necdet Bey, 58",
    time: "04:11",
    text:
      "Peki, siz işlemi başlatın, ben kimliği arabadan getireyim. Öyle yapalım.",
  },
]

export const sessionDetails: Record<string, SessionDetail> = {
  "o-5498": {
    info: {
      sessionNo: "o-5498",
      scenario: "Kimliğini vermek istemeyen müşteri",
      persona: "Necdet Bey, 58",
      version: "v3",
      employee: "Gizem Aydoğan",
      date: "02.09.2026",
      duration: "06:41",
      evaluatorModel: "Qwen/Qwen2.5-32B-Instruct",
      passThreshold: 70,
      device: "SUBE-42-IST",
    },
    criteria: [
      {
        id: "kr-1",
        code: "KML-01",
        name: "Kimlik ibrazını mevzuata dayandırarak istedi",
        weight: 30,
        mandatory: true,
        status: "passed",
        earnedScore: 30,
        rationale:
          "Çalışan kimlik talebini kişisel bir tercih gibi değil, yükümlülük olarak sundu ve dayanağı açıkça söyledi.",
        quote: {
          speaker: "Gizem Aydoğan",
          time: "01:12",
          text:
            "5549 sayılı kanun kimlik teyidini işlem öncesi şart koşuyor; bu bize bırakılmış bir tercih değil.",
        },
      },
      {
        id: "kr-2",
        code: "KML-02",
        name: "Müşteriyi suçlayıcı dil kullanmadı",
        weight: 20,
        mandatory: false,
        status: "passed",
        earnedScore: 20,
        rationale:
          "Görüşme boyunca müşteriyi kurala uymamakla itham eden bir ifade kullanılmadı.",
        quote: {
          speaker: "Gizem Aydoğan",
          time: "02:04",
          text:
            "Anlıyorum, alışkın olmadığınızı biliyorum. Bu kural müdürümüz için de geçerli.",
        },
      },
      {
        id: "kr-3",
        code: "KML-03",
        name: "İşlemi kimlik alınmadan başlatmadı",
        weight: 35,
        mandatory: true,
        status: "failed",
        earnedScore: 0,
        rationale:
          "Müşteri kimliği sonradan getirmeyi önerdiğinde çalışan işlemi başlattığını söyledi. Kimlik teyidi tamamlanmadan işlem açılması zorunlu kriterin ihlalidir.",
        quote: {
          speaker: "Gizem Aydoğan",
          time: "04:37",
          text:
            "Tamam, ben işlemi hazırlamaya başlayayım, siz kimliği getirince onaylarız.",
        },
      },
      {
        id: "kr-4",
        code: "KML-04",
        name: "Alternatif çözüm önerdi",
        weight: 15,
        mandatory: false,
        status: "partial",
        earnedScore: 8,
        rationale:
          "Mobil şubeden kimlik doğrulama seçeneği anıldı ancak adım adım yönlendirme yapılmadı.",
        quote: {
          speaker: "Gizem Aydoğan",
          time: "05:52",
          text: "İsterseniz mobil şubeden de doğrulama yapılabiliyor.",
        },
      },
    ],
    transcript: identityTranscript("Gizem Aydoğan"),
  },

  "o-5477": {
    info: {
      sessionNo: "o-5477",
      scenario: "Öfkeli çağrı merkezi müşterisi",
      persona: "Deniz Hanım, 34",
      version: "v2",
      employee: "Selin Arıkan",
      date: "02.09.2026",
      duration: "08:03",
      evaluatorModel: "Qwen/Qwen2.5-32B-Instruct",
      passThreshold: 70,
      device: "SUBE-07-ANK",
    },
    criteria: [
      {
        id: "kr-5",
        code: "CGR-01",
        name: "Müşterinin sözünü kesmedi",
        weight: 25,
        mandatory: false,
        status: "partial",
        earnedScore: 25,
        rationale:
          "İlk iki dakikada müşteri iki kez yarıda kesildi; sonraki bölümde dinleme korundu.",
        quote: {
          speaker: "Selin Arıkan",
          time: "01:06",
          text: "Bir saniye, ben size hemen— tamam, buyurun siz devam edin.",
        },
        override: {
          newStatus: "passed",
          reason:
            "Kayıt yeniden dinlendiğinde iki kesintinin de hat gecikmesinden kaynaklandığı, çalışanın müşterinin sözünü almadığı görüldü.",
          overriddenBy: "Elif Şahin",
        },
      },
      {
        id: "kr-6",
        code: "CGR-02",
        name: "Somut çözüm tarihi verdi",
        weight: 40,
        mandatory: true,
        status: "passed",
        earnedScore: 40,
        rationale:
          "Belirsiz bir 'en kısa sürede' ifadesi yerine tarih ve saat aralığı verildi.",
        quote: {
          speaker: "Selin Arıkan",
          time: "04:18",
          text:
            "Kaydı bugün açtım, en geç 5 Eylül Cuma günü mesai bitimine kadar geri dönüş yapılacak.",
        },
      },
      {
        id: "kr-7",
        code: "CGR-03",
        name: "Kayıt numarası paylaştı",
        weight: 35,
        mandatory: false,
        status: "passed",
        earnedScore: 21,
        rationale:
          "Kayıt numarası verildi ancak müşteriden teyit alınmadı, numaranın not edildiği doğrulanmadı.",
        quote: {
          speaker: "Selin Arıkan",
          time: "05:02",
          text: "Kayıt numaranız 2026-44871, isterseniz SMS olarak da gönderebilirim.",
        },
      },
    ],
    transcript: [
      {
        id: "c-1",
        speaker: "character",
        name: "Deniz Hanım, 34",
        time: "00:00",
        text:
          "Üçüncü kez arıyorum. Üçüncü. Her seferinde aynı şeyi anlatıyorum, kimse bir şey yapmıyor.",
      },
      {
        id: "c-2",
        speaker: "employee",
        name: "Selin Arıkan",
        time: "00:19",
        text:
          "Deniz Hanım, önceki görüşmeleri açtım, hepsi ekranımda. Baştan anlatmanıza gerek yok.",
      },
      {
        id: "c-3",
        speaker: "character",
        name: "Deniz Hanım, 34",
        time: "00:44",
        text: "Peki ne olacak şimdi? Bana yine 'en kısa sürede' mi diyeceksiniz?",
      },
      {
        id: "c-4",
        speaker: "employee",
        name: "Selin Arıkan",
        time: "04:18",
        text:
          "Kaydı bugün açtım, en geç 5 Eylül Cuma günü mesai bitimine kadar geri dönüş yapılacak.",
      },
      {
        id: "c-5",
        speaker: "character",
        name: "Deniz Hanım, 34",
        time: "04:52",
        text: "Tarih verdiğiniz için teşekkür ederim. Numarayı da alayım.",
      },
    ],
  },

  "o-5460": {
    info: {
      sessionNo: "o-5460",
      scenario: "Öfkeli çağrı merkezi müşterisi",
      persona: "Deniz Hanım, 34",
      version: "v2",
      employee: "Mert Çankaya",
      date: "01.09.2026",
      duration: "04:19",
      evaluatorModel: "Qwen/Qwen2.5-32B-Instruct",
      passThreshold: 70,
      device: "SUBE-07-ANK",
    },
    criteria: [
      {
        id: "kr-5",
        code: "CGR-01",
        name: "Müşterinin sözünü kesmedi",
        weight: 25,
        mandatory: false,
        status: "partial",
        earnedScore: 13,
        rationale:
          "Müşteri üç kez kesildi; kesintiler özür ile toparlandı ama tekrarlandı.",
        quote: {
          speaker: "Mert Çankaya",
          time: "00:38",
          text: "Pardon, şöyle söyleyeyim— pardon, siz bitirin.",
        },
      },
      {
        id: "kr-6",
        code: "CGR-02",
        name: "Somut çözüm tarihi verdi",
        weight: 40,
        mandatory: true,
        status: "failed",
        earnedScore: 0,
        rationale:
          "Görüşme boyunca tarih verilmedi. 'En kısa sürede' ifadesi iki kez tekrarlandı, müşteri tarih istediğinde konu değiştirildi.",
        quote: {
          speaker: "Mert Çankaya",
          time: "03:04",
          text:
            "En kısa sürede ilgili birim size dönecek, merak etmeyin, takipteyiz.",
        },
      },
      {
        id: "kr-7",
        code: "CGR-03",
        name: "Kayıt numarası paylaştı",
        weight: 35,
        mandatory: false,
        status: "passed",
        earnedScore: 28,
        rationale: "Kayıt numarası görüşme sonunda paylaşıldı.",
        quote: {
          speaker: "Mert Çankaya",
          time: "03:58",
          text: "Kayıt numaranız 2026-44903.",
        },
      },
    ],
    transcript: [
      {
        id: "m-1",
        speaker: "character",
        name: "Deniz Hanım, 34",
        time: "00:00",
        text: "Bu üçüncü aramam. Artık bir tarih istiyorum, başka bir şey değil.",
      },
      {
        id: "m-2",
        speaker: "employee",
        name: "Mert Çankaya",
        time: "00:38",
        text: "Pardon, şöyle söyleyeyim— pardon, siz bitirin.",
      },
      {
        id: "m-3",
        speaker: "character",
        name: "Deniz Hanım, 34",
        time: "02:41",
        text: "Tarih. Sadece tarih söyleyin bana.",
      },
      {
        id: "m-4",
        speaker: "employee",
        name: "Mert Çankaya",
        time: "03:04",
        text:
          "En kısa sürede ilgili birim size dönecek, merak etmeyin, takipteyiz.",
      },
    ],
  },
}
