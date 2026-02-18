import * as React from "react"

type UseSearchStateOptions = {
  isFetching: boolean
  setSearchValue: React.Dispatch<React.SetStateAction<string>>
  onResetPage?: () => void
}

export function useSearchState({ isFetching, setSearchValue, onResetPage }: UseSearchStateOptions) {
  const [isSearching, setIsSearching] = React.useState(false)

  const handleSearchChange = React.useCallback(
    (value: string) => {
      setIsSearching(true)
      setSearchValue(value)
      onResetPage?.()
    },
    [onResetPage, setSearchValue]
  )

  React.useEffect(() => {
    if (!isFetching && isSearching) {
      setIsSearching(false)
    }
  }, [isFetching, isSearching])

  return {
    isSearching,
    setIsSearching,
    handleSearchChange,
  }
}
