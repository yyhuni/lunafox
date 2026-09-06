import { api } from "@/lib/api-client"
import type { FilterOptionsResponse, GetIPAddressesParams, GetIPAddressesResponse } from "@/types/ip-address.types"

// Bulk delete response type
export interface BulkDeleteResponse {
  deletedCount: number
}

type AipHostPortListResponse<T> = {
  results: T[]
  totalSize?: number
  nextPageToken?: string
  total?: number
  page?: number
  pageSize?: number
  totalPages?: number
}

type HostPortAggregateRow = {
  ip: string
  hosts?: string[]
  ports?: number[]
  createdAt: string
}

function normalizePagination<T>(
  response: AipHostPortListResponse<T>,
  params?: GetIPAddressesParams
): Omit<GetIPAddressesResponse, "results"> & { nextPageToken?: string } {
  const page = response.page ?? 1
  const pageSize = response.pageSize ?? params?.pageSize ?? 10
  const total = response.total ?? response.totalSize ?? response.results.length

  return {
    total,
    ...(response.totalSize !== undefined ? { totalSize: response.totalSize } : {}),
    page,
    pageSize,
    totalPages: response.totalPages ?? (total === 0 ? 0 : Math.ceil(total / pageSize)),
    nextPageToken: response.nextPageToken,
  }
}

function normalizeHostPortResponse(
  response: AipHostPortListResponse<HostPortAggregateRow>,
  params?: GetIPAddressesParams
): GetIPAddressesResponse {
  return {
    ...normalizePagination(response, params),
    results: response.results.map((item) => ({
      ip: item.ip,
      hosts: item.hosts ?? [],
      ports: item.ports ?? [],
      createdAt: item.createdAt,
    })),
  }
}

function buildHostPortListParams(params?: GetIPAddressesParams) {
  return {
    pageSize: params?.pageSize ?? 10,
    ...(params?.pageToken && { pageToken: params.pageToken }),
    ...(params?.filter && { filter: params.filter }),
    ...(params?.orderBy && { orderBy: params.orderBy }),
  }
}

export class IPAddressService {
  /**
   * Bulk delete IP addresses
   * POST /v1/hostPorts:batchDelete
   * Note: IP addresses are aggregated, so we pass IP strings instead of IDs
   */
  static async bulkDelete(ips: string[]): Promise<BulkDeleteResponse> {
    const response = await api.post<BulkDeleteResponse>(
      `/hostPorts:batchDelete`,
      { ips }
    )
    return response.data
  }

  static async getTargetIPAddresses(
    targetId: number,
    params?: GetIPAddressesParams
  ): Promise<GetIPAddressesResponse> {
    const response = await api.get<AipHostPortListResponse<HostPortAggregateRow>>(`/targets/${targetId}/hostPorts`, {
      params: buildHostPortListParams(params),
    })
    return normalizeHostPortResponse(response.data, params)
  }

  static async getScanIPAddresses(
    scanId: number,
    params?: GetIPAddressesParams
  ): Promise<GetIPAddressesResponse> {
    const response = await api.get<AipHostPortListResponse<HostPortAggregateRow>>(`/scans/${scanId}/hostPorts`, {
      params: buildHostPortListParams(params),
    })
    return normalizeHostPortResponse(response.data, params)
  }

  static async getTargetPortOptions(targetId: number): Promise<FilterOptionsResponse> {
    const response = await api.get<FilterOptionsResponse>(`/targets/${targetId}/hostPorts/filterOptions`, {
      params: { field: "port" },
    })
    return response.data
  }

  static async getScanPortOptions(scanId: number): Promise<FilterOptionsResponse> {
    const response = await api.get<FilterOptionsResponse>(`/scans/${scanId}/hostPorts/filterOptions`, {
      params: { field: "port" },
    })
    return response.data
  }

  /** Export all IP addresses by target (CSV format) */
  static async exportIPAddressesByTargetId(targetId: number, ips?: string[]): Promise<Blob> {
    const params: Record<string, string> = {}
    if (ips && ips.length > 0) {
      params.ips = ips.join(',')
    }
    const response = await api.get<Blob>(`/targets/${targetId}/hostPorts/exportFiles/current`, {
      params,
      responseType: 'blob',
    })
    return response.data
  }

  /** Export all IP addresses by scan task (CSV format) */
  static async exportIPAddressesByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/hostPorts/exportFiles/current`, {
      responseType: 'blob',
    })
    return response.data
  }
}
