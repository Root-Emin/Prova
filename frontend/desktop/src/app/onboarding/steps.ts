import {
  BadgeCheck,
  CheckCheck,
  Laptop,
  ListChecks,
  MessagesSquare,
  Mic,
  Play,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react"

export type OnboardingPoint = {
  icon: LucideIcon
  text: string
}

export type OnboardingStep = {
  id: string
  title: string
  /** One line only; it sits in the card description slot. */
  description: string
  points: OnboardingPoint[]
  /** The privacy block appears once, on the screen that talks about scoring. */
  privacy?: boolean
}

export const onboardingSteps: OnboardingStep[] = [
  {
    id: "nedir",
    title: "Prova nedir?",
    description: "Gerçek müşteriden önce yapılan görüşme provası.",
    points: [
      {
        icon: MessagesSquare,
        text: "Karşınızda senaryoya göre davranan bir karakter var; onunla sesli konuşursunuz.",
      },
      {
        icon: ListChecks,
        text: "Görüşme, kurumunuzun belirlediği kriterlere göre ölçülür.",
      },
      {
        icon: BadgeCheck,
        text: "Geçtiğiniz oturum yetkinlik kaydınıza işlenir.",
      },
    ],
  },
  {
    id: "oturum",
    title: "Bir oturum nasıl geçer?",
    description: "Senaryoyu siz başlatır, siz bitirirsiniz.",
    points: [
      {
        icon: Play,
        text: "Oturumlarım ekranından atanan senaryoyu başlatırsınız; tipik bir oturum 6–8 dakika sürer.",
      },
      {
        icon: Mic,
        text: "Konuşmak için space tuşunu basılı tutar, bıraktığınızda sözü karaktere verirsiniz.",
      },
      {
        icon: Laptop,
        text: "Sınav yalnızca kayıtlı cihazdan verilir; girişten sonra bu cihazı kaydedersiniz.",
      },
    ],
  },
  {
    id: "degerlendirme",
    title: "Değerlendirme ve gizlilik",
    description: "Puan kriterlerden gelir, ses cihazınızdan çıkmaz.",
    privacy: true,
    points: [
      {
        icon: CheckCheck,
        text: "Her kriter geçti, kısmi veya kaldı olarak işaretlenir; gerekçe transkriptten alıntılanır.",
      },
      {
        icon: TriangleAlert,
        text: "Zorunlu bir kriter düşerse oturum, toplam puandan bağımsız kaldı sayılır.",
      },
    ],
  },
]
