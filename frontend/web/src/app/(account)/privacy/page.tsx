"use client"

import * as React from "react"
import Link from "next/link"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { FormShell } from "@/components/prova/form-shell"
import { privacyClauses } from "./mock"

export default function PrivacyPage() {
  const [consent, setConsent] = React.useState(false)

  return (
    <FormShell
      title="Aydınlatma metni ve onay"
      description="Devam etmek için veri işleme esaslarını onaylamanız gerekir."
      action={
        <Button
          nativeButton={consent ? false : undefined}
          className="w-full"
          size="lg"
          disabled={!consent}
          render={consent ? <Link href="/verify-result" /> : undefined}
        >
          Onaylıyorum
        </Button>
      }
      footer="Onayınızı hesap ayarlarından her zaman geri çekebilirsiniz."
    >
      <dl className="space-y-3">
        {privacyClauses.map((clause) => (
          <div key={clause.id}>
            <dt className="prova-meta uppercase">{clause.title}</dt>
            <dd className="text-sm leading-relaxed text-foreground">
              {clause.text}
            </dd>
          </div>
        ))}
      </dl>

      <div className="flex items-start gap-2.5 border-t border-border pt-4">
        <Checkbox
          id="privacy-consent"
          checked={consent}
          onCheckedChange={(value) => setConsent(value === true)}
          className="mt-0.5"
        />
        <Label htmlFor="privacy-consent" className="text-sm leading-relaxed font-normal">
          Aydınlatma metnini okudum, verilerimin bu kapsamda işlenmesini
          onaylıyorum.
        </Label>
      </div>
    </FormShell>
  )
}
