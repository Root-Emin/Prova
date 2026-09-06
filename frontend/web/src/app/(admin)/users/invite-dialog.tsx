"use client"

import * as React from "react"
import { UserPlus } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { graphqlRequest } from "@/lib/graphql"

import { userRoleLabel, type UserRole } from "./mock"
import { useUsers } from "./users-store"

/** Çalışan masaüstünde sınava girer, kurum yöneticisi bu paneli kullanır. */
const invitableRoles: UserRole[] = ["employee", "org-admin"]

const roleOptions = Object.fromEntries(
  invitableRoles.map((role) => [role, userRoleLabel[role]])
)


function inviteErrorMessage(error: unknown): string {
  const raw = error instanceof Error ? error.message : "Bir hata oluştu."
  if (/account already exists/i.test(raw)) {
    return "Bu e-posta için zaten doğrulanmış bir hesap var."
  }
  return raw
}

export function InviteDialog({
  departmentId,
  lockDepartment = false,
}: {
  departmentId: string
  lockDepartment?: boolean
}) {
  const { departments, addUsers } = useUsers()

  const [open, setOpen] = React.useState(false)
  const [name, setName] = React.useState("")
  const [email, setEmail] = React.useState("")
  const [role, setRole] = React.useState<UserRole>("employee")
  const [target, setTarget] = React.useState(departmentId)
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  const departmentOptions = React.useMemo(
    () =>
      Object.fromEntries(
        departments.map((department) => [department.id, department.name])
      ),
    [departments]
  )

  function openChange(next: boolean) {
    setOpen(next)
    if (next) setTarget(departmentId)
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    const address = email.trim().toLocaleLowerCase("en-US")
    const nameParts = name.trim().split(/\s+/)
    const firstName = nameParts.shift() ?? ""
    const lastName = nameParts.join(" ")

    setIsSubmitting(true)
    try {
      const data = await graphqlRequest<{
        inviteUser: {
          invited: boolean
          email: string
        }
      }>(
        `mutation InviteUser($input: InviteUserInput!) {
          inviteUser(input: $input) {
            invited
            email
          }
        }`,
        {
          input: {
            email: address,
            firstName,
            lastName,
            role: role === "org-admin" ? "ORG_ADMIN" : "EMPLOYEE",
          },
        },
      )

      addUsers(
        [{ email: data.inviteUser.email, name: name.trim(), role }],
        target,
        "invite",
      )
      toast.success("Kullanıcı eklendi", {
        description: `${data.inviteUser.email} Desktop'tan ilk giriş yaptığında doğrulama kodu gönderilecek.`,
      })
      setName("")
      setEmail("")
      setRole("employee")
      setOpen(false)
    } catch (error) {
      toast.error("Davet gönderilemedi", {
        description: inviteErrorMessage(error),
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={openChange}>
      <DialogTrigger render={<Button size="lg" />}>
        <UserPlus aria-hidden />
        Davet et
      </DialogTrigger>
      <DialogContent className="sm:max-w-[420px]">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Kullanıcı davet et</DialogTitle>
            <DialogDescription>
              Kullanıcı yalnızca Desktop uygulamasından kendi adresiyle giriş
              yapabilir. Doğrulama kodu ilk giriş isteğinde gönderilir.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-5">
            <div className="space-y-2">
              <Label htmlFor="davet-ad">Ad soyad</Label>
              <Input
                id="davet-ad"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Ad Soyad"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="davet-eposta">Kurumsal e-posta</Label>
              <Input
                id="davet-eposta"
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="ad.soyad@kurum.tr"
                required
              />
            </div>
            {!lockDepartment && (
              <div className="space-y-2">
                <Label>Departman</Label>
                <Select
                  items={departmentOptions}
                  value={target}
                  onValueChange={(value) => setTarget(value as string)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {departments.map((department) => (
                      <SelectItem key={department.id} value={department.id}>
                        {department.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
            <div className="space-y-2">
              <Label>Rol</Label>
              <Select
                items={roleOptions}
                value={role}
                onValueChange={(value) => setRole(value as UserRole)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {invitableRoles.map((option) => (
                    <SelectItem key={option} value={option}>
                      {userRoleLabel[option]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" type="button" />}>
              Vazgeç
            </DialogClose>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Ekleniyor…" : "Kullanıcıyı ekle"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
