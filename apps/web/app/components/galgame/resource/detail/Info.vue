<script setup lang="ts">
const { id, role } = usePersistUserStore()
const {
  isShowPublish,
  resource: storeResource,
  rewriteResourceId
} = storeToRefs(useTempGalgameResourceStore())

const props = defineProps<{
  resource: GalgameResource
  resourceTypeLabel: string
  refresh: () => void
}>()

// Backend-computed labels (e.g. "百度网盘 / OneDrive"). Falls back to the
// raw domain when the resource pre-dates the backfill or matches no rule.
const providerName = computed(() => {
  const names = props.resource.providerNames
  if (names && names.length > 0) {
    return names.join(' / ')
  }
  return props.resource.linkDomain
})

const isFetching = ref(false)
const detail = ref<null | GalgameResourceDetailLink>(null)
const isResourceExpired = computed(() => props.resource.status === 1)

const handleDeleteResource = async () => {
  const res = await useComponentMessageStore().alert(
    '您确定删除 Galgame 资源链接吗？',
    '这将会扣除您发布 Galgame 资源获得的 5 萌萌点，并且扣除其它人对资源链接的点赞影响（萌萌点和点赞数减一），此操作不可撤销。'
  )
  if (!res) {
    return
  }

  const result = await kunFetch(
    `/galgame/${props.resource.galgameId}/resource`,
    {
      method: 'DELETE',
      query: { galgameResourceId: props.resource.id }
    }
  )

  if (result) {
    useMessage('删除资源成功', 'success')
    await navigateTo(`/galgame/${props.resource.galgameId}`)
  }
}

const handleReportExpire = async () => {
  if (!id) {
    useMessage(10546, 'warn')
    return
  }

  const res = await useComponentMessageStore().alert(
    '您确定报告资源链接失效吗？',
    '这将会通知资源发布者链接失效, 并将该链接标记为失效。若 17 天内资源发布者没有更换有效链接，该资源链接将会被删除。若恶意报告失效, 将会被处罚。'
  )
  if (!res) {
    return
  }

  isFetching.value = true
  const result = await kunFetch(
    `/galgame/${props.resource.galgameId}/resource/expired`,
    {
      method: 'PUT',
      body: { galgameResourceId: props.resource.id }
    }
  )
  isFetching.value = false

  if (result) {
    useMessage(10547, 'success')
    props.refresh()
  }
}

const handleGetResourceLink = async () => {
  if (detail.value) return

  isFetching.value = true
  const result = await kunFetch(
    `/galgame-resource/${props.resource.id}/detail`,
    {
      method: 'GET',
      query: { galgameResourceId: props.resource.id }
    }
  )
  isFetching.value = false

  if (result) {
    detail.value = result
    props.refresh()
    return result
  }
}

const handleRewriteResource = async () => {
  const _handleEdit = () => {
    storeResource.value = detail.value!
    rewriteResourceId.value = detail.value!.id
    isShowPublish.value = true
  }

  if (detail.value) _handleEdit()

  const res = await handleGetResourceLink()
  if (res) _handleEdit()
}

</script>

