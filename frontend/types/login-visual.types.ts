export type LoginVisualKind = "image" | "video"

export interface LoginVisualMedia {
  kind: LoginVisualKind
  contentType: string
  sizeBytes: number
  durationMs?: number
}

export interface LoginVisualSettings {
  draft?: LoginVisualMedia
  published?: LoginVisualMedia
  previewUrl?: string
}

export interface LoginVisualDiscoverability {
  unlocked: boolean
}

export interface PublicLoginVisual {
  kind: LoginVisualKind | "builtIn"
  mediaUrl?: string
  posterUrl?: string
}
