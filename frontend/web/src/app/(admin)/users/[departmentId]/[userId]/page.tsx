"use client"

import * as React from "react"
import Link from "next/link"
import { notFound, useRouter } from "next/navigation"
import { ArrowLeft, Ban, Laptop, RotateCcw, Trash2 } from "lucide-react"
import { toast } from "sonner"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

import {
  initialsOf,
  userRoleLabel,
  userSourceLabel,
  userStatusHint,
  type User,
  type Workstation,
} from "../../mock"
import { UserStatusBadge } from "../../status-badge"
import { UserDelete } from "../../user-delete"
import { useUsers } from "../../users-store"

export default function UserDetailPage({
  params,
}: {
  params: Promise<{ departmentId: string; userId: string }>
}) {
  const { departmentId, userId } = React.use(params)
  const { departments, users, toggleStatus, revokeWorkstation } = useUsers()
  const router = useRouter()

  // Kullanıcı bu ekrandan silinince liste ekranına dönülür; dönüş tamamlanana
  // kadar 404 gösterilmemesi için silme ayrıca işaretlenir.
  const [removed, setRemoved] = React.useState(false)

  const user = users.find((item) => item.id === userId)
  const department = departments.find((item) => item.id === departmentId)
  if (!user || !department) {
    if (removed) return null
    notFound()
  }

  const station =
    user.workstations.find((item) => item.current) ?? user.workstations[0]

  function flipStatus() {
    const before = toggleStatus(user!.id)
    if (!before) return
    toast.success(
      before.status === "inactive"
        ? `${before.name} yeniden etkinleştirildi`
        : `${before.name} pasifleştirildi`,
      { description: "İşlem denetim kaydına yazıldı." }
    )
  }

  function dropStation(workstation: Workstation) {
    revokeWorkstation(user!.id, workstation.id)
    toast.success(`${workstation.hostname} kaydı kaldırıldı`, {
      description: "Kullanıcı bu makineden sınava giremez.",
    })
  }

  return (
    <div className="space-y-6">
      <Button
        nativeButton={false}
        variant="ghost"
        size="sm"
        render={<Link href={`/users/${departmentId}`} />}
      >
        <ArrowLeft aria-hidden />
        {department.name}
      </Button>

      <IdentityHeader
        user={user}
        departmentName={department.name}
        onToggle={flipStatus}
        onRemoved={() => {
          setRemoved(true)
          router.replace(`/users/${departmentId}`)
        }}
      />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <MembershipCard user={user} departmentName={department.name} />
        <MachineCard user={user} station={station} />
      </div>

      <section className="space-y-3">
        <div className="flex items-baseline justify-between gap-4">
          <h2 className="font-heading text-lg">Kayıtlı cihazlar</h2>
          <p className="prova-meta normal-case">
            Kullanıcı başına en fazla iki cihaz kaydedilebilir
          </p>
        </div>

        {user.workstations.length > 0 ? (
          <div className="min-w-0 rounded-lg border border-border bg-card">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[170px]">Bilgisayar adı</TableHead>
                  <TableHead className="w-[130px]">IP adresi</TableHead>
                  <TableHead className="w-[170px]">MAC adresi</TableHead>
                  <TableHead className="w-[160px]">İşletim sistemi</TableHead>
                  <TableHead>Son görülme</TableHead>
                  <TableHead className="w-[120px] text-right">İşlem</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {user.workstations.map((workstation) => (
                  <TableRow key={workstation.id}>
                    <TableCell className="font-mono text-xs">
                      {workstation.hostname}
                      {workstation.current && (
                        <div className="prova-meta normal-case">
                          son giriş bu makineden
                        </div>
                      )}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {workstation.ip}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {workstation.mac}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {workstation.os}
                    </TableCell>
                    <TableCell className="prova-meta">
                      {workstation.lastSeenAt}
                    </TableCell>
                    <TableCell className="text-right">
                      <AlertDialog>
                        <AlertDialogTrigger
                          render={<Button variant="ghost" size="sm" />}
                        >
                          <Trash2 aria-hidden />
                          Kaldır
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>
                              Cihaz kaydı kaldırılsın mı?
                            </AlertDialogTitle>
                            <AlertDialogDescription>
                              {user.name} {workstation.hostname} makinesinden
                              sınava giremez. Yeniden girmek için cihazı baştan
                              kaydetmesi gerekir.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Vazgeç</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => dropStation(workstation)}
                            >
                              Cihazı kaldır
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <p className="rounded-lg border border-border bg-card px-4 py-6 text-sm text-muted-foreground">
            Kayıtlı cihaz yok. İlk giriş tamamlandığında kullanılan makine
            hesaba bağlanır ve burada listelenir.
          </p>
        )}
      </section>

      <p className="prova-meta normal-case">
        Bilgisayar adı, IP ve MAC adresi sınav bütünlüğü için tutulur; kişisel
        veridir, kurum alanı dışına çıkmaz ve bu sayfaya her erişim denetim
        kaydına yazılır.
      </p>
    </div>
  )
}

function IdentityHeader({
  user,
  departmentName,
  onToggle,
  onRemoved,
}: {
  user: User
  departmentName: string
  onToggle: () => void
  onRemoved: () => void
}) {
  return (
    <header className="flex items-start justify-between gap-6 rounded-lg border border-line-strong bg-card p-4">
      <div className="flex min-w-0 items-start gap-4">
        <Avatar size="lg" className="mt-0.5">
          <AvatarFallback>{initialsOf(user.name)}</AvatarFallback>
        </Avatar>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2.5">
            <h1 className="font-heading text-2xl leading-none tracking-tight">
              {user.name}
            </h1>
            <UserStatusBadge status={user.status} />
          </div>
          <p className="mt-1.5 font-mono text-sm text-muted-foreground">
            {user.email}
          </p>
          <p className="mt-2 text-sm">{userStatusHint[user.status]}</p>
        </div>
      </div>

      <div className="flex shrink-0 gap-2">
        <Button variant="outline" onClick={onToggle}>
          {user.status === "inactive" ? (
            <>
              <RotateCcw aria-hidden />
              Etkinleştir
            </>
          ) : (
            <>
              <Ban aria-hidden />
              Pasifleştir
            </>
          )}
        </Button>
        <UserDelete
          user={user}
          departmentName={departmentName}
          onDeleted={onRemoved}
        />
      </div>
    </header>
  )
}

function MembershipCard({
  user,
  departmentName,
}: {
  user: User
  departmentName: string
}) {
  return (
    <section className="prova-kart p-4">
      <h2 className="font-heading text-base leading-none">Kimlik ve üyelik</h2>
      <dl className="mt-3 space-y-2.5">
        <Row label="Kullanıcı no" value={user.id} mono />
        <Row label="Departman" value={departmentName} />
        <Row label="Görev" value={user.title} />
        <Row label="Rol" value={userRoleLabel[user.role]} />
        <Row label="Eklenme" value={user.addedAt} />
        <Row label="Eklenme yolu" value={userSourceLabel[user.source]} />
        <Row label="Son giriş" value={user.lastLoginAt ?? "Hiç giriş yapmadı"} />
      </dl>
    </section>
  )
}

/**
 * Girişin yapıldığı makine. Sertifikanın "kim, nerede, hangi cihazda"
 * iddiasını taşıyan üç alan burada, tek bakışta okunacak biçimde durur.
 */
function MachineCard({
  user,
  station,
}: {
  user: User
  station: Workstation | undefined
}) {
  return (
    <section className="prova-kart flex flex-col p-4">
      <div className="flex items-center gap-2">
        <Laptop size={16} className="text-primary" aria-hidden />
        <h2 className="font-heading text-base leading-none">
          Son girişin yapıldığı makine
        </h2>
      </div>

      {station ? (
        <>
          <dl className="mt-3 space-y-2.5">
            <Row label="Bilgisayar adı" value={station.hostname} mono />
            <Row label="IP adresi" value={station.ip} mono />
            <Row label="MAC adresi" value={station.mac} mono />
            <Row label="İşletim sistemi" value={station.os} />
          </dl>
          <p className="prova-meta mt-3 border-t border-border pt-3 normal-case">
            {user.lastLoginAt} · cihaz {station.registeredAt} tarihinde kaydedildi
          </p>
        </>
      ) : (
        <p className="mt-3 text-sm text-muted-foreground">
          Kullanıcı henüz giriş yapmadı. Masaüstü uygulamasından kendi kurumsal
          adresiyle giriş yaptığında bilgisayar adı, IP adresi ve MAC adresi
          burada görünür.
        </p>
      )}
    </section>
  )
}

function Row({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      <dt className="prova-meta shrink-0 uppercase">{label}</dt>
      <dd className={mono ? "font-mono text-sm" : "text-sm"}>{value}</dd>
    </div>
  )
}
