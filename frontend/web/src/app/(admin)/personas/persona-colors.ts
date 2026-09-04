import type { StaticImageData } from "next/image"

import blueChar from "@/assets/illustrations/blue_char.png"
import greenChar from "@/assets/illustrations/green_char.png"
import redChar from "@/assets/illustrations/red_char.png"
import yellowChar from "@/assets/illustrations/yellow_char.png"

export type PersonaColor = "red" | "yellow" | "green" | "blue"

/** Kart sırası DISC sırasıdır: baskın, etkileyici, uyumlu, analitik. */
export const personaColorOrder: PersonaColor[] = ["red", "yellow", "green", "blue"]

export type PersonaColorProfile = {
  id: PersonaColor
  /** Rengin kendi adı — kartın üst satırında geçer. */
  label: string
  /** Kişilik tipinin adı. */
  title: string
  /** Tek cümlelik özet, başlığın altında. */
  summary: string
  /** Kartın altındaki renk özellikleri. */
  traits: string[]
  /** Görüşmede çalışanın karşılaştığı zorluk. */
  challenge: string
  /** Personanın çözüldüğü an. */
  opening: string
  image: StaticImageData
}

export const personaColors: Record<PersonaColor, PersonaColorProfile> = {
  red: {
    id: "red",
    label: "Kırmızı",
    title: "Baskın",
    summary: "Sonuca gider, gerekçeyi sonra dinler.",
    traits: [
      "Doğrudan konuşur, sözü uzatmaz",
      "Kontrolü elinde tutmak ister",
      "İtiraz görünce tonunu yükseltir",
      "Oyalandığını hissederse yetkili ister",
    ],
    challenge: "Çalışanı savunmaya iter, süreyi baskı aracı olarak kullanır.",
    opening: "Somut karar ve net tarih duyduğunda hızla yumuşar.",
    image: redChar,
  },
  yellow: {
    id: "yellow",
    label: "Sarı",
    title: "Etkileyici",
    summary: "İlişki üzerinden ilerler, ayrıcalık bekler.",
    traits: [
      "Sohbeti açar, samimiyet kurar",
      "Israrı güler yüzlüdür ama bırakmaz",
      "Detaydan sıkılır, konuyu dağıtır",
      "Kural yerine kişisel jest arar",
    ],
    challenge: "Samimiyet çalışanın kuralı esnetmesini kolaylaştırır.",
    opening: "Sıcak tonla verilen net bir sınırı sorunsuz kabul eder.",
    image: yellowChar,
  },
  green: {
    id: "green",
    label: "Yeşil",
    title: "Uyumlu",
    summary: "Sesini yükseltmez, alışkanlığa yaslanır.",
    traits: [
      "Sabırlı ama vazgeçmeyen bir ısrarı vardır",
      "Güvene ve geçmişe atıf yapar",
      "Değişimi kişisel algılar",
      "Reddedilince tartışmaz, kırılır",
    ],
    challenge: "Çatışma çıkmadığı için çalışan usulü atlamaya meyleder.",
    opening: "Değişimin gerekçesi anlatılınca uyum sağlar.",
    image: greenChar,
  },
  blue: {
    id: "blue",
    label: "Mavi",
    title: "Analitik",
    summary: "Dayanak ister, belirsiz cevabı kabul etmez.",
    traits: [
      "Kaynak ve mevzuat maddesi sorar",
      "Aynı soruyu farklı biçimde tekrarlar",
      "Duygu değil, prosedür üzerinden ilerler",
      "Çelişkiyi not eder ve geri döner",
    ],
    challenge: "Ezber cevabı anında yakalar, çalışanı kaynağa zorlar.",
    opening: "Doğru maddeye dayandırılan cevapta konuyu kapatır.",
    image: blueChar,
  },
}

/**
 * Tailwind sınıf adları taranarak toplanır; birleştirilerek üretilemez.
 * Bu yüzden her renk için sınıflar tam yazılır.
 */
export const personaColorClass: Record<
  PersonaColor,
  { accent: string; soft: string; ink: string; border: string; hoverBorder: string }
> = {
  red: {
    accent: "bg-persona-red",
    soft: "bg-persona-red-soft",
    ink: "text-persona-red-ink",
    border: "border-persona-red",
    hoverBorder: "hover:border-persona-red",
  },
  yellow: {
    accent: "bg-persona-yellow",
    soft: "bg-persona-yellow-soft",
    ink: "text-persona-yellow-ink",
    border: "border-persona-yellow",
    hoverBorder: "hover:border-persona-yellow",
  },
  green: {
    accent: "bg-persona-green",
    soft: "bg-persona-green-soft",
    ink: "text-persona-green-ink",
    border: "border-persona-green",
    hoverBorder: "hover:border-persona-green",
  },
  blue: {
    accent: "bg-persona-blue",
    soft: "bg-persona-blue-soft",
    ink: "text-persona-blue-ink",
    border: "border-persona-blue",
    hoverBorder: "hover:border-persona-blue",
  },
}
