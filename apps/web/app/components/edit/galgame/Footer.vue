<script setup lang="ts">
import { createGalgameSchema } from '~/validations/galgame'

const {
  vndbId,
  name,
  contentLimit,
  ageLimit,
  originalLanguage,
  introduction,
  aliases
} = storeToRefs(usePersistEditGalgameStore())

const isPublishing = ref(false)

const handlePublishGalgame = async () => {
  const banner = await getImage('kun-galgame-publish-banner')
  // Wire-format payload uses snake_case keys to match the wiki API
  // (POST /galgame). The Vue store keeps camelCase locally; we rename
  // at the boundary so the schema, the JSON body, and the wiki contract
  // all agree.
  const data: Record<string, number | string | string[] | Blob | null> = {
    vndb_id: vndbId.value,
    name_en_us: name.value['en-us'],
    name_ja_jp: name.value['ja-jp'],
    name_zh_cn: name.value['zh-cn'],
    name_zh_tw: name.value['zh-tw'],
    intro_en_us: introduction.value['en-us'],
    intro_ja_jp: introduction.value['ja-jp'],
    intro_zh_cn: introduction.value['zh-cn'],
    intro_zh_tw: introduction.value['zh-tw'],
    content_limit: contentLimit.value,
    age_limit: ageLimit.value,
    original_language: originalLanguage.value,
    banner,
    aliases: String(aliases.value)
  }
  const result = createGalgameSchema.safeParse(data)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(
      `位置: ${message.path[0]} - 错误提示: ${message.message}`,
      'warn'
    )
    return
  }
  const res = await useComponentMessageStore().alert(
    '确定发布 Galgame 吗?',
    '您要发布的是 Galgame。发布后, 您必须到您发布完成的 Galgame 资源详情页, 添加一条该Galgame 资源的获取 / 下载链接。'
  )
  if (!res) {
    return
  }

  if (isPublishing.value) {
    return
  } else {
    isPublishing.value = true
    useMessage(10525, 'info', 7777)
  }

  // Wiki 新约定 (docs/galgame_wiki/01-galgame.md "Banner 上传"):
  //   data: 整个 JSON 串
  //   file: 可选图片二进制
  const { banner: _bannerBlob, ...jsonFields } = data
  const formData = new FormData()
  formData.append('data', JSON.stringify(jsonFields))
  if (banner instanceof Blob) {
    formData.append('file', banner)
  }
  // POST /galgame returns the created galgame object (`{id, vndb_id, ...}`);
  // extract `id` for the redirect rather than interpolating the whole object.
  const created = await kunFetch<{ id: number }>('/galgame', {
    method: 'POST',
    body: formData
  })
  isPublishing.value = false

  if (created?.id) {
    await deleteImage('kun-galgame-publish-banner')

    useKunLoliInfo('发布 Galgame 成功', 5)
    await navigateTo(`/galgame/${created.id}`)
    usePersistEditGalgameStore().resetEditGalgameStore()
  }
}
</script>

<template>
  <div class="flex justify-end">
    <KunButton
      :disabled="isPublishing"
      :loading="isPublishing"
      size="lg"
      @click="handlePublishGalgame"
    >
      确认发布 Galgame
    </KunButton>
  </div>
</template>
