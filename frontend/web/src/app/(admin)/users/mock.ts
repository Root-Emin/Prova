/**
 * Kullanıcı yaşam döngüsü:
 *
 *   expected  Excel ile yüklendi, adres tanımlı, henüz giriş yok
 *   invited   yönetici tek tek ekledi, adres doğrulanmadı
 *   active    masaüstü uygulamasından kendi adresiyle giriş yaptı
 *   inactive  yönetici pasifleştirdi, oturum açamaz
 *
 * `expected` ile `invited` arasındaki fark eklenme yoludur; kod e-postası
 * ikisinde de yalnızca Desktop'taki ilk giriş isteğinde gönderilir.
 */
export type UserStatus = "active" | "expected" | "invited" | "inactive"

export type UserSource = "bulk" | "invite"

/**
 * İki rol vardır. Çalışan yalnızca masaüstü uygulamasını kullanır, kurum
 * yöneticisi yalnızca web panelini. Üçüncü bir rol yoktur.
 */
export type UserRole = "employee" | "org-admin"

/**
 * Girişin yapıldığı makine. Bu üç alan — bilgisayar adı, IP ve MAC — sınavın
 * kimin, nereden ve hangi makinede verildiğini belirler; sertifikayı bir
 * eğitim kaydından denetim kanıtına dönüştüren de budur.
 */
export type Workstation = {
  id: string
  /** Bilgisayar adı (hostname). */
  hostname: string
  ip: string
  mac: string
  os: string
  registeredAt: string
  lastSeenAt: string
  /** Son girişin yapıldığı makine. */
  current: boolean
}

export type User = {
  id: string
  name: string
  email: string
  title: string
  role: UserRole
  status: UserStatus
  departmentId: string
  source: UserSource
  addedAt: string
  /** Hiç giriş yapmadıysa null; cihaz alanları da o zaman boştur. */
  lastLoginAt: string | null
  workstations: Workstation[]
}

export type Department = {
  id: string
  name: string
  /** Listelerde adın kısaltması olarak görünür. */
  code: string
  lead: string
  site: string
}

export const userStatusLabel: Record<UserStatus, string> = {
  active: "Aktif",
  expected: "Giriş bekleniyor",
  invited: "Davet bekliyor",
  inactive: "Pasif",
}

export const userStatusHint: Record<UserStatus, string> = {
	active: "Masaüstü uygulamasından giriş yaptı, cihazı kayıtlı.",
	expected: "Toplu yükleme ile tanımlandı, Desktop'tan ilk girişi bekleniyor.",
	invited: "Yönetici ekledi; Desktop'tan ilk giriş ve e-posta doğrulaması bekleniyor.",
  inactive: "Pasifleştirildi; oturum açamaz, sınava giremez.",
}

export const userRoleLabel: Record<UserRole, string> = {
  employee: "Çalışan",
  "org-admin": "Kurum yöneticisi",
}

export const userSourceLabel: Record<UserSource, string> = {
  bulk: "Excel ile toplu yükleme",
  invite: "Tek tek davet",
}

export const departments: Department[] = [
  {
    id: "cagri-merkezi",
    name: "Çağrı Merkezi",
    code: "CM",
    lead: "Elif Şahin",
    site: "İstanbul · Kozyatağı",
  },
  {
    id: "sube-operasyon",
    name: "Şube Operasyonları",
    code: "SOP",
    lead: "Burak Yıldırım",
    site: "İstanbul · Genel Müdürlük",
  },
  {
    id: "uyum-risk",
    name: "Uyum ve Risk",
    code: "UYM",
    lead: "Gizem Aydoğan",
    site: "Ankara · Bölge",
  },
  {
    id: "bireysel-satis",
    name: "Bireysel Satış",
    code: "BSA",
    lead: "Onur Kılıçarslan",
    site: "İzmir · Bölge",
  },
  {
    id: "egitim-gelisim",
    name: "Eğitim ve Gelişim",
    code: "EGT",
    lead: "Selin Arıkan",
    site: "İstanbul · Genel Müdürlük",
  },
]

