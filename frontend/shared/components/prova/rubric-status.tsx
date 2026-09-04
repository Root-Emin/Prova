import { cn } from "@/lib/utils"
import { rubricStatusView, type RubricStatus } from "@/lib/rubric"

/**
 * Rubric status is always conveyed by colour + lucide icon + text label
 * together. Colour is never used on its own.
 */
function RubricStatusBadge({
  status,
  className,
}: {
  status: RubricStatus
  className?: string
}) {
  const view = rubricStatusView[status]
  const Icon = view.icon

  return (
    <span
      data-status={status}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium",
        view.surface,
        view.border,
        view.text,
        className
      )}
    >
      <Icon size={16} aria-hidden />
      {view.label}
    </span>
  )
}

/** Inline variant: icon + label, no surface. */
function RubricStatusInline({
  status,
  className,
}: {
  status: RubricStatus
  className?: string
}) {
  const view = rubricStatusView[status]
  const Icon = view.icon

  return (
    <span
      data-status={status}
      className={cn(
        "inline-flex items-center gap-1.5 text-xs font-medium",
        view.text,
        className
      )}
    >
      <Icon size={16} aria-hidden />
      {view.label}
    </span>
  )
}

export { RubricStatusBadge, RubricStatusInline }
