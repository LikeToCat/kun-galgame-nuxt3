import Cookies from 'js-cookie'

/**
 * Unified API response format from Go backend.
 * All endpoints return: { code: 0, message: "成功", data: T }
 *
 * For paginated endpoints, data itself contains { items, total }:
 *   { code: 0, message: "成功", data: { items: [...], total: 42 } }
 */
interface KunApiResponse<T> {
  code: number
  message: string
  data: T
}

const CODE_AUTH_EXPIRED = 205
const CODE_BIZ_ERROR = 233

const getApiBase = () => {
  const config = useRuntimeConfig()
  return config.public.apiBaseUrl as string
}

const handleError = async (code: number, message: string) => {
  if (code === CODE_AUTH_EXPIRED) {
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

  if (code === CODE_BIZ_ERROR) {
    useMessage(message, 'error')
  }
}

/**
 * kunFetch - Imperative fetch for mutations (button clicks, form submits).
 * Unwraps { code, data } and handles errors automatically.
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
  try {
    const resp = await $fetch<KunApiResponse<T>>(
      `${getApiBase()}/api${url}`,
      {
        credentials: 'include',
        ...options
      }
    )

    if (!resp) {
      useMessage('网络请求失败，请稍后重试', 'error')
      return null
    }

    if (resp.code !== 0) {
      await handleError(resp.code, resp.message)
      return null
    }

    return resp.data
  } catch {
    useMessage('网络请求失败，请稍后重试', 'error')
    return null
  }
}

/**
 * useKunFetch - SSR-safe composable for data fetching.
 * Wraps useFetch, automatically unwraps { code, data } via transform.
 * data.value is T | null (already unwrapped).
 *
 * @example
 * const { data } = await useKunFetch<HomeData>('/home')
 * // data.value?.galgames
 *
 * @example
 * const { data, status } = await useKunFetch<{ items: Topic[], total: number }>(
 *   '/topic',
 *   { query: pageData }
 * )
 */
export const useKunFetch = <T>(
  url: string | (() => string),
  options?: Record<string, unknown>
) => {
  const resolvedUrl = typeof url === 'function' ? url : () => url
  const apiBase = getApiBase()

  return useFetch(() => `${apiBase}/api${resolvedUrl()}`, {
    credentials: 'include',
    ...options,
    transform: (resp: KunApiResponse<T>) => {
      if (!resp || resp.code !== 0) {
        return null as T | null
      }
      return resp.data
    },
    onResponse: async (ctx) => {
      const resp = ctx.response._data as KunApiResponse<T> | undefined
      if (resp && resp.code !== 0) {
        await handleError(resp.code, resp.message)
      }
    },
    onResponseError: () => {
      useMessage('网络请求失败，请稍后重试', 'error')
    }
  })
}
