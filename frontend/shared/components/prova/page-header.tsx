import { cn } from "@/lib/utils"

/** Shared header block for the admin routes. No background texture. */
function PageHeader({
  title,
  description,
  meta,
  action,
  className,
}: {
  title: string
  description: string
  /** Single meta line, parts separated by a dot. */
  meta?: (string | false | undefined)[]
  action?: React.ReactNode
  className?: string
}) {
  const metaLine = meta?.filter(Boolean) as string[] | undefined

  return (
    <header
      className={cn(
        "flex items-start justify-between gap-6 border-b border-border pb-4",
        className
      )}
    >
      <div>
        <h1 className="font-heading text-2xl tracking-tight">{title}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
        {metaLine && metaLine.length > 0 ? (
          <p className="prova-meta mt-2 normal-case">{metaLine.join(" · ")}</p>
        ) : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </header>
  )
}

export { PageHeader }