export const users: User[] = [
  {
    id: "k-1041",
    name: "Elif Şahin",
    email: "elif.sahin@akbank-egitim.tr",
    title: "Çağrı merkezi müdürü",
    role: "org-admin",
    status: "active",
    departmentId: "cagri-merkezi",
    source: "invite",
    addedAt: "12.02.2026",
    lastLoginAt: "05.09.2026 08:52",
    workstations: [
      {
        id: "c-8811",
        hostname: "CM-MUDUR-01",
        ip: "10.42.7.18",
        mac: "A4:83:E7:1C:9D:02",
        os: "Windows 11 23H2",
        registeredAt: "12.02.2026",
        lastSeenAt: "05.09.2026 08:52",
        current: true,
      },
    ],
  },
  {
    id: "k-1051",
    name: "Deniz Korkmaz",
    email: "deniz.korkmaz@akbank-egitim.tr",
    title: "Müşteri temsilcisi",
    role: "employee",
    status: "active",
    departmentId: "cagri-merkezi",
    source: "bulk",
    addedAt: "03.03.2026",
    lastLoginAt: "04.09.2026 17:26",
    workstations: [
      {
        id: "c-8812",
        hostname: "CM-IST-114",
        ip: "10.42.7.114",
        mac: "B8:27:EB:5A:31:7F",
        os: "Windows 11 23H2",
        registeredAt: "05.03.2026",
        lastSeenAt: "04.09.2026 17:26",
        current: true,
      },
    ],
  },
  {
    id: "k-1052",
    name: "Kerem Uysal",
    email: "kerem.uysal@akbank-egitim.tr",
    title: "Müşteri temsilcisi",
    role: "employee",
    status: "expected",
    departmentId: "cagri-merkezi",
    source: "bulk",
    addedAt: "01.09.2026",
    lastLoginAt: null,
    workstations: [],
  },
  {
    id: "k-1053",
    name: "Zeynep Aksoy",
    email: "zeynep.aksoy@akbank-egitim.tr",
    title: "Müşteri temsilcisi",
    role: "employee",
    status: "expected",
    departmentId: "cagri-merkezi",
    source: "bulk",
    addedAt: "01.09.2026",
    lastLoginAt: null,
    workstations: [],
  },
  {
    id: "k-1054",
    name: "Tolga Ergün",
    email: "tolga.ergun@akbank-egitim.tr",
    title: "Takım lideri",
    role: "employee",
    status: "active",
    departmentId: "cagri-merkezi",
    source: "invite",
    addedAt: "19.03.2026",
    lastLoginAt: "03.09.2026 11:07",
    workstations: [
      {
        id: "c-8813",
        hostname: "CM-LIDER-07",
        ip: "10.42.7.7",
        mac: "D4:6D:6D:12:8B:90",
        os: "Windows 11 23H2",
        registeredAt: "19.03.2026",
        lastSeenAt: "03.09.2026 11:07",
        current: true,
      },
    ],
  },

  {
    id: "k-1042",
    name: "Burak Yıldırım",
    email: "burak.yildirim@akbank-egitim.tr",
    title: "Operasyon yöneticisi",
    role: "org-admin",
    status: "active",
    departmentId: "sube-operasyon",
    source: "invite",
    addedAt: "14.02.2026",
    lastLoginAt: "05.09.2026 08:12",
    workstations: [
      {
        id: "c-8804",
        hostname: "BURAK-EGITIM",
        ip: "10.18.3.9",
        mac: "9C:8E:CD:44:10:2A",
        os: "macOS 15.3",
        registeredAt: "14.02.2026",
        lastSeenAt: "05.09.2026 08:12",
        current: true,
      },
    ],
  },
  {
    id: "k-1045",
    name: "Gizem Aydoğan",
    email: "gizem.aydogan@akbank-egitim.tr",
    title: "Gişe yetkilisi",
    role: "employee",
    status: "active",
    departmentId: "sube-operasyon",
    source: "bulk",
    addedAt: "11.04.2026",
    lastLoginAt: "02.09.2026 17:08",
    workstations: [
      {
        id: "c-8802",
        hostname: "SUBE-42-IST",
        ip: "10.18.42.66",
        mac: "00:1A:2B:3C:4D:5E",
        os: "Windows 11 23H2",
        registeredAt: "11.04.2026",
        lastSeenAt: "02.09.2026 17:08",
        current: true,
      },
    ],
  },
  {
    id: "k-1055",
    name: "Murat Şentürk",
    email: "murat.senturk@akbank-egitim.tr",
    title: "Gişe yetkilisi",
    role: "employee",
    status: "active",
    departmentId: "sube-operasyon",
    source: "bulk",
    addedAt: "11.04.2026",
    lastLoginAt: "01.09.2026 09:14",
    workstations: [
      {
        id: "c-8814",
        hostname: "SUBE-19-IST",
        ip: "10.18.19.23",
        mac: "54:BF:64:9A:E1:07",
        os: "Windows 11 23H2",
        registeredAt: "13.04.2026",
        lastSeenAt: "01.09.2026 09:14",
        current: true,
      },
    ],
  },
  {
    id: "k-1056",
    name: "Pınar Ateş",
    email: "pinar.ates@akbank-egitim.tr",
    title: "Gişe yetkilisi",
    role: "employee",
    status: "invited",
    departmentId: "sube-operasyon",
    source: "invite",
    addedAt: "28.08.2026",
    lastLoginAt: null,
    workstations: [],
  },
  {
    id: "k-1044",
    name: "Mert Çankaya",
    email: "mert.cankaya@akbank-egitim.tr",
    title: "Operasyon uzmanı",
    role: "employee",
    status: "expected",
    departmentId: "sube-operasyon",
    source: "bulk",
    addedAt: "28.08.2026",
    lastLoginAt: null,
    workstations: [],
  },

  {
    id: "k-1057",
    name: "Hakan Ersoy",
    email: "hakan.ersoy@akbank-egitim.tr",
    title: "Uyum uzmanı",
    role: "org-admin",
    status: "active",
    departmentId: "uyum-risk",
    source: "invite",
    addedAt: "07.01.2026",
    lastLoginAt: "05.09.2026 07:58",
    workstations: [
      {
        id: "c-8815",
        hostname: "UYM-ANK-03",
        ip: "10.61.2.14",
        mac: "AC:DE:48:00:11:22",
        os: "Windows 11 23H2",
        registeredAt: "07.01.2026",
        lastSeenAt: "05.09.2026 07:58",
        current: true,
      },
    ],
  },
  {
    id: "k-1058",
    name: "Nur Bilgin",
    email: "nur.bilgin@akbank-egitim.tr",
    title: "Risk analisti",
    role: "employee",
    status: "active",
    departmentId: "uyum-risk",
    source: "bulk",
    addedAt: "02.05.2026",
    lastLoginAt: "29.08.2026 14:35",
    workstations: [
      {
        id: "c-8816",
        hostname: "UYM-ANK-11",
        ip: "10.61.2.41",
        mac: "E4:5F:01:6C:88:B3",
        os: "Windows 11 23H2",
        registeredAt: "02.05.2026",
        lastSeenAt: "29.08.2026 14:35",
        current: true,
      },
    ],
  },
  {
    id: "k-1059",
    name: "Cem Doğanay",
    email: "cem.doganay@akbank-egitim.tr",
    title: "Uyum uzmanı",
    role: "employee",
    status: "expected",
    departmentId: "uyum-risk",
    source: "bulk",
    addedAt: "02.09.2026",
    lastLoginAt: null,
    workstations: [],
  },

  {
    id: "k-1046",
    name: "Onur Kılıçarslan",
    email: "onur.kilicarslan@akbank-egitim.tr",
    title: "Satış yöneticisi",
    role: "employee",
    status: "inactive",
    departmentId: "bireysel-satis",
    source: "invite",
    addedAt: "19.11.2025",
    lastLoginAt: "14.06.2026 10:02",
    workstations: [
      {
        id: "c-8817",
        hostname: "BSA-IZM-01",
        ip: "10.77.5.10",
        mac: "70:85:C2:33:9F:41",
        os: "macOS 15.2",
        registeredAt: "19.11.2025",
        lastSeenAt: "14.06.2026 10:02",
        current: true,
      },
    ],
  },
  {
    id: "k-1060",
    name: "İrem Yalçın",
    email: "irem.yalcin@akbank-egitim.tr",
    title: "Portföy yöneticisi",
    role: "employee",
    status: "active",
    departmentId: "bireysel-satis",
    source: "bulk",
    addedAt: "21.05.2026",
    lastLoginAt: "04.09.2026 16:44",
    workstations: [
      {
        id: "c-8818",
        hostname: "BSA-IZM-24",
        ip: "10.77.5.24",
        mac: "2C:F0:5D:71:04:E8",
        os: "Windows 11 23H2",
        registeredAt: "21.05.2026",
        lastSeenAt: "04.09.2026 16:44",
        current: true,
      },
    ],
  },
  {
    id: "k-1061",
    name: "Barış Toprak",
    email: "baris.toprak@akbank-egitim.tr",
    title: "Portföy yöneticisi",
    role: "employee",
    status: "expected",
    departmentId: "bireysel-satis",
    source: "bulk",
    addedAt: "02.09.2026",
    lastLoginAt: null,
    workstations: [],
  },
  {
    id: "k-1062",
    name: "Sinem Kavak",
    email: "sinem.kavak@akbank-egitim.tr",
    title: "Portföy yöneticisi",
    role: "employee",
    status: "expected",
    departmentId: "bireysel-satis",
    source: "bulk",
    addedAt: "02.09.2026",
    lastLoginAt: null,
    workstations: [],
  },

  {
    id: "k-1047",
    name: "Selin Arıkan",
    email: "selin.arikan@akbank-egitim.tr",
    title: "Eğitim tasarımcısı",
    role: "employee",
    status: "active",
    departmentId: "egitim-gelisim",
    source: "invite",
    addedAt: "05.06.2026",
    lastLoginAt: "05.09.2026 10:12",
    workstations: [
      {
        id: "c-8803",
        hostname: "SUBE-07-ANK",
        ip: "10.5.9.7",
        mac: "48:2A:E3:0B:6C:D9",
        os: "Windows 11 23H2",
        registeredAt: "05.06.2026",
        lastSeenAt: "05.09.2026 10:12",
        current: true,
      },
    ],
  },
  {
    id: "k-1063",
    name: "Emre Balcı",
    email: "emre.balci@akbank-egitim.tr",
    title: "Eğitim uzmanı",
    role: "employee",
    status: "active",
    departmentId: "egitim-gelisim",
    source: "invite",
    addedAt: "18.06.2026",
    lastLoginAt: "03.09.2026 13:20",
    workstations: [
      {
        id: "c-8819",
        hostname: "EGT-GM-05",
        ip: "10.5.9.35",
        mac: "6C:4B:90:2D:77:1A",
        os: "macOS 15.4",
        registeredAt: "18.06.2026",
        lastSeenAt: "03.09.2026 13:20",
        current: true,
      },
    ],
  },
]

