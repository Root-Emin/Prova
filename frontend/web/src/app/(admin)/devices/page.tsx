"use client"

import * as React from "react"
import { toast } from "sonner"

import { PageHeader } from "@/components/prova/page-header"
import { DeviceTable } from "./device-table"
import { devices as initialRows, type Device } from "./mock"

export default function DevicesPage() {
  const [rows, setRows] = React.useState<Device[]>(initialRows)

  function remove(id: string) {
    const device = rows.find((row) => row.id === id)
    setRows((prev) => prev.filter((row) => row.id !== id))
    if (device) {
      toast.success(`${device.deviceName} kaydı kaldırıldı`)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Cihazlar"
        description="Sınav yalnızca kayıtlı cihazdan verilebilir. Her kullanıcı en fazla iki cihaz kaydedebilir."
      />
      <DeviceTable rows={rows} onRemove={remove} />
      <p className="prova-meta">{rows.length} kayıtlı cihaz</p>
    </div>
  )
}
