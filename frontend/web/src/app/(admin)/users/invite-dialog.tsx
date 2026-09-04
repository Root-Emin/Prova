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
import { userRoleLabel, type User } from "./mock"

const invitableRoles: User["role"][] = [
  "employee",
  "trainer",
  "org-admin",
]

const roleOptions = Object.fromEntries(
  invitableRoles.map((role) => [role, userRoleLabel[role]])
)

export function InviteDialog({
  onInvite,
}: {
  onInvite: (user: { name: string; email: string; role: User["role"] }) => void
}) {
  const [open, setOpen] = React.useState(false)
  const [name, setAd] = React.useState("")
  const [email, setEmail] = React.useState("")
  const [role, setRole] = React.useState<User["role"]>("employee")

  function submit(event: React.FormEvent) {
    event.preventDefault()
    onInvite({ name, email, role })
    toast.success("Davet gönderildi", {
      description: `${email} adresine doğrulama bağlantısı iletildi.`,
    })
    setAd("")
    setEmail("")
    setRole("employee")
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button size="lg" />}>
        <UserPlus aria-hidden />
        Davet et
      </DialogTrigger>
      <DialogContent className="sm:max-w-[420px]">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Kullanıcı davet et</DialogTitle>
            <DialogDescription>
              Davet edilen kişi e-postasını doğrulayana kadar oturum açamaz.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-5">
            <div className="space-y-2">
              <Label htmlFor="davet-ad">Ad soyad</Label>
              <Input
                id="davet-ad"
                value={name}
                onChange={(event) => setAd(event.target.value)}
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
            <div className="space-y-2">
              <Label>Rol</Label>
              <Select
                items={roleOptions}
                value={role}
                onValueChange={(value) => setRole(value as User["role"])}
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
            <Button type="submit">Daveti gönder</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
