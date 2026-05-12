<script setup lang="ts">
// Wiki accepts these fields on POST /galgame; defaults (ja-jp / r18) are
// fine for most Japanese visual novels but we expose them as choices so
// authors can correct non-default games (English VNs, all-ages titles) at
// publish time instead of needing a follow-up PR. See audit batch 2 §10.

const { ageLimit, originalLanguage } = storeToRefs(usePersistEditGalgameStore())

const ageLimitOptions = [
  { value: 'all', label: '全年龄 (本游戏不含成人内容)' },
  { value: 'r18', label: 'R18 (本游戏含成人内容)' }
] as const

const originalLanguageOptions = [
  { value: 'ja-jp', label: '日语' },
  { value: 'en-us', label: '英语' },
  { value: 'zh-cn', label: '简体中文' },
  { value: 'zh-tw', label: '繁体中文' },
  { value: 'others', label: '其它' }
] as const
</script>

<template>
  <div class="space-y-2">
    <h2 class="text-xl">游戏基础信息</h2>
    <p class="text-default-500 text-sm">
      默认假设是日语原版、含成人内容的视觉小说。如果游戏与默认不同，请在此调整。
    </p>
    <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
      <KunSelect
        v-model="ageLimit"
        label="年龄分级"
        :options="ageLimitOptions"
      />
      <KunSelect
        v-model="originalLanguage"
        label="原始语言"
        :options="originalLanguageOptions"
      />
    </div>
  </div>
</template>
