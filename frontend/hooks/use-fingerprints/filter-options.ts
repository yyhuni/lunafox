"use client"

import * as React from "react"
import { keepPreviousData, useQueries } from "@tanstack/react-query"

import { FingerprintService } from "@/services/fingerprint.service"
import type {
  FingerprintFilterOption,
  FingerprintFilterOptionField,
  FingerprintLibrary,
} from "@/types/fingerprint.types"

import { fingerprintKeys } from "./keys"

/**
 * Facet options come from a library-wide server aggregation. Keeping them in
 * their own queries prevents a cursor page from silently becoming the option
 * source after search, pagination, or mutation changes.
 */
export function useFingerprintFacetOptions(
  library: FingerprintLibrary,
  fields: readonly FingerprintFilterOptionField[]
) {
  const libraryKeys = fingerprintKeys[library]
  const queries = useQueries({
    queries: fields.map((field) => ({
      queryKey: libraryKeys.filterOptions(field),
		queryFn: () => FingerprintService.getFilterOptions(),
      placeholderData: keepPreviousData,
    })),
  })

  return React.useMemo(() => {
    const options: Partial<Record<FingerprintFilterOptionField, FingerprintFilterOption[]>> = {}
    fields.forEach((field, index) => {
      const results = queries[index]?.data?.results
      if (results) {
        options[field] = results
      }
    })
    const failedQuery = queries.find((query) => query.error)
    return {
      options,
      isLoading: queries.some((query) => query.isLoading),
      error: failedQuery?.error ?? null,
      refetch: () => Promise.all(queries.map((query) => query.refetch())),
    }
  }, [fields, queries])
}
