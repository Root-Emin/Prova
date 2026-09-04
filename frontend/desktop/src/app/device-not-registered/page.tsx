"use client"

import Link from "next/link"
import { Laptop } from "lucide-react"

import { Button } from "@/components/ui/button"
import { FormShell } from "@/components/prova/form-shell"

export default function DeviceNotRegisteredPage() {
  return (
    <FormShell
      title="Bu cihaz kayıtlı değil"
      description="Sınava başlamadan önce cihazınızı kaydetmeniz gerekiyor."
      action={
        <Button nativeButton={false} className="w-full" size="lg" render={<Link href="/device-enrollment" />}>
          Cihazı kaydet
        </Button>
      }
      footer="Cihazınızı kaydedemiyorsanız kurum yöneticinize başvurun."
    >
      <div className="flex items-start gap-3 rounded-lg border border-line-strong bg-muted px-4 py-3">
        <Laptop size={20} className="mt-0.5 shrink-0 text-primary" aria-hidden />
        <p className="text-sm leading-relaxed text-foreground">
          Sertifikanın geçerli olabilmesi için sınavın kayıtlı bir cihazdan
          verilmesi gerekir. Kayıtsız cihazdan başlatılan oturum
          değerlendirilmez.
        </p>
      </div>
    </FormShell>
  )
}