<template>
  <div class="flex h-full flex-col gap-3" v-if="resource">
    <div class="flex items-center gap-2">
      <KunAvatar :user="resource.user" />
      <span>{{ resource.user.name }}</span>
      <span class="text-default-500 text-sm">
        {{ `发布于 ${formatTimeDifference(resource.created)}` }}
      </span>
    </div>

    <KunInfo
      variant="bordered"
      v-if="resource.note"
      color="info"
      title="下载备注信息"
    >
      <pre class="break-all whitespace-pre-line">
        {{ resource.note }}
      </pre>
    </KunInfo>

    <KunAdDZMMBanner class-name="block lg:hidden" />

    <KunInfo
      :color="isResourceExpired ? 'warning' : 'success'"
      variant="bordered"
      class-name="relative"
    >
      <template #title>
        <div class="flex w-full flex-wrap items-center gap-2">
          <span>
            {{ `${props.resourceTypeLabel}下载链接` }}
          </span>
          <span class="text-default-500 text-sm">{{ providerName }}</span>
          <KunButton
            class-name="ml-auto whitespace-nowrap"
            :color="isResourceExpired ? 'warning' : 'success'"
            :loading="isFetching"
            @click="handleGetResourceLink"
          >
            获取链接
          </KunButton>
        </div>
      </template>

      <template #default v-if="detail">
        <div class="space-y-3">
          <p class="text-default-500 text-sm">点击下面的链接以下载</p>
          <KunLink
            v-for="(kun, index) in detail.link"
            :key="index"
            :to="kun"
            target="_blank"
            rel="noopener noreferrer"
            :is-show-anchor-icon="true"
          >
            {{ kun }}
          </KunLink>

          <div class="flex items-center gap-2">
            <KunCopy
              variant="solid"
              :color="isResourceExpired ? 'warning' : 'success'"
              v-if="detail.code"
              :name="`提取码 ${detail.code}`"
              :text="detail.code"
            />
            <KunCopy
              variant="solid"
              :color="isResourceExpired ? 'warning' : 'success'"
              v-if="detail.password"
              :name="`解压码 ${detail.password}`"
              :text="detail.password"
            />
          </div>

          <KunInfo title="补票提示信息" color="danger">
            <p>
              须知 Galgame 厂商制作游戏不易, 很多厂商如今都在炒冷饭,
              可见经济并不宽裕。如果条件允许, 请尽可能前往
              <KunLink size="sm" :to="`/galgame/${resource.galgameId}`">
                Galgame 详情
              </KunLink>
              中的 Galgame 制作商部分 进行正版 Galgame 补票, 感谢您对 Galgame
              业界做出的贡献
            </p>
          </KunInfo>

          <div class="flex justify-end">
            <KunBadge
              :color="isResourceExpired ? 'danger' : 'success'"
              variant="solid"
            >
              {{
                isResourceExpired
                  ? '该资源链接被其它用户标记为失效'
                  : '该资源链接可用'
              }}
            </KunBadge>
          </div>
        </div>
      </template>
    </KunInfo>

    <KunInfo title="鲲的小请求">
      <p>
        在您下载这部 Galgame 并游玩之后, 可否请您在本网站的
        <KunLink size="sm" :to="`/galgame/${resource.galgameId}`">
          Galgame 评分页面
        </KunLink>
        为这部 Galgame 提交一个评分, 这将有助于我们把优秀的 Galgame
        推荐给更多人, 谢谢您的支持
      </p>
    </KunInfo>

    <div class="mt-auto flex flex-wrap items-center justify-end gap-1">
      <KunButton
        :is-icon-only="true"
        variant="flat"
        @click="handleRewriteResource"
        :loading="isFetching"
        v-if="resource.user.id === id || role > 1"
      >
        编辑资源
        <KunIcon name="lucide:pencil" />
      </KunButton>
      <KunButton
        :is-icon-only="true"
        color="danger"
        variant="flat"
        @click="handleDeleteResource"
        :loading="isFetching"
        v-if="resource.user.id === id || role > 1"
      >
        删除资源
        <KunIcon name="lucide:trash-2" />
      </KunButton>

      <div v-if="id !== resource.user.id && !resource.status">
        <KunButton
          variant="flat"
          color="danger"
          @click="handleReportExpire"
          :loading="isFetching"
        >
          报告链接过期
        </KunButton>
      </div>

      <KunButton variant="flat" :href="`/galgame/${resource.galgameId}`">
        反馈资源问题
      </KunButton>
    </div>

    <KunModal
      :is-dismissable="false"
      :is-show-close-button="false"
      :modal-value="isShowPublish"
      @update:modal-value="(value) => (isShowPublish = value)"
    >
      <GalgameResourcePublish
        :refresh="refresh"
        @close="isShowPublish = false"
        :galgame-id="resource.galgameId"
      />
    </KunModal>
  </div>
</template>
