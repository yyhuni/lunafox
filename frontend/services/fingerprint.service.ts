/**
 * Fingerprint management API service
 */

import apiClient from "@/lib/api-client"
import type { PaginatedResponse } from "@/types/api-response.types"
import type { 
  EholeFingerprint,
  GobyFingerprint,
  WappalyzerFingerprint,
  FingersFingerprint,
  FingerPrintHubFingerprint,
  ARLFingerprint,
  BatchCreateResponse, 
  BulkDeleteResponse,
  FingerprintStats 
} from "@/types/fingerprint.types"

// Paginated query parameters
interface QueryParams {
  page?: number
  pageSize?: number
  filter?: string
}

export const FingerprintService = {
  // ==================== EHole ====================
  
  /**
   * Get EHole fingerprint list
   */
  async getEholeFingerprints(params: QueryParams = {}): Promise<PaginatedResponse<EholeFingerprint>> {
    const response = await apiClient.get("/fingerprints/ehole/", { params })
    return response.data
  },

  /**
   * Get EHole fingerprint details
   */
  async getEholeFingerprint(id: number): Promise<EholeFingerprint> {
    const response = await apiClient.get(`/fingerprints/ehole/${id}/`)
    return response.data
  },

  /**
   * Create single EHole fingerprint
   */
  async createEholeFingerprint(data: Omit<EholeFingerprint, 'id' | 'createdAt'>): Promise<EholeFingerprint> {
    const response = await apiClient.post("/fingerprints/ehole/", data)
    return response.data
  },

  /**
   * Update EHole fingerprint
   */
  async updateEholeFingerprint(id: number, data: Partial<EholeFingerprint>): Promise<EholeFingerprint> {
    const response = await apiClient.put(`/fingerprints/ehole/${id}/`, data)
    return response.data
  },

  /**
   * Delete single EHole fingerprint
   */
  async deleteEholeFingerprint(id: number): Promise<void> {
    await apiClient.delete(`/fingerprints/ehole/${id}/`)
  },

  /**
   * Batch create EHole fingerprints
   */
  async batchCreateEholeFingerprints(fingerprints: Omit<EholeFingerprint, 'id' | 'createdAt'>[]): Promise<BatchCreateResponse> {
    const response = await apiClient.post("/fingerprints/ehole/batch_create/", { fingerprints })
    return response.data
  },

  /**
   * File import EHole fingerprints
   */
  async importEholeFingerprints(file: File): Promise<BatchCreateResponse> {
    const formData = new FormData()
    formData.append("file", file)
    const response = await apiClient.post("/fingerprints/ehole/import_file/", formData, {
      headers: { "Content-Type": "multipart/form-data" }
    })
    return response.data
  },

  /**
   * Bulk delete EHole fingerprints
   */
  async bulkDeleteEholeFingerprints(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/ehole/bulk-delete/", { ids })
    return response.data
  },

  /**
   * Delete all EHole fingerprints
   */
  async deleteAllEholeFingerprints(): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/ehole/delete-all/")
    return response.data
  },

  /**
   * Export EHole fingerprints
   */
  async exportEholeFingerprints(): Promise<Blob> {
    const response = await apiClient.get("/fingerprints/ehole/export/", {
      responseType: "blob"
    })
    return response.data
  },

  /**
   * Get EHole fingerprint count
   */
  async getEholeCount(): Promise<number> {
    const response = await apiClient.get("/fingerprints/ehole/", { params: { pageSize: 1 } })
    return response.data.total || 0
  },

  // ==================== Goby ====================
  
  /**
   * Get Goby fingerprint list
   */
  async getGobyFingerprints(params: QueryParams = {}): Promise<PaginatedResponse<GobyFingerprint>> {
    const response = await apiClient.get("/fingerprints/goby/", { params })
    return response.data
  },

  /**
   * Get Goby fingerprint details
   */
  async getGobyFingerprint(id: number): Promise<GobyFingerprint> {
    const response = await apiClient.get(`/fingerprints/goby/${id}/`)
    return response.data
  },

  /**
   * Create a single Goby fingerprint
   */
  async createGobyFingerprint(data: Omit<GobyFingerprint, 'id' | 'createdAt'>): Promise<GobyFingerprint> {
    const response = await apiClient.post("/fingerprints/goby/", data)
    return response.data
  },

  /**
   * Update Goby fingerprint
   */
  async updateGobyFingerprint(id: number, data: Partial<GobyFingerprint>): Promise<GobyFingerprint> {
    const response = await apiClient.put(`/fingerprints/goby/${id}/`, data)
    return response.data
  },

  /**
   * Delete a single Goby fingerprint
   */
  async deleteGobyFingerprint(id: number): Promise<void> {
    await apiClient.delete(`/fingerprints/goby/${id}/`)
  },

  /**
   * File import Goby fingerprint
   */
  async importGobyFingerprints(file: File): Promise<BatchCreateResponse> {
    const formData = new FormData()
    formData.append("file", file)
    const response = await apiClient.post("/fingerprints/goby/import_file/", formData, {
      headers: { "Content-Type": "multipart/form-data" }
    })
    return response.data
  },

  /**
   * Delete Goby fingerprints in batches
   */
  async bulkDeleteGobyFingerprints(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/goby/bulk-delete/", { ids })
    return response.data
  },

  /**
   * Delete all Goby fingerprints
   */
  async deleteAllGobyFingerprints(): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/goby/delete-all/")
    return response.data
  },

  /**
   * Export Goby fingerprint
   */
  async exportGobyFingerprints(): Promise<Blob> {
    const response = await apiClient.get("/fingerprints/goby/export/", {
      responseType: "blob"
    })
    return response.data
  },

  /**
   * Get the number of Goby fingerprints
   */
  async getGobyCount(): Promise<number> {
    const response = await apiClient.get("/fingerprints/goby/", { params: { pageSize: 1 } })
    return response.data.total || 0
  },

  // ==================== Wappalyzer ====================
  
  /**
   * Get Wappalyzer fingerprint list
   */
  async getWappalyzerFingerprints(params: QueryParams = {}): Promise<PaginatedResponse<WappalyzerFingerprint>> {
    const response = await apiClient.get("/fingerprints/wappalyzer/", { params })
    return response.data
  },

  /**
   * Get Wappalyzer fingerprint details
   */
  async getWappalyzerFingerprint(id: number): Promise<WappalyzerFingerprint> {
    const response = await apiClient.get(`/fingerprints/wappalyzer/${id}/`)
    return response.data
  },

  /**
   * Create a single Wappalyzer fingerprint
   */
  async createWappalyzerFingerprint(data: Omit<WappalyzerFingerprint, 'id' | 'createdAt'>): Promise<WappalyzerFingerprint> {
    const response = await apiClient.post("/fingerprints/wappalyzer/", data)
    return response.data
  },

  /**
   * Update Wappalyzer fingerprint
   */
  async updateWappalyzerFingerprint(id: number, data: Partial<WappalyzerFingerprint>): Promise<WappalyzerFingerprint> {
    const response = await apiClient.put(`/fingerprints/wappalyzer/${id}/`, data)
    return response.data
  },

  /**
   * Delete a single Wappalyzer fingerprint
   */
  async deleteWappalyzerFingerprint(id: number): Promise<void> {
    await apiClient.delete(`/fingerprints/wappalyzer/${id}/`)
  },

  /**
   * File import Wappalyzer fingerprint
   */
  async importWappalyzerFingerprints(file: File): Promise<BatchCreateResponse> {
    const formData = new FormData()
    formData.append("file", file)
    const response = await apiClient.post("/fingerprints/wappalyzer/import_file/", formData, {
      headers: { "Content-Type": "multipart/form-data" }
    })
    return response.data
  },

  /**
   * Delete Wappalyzer fingerprints in batches
   */
  async bulkDeleteWappalyzerFingerprints(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/wappalyzer/bulk-delete/", { ids })
    return response.data
  },

  /**
   * Remove all Wappalyzer fingerprints
   */
  async deleteAllWappalyzerFingerprints(): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/wappalyzer/delete-all/")
    return response.data
  },

  /**
   * Export Wappalyzer fingerprints
   */
  async exportWappalyzerFingerprints(): Promise<Blob> {
    const response = await apiClient.get("/fingerprints/wappalyzer/export/", {
      responseType: "blob"
    })
    return response.data
  },

  /**
   * Get the number of Wappalyzer fingerprints
   */
  async getWappalyzerCount(): Promise<number> {
    const response = await apiClient.get("/fingerprints/wappalyzer/", { params: { pageSize: 1 } })
    return response.data.total || 0
  },

  // ==================== Fingers ====================

  /**
   * Get Fingers fingerprint list
   */
  async getFingersFingerprints(params: QueryParams = {}): Promise<PaginatedResponse<FingersFingerprint>> {
    const response = await apiClient.get("/fingerprints/fingers/", { params })
    return response.data
  },

  /**
   * Get Fingers fingerprint details
   */
  async getFingersFingerprint(id: number): Promise<FingersFingerprint> {
    const response = await apiClient.get(`/fingerprints/fingers/${id}/`)
    return response.data
  },

  /**
   * Create a single Fingerprint
   */
  async createFingersFingerprint(data: Omit<FingersFingerprint, 'id' | 'createdAt'>): Promise<FingersFingerprint> {
    const response = await apiClient.post("/fingerprints/fingers/", data)
    return response.data
  },

  /**
   * Update Fingers
   */
  async updateFingersFingerprint(id: number, data: Partial<FingersFingerprint>): Promise<FingersFingerprint> {
    const response = await apiClient.put(`/fingerprints/fingers/${id}/`, data)
    return response.data
  },

  /**
   * Delete a single Fingerprint
   */
  async deleteFingersFingerprint(id: number): Promise<void> {
    await apiClient.delete(`/fingerprints/fingers/${id}/`)
  },

  /**
   * File import Fingers fingerprint
   */
  async importFingersFingerprints(file: File): Promise<BatchCreateResponse> {
    const formData = new FormData()
    formData.append("file", file)
    const response = await apiClient.post("/fingerprints/fingers/import_file/", formData, {
      headers: { "Content-Type": "multipart/form-data" }
    })
    return response.data
  },

  /**
   * Delete Fingers in batches
   */
  async bulkDeleteFingersFingerprints(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/fingers/bulk-delete/", { ids })
    return response.data
  },

  /**
   * Delete all Fingers
   */
  async deleteAllFingersFingerprints(): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/fingers/delete-all/")
    return response.data
  },

  /**
   * Export Fingers
   */
  async exportFingersFingerprints(): Promise<Blob> {
    const response = await apiClient.get("/fingerprints/fingers/export/", {
      responseType: "blob"
    })
    return response.data
  },

  /**
   * Get the number of Fingers
   */
  async getFingersCount(): Promise<number> {
    const response = await apiClient.get("/fingerprints/fingers/", { params: { pageSize: 1 } })
    return response.data.total || 0
  },

  // ==================== FingerPrintHub ====================

  /**
   * Get FingerPrintHub fingerprint list
   */
  async getFingerPrintHubFingerprints(params: QueryParams = {}): Promise<PaginatedResponse<FingerPrintHubFingerprint>> {
    const response = await apiClient.get("/fingerprints/fingerprinthub/", { params })
    return response.data
  },

  /**
   * Get FingerPrintHub fingerprint details
   */
  async getFingerPrintHubFingerprint(id: number): Promise<FingerPrintHubFingerprint> {
    const response = await apiClient.get(`/fingerprints/fingerprinthub/${id}/`)
    return response.data
  },

  /**
   * Create a single FingerPrintHub fingerprint
   */
  async createFingerPrintHubFingerprint(data: Omit<FingerPrintHubFingerprint, 'id' | 'createdAt'>): Promise<FingerPrintHubFingerprint> {
    const response = await apiClient.post("/fingerprints/fingerprinthub/", data)
    return response.data
  },

  /**
   * Update FingerPrintHub fingerprint
   */
  async updateFingerPrintHubFingerprint(id: number, data: Partial<FingerPrintHubFingerprint>): Promise<FingerPrintHubFingerprint> {
    const response = await apiClient.put(`/fingerprints/fingerprinthub/${id}/`, data)
    return response.data
  },

  /**
   * Delete a single FingerPrintHub fingerprint
   */
  async deleteFingerPrintHubFingerprint(id: number): Promise<void> {
    await apiClient.delete(`/fingerprints/fingerprinthub/${id}/`)
  },

  /**
   * File import FingerPrintHub fingerprint
   */
  async importFingerPrintHubFingerprints(file: File): Promise<BatchCreateResponse> {
    const formData = new FormData()
    formData.append("file", file)
    const response = await apiClient.post("/fingerprints/fingerprinthub/import_file/", formData, {
      headers: { "Content-Type": "multipart/form-data" }
    })
    return response.data
  },

  /**
   * Delete FingerPrintHub fingerprints in batches
   */
  async bulkDeleteFingerPrintHubFingerprints(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/fingerprinthub/bulk-delete/", { ids })
    return response.data
  },

  /**
   * Delete all FingerPrintHub fingerprints
   */
  async deleteAllFingerPrintHubFingerprints(): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/fingerprinthub/delete-all/")
    return response.data
  },

  /**
   * Export FingerPrintHub fingerprints
   */
  async exportFingerPrintHubFingerprints(): Promise<Blob> {
    const response = await apiClient.get("/fingerprints/fingerprinthub/export/", {
      responseType: "blob"
    })
    return response.data
  },

  /**
   * Get the number of FingerPrintHub fingerprints
   */
  async getFingerPrintHubCount(): Promise<number> {
    const response = await apiClient.get("/fingerprints/fingerprinthub/", { params: { pageSize: 1 } })
    return response.data.total || 0
  },

  // ==================== ARL ====================

  /**
   * Get a list of ARL fingerprints
   */
  async getARLFingerprints(params: QueryParams = {}): Promise<PaginatedResponse<ARLFingerprint>> {
    const response = await apiClient.get("/fingerprints/arl/", { params })
    return response.data
  },

  /**
   * Get ARL fingerprint details
   */
  async getARLFingerprint(id: number): Promise<ARLFingerprint> {
    const response = await apiClient.get(`/fingerprints/arl/${id}/`)
    return response.data
  },

  /**
   * Create a single ARL fingerprint
   */
  async createARLFingerprint(data: Omit<ARLFingerprint, 'id' | 'createdAt'>): Promise<ARLFingerprint> {
    const response = await apiClient.post("/fingerprints/arl/", data)
    return response.data
  },

  /**
   * Update ARL fingerprint
   */
  async updateARLFingerprint(id: number, data: Partial<ARLFingerprint>): Promise<ARLFingerprint> {
    const response = await apiClient.put(`/fingerprints/arl/${id}/`, data)
    return response.data
  },

  /**
   * Delete a single ARL fingerprint
   */
  async deleteARLFingerprint(id: number): Promise<void> {
    await apiClient.delete(`/fingerprints/arl/${id}/`)
  },

  /**
   * File import ARL fingerprint (supports YAML and JSON)
   */
  async importARLFingerprints(file: File): Promise<BatchCreateResponse> {
    const formData = new FormData()
    formData.append("file", file)
    const response = await apiClient.post("/fingerprints/arl/import_file/", formData, {
      headers: { "Content-Type": "multipart/form-data" }
    })
    return response.data
  },

  /**
   * Delete ARL fingerprints in batches
   */
  async bulkDeleteARLFingerprints(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/arl/bulk-delete/", { ids })
    return response.data
  },

  /**
   * Remove all ARL fingerprints
   */
  async deleteAllARLFingerprints(): Promise<BulkDeleteResponse> {
    const response = await apiClient.post("/fingerprints/arl/delete-all/")
    return response.data
  },

  /**
   * Export ARL fingerprint (YAML format)
   */
  async exportARLFingerprints(): Promise<Blob> {
    const response = await apiClient.get("/fingerprints/arl/export/", {
      responseType: "blob"
    })
    return response.data
  },

  /**
   * Get the number of ARL fingerprints
   */
  async getARLCount(): Promise<number> {
    const response = await apiClient.get("/fingerprints/arl/", { params: { pageSize: 1 } })
    return response.data.total || 0
  },

  // ==================== Statistics ====================

  /**
   * Get all fingerprint database statistics
   */
  async getStats(): Promise<FingerprintStats> {
    // Obtain the number of each fingerprint database in parallel
    const [eholeCount, gobyCount, wappalyzerCount, fingersCount, fingerprinthubCount, arlCount] = await Promise.all([
      this.getEholeCount(),
      this.getGobyCount(),
      this.getWappalyzerCount(),
      this.getFingersCount(),
      this.getFingerPrintHubCount(),
      this.getARLCount(),
    ])
    return {
      ehole: eholeCount,
      goby: gobyCount,
      wappalyzer: wappalyzerCount,
      fingers: fingersCount,
      fingerprinthub: fingerprinthubCount,
      arl: arlCount,
    }
  },
}
