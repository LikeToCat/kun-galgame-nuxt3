export const KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM = [
  'user',
  'topic',
  'topic_reply',
  'topic_comment',
  'galgame',
  'galgame_resource',
  'galgame_comment',
  'galgame_website',
  'galgame_website_comment',
  'chat_message',
  'galgame_tag',
  'galgame_official',
  'galgame_engine',
  'galgame_series',
  'galgame_link',
  'galgame_pr',
  'galgame_revision'
] as const

export type StatsModelType =
  (typeof KUN_ADMIN_OVERVIEW_STATS_MODEL_ITEM)[number]

export interface ChartItem {
  label: string
  color: string
}

export const KUN_ADMIN_OVERVIEW_STATS_MODEL_MAP: Record<
  StatsModelType,
  ChartItem
> = {
  user: { label: '用户', color: '#006FEE' },
  topic: { label: '话题', color: '#7828C8' },
  topic_reply: { label: '话题回复', color: '#17C964' },
  topic_comment: { label: '话题评论', color: '#F31260' },
  galgame: { label: 'Galgame', color: '#FF4ECD' },
  galgame_resource: { label: 'Galgame 资源', color: '#F5A524' },
  galgame_comment: { label: 'Galgame 评论', color: '#7EE7FC' },
  galgame_website: { label: 'Galgame 网站', color: '#7ccf00' },
  galgame_website_comment: { label: 'Galgame 网站评论', color: '#ff637e' },
  chat_message: { label: '聊天消息', color: '#ff8904' },
  galgame_tag: { label: 'Galgame 标签', color: '#10b981' },
  galgame_official: { label: 'Galgame 会社', color: '#8b5cf6' },
  galgame_engine: { label: 'Galgame 引擎', color: '#ec4899' },
  galgame_series: { label: 'Galgame 系列', color: '#14b8a6' },
  galgame_link: { label: 'Galgame 链接', color: '#f97316' },
  galgame_pr: { label: 'Galgame PR', color: '#6366f1' },
  galgame_revision: { label: 'Galgame 编辑历史', color: '#84cc16' }
} as const

export const KUN_ADMIN_PAGE_ROUTE = ['overview', 'user', 'setting']

export type KUN_ADMIN_PAGE_ROUTE_TYPE = (typeof KUN_ADMIN_PAGE_ROUTE)[number]

export interface KunAdminPageAsideItem {
  name: KUN_ADMIN_PAGE_ROUTE_TYPE
  label: string
  icon?: string
  router?: KUN_ADMIN_PAGE_ROUTE_TYPE
}

export const KUN_ADMIN_PAGE_ASIDE_NAV_ITEM: KunAdminPageAsideItem[] = [
  {
    name: 'overview',
    label: '数据总览',
    icon: 'lucide:chart-area',
    router: 'overview'
  },
  {
    name: 'user',
    label: '用户管理',
    icon: 'lucide:user',
    router: 'user'
  },
  {
    name: 'setting',
    label: '网站设置',
    icon: 'lucide:settings',
    router: 'setting'
  }
]
