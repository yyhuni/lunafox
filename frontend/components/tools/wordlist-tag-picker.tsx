"use client"

import { useEffect, useMemo, useRef, useState } from "react"
import { Check, Plus, Tag, X } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useWordlistTags } from "@/hooks/use-wordlists"
import { textRole } from "@/lib/typography"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WordlistTagPickerProps {
  t: TranslationFn
  value: string[]
  onChange: (tags: string[]) => void
}

function normalizeTag(value: string) {
  return value.trim()
}

export function WordlistTagPicker({ t, value, onChange }: WordlistTagPickerProps) {
  const [draftTag, setDraftTag] = useState("")
  const [isAddingTag, setIsAddingTag] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const { data } = useWordlistTags({ pageSize: 20 })
  const selected = useMemo(() => new Set(value), [value])
  const suggestions = data?.results ?? []

  useEffect(() => {
    if (isAddingTag) {
      inputRef.current?.focus()
    }
  }, [isAddingTag])

  const addTag = (tag: string) => {
    const normalized = normalizeTag(tag)
    if (!normalized || selected.has(normalized)) return
    onChange([...value, normalized])
    setDraftTag("")
    setIsAddingTag(false)
  }

  const commitDraftTag = () => {
    const normalized = normalizeTag(draftTag)
    if (!normalized) {
      cancelDraftTag()
      return
    }
    if (selected.has(normalized)) {
      cancelDraftTag()
      return
    }
    addTag(normalized)
  }

  const cancelDraftTag = () => {
    setDraftTag("")
    setIsAddingTag(false)
  }

  const removeTag = (tag: string) => {
    onChange(value.filter((item) => item !== tag))
  }

  const renderAddTagControl = () => {
    if (isAddingTag) {
      return (
        <Badge variant="outline" className="h-7 gap-1.5 px-2">
          <Plus className="h-3.5 w-3.5" />
          <input
            ref={inputRef}
            name="tagPickerInlineInput"
            autoComplete="off"
            value={draftTag}
            onChange={(event) => setDraftTag(event.target.value)}
            onBlur={commitDraftTag}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault()
                commitDraftTag()
              }
              if (event.key === "Escape") {
                event.preventDefault()
                cancelDraftTag()
              }
            }}
            placeholder={t("inlineTagPlaceholder")}
            aria-label={t("tagPickerPlaceholder")}
            className="w-24 bg-transparent p-0 text-inherit outline-none placeholder:text-muted-foreground"
          />
          <Button
            type="button"
            variant="ghost"
            size="chip-icon"
            onPointerDown={(event) => event.preventDefault()}
            onClick={cancelDraftTag}
            aria-label={t("cancel")}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </Badge>
      )
    }

    return (
      <Badge
        variant="outline"
        className="h-7 cursor-pointer gap-1.5 px-2 hover:bg-secondary/80"
        render={(<button type="button" onClick={() => setIsAddingTag(true)} />)}
      >
        <Plus className="h-3.5 w-3.5" />
        <span>{t("addTag")}</span>
      </Badge>
    )
  }

  return (
    <div className="space-y-3">
      <div className="space-y-1.5">
        <div className="flex min-h-7 flex-wrap items-center gap-1.5">
          {value.length ? value.map((tag) => (
            <Badge key={tag} variant="secondary" className="h-7 gap-1 rounded-md px-2">
              <span>{tag}</span>
              <button
                type="button"
                onClick={() => removeTag(tag)}
                className="rounded-sm text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
                aria-label={t("removeTag", { tag })}
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </Badge>
          )) : (
            <span className={textRole.helperText}>{t("noTagsSelected")}</span>
          )}
          {renderAddTagControl()}
        </div>
      </div>

      <div className="space-y-2 rounded-md border bg-muted/20 p-2.5">
        <p className={textRole.helperText}>{t("recommendedTags")}</p>
        <div className="flex flex-wrap gap-1.5">
          {suggestions.map((tag) => {
            const isSelected = selected.has(tag.displayName)
            return (
              <Badge
                key={tag.name}
                variant={isSelected ? "secondary" : "outline"}
                className="h-7 cursor-pointer gap-1.5 px-2 hover:bg-secondary/80 disabled:cursor-default disabled:opacity-60"
                render={(
                  <button
                    type="button"
                    onClick={() => addTag(tag.displayName)}
                    disabled={isSelected}
                    aria-pressed={isSelected}
                  />
                )}
              >
                {isSelected ? <Check className="h-3.5 w-3.5" /> : <Tag className="h-3.5 w-3.5" />}
                <span>{tag.displayName}</span>
                <Badge variant="count" className="h-4 min-w-4 rounded-full px-1">
                  {tag.wordlistCount}
                </Badge>
              </Badge>
            )
          })}
        </div>
      </div>
    </div>
  )
}
