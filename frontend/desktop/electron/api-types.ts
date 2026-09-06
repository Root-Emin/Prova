export type NormalizedPlatform = "macos" | "windows" | "linux"

export type MicrophonePermissionStatus =
  | "granted"
  | "denied"
  | "restricted"
  | "not-determined"
  | "unknown"

export type SystemInfo = {
  platform: NormalizedPlatform
  osRelease: string
  appVersion: string
}

export type DeviceInfo = SystemInfo & {
  displayName: string
}

export type DeviceRegistrationInfo = DeviceInfo & {
  /** Random per-install identifier; never derived from hardware or hostname. */
  fingerprint: string
  /** Raw 32-byte Ed25519 public key encoded as standard base64. */
  publicKey: string
  /** Active network adapter MAC address; unavailable on some systems. */
  macAddress: string
}

export type LoginCodeResponse = {
  sent: boolean
  expiresInSeconds: number
  resendAfterSeconds: number
}

export type OnboardingState = {
  /** True once the three intro screens have been seen or skipped. */
  completed: boolean
}

export type LoginResult = {
  deviceIsNew: boolean
  expiresAt: string
}

export type ProvaAPI = {
  system: {
    getInfo(): Promise<SystemInfo>
  }
  device: {
    getInfo(): Promise<DeviceInfo>
    getRegistrationInfo(): Promise<DeviceRegistrationInfo>
  }
  permissions: {
    getMicrophoneStatus(): Promise<MicrophonePermissionStatus>
  }
  auth: {
    login(email: string, password: string): Promise<LoginResult>
    requestLoginCode(email: string): Promise<LoginCodeResponse>
    verifyLoginCode(email: string, code: string): Promise<LoginResult>
  }
  onboarding: {
    getState(): Promise<OnboardingState>
    complete(): Promise<OnboardingState>
  }
}
