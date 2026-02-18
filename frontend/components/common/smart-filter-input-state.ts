import * as React from "react"
import {
  buildDefaultPlaceholder,
  getGhostText,
  parseFilterExpression,
  saveQueryHistory,
  type FilterField,
  type ParsedFilter,
} from "@/lib/smart-filter"

type SmartFilterInputStateOptions = {
  fields: FilterField[]
  examples?: string[]
  value?: string
  onSearch?: (filters: ParsedFilter[], rawQuery: string) => void
}

export function useSmartFilterInputState({
  fields,
  examples,
  value,
  onSearch,
}: SmartFilterInputStateOptions) {
  const [open, setOpen] = React.useState(false)
  const [inputValue, setInputValue] = React.useState(value ?? "")
  const inputRef = React.useRef<HTMLInputElement>(null)
  const ghostRef = React.useRef<HTMLSpanElement>(null)
  const listRef = React.useRef<HTMLDivElement>(null)
  const savedScrollTop = React.useRef<number | null>(null)
  const hasInitialized = React.useRef(false)

  const ghostText = React.useMemo(() => getGhostText(inputValue, fields), [fields, inputValue])

  React.useEffect(() => {
    if (value !== undefined) {
      setInputValue(value)
    }
  }, [value])

  React.useEffect(() => {
    if (open) {
      const restoreScroll = () => {
        if (listRef.current) {
          if (!hasInitialized.current) {
            listRef.current.scrollTop = 0
            hasInitialized.current = true
          } else if (savedScrollTop.current !== null) {
            listRef.current.scrollTop = savedScrollTop.current
          }
        }
      }

      restoreScroll()
      const timer = setTimeout(restoreScroll, 50)
      return () => clearTimeout(timer)
    }

    if (listRef.current) {
      savedScrollTop.current = listRef.current.scrollTop
    }
  }, [open])

  const defaultPlaceholder = React.useMemo(
    () => buildDefaultPlaceholder(fields, examples),
    [fields, examples]
  )

  const parsedFilters = React.useMemo(
    () => parseFilterExpression(inputValue),
    [inputValue]
  )

  const currentWord = React.useMemo(() => {
    const words = inputValue.split(/\s+/)
    return words[words.length - 1] || ""
  }, [inputValue])

  const showFieldSuggestions = !currentWord.includes("=")

  const acceptGhostText = React.useCallback(() => {
    if (ghostText) {
      setInputValue((prev) => prev + ghostText)
      return true
    }
    return false
  }, [ghostText])

  const handleSelectSuggestion = React.useCallback(
    (suggestion: string) => {
      const words = inputValue.split(/\s+/)
      words[words.length - 1] = suggestion
      const newValue = words.join(" ")
      setInputValue(newValue)
      setOpen(false)
      inputRef.current?.blur()
    },
    [inputValue]
  )

  const handleSearch = React.useCallback(() => {
    saveQueryHistory(inputValue)
    onSearch?.(parsedFilters, inputValue)
    setOpen(false)
  }, [inputValue, onSearch, parsedFilters])

  const handleKeyDown = React.useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Tab" && ghostText) {
        e.preventDefault()
        acceptGhostText()
      }
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault()
        handleSearch()
      }
      if (e.key === "Escape") {
        setOpen(false)
      }
      if (e.key === "ArrowRight" && ghostText) {
        const input = inputRef.current
        if (input && input.selectionStart === input.value.length) {
          e.preventDefault()
          acceptGhostText()
        }
      }
    },
    [acceptGhostText, ghostText, handleSearch]
  )

  const handleAppendExample = React.useCallback(
    (example: string) => {
      const trimmed = inputValue.trim()
      const newValue = trimmed ? `${trimmed} ${example}` : example
      setInputValue(newValue)
      setOpen(false)
      inputRef.current?.blur()
    },
    [inputValue]
  )

  const handleInputChange = React.useCallback(
    (value: string) => {
      setInputValue(value)
      if (!open) {
        setOpen(true)
      }
    },
    [open]
  )

  const handleBlur = React.useCallback(
    (e: React.FocusEvent<HTMLInputElement>) => {
      const relatedTarget = e.relatedTarget as HTMLElement | null
      if (relatedTarget?.closest('[data-radix-popper-content-wrapper]')) {
        return
      }
      setTimeout(() => setOpen(false), 150)
    },
    []
  )

  return {
    open,
    setOpen,
    inputValue,
    setInputValue,
    inputRef,
    ghostRef,
    listRef,
    ghostText,
    parsedFilters,
    defaultPlaceholder,
    currentWord,
    showFieldSuggestions,
    handleSelectSuggestion,
    handleSearch,
    handleKeyDown,
    handleAppendExample,
    handleInputChange,
    handleBlur,
  }
}
