"use client"

import Link from "next/link"

import { Button } from "@/components/ui/button"
import { FormShell } from "@/components/prova/form-shell"
import { useDeviceState } from "@/hooks/use-device-state"
import { enrollmentInfo } from "./mock"

export default function DeviceEnrollmentPage() {
  const device = useDeviceState()

  return (
    <FormShell
      title="Bu cihazı kaydedin"
      description="Sınav yalnızca kayıtlı cihazdan verilebilir."
      action={
        <Button nativeButton={false} className="w-full" size="lg" render={<Link href="/" />}>
          Cihazı kaydet
        </Button>
      }
      footer={`Kayıtlı cihaz sınırı ${enrollmentInfo.deviceLimit}; şu an ${enrollmentInfo.registeredDeviceCount} cihaz kayıtlı.`}
    >
      <dl className="space-y-3">
        <div className="flex items-baseline justify-between gap-4">
          <dt className="prova-meta uppercase">Cihaz adı</dt>
          <dd className="font-mono text-sm">{device.deviceName}</dd>
        </div>
        <div className="flex items-baseline justify-between gap-4">
          <dt className="prova-meta uppercase">İşletim sistemi</dt>
          <dd className="text-sm">{device.os}</dd>
        </div>
        <div className="flex items-baseline justify-between gap-4">
          <dt className="prova-meta uppercase">Parmak izi</dt>
          <dd className="font-mono text-sm">{shortFingerprint(device.fingerprint)}</dd>
        </div>
        <div className="flex items-baseline justify-between gap-4 border-t border-border pt-3">
          <dt className="prova-meta uppercase">Eşleştirme kodu</dt>
          <dd className="font-mono text-base tracking-widest">
            {enrollmentInfo.pairingCode}
          </dd>
        </div>
      </dl>
      <p className="prova-meta normal-case">
        Bu kodu web panelindeki cihaz ekranına girerek eşleştirmeyi
        tamamlayabilirsiniz.
      </p>
    </FormShell>
  )
}

function shortFingerprint(fingerprint: string | null): string {
  if (!fingerprint) return "Kullanılamıyor"
  return `${fingerprint.slice(0, 4)}·${fingerprint.slice(4, 8)}·${fingerprint.slice(8, 12)}`
}
