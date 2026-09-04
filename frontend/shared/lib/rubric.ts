import { AlertTriangle, Check, Circle, X, type LucideIcon } from "lucide-react"

/** The four states of a rubric criterion. All start as "unevaluated". */
export type RubricStatus = "unevaluated" | "passed" | "partial" | "failed"

type RubricStatusView = {
  /** Text label — colour alone is not enough. */
  label: string
  icon: LucideIcon
  /** Colour of the icon and the text. */
  text: string
  /** Badge surface. */
  surface: string
  /** Badge border. */
  border: string
}

export const rubricStatusView: Record<RubricStatus, RubricStatusView> = {
  unevaluated: {
    label: "Değerlendirilmedi",
    icon: Circle,
    text: "text-muted-foreground",
    surface: "bg-muted",
    border: "border-border",
  },
  passed: {
    label: "Geçti",
    icon: Check,
    text: "text-pass",
    surface: "bg-pass/10",
    border: "border-pass/35",
  },
  partial: {
    label: "Kısmi",
    icon: AlertTriangle,
    text: "text-partial",
    surface: "bg-partial/10",
    border: "border-partial/35",
  },
  failed: {
    label: "Kaldı",
    icon: X,
    text: "text-fail",
    surface: "bg-fail/10",
    border: "border-fail/35",
  },
}

export const rubricStatusOrder: RubricStatus[] = [
  "passed",
  "partial",
  "failed",
  "unevaluated",
]
