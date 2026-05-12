/**
 * Unified API response format from Go backend.
 * All endpoints return: { code: 0, message: "成功", data: T }
 */
interface KunApiResponse<T> {
  code: number
  message: string
  data: T
}

const CODE_AUTH_EXPIRED = 205
// CODE_BANNED comes from kungal's mapping of OAuth 10014. We MUST NOT
// redirect banned users to /login — a fresh login would just hit 10014
// again at the next refresh, putting them in an infinite loop. Instead
// surface the message prominently and stop there.
const CODE_BANNED = 234

const handleApiError = async (code: number, message: string) => {
  if (import.meta.server) return

  if (code === CODE_BANNED) {
    const userStore = usePersistUserStore()
    if (userStore.id) {
      userStore.resetUser()
    }
    useMessage(message || '您的账号已被封禁', 'error', 10000)
    return
  }

  if (code === CODE_AUTH_EXPIRED) {
    const { default: Cookies } = await import('js-cookie')
    const navigateCookie = Cookies.get('kun-is-navigate-to-login')
    const userStore = usePersistUserStore()

    if (!navigateCookie && userStore.id) {
      userStore.resetUser()
      useMessage(message || '登录已失效，请重新登录', 'error', 7777)
      Cookies.set('kun-is-navigate-to-login', 'navigated', {
        expires: 1 / 1440
      })
      await navigateTo('/login')
    }
    return
  }

  if (code !== 0) {
    useMessage(message, 'error')
  }
}

/**
 * useKunFetch — SSR-safe composable built on Nuxt 4 `createUseFetch`.
 *
 * Automatically:
 * - Resolves baseURL for SSR/CSR
 * - Forwards cookies during SSR via credentials
 * - Unwraps `{ code, data }` response
 * - Handles auth/biz errors client-side only
 *
 * The response type T is what the Go backend returns inside `data`.
 * The transform unwraps it so `data.value` is `T | null` directly.
 *
 * @example
 * const { data } = await useKunFetch<HomeData>('/home')
 * // data.value?.galgames
 *
 * @example
 * const { data } = await useKunFetch<{ items: T[], total: number }>(
 *   '/topic',
 *   { query: pageData }
 * )
 */
export const useKunFetch = createUseFetch({
  baseURL: computed(() => {
    const config = useRuntimeConfig()
    const base = import.meta.server
      ? config.apiBaseUrl
      : config.public.apiBaseUrl
    return `${base}/api`
  }),
  credentials: 'include',
  // SSR is cross-origin from Nuxt → Go API, so `credentials: 'include'` is a
  // no-op on the server (there's no browser-side cookie store to pull from).
  // Manually forward the incoming request's Cookie header so the Go backend
  // sees the authenticated session during SSR. Without this, per-user flags
  // like isLiked / isFavorited / isUpvoted come back false on the first
  // render and stay un-highlighted until the user interacts.
  onRequest({ options }) {
    if (import.meta.server) {
      const headers = useRequestHeaders(['cookie'])
      if (headers.cookie) {
        const merged = new Headers(options.headers as HeadersInit | undefined)
        merged.set('cookie', headers.cookie)
        options.headers = merged
      }
    }
  },
  async onResponseError({ response }) {
    const resp = response._data as KunApiResponse<unknown> | undefined
    if (resp && resp.code !== 0) {
      await handleApiError(resp.code, resp.message)
    }
  },
  transform(resp: unknown) {
    const envelope = resp as KunApiResponse<unknown> | null | undefined
    if (!envelope || envelope.code !== 0) {
      return null
    }
    // Go's response.OKMessage omits `data`. Callers typically gate optimistic
    // updates on `if (result)`, so returning the message keeps the truthy
    // success signal intact while still distinguishing from the `null`
    // returned for real failures.
    return envelope.data !== undefined ? envelope.data : envelope.message
  }
})

/**
 * kunFetch — Imperative fetch for mutations (button clicks, form submits).
 * Client-side only. Unwraps { code, data } and handles errors.
 * Returns the unwrapped data, or null on error.
 *
 * @example
 * const result = await kunFetch<string>('/user/bio', {
 *   method: 'PUT',
 *   body: { bio: 'hello' }
 * })
 * if (result) { useMessage('更新成功', 'success') }
 */
export const kunFetch = async <T>(
  url: string,
  options?: Record<string, unknown>
): Promise<T | null> => {
  const config = useRuntimeConfig()
  // Prefer the server-only base URL when running on SSR — it bypasses
  // the public proxy and goes straight to the Go backend.
  const apiBase = import.meta.server
    ? `${config.apiBaseUrl}/api`
    : `${config.public.apiBaseUrl}/api`

  // Cookie forwarding (SSR): same rationale as useKunFetch above — without
  // this, server-side callers (e.g. loadInitialReplies awaited in setup())
  // are unauthenticated and the reply list comes back with isLiked=false.
  const headers = new Headers(
    (options as { headers?: HeadersInit } | undefined)?.headers
  )
  if (import.meta.server) {
    const requestHeaders = useRequestHeaders(['cookie'])
    if (requestHeaders.cookie) {
      headers.set('cookie', requestHeaders.cookie)
    }
  }

  try {
    const resp = await $fetch<KunApiResponse<T>>(`${apiBase}${url}`, {
      credentials: 'include',
      ...options,
      headers
    })

    if (!resp || resp.code !== 0) {
      if (resp) {
        await handleApiError(resp.code, resp.message)
      }
      return null
    }

    // Same fallback rationale as useKunFetch above: OKMessage responses
    // have no `data`, but callers check `if (result)` to confirm success.
    return resp.data !== undefined ? resp.data : (resp.message as T)
  } catch {
    if (import.meta.client) {
      useMessage('网络请求失败，请稍后重试', 'error')
    }
    return null
  }
}
