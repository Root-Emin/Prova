/** Convert transport, GraphQL, and domain errors into safe user-facing text. */
export function authErrorMessage(
  error: unknown,
  fallback = "İşlem tamamlanamadı. Lütfen tekrar deneyin.",
): string {
  const raw =
    error instanceof Error
      ? error.message
      : typeof error === "string"
        ? error
        : "";
  const message = raw.toLocaleLowerCase("en-US");

  if (
    message.includes("invalid or expired code") ||
    message.includes("invalid or expired email verification code") ||
    message.includes("geçersiz veya süresi dolmuş")
  ) {
    return "Kod geçersiz veya süresi dolmuş. Yeni bir kod isteyin.";
  }

  if (
    message.includes("too many") ||
    message.includes("rate limited") ||
    message.includes("rate_limit") ||
    message.includes("çok fazla")
  ) {
    return "Çok fazla deneme yapıldı. Lütfen biraz sonra tekrar deneyin.";
  }

  if (
    message.includes("already sent") ||
    message.includes("retry in") ||
    message.includes("cooldown")
  ) {
    return "Yeni kod göndermek için lütfen biraz bekleyin.";
  }

  if (
    message.includes("email is required") ||
    message.includes("valid email") ||
    message.includes("geçerli bir e-posta")
  ) {
    return "Geçerli bir e-posta adresi girin.";
  }

  if (message.includes("6 haneli") || message.includes("six digit")) {
    return "6 haneli kodu girin.";
  }

  if (
    message.includes("account is not active") ||
    message.includes("hesap etkin değil")
  ) {
    return "Bu hesap henüz aktif değil.";
  }

  if (
    message.includes("secure storage") ||
    message.includes("güvenli oturum deposu") ||
    message.includes("güvenli cihaz deposu")
  ) {
    return "Güvenli cihaz depolaması kullanılamıyor. Uygulamayı yeniden açın.";
  }

  if (
    message.includes("device") ||
    message.includes("cihaz") ||
    message.includes("challenge") ||
    message.includes("signature")
  ) {
    return "Bu cihazla giriş yapılamadı. Uygulamayı yeniden açıp tekrar deneyin.";
  }

  if (
    message.includes("fetch failed") ||
    message.includes("connection") ||
    message.includes("connect") ||
    message.includes("server") ||
    message.includes("resend") ||
    message.includes("delivery") ||
    message.includes("internal") ||
    message.includes("sunucu")
  ) {
    return "Sunucuya bağlanılamadı. Lütfen daha sonra tekrar deneyin.";
  }

  return fallback;
}
