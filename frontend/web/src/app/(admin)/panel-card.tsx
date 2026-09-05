import Link from "next/link"
import { ChevronRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

/**
 * Panel bölümlerinin ortak kabuğu: başlık şeridi ince ayraçla gövdeden
 * ayrılır, sağ üstte tek bir "tümü" bağlantısı durur. Gölge yok.
 */
export function PanelCard({
  title,
  description,
  href,
  linkLabel = "Tümü",
  action,
  className,
  bodyClassName,
  children,
}: {
  title: string
  description?: string
  /** Bölümün tam listesine giden bağlantı. */
  href?: string
  linkLabel?: string
  /** Bağlantı yerine geçen serbest denetim (aralık seçimi gibi). */
  action?: React.ReactNode
  className?: string
  bodyClassName?: string
  children: React.ReactNode
}) {
  return (
    <section
      className={cn(
        "flex min-w-0 flex-col rounded-lg border border-border bg-card",
        className
      )}
    >
      <header className="flex items-center justify-between gap-4 border-b border-border px-4 py-3">
        <div className="min-w-0">
          <h2 className="font-heading text-base tracking-tight text-primary">
            {title}
          </h2>
          {description ? (
            <p className="prova-meta mt-1 normal-case">{description}</p>
          ) : null}
        </div>
        {action ??
          (href ? (
            <Button
              nativeButton={false}
              variant="ghost"
              size="sm"
              className="shrink-0"
              render={<Link href={href} />}
            >
              {linkLabel}
              <ChevronRight aria-hidden />
            </Button>
          ) : null)}
      </header>
      <div className={cn("flex-1 p-4", bodyClassName)}>{children}</div>
    </section>
  )
}
