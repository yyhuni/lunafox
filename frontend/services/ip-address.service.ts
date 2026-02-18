import { api } from "@/lib/api-client"
import { USE_MOCK, mockDelay, mockIPAddresses, getMockTargetById, getMockScanById } from "@/mock"
import type { GetIPAddressesParams, GetIPAddressesResponse } from "@/types/ip-address.types"

// Bulk delete response type
export interface BulkDeleteResponse {
  deletedCount: number
}

export class IPAddressService {
  /**
   * Bulk delete IP addresses
   * POST /api/host-ports/bulk-delete
   * Note: IP addresses are aggregated, so we pass IP strings instead of IDs
   */
  static async bulkDelete(ips: string[]): Promise<BulkDeleteResponse> {
    const response = await api.post<BulkDeleteResponse>(
      `/host-ports/bulk-delete`,
      { ips }
    )
    return response.data
  }

  static async getTargetIPAddresses(
    targetId: number,
    params?: GetIPAddressesParams
  ): Promise<GetIPAddressesResponse> {
    if (USE_MOCK) {
      await mockDelay()
      const page = params?.page || 1
      const pageSize = params?.pageSize || 10
      const filter = params?.filter?.toLowerCase() || ""
      const target = getMockTargetById(targetId)
      const domain = target?.name?.toLowerCase()

      let filtered = mockIPAddresses

      if (domain) {
        filtered = filtered.filter((ip) =>
          ip.hosts.some((host) => host.toLowerCase().includes(domain))
        )
      }

      if (filter) {
        filtered = filtered.filter((ip) =>
          ip.ip.toLowerCase().includes(filter) ||
          ip.hosts.some((host) => host.toLowerCase().includes(filter))
        )
      }

      const total = filtered.length
      const totalPages = Math.ceil(total / pageSize)
      const start = (page - 1) * pageSize
      const results = filtered.slice(start, start + pageSize)

      return {
        results,
        total,
        page,
        pageSize,
        totalPages,
      }
    }
    const response = await api.get<GetIPAddressesResponse>(`/targets/${targetId}/host-ports`, {
      params: {
        page: params?.page || 1,
        pageSize: params?.pageSize || 10,
        ...(params?.filter && { filter: params.filter }),
      },
    })
    return response.data
  }

  static async getScanIPAddresses(
    scanId: number,
    params?: GetIPAddressesParams
  ): Promise<GetIPAddressesResponse> {
    if (USE_MOCK) {
      await mockDelay()
      const page = params?.page || 1
      const pageSize = params?.pageSize || 10
      const filter = params?.filter?.toLowerCase() || ""
      const scan = getMockScanById(scanId)
      const domain = scan?.target?.name?.toLowerCase()

      let filtered = mockIPAddresses

      if (domain) {
        filtered = filtered.filter((ip) =>
          ip.hosts.some((host) => host.toLowerCase().includes(domain))
        )
      }

      if (filter) {
        filtered = filtered.filter((ip) =>
          ip.ip.toLowerCase().includes(filter) ||
          ip.hosts.some((host) => host.toLowerCase().includes(filter))
        )
      }

      const total = filtered.length
      const totalPages = Math.ceil(total / pageSize)
      const start = (page - 1) * pageSize
      const results = filtered.slice(start, start + pageSize)

      return {
        results,
        total,
        page,
        pageSize,
        totalPages,
      }
    }
    const response = await api.get<GetIPAddressesResponse>(`/scans/${scanId}/host-ports`, {
      params: {
        page: params?.page || 1,
        pageSize: params?.pageSize || 10,
        ...(params?.filter && { filter: params.filter }),
      },
    })
    return response.data
  }

  /** Export all IP addresses by target (CSV format) */
  static async exportIPAddressesByTargetId(targetId: number, ips?: string[]): Promise<Blob> {
    const params: Record<string, string> = {}
    if (ips && ips.length > 0) {
      params.ips = ips.join(',')
    }
    const response = await api.get<Blob>(`/targets/${targetId}/host-ports/export`, {
      params,
      responseType: 'blob',
    })
    return response.data
  }

  /** Export all IP addresses by scan task (CSV format) */
  static async exportIPAddressesByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/host-ports/export`, {
      responseType: 'blob',
    })
    return response.data
  }
}
