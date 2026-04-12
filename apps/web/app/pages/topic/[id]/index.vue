<script setup lang="ts">
import { KUN_TOPIC_SECTION } from '~/constants/topic'
import type {
  DiscussionForumPosting,
  WithContext,
  Person,
  Comment,
  InteractionCounter
} from 'schema-dts'

const route = useRoute()

const { isReplyRewriting } = storeToRefs(useTempReplyStore())
const { isEdit } = storeToRefs(useTempReplyStore())

const topicId = computed(() => {
  return parseInt((route.params as { id: string }).id)
})
provide<number>('topicId', topicId.value)

const { data } = await useKunFetch<TopicDetail>(
  `/topic/${topicId.value}`,
  {
    method: 'GET',
    watch: false,
    query: { topicId: topicId.value }
  }
)

onBeforeRouteLeave(async (_, __, next) => {
  if (isReplyRewriting.value) {
    const res =
      await useComponentMessageStore().alert(
        '确认离开界面吗？您的更改将不会保存。'
      )
    if (res) {
      useTempReplyStore().resetRewriteReplyData()
      isEdit.value = false
      next()
    } else {
      next(false)
    }
  } else {
    next()
  }
  isEdit.value = false
})

onBeforeMount(() => {
  isEdit.value = false
})

const getFirstImageSrc = (htmlString: string) => {
  const imgRegex = /<img[^>]+src="([^">]+)"/i
  const match = htmlString.match(imgRegex)

  return match ? match[1] : `${kungal.domain.main}/kungalgame.webp`
}

if (data.value && data.value !== 'banned') {
  const topic = data.value

  const markdown = topic.contentMarkdown
  const banner = getFirstImageSrc(topic.contentHtml)
  const created = new Date(topic.created).toString()
  const updated = topic.edited ? new Date(topic.edited).toString() : ''
  const description = computed(() =>
    markdownToText(markdown).trim().slice(0, 233).replace(/\\|\n/g, '')
  )

  const jsonLd = computed<WithContext<DiscussionForumPosting>>(() => {
    const topicUrl = `${kungal.domain.main}/topic/${topic.id}`

    const authorSchema: Person = {
      '@type': 'Person',
      name: topic.user.name,
      url: `${kungal.domain.main}/user/${topic.user.id}/info`,
      image: topic.user.avatar
    }

    const interactionStatistics: InteractionCounter[] = [
      {
        '@type': 'InteractionCounter',
        interactionType: {
          '@type': 'CommentAction'
        },
        userInteractionCount: topic.replyCount
      },
      {
        '@type': 'InteractionCounter',
        interactionType: {
          '@type': 'LikeAction'
        },
        userInteractionCount: topic.likeCount
      },
      {
        '@type': 'InteractionCounter',
        interactionType: {
          '@type': 'VoteAction'
        },
        userInteractionCount: topic.upvoteCount
      }
    ]

    // TODO: bestAnswer is no longer embedded in TopicDetail from Go backend.
    // Best answers are replies with isBestAnswer: true, loaded separately.
    let acceptedAnswerSchema: Comment | undefined = undefined

    return {
      '@context': 'https://schema.org',
      '@type': 'DiscussionForumPosting',
      mainEntityOfPage: topicUrl,
      headline: topic.title,
      description: description.value,
      image: banner,
      author: authorSchema,
      datePublished: new Date(topic.created).toISOString(),
      dateModified: topic.edited
        ? new Date(topic.edited).toISOString()
        : new Date(topic.created).toISOString(),
      interactionStatistic: interactionStatistics,
      commentCount: topic.replyCount,
      ...(acceptedAnswerSchema && { acceptedAnswer: acceptedAnswerSchema }),
      keywords: [
        ...topic.section.map((s) => KUN_TOPIC_SECTION[s]).filter(Boolean),
        ...topic.tag
      ].join(', ')
    }
  })

  useHead({
    script: [
      {
        id: 'schema-org-qa-page',
        type: 'application/ld+json',
        innerHTML: jsonLd.value
      }
    ]
  })

  if (topic.isNSFW) {
    useKunDisableSeo(topic.title)
  } else {
    useKunSeoMeta({
      title: data.value.title,
      description: description.value,
      ogImage: banner,
      ogType: 'article',
      articleAuthor: [`${kungal.domain.main}/user/${data.value.user.id}/info`],
      articlePublishedTime: created,
      articleModifiedTime: updated
    })
  }
} else {
  useKunDisableSeo(data.value ? '话题已被封禁' : '未找到此话题')
}
</script>

<template>
  <div>
    <TopicDetail v-if="data && data.status !== 1" :topic="data" />

    <KunNull v-if="!data" description="未找到这个话题" />

    <KunNull
      v-if="data && data.status === 1"
      description="话题被隐藏, 或您未开启网站 NSFW 模式"
    />
  </div>
</template>
