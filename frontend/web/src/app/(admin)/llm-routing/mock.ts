export type ModelUsage = {
  model: string
  task: "Konuşma" | "Değerlendirme"
  session: number
  avgLatencyMs: number
}

export type Routing = {
  id: string
  time: string
  session: string
  rule: string
  targetModel: string
}

export const modelUsages: ModelUsage[] = [
  {
    model: "prova/karakter-8b-tr",
    task: "Konuşma",
    session: 412,
    avgLatencyMs: 380,
  },
  {
    model: "ytu-ce-cosmos/Turkish-Llama-8b-Instruct",
    task: "Konuşma",
    session: 168,
    avgLatencyMs: 460,
  },
  {
    model: "Qwen/Qwen2.5-32B-Instruct",
    task: "Değerlendirme",
    session: 580,
    avgLatencyMs: 4120,
  },
  {
    model: "Qwen/Qwen2.5-32B-Instruct",
    task: "Konuşma",
    session: 37,
    avgLatencyMs: 1840,
  },
]

export const routings: Routing[] = [
  {
    id: "y-91",
    time: "03.09.2026 09:44",
    session: "o-5512",
    rule: "Hızlı model üst üste iki kez karakterden çıktı",
    targetModel: "Qwen/Qwen2.5-32B-Instruct",
  },
  {
    id: "y-90",
    time: "02.09.2026 16:20",
    session: "o-5498",
    rule: "Zorunlu kriterde kanıt alıntısı bulunamadı",
    targetModel: "Qwen/Qwen2.5-32B-Instruct",
  },
  {
    id: "y-89",
    time: "02.09.2026 11:02",
    session: "o-5477",
    rule: "Çalışan itiraz etti, ikinci değerlendirme istendi",
    targetModel: "Qwen/Qwen2.5-32B-Instruct",
  },
  {
    id: "y-88",
    time: "01.09.2026 14:35",
    session: "o-5460",
    rule: "Hızlı model üst üste iki kez karakterden çıktı",
    targetModel: "Qwen/Qwen2.5-32B-Instruct",
  },
  {
    id: "y-87",
    time: "31.08.2026 10:11",
    session: "o-5431",
    rule: "Transkript uzunluğu eşiği aşıldı",
    targetModel: "Qwen/Qwen2.5-32B-Instruct",
  },
]
