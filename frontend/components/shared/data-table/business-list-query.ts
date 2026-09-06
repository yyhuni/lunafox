import type { CursorPaginationNavigation } from "@/types/data-table.types"

export type BusinessListSortDirection = "asc" | "desc"

export type BusinessListSorting = {
  field: string
  direction: BusinessListSortDirection
}

export type BusinessListQuery = {
  search?: string
  filters: Record<string, string[]>
  sorting?: BusinessListSorting
  pageSize: number
  pageToken?: string
  pageIndex?: number
}

export type BusinessListFacetFilterConfig = {
  field: string
  operator?: "=" | "==" | "!="
}

export type BusinessListFilterCompilerConfig = {
  search?: BusinessListFacetFilterConfig | BusinessListFacetFilterConfig[]
  facets: Record<string, BusinessListFacetFilterConfig>
}

export type BusinessListSortableFieldConfig = {
  orderBy: string
  firstDirection?: BusinessListSortDirection
}

type CursorNavigationInput = {
  currentPage: number
  pageTokens: Record<number, string | undefined>
  nextPageToken?: string
  firstPage?: number
  onFirstPage?: () => void
}

type CursorPageTransitionInput = CursorNavigationInput & {
  requestedPage: number
}

export type CursorPageTransition = {
  reachable: boolean
  pageToken?: string
}

/**
 * React Query may retain the prior result while a new cursor query resolves.
 * That data belongs to a different request and must not authorize another
 * continuation from the page currently being requested.
 */
export function getCurrentCursorNextPageToken(
  nextPageToken: string | undefined,
  isPlaceholderData?: boolean,
) {
  return isPlaceholderData ? undefined : nextPageToken
}

/**
 * Cursor caches store the token needed to request each visited page. The first
 * page deliberately has an undefined token, so property presence determines
 * whether a prior page is reachable rather than the token value itself.
 */
export function getCursorPaginationNavigation({
  currentPage,
  pageTokens,
  nextPageToken,
  firstPage = 1,
  onFirstPage,
}: CursorNavigationInput): CursorPaginationNavigation {
  const previousPage = currentPage - 1

  return {
    mode: "cursor",
    canFirstPage: currentPage > firstPage,
    canPreviousPage: currentPage > firstPage
      && Object.prototype.hasOwnProperty.call(pageTokens, previousPage),
    canNextPage: Boolean(nextPageToken),
    ...(onFirstPage ? { onFirstPage } : {}),
  }
}

/**
 * AIP page tokens only authorize movement to the immediate continuation or an
 * already visited predecessor. This intentionally rejects cached arbitrary
 * jumps so route state cannot reintroduce numbered-pagination semantics.
 */
export function getCursorPageTransition({
  currentPage,
  pageTokens,
  nextPageToken,
  requestedPage,
  firstPage = 1,
}: CursorPageTransitionInput): CursorPageTransition {
  // Keep the internal ordinal bounded even though the UI exposes only adjacent
  // controls. A malformed caller must not turn a cached first-page token into
  // a transition before the collection start.
  if (currentPage < firstPage || requestedPage < firstPage) {
    return { reachable: false }
  }

  // Resetting to the first cursor page deliberately omits pageToken. Unlike
  // adjacent transitions, it does not require a cached token from the service.
  if (requestedPage === firstPage) {
    return { reachable: true, pageToken: undefined }
  }

  if (requestedPage === currentPage - 1) {
    if (!Object.prototype.hasOwnProperty.call(pageTokens, requestedPage)) {
      return { reachable: false }
    }

    return { reachable: true, pageToken: pageTokens[requestedPage] }
  }

  if (requestedPage === currentPage + 1 && nextPageToken) {
    return { reachable: true, pageToken: nextPageToken }
  }

  return { reachable: false }
}

function oppositeSortDirection(direction: BusinessListSortDirection): BusinessListSortDirection {
  return direction === "asc" ? "desc" : "asc"
}

export function createBusinessListQuery(input?: Partial<BusinessListQuery>): BusinessListQuery {
  return {
    search: input?.search,
    filters: input?.filters ?? {},
    sorting: input?.sorting,
    pageSize: input?.pageSize ?? 20,
    pageToken: input?.pageToken,
    pageIndex: input?.pageIndex ?? 1,
  }
}

