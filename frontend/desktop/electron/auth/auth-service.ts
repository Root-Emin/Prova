import type { LoginCodeResponse, LoginResult } from "../api-types";
import type { DeviceIdentityService } from "../device/device-identity";
import type { SecureStorage } from "../security/secure-storage";

const AUTH_SESSION_KEY = "auth-session-v1";
const DEFAULT_GRAPHQL_URL = "http://127.0.0.1:8080/graphql";
const REQUEST_TIMEOUT_MS = 15_000;

type GraphQLResponse<T> = {
  data?: T;
  errors?: Array<{ message?: string }>;
};

type LoginCodeData = {
  requestLoginCode: LoginCodeResponse;
};

type PasswordLoginData = {
  login: {
    expiresAt: string;
    accessToken: string;
    refreshToken: string;
    organizationId: string;
    user: { email: string };
  };
};

type ChallengeData = {
  requestDeviceChallenge: {
    challenge: string;
  };
};

type VerifyLoginData = {
  verifyLoginCode: {
    expiresAt: string;
    accessToken: string;
    refreshToken: string;
    organizationId: string;
    user: { email: string };
    device: { isNew: boolean } | null;
  };
};

type StoredAuthSession = {
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  organizationId: string;
  email: string;
};

export class DesktopAuthService {
  private readonly endpoint: string;

  constructor(
    private readonly identity: DeviceIdentityService,
    private readonly storage: SecureStorage,
    endpoint = process.env.PROVA_GRAPHQL_URL ??
      process.env.NEXT_PUBLIC_GRAPHQL_URL ??
      DEFAULT_GRAPHQL_URL,
  ) {
    this.endpoint = validateEndpoint(endpoint);
  }

  async login(email: string, password: string): Promise<LoginResult> {
    const normalizedEmail = normalizeEmail(email);
    if (!password || password.length < 8) throw new Error("Şifre en az 8 karakter olmalı");

    const storageStatus = await this.storage.status();
    if (!storageStatus.available) {
      throw new Error(
        `Güvenli oturum deposu kullanılamıyor: ${storageStatus.reason ?? "bilinmeyen neden"}`,
      );
    }

    const data = await this.graphql<PasswordLoginData>(
      `mutation Login($input: PasswordLoginInput!) {
        login(input: $input) {
          accessToken
          refreshToken
          expiresAt
          organizationId
          user { email }
        }
      }`,
      { input: { email: normalizedEmail, password } },
    );
    const result = data.login;
    if (
      !result ||
      !result.accessToken ||
      !result.refreshToken ||
      !result.expiresAt ||
      !result.organizationId ||
      result.user?.email !== normalizedEmail
    ) {
      throw new Error("Sunucudan geçersiz giriş yanıtı alındı");
    }
    await this.storage.set(
      AUTH_SESSION_KEY,
      JSON.stringify({
        accessToken: result.accessToken,
        refreshToken: result.refreshToken,
        expiresAt: result.expiresAt,
        organizationId: result.organizationId,
        email: result.user.email,
      } satisfies StoredAuthSession),
    );
    return { deviceIsNew: false, expiresAt: result.expiresAt };
  }

  async requestLoginCode(email: string): Promise<LoginCodeResponse> {
    const normalizedEmail = normalizeEmail(email);
    const data = await this.graphql<LoginCodeData>(
      `mutation RequestLoginCode($input: RequestLoginCodeInput!) {
        requestLoginCode(input: $input) {
          sent
          expiresInSeconds
          resendAfterSeconds
        }
      }`,
      { input: { email: normalizedEmail } },
    );
    const result = data.requestLoginCode;
    if (
      typeof result?.sent !== "boolean" ||
      !Number.isInteger(result.expiresInSeconds) ||
      result.expiresInSeconds < 0 ||
      !Number.isInteger(result.resendAfterSeconds) ||
      result.resendAfterSeconds < 0
    ) {
      throw new Error("Sunucudan geçersiz kod yanıtı alındı");
    }
    return result;
  }

