import { api, ensureMockInterceptionReady } from "@/lib/api-client"
import { buildBackendUrl } from "@/lib/env"
import type { LoginVisualDiscoverability, LoginVisualSettings, PublicLoginVisual } from "@/types/login-visual.types"

const settingsPath = "/settings/loginVisual"

function toAuthenticatedApiPath(previewURL: string): string {
  return previewURL.startsWith("/v1/") ? previewURL.slice("/v1".length) : previewURL
}

export class LoginVisualService {
  static async checkDiscoverability(): Promise<LoginVisualDiscoverability> {
    const response = await api.get<LoginVisualDiscoverability>(`${settingsPath}:checkDiscoverability`)
    return response.data
  }

  static async unlockDiscoverability(): Promise<LoginVisualDiscoverability> {
    const response = await api.post<LoginVisualDiscoverability>(`${settingsPath}:unlockDiscoverability`)
    return response.data
  }

  static async getSettings(): Promise<LoginVisualSettings> {
    const response = await api.get<LoginVisualSettings>(settingsPath)
    return response.data
  }

  static async upload(file: File): Promise<LoginVisualSettings> {
    const data = new FormData()
    data.set("file", file)
    const response = await api.post<LoginVisualSettings>(`${settingsPath}:upload`, data, {
      headers: { "Content-Type": "multipart/form-data" },
    })
    return response.data
  }

  static async publish(): Promise<LoginVisualSettings> {
    const response = await api.post<LoginVisualSettings>(`${settingsPath}:publish`)
    return response.data
  }

  static async restoreDefault(): Promise<LoginVisualSettings> {
    const response = await api.post<LoginVisualSettings>(`${settingsPath}:restoreDefault`)
    return response.data
  }

  static async getPreview(previewURL: string): Promise<Blob> {
    // Native media tags cannot attach the session token; keep this private read in api.
    const response = await api.get<Blob>(toAuthenticatedApiPath(previewURL), { responseType: "blob" })
    return response.data
  }

  static async getPublicCurrent(): Promise<PublicLoginVisual> {
    await ensureMockInterceptionReady()
    const response = await fetch(buildBackendUrl("/v1/loginVisual/current"), { cache: "no-store" })
    if (!response.ok) {
      throw new Error("Unable to load login visual")
    }
    return response.json() as Promise<PublicLoginVisual>
  }
}
