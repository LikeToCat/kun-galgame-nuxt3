<script setup lang="ts">
definePageMeta({ layout: 'blank' })

const route = useRoute()
const error = ref('')

onMounted(async () => {
  const code = route.query.code as string
  const returnedState = route.query.state as string
  const savedState = sessionStorage.getItem('oauth_state')
  const codeVerifier = sessionStorage.getItem('oauth_code_verifier')

  // Clean up
  sessionStorage.removeItem('oauth_state')
  sessionStorage.removeItem('oauth_code_verifier')

  if (!code) {
    error.value = '未收到授权码'
    redirectToLogin()
    return
  }

  if (returnedState !== savedState) {
    error.value = 'State 不匹配，可能存在安全风险'
    redirectToLogin()
    return
  }

  if (!codeVerifier) {
    error.value = 'PKCE 验证器丢失，请重新登录'
    redirectToLogin()
    return
  }

  const result = await kunFetch<{
    id: number
    name: string
    email: string
    avatar: string
    role: number
    moemoepoint: number
    bio: string
  }>('/auth/oauth/callback', {
    method: 'POST',
    body: { code, code_verifier: codeVerifier }
  })

  if (result) {
    const userStore = usePersistUserStore()
    userStore.setUserInfo({
      id: result.id,
      name: result.name,
      avatar: result.avatar,
      avatarMin: result.avatar
        ? result.avatar.replace(/\.webp$/, '-100.webp')
        : '',
      moemoepoint: result.moemoepoint,
      role: result.role,
      isCheckIn: false,
      dailyToolsetUploadCount: 0
    })

    useKunLoliInfo(`登录成功! 欢迎来到 ${kungal.name}`)
    await navigateTo('/')
  } else {
    error.value = '登录失败，请重试'
    redirectToLogin()
  }
})

const redirectToLogin = () => {
  setTimeout(() => navigateTo('/login'), 2000)
}
</script>

<template>
  <div class="flex size-full items-center justify-center">
    <div class="text-center">
      <p v-if="!error" class="text-lg">正在登录...</p>
      <p v-else class="text-red-500">{{ error }}</p>
    </div>
  </div>
</template>
