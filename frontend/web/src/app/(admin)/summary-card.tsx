import type { LucideIcon } from "lucide-react"

/** Dashboard summary tile. No shadow; separation comes from a thin rule and a surface shift. */
export function SummaryCard({
  title,
  value,
  description,
  icon: Icon,
}: {
  title: string
  value: string
  description: string
  icon: LucideIcon
}) {
  return (
    <div className="rounded-lg border border-border bg-card px-4 py-3">
      <div className="flex items-center gap-2">
        <Icon size={16} className="text-primary" aria-hidden />
        <span className="prova-meta uppercase">{title}</span>
      </div>
      <p className="mt-2 font-heading text-3xl leading-none text-primary">
        {value}
      </p>
      <p className="prova-meta mt-1.5 normal-case">{description}</p>
    </div>
  )
}
