export interface DocTocLink {
  id: string
  text: string
  depth: number
  children?: DocTocLink[]
}

export interface DocCategoryItem {
  id: number
  slug: string
  title: string
  description: string
  icon: string
  sort_order: number
  created: Date | string
  updated: Date | string
}

export interface DocCategoryListResponse {
  categories: DocCategoryItem[]
  totalCount: number
  page: number
  limit: number
}

export interface DocTagItem {
  id: number
  slug: string
  title: string
  description: string
  created: Date | string
  updated: Date | string
}

export interface DocTagListResponse {
  tags: DocTagItem[]
  totalCount: number
  page: number
  limit: number
}

export interface DocArticle {
  id: number
  title: string
  slug: string
  path: string
  description: string
  banner: string
  status: number
  is_pin: boolean
  view: number
  published_time: Date | string
  edited_time: Date | string | null
  content_markdown: string
  category_id: number
  author_id: number
  created: Date | string
  updated: Date | string
  // TODO: server-side markdown rendering
  contentHtml?: string
  toc?: DocTocLink[]
}

// Kept for backward compatibility with components that use the old shape
export type DocArticleSummary = DocArticle
export type DocArticleDetail = DocArticle

export interface DocArticleListResponse {
  items: DocArticle[]
  total: number
}
