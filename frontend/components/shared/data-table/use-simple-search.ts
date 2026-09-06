import * as React from "react"

export const SEARCH_COMMIT_DEBOUNCE_MS = 300

type UseSimpleSearchStateOptions = {
  searchValue?: string
  onSearch?: (value: string) => void
}

export function useSimpleSearchState({ searchValue, onSearch }: UseSimpleSearchStateOptions) {
  const [value, setDraftValue] = React.useState(searchValue ?? "")
  const committedValueRef = React.useRef(searchValue ?? "")
  const commitTimeoutRef = React.useRef<number | null>(null)

  const clearPendingCommit = React.useCallback(() => {
    if (commitTimeoutRef.current === null) return
    window.clearTimeout(commitTimeoutRef.current)
    commitTimeoutRef.current = null
  }, [])

  const commitNow = React.useCallback(
    (nextValue = value) => {
      clearPendingCommit()
      if (nextValue === committedValueRef.current) return

      committedValueRef.current = nextValue
      onSearch?.(nextValue)
    },
    [clearPendingCommit, onSearch, value]
  )

  const handleSearchInputChange = React.useCallback(
    (nextValue: string) => {
      setDraftValue(nextValue)
      clearPendingCommit()
      commitTimeoutRef.current = window.setTimeout(() => {
        commitNow(nextValue)
      }, SEARCH_COMMIT_DEBOUNCE_MS)
    },
    [clearPendingCommit, commitNow]
  )

  React.useEffect(() => {
    const nextValue = searchValue ?? ""
    committedValueRef.current = nextValue
    setDraftValue(nextValue)
  }, [searchValue])

  React.useEffect(() => clearPendingCommit, [clearPendingCommit])

  return {
    value,
    handleSearchInputChange,
    commitNow,
  }
}
