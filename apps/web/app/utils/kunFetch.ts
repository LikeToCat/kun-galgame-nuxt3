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

const handleApiError = async (code: number, message: string) => {
  if (import.meta.server) return

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
  async onResponseError({ response }) {
    const resp = response._data as KunApiResponse<unknown> | undefined
    if (resp && resp.code !== 0) {
      await handleApiError(resp.code, resp.message)
    }
  },
  transform(resp: any) {
    if (!resp || resp.code !== 0) {
      return null
    }
    return resp.data
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
  const apiBase = `${config.public.apiBaseUrl}/api`

  try {
    const resp = await $fetch<KunApiResponse<T>>(`${apiBase}${url}`, {
      credentials: 'include',
      ...options
    })

    if (!resp || resp.code !== 0) {
      if (resp) {
        await handleApiError(resp.code, resp.message)
      }
      return null
    }

    return resp.data
  } catch {
    if (import.meta.client) {
      useMessage('网络请求失败，请稍后重试', 'error')
    }
    return null
  }
}
