import Image from "next/image"

import { cn } from "@/lib/utils"
import provaMark from "../../assets/logo/prova-mark.png"

type LogoSize = "sm" | "md" | "lg"

/** The mark is a line engraving; below ~18px the hatching turns to mush. */
const markPx: Record<LogoSize, number> = { sm: 20, md: 28, lg: 56 }
const wordClass: Record<LogoSize, string> = {
  sm: "text-sm",
  md: "text-xl",
  lg: "text-2xl",
}

/**
 * The bird engraving on its own, without the wordmark. Decorative by default;
 * pass a label where the mark stands in for the product name.
 */
function ProvaMark({
  size = "md",
  className,
  label,
}: {
  size?: LogoSize
  className?: string
  label?: string
}) {
  const px = markPx[size]

  return (
    <Image
      src={provaMark}
      alt={label ?? ""}
      aria-hidden={label ? undefined : true}
      width={px}
      height={px}
      priority
      className={cn("shrink-0 select-none object-contain", className)}
      style={{ width: px, height: px }}
    />
  )
}

/**
 * Mark + "Prova" wordmark lockup. Horizontal in chrome (sidebar, top bar),
 * stacked above the form card on the account screens.
 */
function ProvaLogo({
  size = "md",
  stacked = false,
  tagline,
  className,
}: {
  size?: LogoSize
  stacked?: boolean
  /** Single line under the wordmark, e.g. the tenant or the product line. */
  tagline?: string
  className?: string
}) {
  return (
    <div
      className={cn(
        stacked
          ? "flex flex-col items-center gap-1.5"
          : "flex min-w-0 items-center gap-2.5",
        className
      )}
    >
      <ProvaMark size={size} />
      <div className={cn("min-w-0", stacked && "text-center")}>
        <span
          className={cn(
            "block font-heading tracking-tight text-primary",
            wordClass[size]
          )}
        >
          Prova
        </span>
        {tagline ? (
          <span className="prova-meta mt-0.5 block truncate uppercase">
            {tagline}
          </span>
        ) : null}
      </div>
    </div>
  )
}

export { ProvaLogo, ProvaMark }
