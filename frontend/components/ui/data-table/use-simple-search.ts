import * as React from "react"

type UseSimpleSearchStateOptions = {
  searchValue?: string
  onSearch?: (value: string) => void
}

export function useSimpleSearchState({ searchValue, onSearch }: UseSimpleSearchStateOptions) {
  const [value, setValue] = React.useState(searchValue ?? "")

  React.useEffect(() => {
    setValue(searchValue ?? "")
  }, [searchValue])

  const submit = React.useCallback(() => {
    onSearch?.(value)
  }, [onSearch, value])

  return {
    value,
    setValue,
    submit,
  }
}
