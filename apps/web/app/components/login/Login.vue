<script setup lang="ts">
const config = useRuntimeConfig()

const isLoading = ref(false)

const handleOAuthLogin = async () => {
  isLoading.value = true

  const codeVerifier = generateCodeVerifier()
  const codeChallenge = await generateCodeChallenge(codeVerifier)
  const state = generateState()

  sessionStorage.setItem('oauth_code_verifier', codeVerifier)
  sessionStorage.setItem('oauth_state', state)

  const params = new URLSearchParams({
    client_id: config.public.oauthClientId as string,
    redirect_uri: config.public.oauthRedirectUri as string,
    response_type: 'code',
    scope: 'openid profile',
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256'
  })

  window.location.href = `${config.public.oauthServerUrl}/oauth/authorize?${params}`
}
</script>

<template>
  <div class="flex size-full items-center justify-center">
    <DocDetailBackgroundImage src="/login.webp" />

    <KunCard
      :is-transparent="false"
      :is-hoverable="false"
      class-name="w-88 p-8 select-none"
    >
      <div class="flex h-full flex-col justify-center">
        <div class="my-6">
          <h1 class="mb-6 flex items-center gap-2 text-2xl">
            <KunImage
              src="/favicon.webp"
              class-name="h-8 w-8 rounded-2xl"
            />登录
          </h1>
          <p class="text-default-500 mb-1">你好呀鲲的朋友! 欢迎回家!</p>
          <p class="text-default-500">
            {{ `${kungal.titleShort}给你最温暖的拥抱!` }}
          </p>
        </div>

        <KunButton
          @click="handleOAuthLogin"
          :disabled="isLoading"
          class="bg-primary w-full rounded-3xl text-base tracking-wider text-white uppercase"
        >
          {{ isLoading ? '跳转中...' : '使用 KUN 账号登录' }}
        </KunButton>

        <p class="text-default-400 mt-4 text-center text-xs">
          点击登录将跳转至鲲 Galgame OAuth 统一认证系统
        </p>
      </div>
    </KunCard>
  </div>
</template>
