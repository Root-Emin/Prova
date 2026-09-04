import { cn } from "@/lib/utils"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

/**
 * The single layout behind the eight form screens (register, login, verify,
 * device, privacy notice, account deletion). One centred column, max 420px,
 * inside a Card: title + one-line description + one primary button.
 * The background texture is visible on these screens.
 */
function FormShell({
  title,
  description,
  children,
  action,
  footer,
  aside,
  className,
}: {
  title: string
  /** One line only. */
  description: string
  children?: React.ReactNode
  /** Exactly one primary button. */
  action?: React.ReactNode
  footer?: React.ReactNode
  /** Set-apart block rendered before the card (e.g. the privacy notice). */
  aside?: React.ReactNode
  className?: string
}) {
  return (
    <main
      className={cn(
        "prova-texture flex min-h-svh items-center justify-center bg-background px-6 py-12",
        className
      )}
    >
      <div className="w-[420px] max-w-[420px]">
        {/* Drag handle for the frameless window in Electron. */}
        <div className="prova-drag mb-6 text-center">
          <span className="font-heading text-2xl tracking-tight text-primary">
            Prova
          </span>
          <p className="prova-meta mt-1 uppercase">Yetkinlik sertifikasyonu</p>
        </div>

        {aside ? <div className="mb-4">{aside}</div> : null}

        <Card className="border-line-strong">
          <CardHeader>
            <CardTitle className="text-lg">{title}</CardTitle>
            <CardDescription>{description}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            {children}
            {action ? <div className="pt-1">{action}</div> : null}
          </CardContent>
        </Card>

        {footer ? (
          <div className="mt-4 text-center text-sm text-muted-foreground">
            {footer}
          </div>
        ) : null}
      </div>
    </main>
  )
}

export { FormShell }