  async verifyLoginCode(email: string, code: string): Promise<LoginResult> {
    const normalizedEmail = normalizeEmail(email);
    if (!/^\d{6}$/.test(code)) throw new Error("6 haneli kodu girin");

    const storageStatus = await this.storage.status();
    if (!storageStatus.available) {
      throw new Error(
        `Güvenli oturum deposu kullanılamıyor: ${storageStatus.reason ?? "bilinmeyen neden"}`,
      );
    }

    const registration = await this.identity.getRegistrationInfo();
    const challengeData = await this.graphql<ChallengeData>(
      `mutation RequestDeviceChallenge($input: DeviceChallengeInput!) {
        requestDeviceChallenge(input: $input) { challenge }
      }`,
      {
        input: {
          email: normalizedEmail,
          fingerprint: registration.fingerprint,
        },
      },
    );
    const signature = await this.identity.signChallenge(
      challengeData.requestDeviceChallenge.challenge,
    );

    const data = await this.graphql<VerifyLoginData>(
      `mutation VerifyLoginCode($input: VerifyLoginCodeInput!) {
        verifyLoginCode(input: $input) {
          accessToken
          refreshToken
          expiresAt
          organizationId
          user { email }
          device { isNew }
        }
      }`,
      {
        input: {
          email: normalizedEmail,
          code,
          device: {
            fingerprint: registration.fingerprint,
            name: registration.displayName,
            platform: registration.platform,
            publicKey: registration.publicKey,
            macAddress: registration.macAddress,
          },
          deviceSignature: signature,
        },
      },
    );

    const result = data.verifyLoginCode;
    if (
      !result ||
      !result.accessToken ||
      !result.refreshToken ||
      !result.expiresAt ||
      !result.organizationId ||
      result.user?.email !== normalizedEmail
    ) {
      throw new Error("Sunucudan geçersiz giriş yanıtı alındı");
    }

    const storedSession: StoredAuthSession = {
      accessToken: result.accessToken,
      refreshToken: result.refreshToken,
      expiresAt: result.expiresAt,
      organizationId: result.organizationId,
      email: result.user.email,
    };
    await this.storage.set(AUTH_SESSION_KEY, JSON.stringify(storedSession));

    return {
      deviceIsNew: result.device?.isNew === true,
      expiresAt: result.expiresAt,
    };
  }

  private async graphql<T>(
    query: string,
    variables: Record<string, unknown>,
  ): Promise<T> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
    try {
      const response = await fetch(this.endpoint, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ query, variables }),
        signal: controller.signal,
      });
      const payload: unknown = await response.json();
      if (!payload || typeof payload !== "object") {
        throw new Error("Sunucudan geçersiz yanıt alındı");
      }
      const result = payload as GraphQLResponse<T>;
      if (!response.ok || result.errors?.length || !result.data) {
        throw new Error(
          result.errors?.[0]?.message ?? "Sunucu işlemi tamamlayamadı",
        );
      }
      return result.data;
    } catch (error) {
      if (error instanceof Error && error.name === "AbortError") {
        throw new Error("Sunucu yanıt vermedi; lütfen tekrar deneyin");
      }
      if (error instanceof TypeError && error.message === "fetch failed") {
        throw new Error(
          "Sunucuya bağlanılamadı; backend'in çalıştığından emin olun",
        );
      }
      throw error;
    } finally {
      clearTimeout(timeout);
    }
  }
}

function normalizeEmail(value: string): string {
  const email = typeof value === "string" ? value.trim().toLowerCase() : "";
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || email.length > 320) {
    throw new Error("Geçerli bir e-posta girin");
  }
  return email;
}

function validateEndpoint(rawEndpoint: string): string {
  let endpoint: URL;
  try {
    endpoint = new URL(rawEndpoint);
  } catch {
    throw new Error("PROVA_GRAPHQL_URL geçersiz");
  }
  if (
    (endpoint.protocol !== "http:" && endpoint.protocol !== "https:") ||
    endpoint.username ||
    endpoint.password ||
    endpoint.hash ||
    (endpoint.protocol === "http:" && !isLoopbackHost(endpoint.hostname))
  ) {
    throw new Error(
      "PROVA_GRAPHQL_URL HTTPS olmalı; HTTP yalnızca yerel geliştirmede kullanılabilir",
    );
  }
  return endpoint.toString();
}

function isLoopbackHost(hostname: string): boolean {
  return (
    hostname === "localhost" ||
    hostname === "127.0.0.1" ||
    hostname === "[::1]" ||
    hostname === "::1"
  );
}
