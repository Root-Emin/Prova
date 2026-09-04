import type { ResultInfo, ResultCriterion } from "@/lib/result"

export const resultInfo: ResultInfo = {
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
}

export const resultCriteria: ResultCriterion[] = [
  {
    id: "kr-1",
    code: "KML-01",
    name: "Kimlik ibrazını mevzuata dayandırarak istedi",
    weight: 30,
    mandatory: true,
    status: "passed",
    earnedScore: 30,
    rationale:
      "Çalışan kimlik talebini kişisel bir tercih gibi değil, yükümlülük olarak sundu ve dayanağı açıkça söyledi. Müşterinin itirazı sonrası da aynı çerçeveyi korudu.",
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
      "Görüşme boyunca müşteriyi kurala uymamakla itham eden bir ifade kullanılmadı; itiraz karşısında ton korundu.",
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
      "Mobil şubeden kimlik doğrulama seçeneği anıldı ancak adım adım yönlendirme yapılmadı, müşteri seçeneği kullanmadan görüşme kapandı.",
    quote: {
      speaker: "Gizem Aydoğan",
      time: "05:52",
      text: "İsterseniz mobil şubeden de doğrulama yapılabiliyor.",
    },
  },
]
