import type {
  KunGalgameResourceTypeOptions,
  KunGalgameResourceLanguageOptions,
  KunGalgameResourcePlatformOptions
} from '~/constants/galgame'
import type { GalgameSeries } from './galgame-series'
import type { GalgameEngineItem } from './galgame-engine'
import type { GalgameOfficialItem } from './galgame-official'
import type { GalgameTagItem } from './galgame-tag'
import type { GalgameRatingCardOnGalgamePage } from './galgame-rating'

export interface GalgameDetailTag extends GalgameTagItem {
  spoilerLevel: number
}

export interface GalgameDetail {
  id: number
  vndbId: string
  user: KunUser
  name: KunLanguage
  banner: string
  introduction: KunLanguage
  contentLimit: string
  markdown: KunLanguage
  resourceUpdateTime: Date | string
  view: number
  originalLanguage: string
  ageLimit: 'all' | 'r18'
  platform: string[]
  language: string[]
  type: string[]
  contributor: KunUser[]
  likeCount: number
  isLiked: boolean
  favoriteCount: number
  isFavorited: boolean
  alias: string[]
  series: GalgameSeries | null
  engine: GalgameEngineItem[]
  official: GalgameOfficialItem[]
  tag: GalgameDetailTag[]
  ratings: GalgameRatingCardOnGalgamePage[]
  created: Date | string
  updated: Date | string
}

export interface GalgamePageRequestData {
  page: string
  limit: string
  type: KunGalgameResourceTypeOptions
  language: KunGalgameResourceLanguageOptions
  platform: KunGalgameResourcePlatformOptions
  sortField: 'time' | 'views'
  sortOrder: KunOrder
}

export interface GalgameCard {
  id: number
  name: KunLanguage
  banner: string
  user: KunUser
  contentLimit: string
  view: number
  likeCount: number
  platform: string[]
  language: string[]
  resourceUpdateTime: Date | string
}

// MineGalgameItem matches the per-row shape of GET /api/galgame/mine.
// Wire-format keeps snake_case (passed through verbatim from wiki).
// See docs/galgame_wiki/07-submission.md §GET /galgame/mine.
export interface MineGalgameItem {
  id: number
  status: number
  vndb_id: string
  name_en_us: string
  name_ja_jp: string
  name_zh_cn: string
  name_zh_tw: string
  banner: string
  banner_image_hash: string
  created: string
  updated: string
}

export interface MineGalgameList {
  items: MineGalgameItem[]
  total: number
}

// Galgame draft status — see docs/galgame_wiki/07-submission.md §Status 取值.
export const GalgameStatus = {
  Published: 0,
  Banned: 1,
  VndbDraft: 2,
  Pending: 3,
  Declined: 4
} as const
