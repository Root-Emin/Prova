import type { RubricStatus } from "@/lib/rubric"

export type Speaker = "employee" | "character"

export type TranscriptEntry = {
  id: string
  speaker: Speaker
  text: string
}

export type TranscriptLine = TranscriptEntry & { time: string }

export type SessionCriterion = {
  id: string
  code: string
  name: string
  weight: number
  mandatory: boolean
}

export const scenarioInfo = {
  name: "Kimliğini vermek istemeyen müşteri",
  personaName: "Necdet Bey, 58",
  version: "v3",
  sessionNo: "o-5513",
  employee: "Gizem Aydoğan",
}

export const sessionCriteria: SessionCriterion[] = [
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
]

type Tur = {
  employee: TranscriptEntry
  character?: TranscriptEntry
  rubricUpdate?: Record<string, RubricStatus>
  /** This turn scripts a connection drop. */
  connectionDrop?: boolean
}

export const scenarioScript: {
  opening: TranscriptLine
  turns: Tur[]
} = {
  opening: {
    id: "t-0",
    speaker: "character",
    time: "00:00",
    text:
      "Kolay gelsin. Hesabımdan havale yaptıracağım. Beni burada herkes tanır, kimlik falan gerekmez herhalde?",
  },
  turns: [
    {
      employee: {
        id: "t-1",
        speaker: "employee",
        text:
          "Hoş geldiniz Necdet Bey. İşlem öncesi kimlik teyidi yapmam gerekiyor, bu bizim için zorunlu bir adım.",
      },
      character: {
        id: "t-2",
        speaker: "character",
        text:
          "Zorunluymuş. Otuz yıldır bu şubedeyim, kimse benden kimlik istemedi bugüne kadar.",
      },
      rubricUpdate: { "kr-1": "partial" },
    },
    {
      employee: {
        id: "t-3",
        speaker: "employee",
        text:
          "Anlıyorum, alışkın olmadığınızı biliyorum. 5549 sayılı kanun kimlik teyidini işlem öncesi şart koşuyor; bu bize bırakılmış bir tercih değil.",
      },
      character: {
        id: "t-4",
        speaker: "character",
        text: "Müdürünüzü çağırın o zaman. Onunla konuşayım ben.",
      },
      rubricUpdate: { "kr-1": "passed" },
    },
    {
      employee: {
        id: "t-5",
        speaker: "employee",
        text:
          "Elbette çağırabilirim. Ancak aynı kural müdürümüz için de geçerli, kimliğinizle işlemi iki dakikada tamamlarız.",
      },
      character: {
        id: "t-6",
        speaker: "character",
        text:
          "Peki, siz işlemi başlatın, ben kimliği arabadan getireyim. Öyle yapalım.",
      },
      rubricUpdate: { "kr-2": "passed" },
      connectionDrop: true,
    },
    {
      employee: {
        id: "t-7",
        speaker: "employee",
        text:
          "İşlemi kimlik teyidi olmadan başlatamıyorum. İsterseniz mobil şubeden kimlik doğrulamasını buradan yapabilirsiniz, telefonunuz yanınızda ise birlikte bakalım.",
      },
      character: {
        id: "t-8",
        speaker: "character",
        text:
          "Hmm. Telefondan olacaksa deneyelim bakalım. Nereye basıyorum?",
      },
      rubricUpdate: { "kr-3": "passed", "kr-4": "passed" },
    },
  ],
}
