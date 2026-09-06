"use client"

import { useCallback } from "react"
import { useQuery, keepPreviousData } from "@tanstack/react-query"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { IPAddressService } from "@/services/ip-address.service"
import type { GetIPAddressesParams, GetIPAddressesResponse } from "@/types/ip-address.types"
import type { WebsiteAssetScope } from "@/types/website.types"

const ipAddressKeyBase = createResourceKeys("ip-addresses")

const ipAddressKeys = {
  ...ipAddressKeyBase,
  target: (targetId: number, params: GetIPAddressesParams, websiteScope?: WebsiteAssetScope) =>
    [...ipAddressKeyBase.all, "target", targetId, params, websiteScope] as const,
  scan: (scanId: number, params: GetIPAddressesParams) =>
    [...ipAddressKeyBase.all, "scan", scanId, params] as const,
  portOptions: (scope: "target" | "scan", id: number) =>
    [...ipAddressKeyBase.all, "portOptions", scope, id] as const,
}

function ipAddressCascadeInvalidates() {
  return [
    { queryKey: ipAddressKeys.all },
    { queryKey: ['targets'] as const },
    { queryKey: ['scans'] as const },
  ]
}

function normalizeParams(params?: GetIPAddressesParams): GetIPAddressesParams {
  return {
    pageSize: params?.pageSize ?? 10,
    pageToken: params?.pageToken,
    filter: params?.filter ?? "",
    orderBy: params?.orderBy ?? "",
  }
}

export function useTargetIPAddresses(
  targetId: number,
  params?: GetIPAddressesParams,
  options?: { enabled?: boolean; websiteScope?: WebsiteAssetScope }
) {
  const normalizedParams = normalizeParams(params)

  return useQuery({
    queryKey: ipAddressKeys.target(targetId, normalizedParams, options?.websiteScope),
    queryFn: () => IPAddressService.getTargetIPAddresses(targetId, normalizedParams),
    enabled: options?.enabled ?? !!targetId,
    select: (response: GetIPAddressesResponse) => response,
    placeholderData: keepPreviousData,
  })
}

export function useScanIPAddresses(
  scanId: number,
  params?: GetIPAddressesParams,
  options?: { enabled?: boolean }
) {
  const normalizedParams = normalizeParams(params)

  return useQuery({
    queryKey: ipAddressKeys.scan(scanId, normalizedParams),
    queryFn: () => IPAddressService.getScanIPAddresses(scanId, normalizedParams),
    enabled: options?.enabled ?? !!scanId,
    select: (response: GetIPAddressesResponse) => response,
    placeholderData: keepPreviousData,
  })
}

export function useTargetPortOptions(
  targetId: number,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: ipAddressKeys.portOptions("target", targetId),
    queryFn: () => IPAddressService.getTargetPortOptions(targetId),
    enabled: options?.enabled ?? !!targetId,
    placeholderData: keepPreviousData,
  })
}

export function useScanPortOptions(
  scanId: number,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: ipAddressKeys.portOptions("scan", scanId),
    queryFn: () => IPAddressService.getScanPortOptions(scanId),
    enabled: options?.enabled ?? !!scanId,
    placeholderData: keepPreviousData,
  })
}

export function useBulkDeleteIPAddresses() {
  return useResourceMutation({
    mutationFn: (ips: string[]) => IPAddressService.bulkDelete(ips),
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: 'bulk-delete-ip-addresses',
    },
    invalidate: ipAddressCascadeInvalidates(),
    onSuccess: ({ data, toast }) => {
      toast.success('toast.asset.ipAddress.delete.bulkSuccess', {
        count: data.deletedCount,
      })
    },
    errorFallbackKey: 'toast.asset.ipAddress.delete.error',
  })
}

export function useExportIPAddresses({
  targetId,
  scanId,
}: {
  targetId?: number
  scanId?: number
}) {
  return useCallback((ips?: string[]) => {
    if (targetId) {
      return IPAddressService.exportIPAddressesByTargetId(targetId, ips)
    }
    if (scanId) {
      return IPAddressService.exportIPAddressesByScanId(scanId)
    }
    return Promise.resolve(null)
  }, [scanId, targetId])
}
