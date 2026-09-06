type GraphQLError = { message?: string }

type GraphQLResponse<T> = {
  data?: T
  errors?: GraphQLError[]
}

const graphqlURL =
  process.env.NEXT_PUBLIC_GRAPHQL_URL ?? "http://localhost:8080/graphql"

export async function graphqlRequest<T>(
  query: string,
  variables: Record<string, unknown>,
): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" }
  // Yönetim çağrıları, aynı tarayıcıdaki parolasız girişin access token'ını
  // taşır. SSR'da window olmadığı için anonim akışlar etkilenmez.
  if (typeof window !== "undefined") {
    const accessToken = window.localStorage.getItem("prova:access-token")
    if (accessToken) headers.Authorization = `Bearer ${accessToken}`
  }
  const response = await fetch(graphqlURL, {
    method: "POST",
    headers,
    body: JSON.stringify({ query, variables }),
  })
  const payload = (await response.json()) as GraphQLResponse<T>
  if (!response.ok || payload.errors?.length || !payload.data) {
    throw new Error(payload.errors?.[0]?.message ?? "İşlem tamamlanamadı")
  }
  return payload.data
}

export const accountFlowStorage = {
  email: "prova:account-flow:email",
  mode: "prova:account-flow:mode",
  resendAfter: "prova:account-flow:resend-after",
} as const
