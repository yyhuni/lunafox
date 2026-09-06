import type { LoginVisualDiscoverability, LoginVisualSettings, PublicLoginVisual } from "@/types/login-visual.types"

const mockPreview = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 1600 900'%3E%3Crect width='1600' height='900' fill='%230b1220'/%3E%3Cpath d='M0 640 360 360l240 180 280-300 720 400v260H0z' fill='%231a365d'/%3E%3Ccircle cx='1230' cy='210' r='100' fill='%2338bdf8' fill-opacity='.55'/%3E%3C/svg%3E"

let settings: LoginVisualSettings = {}
let discoverability: LoginVisualDiscoverability = { unlocked: false }

export function getMockLoginVisualDiscoverability(): LoginVisualDiscoverability {
  return { ...discoverability }
}

export function unlockMockLoginVisualDiscoverability(): LoginVisualDiscoverability {
  discoverability = { unlocked: true }
  return getMockLoginVisualDiscoverability()
}

export function resetMockLoginVisualDiscoverability() {
  discoverability = { unlocked: false }
}

export function getMockLoginVisualSettings(): LoginVisualSettings {
  return { ...settings, draft: settings.draft ? { ...settings.draft } : undefined, published: settings.published ? { ...settings.published } : undefined }
}

export function uploadMockLoginVisual(contentType: string): LoginVisualSettings {
  const kind = contentType.startsWith("video/") ? "video" : "image"
  settings = {
    ...settings,
    draft: { kind, contentType: contentType || "image/png", sizeBytes: 1024 },
    previewUrl: mockPreview,
  }
  return getMockLoginVisualSettings()
}

export function publishMockLoginVisual(): LoginVisualSettings {
  if (settings.draft) settings = { published: settings.draft, previewUrl: mockPreview }
  return getMockLoginVisualSettings()
}

export function restoreMockLoginVisual(): LoginVisualSettings {
  settings = {}
  return getMockLoginVisualSettings()
}

export function getMockPublicLoginVisual(): PublicLoginVisual {
  if (!settings.published) return { kind: "builtIn" }
  return {
    kind: settings.published.kind,
    mediaUrl: mockPreview,
    ...(settings.published.kind === "video" ? { posterUrl: mockPreview } : {}),
  }
}
