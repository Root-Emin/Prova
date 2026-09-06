export const IPC_CHANNELS = {
  systemGetInfo: "prova:system:get-info",
  deviceGetInfo: "prova:device:get-info",
  deviceGetRegistrationInfo: "prova:device:get-registration-info",
  permissionsGetMicrophoneStatus: "prova:permissions:get-microphone-status",
  authRequestLoginCode: "prova:auth:request-login-code",
  authLogin: "prova:auth:login",
  authVerifyLoginCode: "prova:auth:verify-login-code",
  onboardingGetState: "prova:onboarding:get-state",
  onboardingComplete: "prova:onboarding:complete",
} as const
