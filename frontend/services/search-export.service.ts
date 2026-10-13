import { api } from "@/lib/api-client"
import type { AssetType } from "@/types/search.types"

export interface GlobalAssetExportRequest {
  q: string
  assetType: Extract<AssetType, "website" | "endpoint">
}

export interface GlobalAssetExportGrantResponse {
  downloadUrl: string
  expiresAt: string
}

/**
 * The grant request is authenticated by api-client; the returned URL is safe
 * to hand to the browser because it contains only a short-lived, one-use grant.
 */
export class SearchExportService {
  static async issueDownloadGrant(
    request: GlobalAssetExportRequest,
  ): Promise<GlobalAssetExportGrantResponse> {
    if (!request.q.trim()) {
      throw new Error("Global asset export query is required")
    }

    const response = await api.post<GlobalAssetExportGrantResponse>("/assets:export", request)
    const grant = response.data
    if (!grant || typeof grant.downloadUrl !== "string" || !grant.downloadUrl.trim()) {
      throw new Error("Global asset export response did not include a download URL")
    }
    if (typeof grant.expiresAt !== "string" || !grant.expiresAt.trim() || !Number.isFinite(Date.parse(grant.expiresAt))) {
      throw new Error("Global asset export response did not include a valid expiry")
    }
    return grant
  }

  static startNativeDownload(downloadUrl: string): void {
    if (!downloadUrl.trim()) {
      throw new Error("Global asset export download URL is required")
    }
    const anchor = document.createElement("a")
    anchor.href = downloadUrl
    anchor.rel = "noreferrer"
    anchor.style.display = "none"
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
  }

  static async exportAll(request: GlobalAssetExportRequest): Promise<void> {
    const grant = await this.issueDownloadGrant(request)
    this.startNativeDownload(grant.downloadUrl)
  }
}
