export interface GalgamePR {
  id: number
  galgameId: number
  status: number
  note: string
  baseRevision: number
  user: KunUser
  completedTime: Date | string | null
  created: Date | string
}

export interface GalgamePRDetails extends GalgamePR {
  snapshot: Record<string, unknown>
}
