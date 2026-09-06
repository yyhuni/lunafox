import * as React from "react"

export const SEARCH_COMMIT_DEBOUNCE_MS = 300

type UseSearchStateOptions = {
  isFetching: boolean
  searchValue: string
  setSearchValue: React.Dispatch<React.SetStateAction<string>>
  onResetPage?: () => void
}

export function useSearchState({ isFetching, searchValue, setSearchValue, onResetPage }: UseSearchStateOptions) {
  const [searchInput, setSearchInput] = React.useState(searchValue)
  const [isSearching, setIsSearching] = React.useState(false)
  const committedSearchValueRef = React.useRef(searchValue)
  const commitTimeoutRef = React.useRef<number | null>(null)

  const clearPendingCommit = React.useCallback(() => {
    if (commitTimeoutRef.current === null) return
    window.clearTimeout(commitTimeoutRef.current)
    commitTimeoutRef.current = null
  }, [])

  const commitSearch = React.useCallback(
    (value = searchInput) => {
      clearPendingCommit()
      setSearchInput(value)
      if (value === committedSearchValueRef.current) return

      committedSearchValueRef.current = value
      setIsSearching(true)
      setSearchValue(value)
      onResetPage?.()
    },
    [clearPendingCommit, onResetPage, searchInput, setSearchValue]
  )

  const handleSearchInputChange = React.useCallback(
    (value: string) => {
      setSearchInput(value)
      clearPendingCommit()
      commitTimeoutRef.current = window.setTimeout(() => {
        commitSearch(value)
      }, SEARCH_COMMIT_DEBOUNCE_MS)
    },
    [clearPendingCommit, commitSearch]
  )

  React.useEffect(() => {
    committedSearchValueRef.current = searchValue
    setSearchInput(searchValue)
  }, [searchValue])

  React.useEffect(() => {
    if (!isFetching && isSearching) {
      setIsSearching(false)
    }
  }, [isFetching, isSearching])

  React.useEffect(() => clearPendingCommit, [clearPendingCommit])

  return {
    searchInput,
    isSearching,
    setIsSearching,
    handleSearchInputChange,
    commitSearch,
  }
}
