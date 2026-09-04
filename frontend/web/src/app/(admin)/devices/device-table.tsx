"use client"

import { Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { Device } from "./mock"

export function DeviceTable({
  rows,
  onRemove,
}: {
  rows: Device[]
  onRemove: (id: string) => void
}) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[220px]">Kullanıcı</TableHead>
            <TableHead className="w-[180px]">Cihaz adı</TableHead>
            <TableHead className="w-[180px]">İşletim sistemi</TableHead>
            <TableHead className="w-[140px]">Kayıt tarihi</TableHead>
            <TableHead>Son görülme</TableHead>
            <TableHead className="w-[120px] text-right">İşlem</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((device) => (
            <TableRow key={device.id}>
              <TableCell className="font-medium">
                {device.user}
                <div className="prova-meta normal-case">{device.email}</div>
              </TableCell>
              <TableCell className="font-mono text-xs">
                {device.deviceName}
              </TableCell>
              <TableCell className="text-muted-foreground">
                {device.os}
              </TableCell>
              <TableCell className="prova-meta">{device.registeredAt}</TableCell>
              <TableCell className="prova-meta">{device.lastSeen}</TableCell>
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
                      <AlertDialogTitle>Cihaz kaydı kaldırılsın mı?</AlertDialogTitle>
                      <AlertDialogDescription>
                        {device.user} bu cihazdan sınava giremez. Yeniden
                        girmek için cihazı baştan kaydetmesi gerekir.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Vazgeç</AlertDialogCancel>
                      <AlertDialogAction onClick={() => onRemove(device.id)}>
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
  )
}
