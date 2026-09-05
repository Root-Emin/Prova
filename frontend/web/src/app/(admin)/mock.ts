export const dashboardSummary = {
  sessionsThisWeek: 12,
  passRate: "%64",
  pendingAssignments: 3,
  certificatesToRenew: 2,
  updatedAt: "03.09.2026 09:44",
}

/**
 * Haftalık oturum hacmi ve o haftanın geçme oranı. Panelin eğilim grafiği
 * bu diziyi okur; son hafta `dashboardSummary.passRate` ile aynı değeri taşır.
 */
export type TrendPoint = {
  /** Haftanın ilk günü, grafik ekseninde göründüğü biçimde. */
  week: string
  sessions: number
  /** Yüzde, 0–100. */
  passRate: number
}

export const weeklyTrend: TrendPoint[] = [
  { week: "15 Haz", sessions: 14, passRate: 47 },
  { week: "22 Haz", sessions: 12, passRate: 44 },
  { week: "29 Haz", sessions: 17, passRate: 50 },
  { week: "06 Tem", sessions: 15, passRate: 48 },
  { week: "13 Tem", sessions: 18, passRate: 52 },
  { week: "20 Tem", sessions: 19, passRate: 55 },
  { week: "27 Tem", sessions: 21, passRate: 57 },
  { week: "03 Ağu", sessions: 23, passRate: 54 },
  { week: "10 Ağu", sessions: 26, passRate: 59 },
  { week: "17 Ağu", sessions: 27, passRate: 62 },
  { week: "24 Ağu", sessions: 28, passRate: 61 },
  { week: "31 Ağu", sessions: 33, passRate: 64 },
]

/** Grafiğin sağ üstündeki aralık seçimi. Değer, gösterilecek hafta sayısıdır. */
export const trendRanges = [
  { value: "8", label: "Son 8 hafta" },
  { value: "12", label: "Son 12 hafta" },
] as const

/**
 * Kimlik zincirinin panelde sayıya dönüşen tek parçası: son 24 saatte
 * gönderilen tek kullanımlık kod. Diğer sayaçlar kullanıcı ve denetim
 * kayıtlarından türetilir, burada tekrarlanmaz.
 */
export const otpSentLastDay = 24