/**
 * Yeni departmanın kimliği adından türetilir; rota parçası olduğu için
 * yalnızca harf, rakam ve tire taşır.
 */
export function departmentIdFrom(name: string) {
  const base = name
    .toLocaleLowerCase("tr-TR")
    .replace(/ı/g, "i")
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 40)
  return base || `departman-${Date.now().toString(36)}`
}

export type DepartmentStats = {
  total: number
  active: number
  waiting: number
  inactive: number
}

export function statsFor(rows: User[], departmentId: string): DepartmentStats {
  const owned = rows.filter((user) => user.departmentId === departmentId)
  return {
    total: owned.length,
    active: owned.filter((user) => user.status === "active").length,
    waiting: owned.filter(
      (user) => user.status === "expected" || user.status === "invited"
    ).length,
    inactive: owned.filter((user) => user.status === "inactive").length,
  }
}

/** Avatar başharfleri. Tek kelimelik adlarda ilk iki harf alınır. */
export function initialsOf(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return "?"
  if (parts.length === 1) return parts[0].slice(0, 2).toLocaleUpperCase("tr-TR")
  return (parts[0][0] + parts[parts.length - 1][0]).toLocaleUpperCase("tr-TR")
}

/** Bugünün tarihi; toplu yüklemede eklenme tarihi olarak yazılır. */
export function today() {
  return new Date().toLocaleDateString("tr-TR")
}
