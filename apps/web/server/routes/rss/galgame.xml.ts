import { fetchKunApi, useKunFeed } from '../../utils/kunFeed'

interface GalgameRSSItem {
  id: number
  name: string
  banner: string
  user: { id: number; name: string; avatar: string }
  description: string
  created: string
}

export default defineEventHandler(async (event) => {
  const baseUrl = useRuntimeConfig().public.KUN_GALGAME_URL || ''
  const feed = useKunFeed(baseUrl, 'galgame')

  const items = await fetchKunApi<GalgameRSSItem[]>('/rss/galgame')

  for (const g of items) {
    feed.addItem({
      link: `${baseUrl}/galgame/${g.id}`,
      title: g.name,
      date: new Date(g.created),
      description: g.description,
      image: g.banner,
      author: [
        {
          name: g.user.name,
          link: `${baseUrl}/user/${g.user.id}/info`
        }
      ]
    })
  }

  setHeader(event, 'Content-Type', 'application/xml')
  return feed.rss2()
})
