export type Device = {
  id: string
  user: string
  email: string
  deviceName: string
  os: string
  registeredAt: string
  lastSeen: string
}

export const devices: Device[] = [
  {
    id: "c-8801",
    user: "Ayşe Demirtaş",
    email: "ayse.demirtas@akbank-egitim.tr",
    deviceName: "AYSE-MBP-14",
    os: "macOS 15.4",
    registeredAt: "03.03.2026",
    lastSeen: "03.09.2026 09:41",
  },
  {
    id: "c-8802",
    user: "Gizem Aydoğan",
    email: "gizem.aydogan@akbank-egitim.tr",
    deviceName: "SUBE-42-IST",
    os: "Windows 11 23H2",
    registeredAt: "11.04.2026",
    lastSeen: "02.09.2026 17:08",
  },
  {
    id: "c-8803",
    user: "Selin Arıkan",
    email: "selin.arikan@akbank-egitim.tr",
    deviceName: "SUBE-07-ANK",
    os: "Windows 11 23H2",
    registeredAt: "05.06.2026",
    lastSeen: "01.09.2026 11:22",
  },
  {
    id: "c-8804",
    user: "Burak Yıldırım",
    email: "burak.yildirim@akbank-egitim.tr",
    deviceName: "BURAK-EGITIM",
    os: "macOS 15.3",
    registeredAt: "14.02.2026",
    lastSeen: "03.09.2026 08:12",
  },
  {
    id: "c-8805",
    user: "Ayşe Demirtaş",
    email: "ayse.demirtas@akbank-egitim.tr",
    deviceName: "AYSE-EV-PC",
    os: "Ubuntu 24.04",
    registeredAt: "22.07.2026",
    lastSeen: "18.08.2026 21:03",
  },
]
