import Cookies from 'js-cookie'
import type { FetchContext, FetchResponse } from 'ofetch'

interface ResponseMap {
  blob: Blob
  text: string
  arrayBuffer: ArrayBuffer
  stream: ReadableStream<Uint8Array>
}
type ResponseType = keyof ResponseMap | 'json'

type KunOnResponseContext = FetchContext<any, ResponseType> & {
  response: FetchResponse<ResponseType>
}
type KunOnResponseErrorContext<R extends ResponseType = 'json'> =
  FetchContext & {
    response: FetchResponse<R>
  }

export const onResponse = (context: KunOnResponseContext) => {
  if (context.response.status === 205) {
    const navigateCookie = Cookies.get('kungalgame-is-navigate-to-login')
    if (!navigateCookie) {
      usePersistUserStore().$reset()
      useMessage(
        'Login expired, please login again',
        '登录过期, 请重新登陆',
        'error',
        7777
      )

      const nuxt = useNuxtApp()
      navigateTo(nuxt.$localePath('/login'))
      Cookies.set('kungalgame-is-navigate-to-login', 'navigated')
      return
    }
  }

  if (context.response.status === 233) {
    kungalgameErrorHandler(context.response.headers.get('Kun-Error') || '')
  }
}

const onResponseError = (context: KunOnResponseErrorContext) => {}

export const kungalgameResponseHandler = { onResponse, onResponseError }
