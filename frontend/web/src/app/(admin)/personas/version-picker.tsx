"use client"

import { GitBranch, Plus } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { Version } from "./mock"

export function VersionPicker({
  versions,
  activeVersionId,
  onVersionChange,
  onNewVersion,
}: {
  versions: Version[]
  activeVersionId: string
  onVersionChange: (id: string) => void
  onNewVersion: () => void
}) {
  const options = Object.fromEntries(
    versions.map((version) => [
      version.id,
      `${version.label} · ${version.date}${version.published ? " · yayında" : ""}`,
    ])
  )

  return (
    <div className="flex items-center gap-2">
      <GitBranch size={16} className="text-muted-foreground" aria-hidden />
      <Select
        items={options}
        value={activeVersionId}
        onValueChange={(value) => onVersionChange(value as string)}
      >
        <SelectTrigger size="default" className="w-[260px] bg-card">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {versions.map((version) => (
            <SelectItem key={version.id} value={version.id}>
              {version.label} · {version.date}
              {version.published ? " · yayında" : ""}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button variant="outline" onClick={onNewVersion}>
        <Plus aria-hidden />
        Yeni sürüm oluştur
      </Button>
    </div>
  )
}