export function applyBusinessListControlChange(
  query: BusinessListQuery,
  change: Partial<Pick<BusinessListQuery, "search" | "filters" | "sorting" | "pageSize">>
): BusinessListQuery {
  return {
    ...query,
    ...change,
    pageIndex: 1,
    pageToken: undefined,
  }
}

export function setBusinessListPage(
  query: BusinessListQuery,
  page: Pick<BusinessListQuery, "pageIndex" | "pageToken">
): BusinessListQuery {
  return {
    ...query,
    pageIndex: page.pageIndex,
    pageToken: page.pageToken,
  }
}

export function toggleBusinessListSorting(
  query: BusinessListQuery,
  field: string,
  registry: Record<string, BusinessListSortableFieldConfig>,
  defaultSorting?: BusinessListSorting
): BusinessListQuery {
  const sortableField = registry[field]
  if (!sortableField) {
    return query
  }

  const firstDirection = sortableField.firstDirection ?? "asc"
  const nextSorting = (() => {
    if (query.sorting?.field !== field) {
      return { field, direction: firstDirection }
    }
    if (query.sorting.direction === firstDirection) {
      return { field, direction: oppositeSortDirection(firstDirection) }
    }
    return defaultSorting
  })()

  return applyBusinessListControlChange(query, { sorting: nextSorting })
}

function quoteFilterValue(value: string) {
  return value.replace(/\\/g, "\\\\").replace(/"/g, "\\\"")
}

function uniqueNonEmptyValues(values: string[] | undefined) {
  return Array.from(new Set((values ?? []).map((value) => value.trim()).filter(Boolean)))
}

function compilePredicate(config: BusinessListFacetFilterConfig, value: string) {
  return `${config.field}${config.operator ?? "="}\"${quoteFilterValue(value)}\"`
}

function compileSearchPredicate(
  config: BusinessListFacetFilterConfig | BusinessListFacetFilterConfig[],
  value: string
) {
  const searchConfigs = Array.isArray(config) ? config : [config]
  const predicates = searchConfigs.map((item) => compilePredicate(item, value))
  if (predicates.length === 1) {
    return predicates[0]
  }
  return `(${predicates.join(" || ")})`
}

function isRawURLSearch(
  config: BusinessListFacetFilterConfig | BusinessListFacetFilterConfig[] | undefined,
) {
  const configs = Array.isArray(config) ? config : config ? [config] : []
  return configs.length > 0 && configs.every((item) => item.field === "url")
}

// URL filters identify observed assets exactly. Whitespace may decide whether a
// query is absent, but must not be removed from a non-blank URL value.
export function preserveRawURLSearchInput(value: string | undefined): string | undefined {
  if (value === undefined || value.trim() === "") return undefined
  return value
}

export function compileBusinessListFilter(
  input: Pick<BusinessListQuery, "search" | "filters">,
  config: BusinessListFilterCompilerConfig
) {
  const clauses: string[] = []

  for (const [facetKey, values] of Object.entries(input.filters ?? {})) {
    const facetConfig = config.facets[facetKey]
    if (!facetConfig) {
      continue
    }
    const predicates = uniqueNonEmptyValues(values).map((value) => compilePredicate(facetConfig, value))
    if (predicates.length === 1) {
      clauses.push(predicates[0]!)
    } else if (predicates.length > 1) {
      clauses.push(`(${predicates.join(" || ")})`)
    }
  }

  // URL search values are exact stored identities. Keep their bytes intact;
  // ordinary text search retains the established presentation trimming.
  const search = isRawURLSearch(config.search)
    ? preserveRawURLSearchInput(input.search)
    : input.search?.trim()
  if (search && config.search) {
    clauses.push(compileSearchPredicate(config.search, search))
  }

  return clauses.length > 0 ? clauses.join(" && ") : undefined
}

export function compileBusinessListOrderBy(
  sorting: BusinessListSorting | undefined,
  registry: Record<string, BusinessListSortableFieldConfig>
) {
  if (!sorting) {
    return undefined
  }
  const field = registry[sorting.field]
  if (!field) {
    return undefined
  }
  return sorting.direction === "desc" ? `${field.orderBy} desc` : field.orderBy
}
