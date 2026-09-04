import { Wifi, WifiOff, RefreshCw } from "lucide-react"

import { cn } from "@/lib/utils"
import type { ConnectionState } from "@/hooks/use-session"

const view = {
  connected: { icon: Wifi, label: "Bağlı" },
  reconnecting: { icon: RefreshCw, label: "Yeniden bağlanıyor" },
  disconnected: { icon: WifiOff, label: "Bağlantı koptu" },
} as const

/** All three states are shown with an icon and a label; colour never carries the meaning alone. */
export function ConnectionStatus({
  status,
  className,
}: {
  status: ConnectionState
  className?: string
}) {
  const { icon: Icon, label } = view[status]

  return (
    <span
      data-connection={status}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border px-2 py-1",
        status === "connected"
          ? "border-border bg-card"
          : "border-line-strong bg-muted",
        className
      )}
    >
      <Icon size={16} className="text-foreground" aria-hidden />
      <span className="prova-meta">{label}</span>
    </span>
  )
}
