export type AuditEntry = {
  id: string
  time: string
  actor: string
  action: string
  target: string
  sourceIp: string
}

export const auditEntries: AuditEntry[] = [
  {
    id: "d-7712",
    time: "03.09.2026 09:44:12",
    actor: "burak.yildirim@akbank-egitim.tr",
    action: "Persona sürümü yayınlandı",
    target: "s-201 · v3",
    sourceIp: "10.42.7.19",
  },
  {
    id: "d-7711",
    time: "03.09.2026 09:12:03",
    actor: "elif.sahin@akbank-egitim.tr",
    action: "Puan ezildi",
    target: "o-5512 · KML-03",
    sourceIp: "10.42.7.4",
  },
  {
    id: "d-7710",
    time: "02.09.2026 17:31:55",
    actor: "sistem",
    action: "Değerlendirme güçlü modele yönlendirildi",
    target: "o-5498",
    sourceIp: "—",
  },
  {
    id: "d-7709",
    time: "02.09.2026 16:04:41",
    actor: "ayse.demirtas@akbank-egitim.tr",
    action: "Cihaz kaydedildi",
    target: "c-8805 · AYSE-EV-PC",
    sourceIp: "88.243.11.207",
  },
  {
    id: "d-7708",
    time: "02.09.2026 11:20:08",
    actor: "elif.sahin@akbank-egitim.tr",
    action: "Kullanıcı davet edildi",
    target: "mert.cankaya@akbank-egitim.tr",
    sourceIp: "10.42.7.4",
  },
  {
    id: "d-7707",
    time: "01.09.2026 14:35:22",
    actor: "sistem",
    action: "Ham ses tamponu silindi",
    target: "o-5460",
    sourceIp: "—",
  },
  {
    id: "d-7706",
    time: "01.09.2026 09:02:17",
    actor: "onur.kilicarslan@akbank-egitim.tr",
    action: "Oturum açma reddedildi — kayıtlı cihaz yok",
    target: "k-1046",
    sourceIp: "78.161.44.90",
  },
  {
    id: "d-7705",
    time: "31.08.2026 18:47:36",
    actor: "elif.sahin@akbank-egitim.tr",
    action: "Hesap silme talebi onaylandı",
    target: "k-1039",
    sourceIp: "10.42.7.4",
  },
]
