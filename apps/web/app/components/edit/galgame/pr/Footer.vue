<script setup lang="ts">
import { updateGalgameSchema } from '~/validations/galgame'

const { galgamePR } = storeToRefs(useTempGalgamePRStore())

const isPublishing = ref(false)

const handlePublishGalgamePR = async () => {
  const galgame = galgamePR.value[0]
  if (!galgame) return

  const data: Record<string, number | string | string[]> = {
    vndbId: galgame.vndbId,
    name_en_us: galgame.name['en-us'],
    name_ja_jp: galgame.name['ja-jp'],
    name_zh_cn: galgame.name['zh-cn'],
    name_zh_tw: galgame.name['zh-tw'],
    intro_en_us: galgame.introduction['en-us'],
    intro_ja_jp: galgame.introduction['ja-jp'],
    intro_zh_cn: galgame.introduction['zh-cn'],
    intro_zh_tw: galgame.introduction['zh-tw'],
    contentLimit: galgame.contentLimit,
    aliases: String(galgame.alias)
  }

  const result = updateGalgameSchema.safeParse(data)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(
      `位置: ${message.path[0]} - 错误提示: ${message.message}`,
      'warn'
    )
    return
  }
  const res = await useComponentMessageStore().alert(
    '确定发布 Galgame 信息更新请求吗?'
  )
  if (!res) {
    return
  }

  if (isPublishing.value) {
    return
  } else {
    isPublishing.value = true
  }

  // 可选 banner：用户在 PR 页改了 banner 就一并提交，没改则只发 JSON 字段。
  // multipart 约定见 docs/galgame_wiki/api-reference.md "Banner 上传"。
  const banner = await getImage('kun-galgame-publish-banner')

  let response: unknown
  if (banner instanceof Blob) {
    const formData = new FormData()
    formData.append('data', JSON.stringify(data))
    formData.append('file', banner)
    response = await kunFetch(`/galgame/${galgame.id}/prs`, {
      method: 'POST',
      body: formData
    })
  } else {
    response = await kunFetch(`/galgame/${galgame.id}/prs`, {
      method: 'POST',
      body: data
    })
  }
  isPublishing.value = false

  if (response) {
    if (banner instanceof Blob) {
      await deleteImage('kun-galgame-publish-banner')
    }
    useKunLoliInfo('创建更新请求成功', 5)
    await navigateTo(`/galgame/${galgame.id}`, {
      replace: true
    })
  }
}
</script>

<template>
  <div class="flex justify-end">
    <KunButton
      :disabled="isPublishing"
      :loading="isPublishing"
      size="lg"
      @click="handlePublishGalgamePR"
    >
      确定发布
    </KunButton>
  </div>
</template>
