import Link from "next/link"

import { Button } from "@/components/ui/button"
import { FormShell } from "@/components/prova/form-shell"
import { PrivacyNotice } from "@/components/prova/privacy-notice"

export default function RegisterPage() {
  return (
    <FormShell
      title="Hesabınız yönetici tarafından tanımlanır"
      description="Prova'da çalışan hesapları kurum yöneticisi kurumsal e-posta adresiyle oluşturur. İlk girişte doğrulama kodu Desktop uygulamasından gönderilir."
      aside={<PrivacyNotice />}
      footer="Hesabınız tanımlandıysa Desktop uygulamasından giriş yapın."
    >
      <Button
        nativeButton={false}
        className="w-full"
        size="lg"
        render={<Link href="/login" />}
      >
        Web girişine dön
      </Button>
    </FormShell>
  )
}
