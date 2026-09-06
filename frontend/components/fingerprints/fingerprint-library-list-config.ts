import type {
  BusinessListFilterCompilerConfig,
  BusinessListSortableFieldConfig,
  BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import type { FingerprintLibrary } from "@/types/fingerprint.types"

export type FingerprintLibraryListConfig = {
  filterConfig: BusinessListFilterCompilerConfig
  sortableFields: Record<string, BusinessListSortableFieldConfig>
  defaultSorting: BusinessListSorting
}

const createdAtSort: BusinessListSortableFieldConfig = {
  orderBy: "createdAt",
  firstDirection: "desc",
}

const primaryNameSort: BusinessListSortableFieldConfig = {
  orderBy: "displayName",
  firstDirection: "asc",
}

const defaultSorting: BusinessListSorting = {
  field: "createdAt",
  direction: "desc",
}

export const FINGERPRINT_LIBRARY_LIST_CONFIG: Record<FingerprintLibrary, FingerprintLibraryListConfig> = {
  fingerprinthub: {
    filterConfig: {
      search: { field: "displayName" },
      facets: { severity: { field: "severity" } },
    },
    sortableFields: { displayName: primaryNameSort, createdAt: createdAtSort },
    defaultSorting,
	  },
}
