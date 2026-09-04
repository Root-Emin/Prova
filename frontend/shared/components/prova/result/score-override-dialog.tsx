"use client"

import * as React from "react"
import { Scale } from "lucide-react"

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
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { rubricStatusView, type RubricStatus } from "@/lib/rubric"

const overridableStatuses: RubricStatus[] = ["passed", "partial", "failed"]

const statusOptions = Object.fromEntries(
  overridableStatuses.map((status) => [status, rubricStatusView[status].label])
)

export function ScoreOverrideDialog({
  criterionCode,
  criterionName,
  currentStatus,
  onOverride,
}: {
  criterionCode: string
  criterionName: string
  currentStatus: RubricStatus
  onOverride: (newStatus: RubricStatus, reason: string) => void
}) {
  const [open, setOpen] = React.useState(false)
  const [status, setStatus] = React.useState<RubricStatus>(currentStatus)
  const [reason, setReason] = React.useState("")

  function submit(event: React.FormEvent) {
    event.preventDefault()
    onOverride(status, reason.trim())
    setReason("")
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" size="sm" />}>
        <Scale aria-hidden />
        Puanı applyOverride
      </DialogTrigger>
      <DialogContent className="sm:max-w-[460px]">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Puanı ez</DialogTitle>
            <DialogDescription>
              {criterionCode} · {criterionName}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-5">
            <div className="space-y-2">
              <Label>Yeni durum</Label>
              <Select
                items={statusOptions}
                value={status}
                onValueChange={(value) => setStatus(value as RubricStatus)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {overridableStatuses.map((option) => (
                    <SelectItem key={option} value={option}>
                      {rubricStatusView[option].label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="ezme-nedeni">Gerekçe</Label>
              <Textarea
                id="ezme-nedeni"
                rows={4}
                required
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Model değerlendirmesini neden değiştirdiğinizi yazın."
              />
              <p className="prova-meta normal-case">
                Gerekçe denetim kaydına düşer ve sertifikada görünür.
              </p>
            </div>
          </div>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" type="button" />}>
              Vazgeç
            </DialogClose>
            <Button type="submit" disabled={reason.trim().length === 0}>
              Puanı ez
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
