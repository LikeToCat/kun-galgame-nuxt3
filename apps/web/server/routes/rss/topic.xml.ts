import { fetchKunApi, useKunFeed } from '../../utils/kunFeed'

interface TopicRSSItem {
  id: number
  title: string
  description: string
  userId: number
  userName: string
  created: string
}

export default defineEventHandler(async (event) => {
  const baseUrl = useRuntimeConfig().public.KUN_GALGAME_URL || ''
  const feed = useKunFeed(baseUrl, 'topic')

  const topics = await fetchKunApi<TopicRSSItem[]>('/rss/topic')

  for (const t of topics) {
    feed.addItem({
      link: `${baseUrl}/topic/${t.id}`,
      title: t.title,
      date: new Date(t.created),
      description: t.description,
      author: [
        {
          name: t.userName,
          link: `${baseUrl}/user/${t.userId}/info`
        }
      ]
    })
  }

  setHeader(event, 'Content-Type', 'application/xml')
  return feed.rss2()
})
