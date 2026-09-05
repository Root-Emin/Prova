"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { ArrowLeft } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { FormShell } from "@/components/prova/form-shell"
import { PrivacyNotice } from "@/components/prova/privacy-notice"
import { onboardingSteps } from "./steps"

/**
 * The three intro screens shown once, before the first login. Deliberately
 * the same shell as the login card that follows on the fourth screen: same
 * width, same logo position, so only the card content changes underneath.
 */
export default function OnboardingPage() {
  const router = useRouter()
  const [index, setIndex] = React.useState(0)
  const [isLeaving, setIsLeaving] = React.useState(false)

  const step = onboardingSteps[index]
  const isLastStep = index === onboardingSteps.length - 1

  async function goToLogin() {
    setIsLeaving(true)
    // The flag is owned by the main process. A failed write would only mean
    // the intro is shown again; it must never trap anyone on this screen.
    try {
      await window.prova?.onboarding.complete()
    } catch {
      // Deliberately ignored; the navigation below happens either way.
    }
    router.replace("/login")
  }

  return (
    <FormShell
      title={step.title}
      description={step.description}
      aside={step.privacy ? <PrivacyNotice /> : undefined}
      action={
        <Button
          className="w-full"
          size="lg"
          disabled={isLeaving}
          onClick={() => {
            if (isLastStep) {
              void goToLogin()
              return
            }
            setIndex((current) => current + 1)
          }}
        >
          {isLastStep ? "Girişe geç" : "Devam"}
        </Button>
      }
      footer={
        <div className="grid grid-cols-3 items-center">
          <Button
            variant="ghost"
            size="sm"
            className={cn("justify-self-start -ml-2.5", index === 0 && "invisible")}
            disabled={isLeaving}
            onClick={() => setIndex((current) => current - 1)}
          >
            <ArrowLeft aria-hidden />
            Geri
          </Button>

          <StepIndicator current={index} total={onboardingSteps.length} />


          <Button
            variant="ghost"
            size="sm"
            className={cn("justify-self-end -mr-2.5", isLastStep && "invisible")}
            disabled={isLeaving}
            onClick={() => void goToLogin()}
          >
            Atla
          </Button>
        </div>
      }
    >
      <ul className="space-y-3">
        {step.points.map((point) => (
          <li key={point.text} className="flex gap-3">
            <point.icon
              size={16}
              className="mt-0.5 shrink-0 text-primary"
              aria-hidden
            />
            <span className="leading-relaxed text-foreground">{point.text}</span>
          </li>
        ))}
      </ul>
    </FormShell>
  )
}

/**
 * Three segments, slate for seen and steel for upcoming. No orange: in this
 * palette orange has only three standing jobs and a step counter is not one.
 */
function StepIndicator({ current, total }: { current: number; total: number }) {
  return (
    <div
      className="flex items-center justify-self-center gap-1"
      role="img"
      aria-label={`Adım ${current + 1} / ${total}`}
    >
      {Array.from({ length: total }, (_, position) => (
        <span
          key={position}
          className={cn(
            "h-[3px] w-6 rounded-full",
            position <= current ? "bg-primary" : "bg-border"
          )}
        />
      ))}
    </div>
  )
}
