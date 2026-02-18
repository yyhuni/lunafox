"use client"

import { useQuery, keepPreviousData } from "@tanstack/react-query"
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { IPAddressService } from "@/services/ip-address.service"
import type { GetIPAddressesParams, GetIPAddressesResponse } from "@/types/ip-address.types"

const ipAddressKeyBase = createResourceKeys("ip-addresses")

const ipAddressKeys = {
  ...ipAddressKeyBase,
  target: (targetId: number, params: GetIPAddressesParams) =>
    [...ipAddressKeyBase.all, "target", targetId, params] as const,
  scan: (scanId: number, params: GetIPAddressesParams) =>
    [...ipAddressKeyBase.all, "scan", scanId, params] as const,
}

function normalizeParams(params?: GetIPAddressesParams): Required<GetIPAddressesParams> {
  return {
    page: params?.page ?? 1,
    pageSize: params?.pageSize ?? 10,
    filter: params?.filter ?? "",
  }
}

export function useTargetIPAddresses(
  targetId: number,
  params?: GetIPAddressesParams,
  options?: { enabled?: boolean }
) {
  const normalizedParams = normalizeParams(params)

  return useQuery({
    queryKey: ipAddressKeys.target(targetId, normalizedParams),
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
