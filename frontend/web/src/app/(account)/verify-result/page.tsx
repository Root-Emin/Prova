import Link from "next/link"
import { Check } from "lucide-react"

import { Button } from "@/components/ui/button"
import { FormShell } from "@/components/prova/form-shell"

export default function VerifyResultPage() {
  return (
    <FormShell
      title="E-posta doğrulandı"
      description="Bağlantı geçerliydi, hesabınız artık kullanıma açık."
      action={
        <Button nativeButton={false} className="w-full" size="lg" render={<Link href="/" />}>
          Panele git
        </Button>
      }
      footer="Bu bağlantı tek kullanımlıktı ve artık geçersiz."
    >
      <div className="flex items-start gap-3 rounded-lg border border-border bg-muted px-4 py-3">
        <Check size={20} className="mt-0.5 shrink-0 text-primary" aria-hidden />
        <p className="text-sm leading-relaxed text-foreground">
          Sınava girebilmek için bir sonraki adımda masaüstü uygulamasından
          cihazınızı kaydetmeniz gerekir.
        </p>
      </div>
    </FormShell>
  )
}
