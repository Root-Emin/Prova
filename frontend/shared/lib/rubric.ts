import { AlertTriangle, Check, Circle, X, type LucideIcon } from "lucide-react"

/** The four states of a rubric criterion. All start as "unevaluated". */
export type RubricStatus = "unevaluated" | "passed" | "partial" | "failed"

type RubricStatusView = {
  /** Text label — colour alone is not enough. */
  label: string
  icon: LucideIcon
  /** Colour of the icon and the text inside the filled badge. */
  text: string
  /**
   * Colour for the surfaceless inline variant. The badge separates states by
   * fill weight, which an inline label has no room for, so there it falls back
   * to a text tone that stays readable on the page background.
   */
  inline: string
  /** Badge surface. */
  surface: string
  /** Badge border. */
  border: string
}

/**
 * The palette holds four colours, so the states are not separated by hue but
 * by fill weight: unevaluated is a quiet outline, passed is a solid slate
 * fill, partial is an ember outline, failed is a solid ember fill. Icon and
 * label always ride along — see .notes/DESIGN.md.
 */
export const rubricStatusView: Record<RubricStatus, RubricStatusView> = {
  unevaluated: {
    label: "Değerlendirilmedi",
    icon: Circle,
    inline: "text-muted-foreground",
    text: "text-muted-foreground",
    surface: "bg-muted",
    border: "border-border",
  },
  passed: {
    label: "Geçti",
    icon: Check,
    inline: "text-pass",
    text: "text-primary-foreground",
    surface: "bg-pass",
    border: "border-pass",
  },
  partial: {
    label: "Kısmi",
    icon: AlertTriangle,
    inline: "text-ember-ink",
    text: "text-foreground",
    surface: "bg-partial/20",
    border: "border-partial",
  },
  failed: {
    label: "Kaldı",
    icon: X,
    inline: "text-ember-ink",
    text: "text-foreground",
    surface: "bg-fail",
    border: "border-fail",
  },
}

export const rubricStatusOrder: RubricStatus[] = [
  "passed",
  "partial",
  "failed",
  "unevaluated",
]
