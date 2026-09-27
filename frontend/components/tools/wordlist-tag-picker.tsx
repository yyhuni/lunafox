"use client"

import { useEffect, useMemo, useRef, useState } from "react"
import { Check, Plus, Tag, X } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useWordlistTags } from "@/hooks/use-wordlists"
import { textRole } from "@/lib/typography"
import { compactSurfaceClassNames } from "@/lib/ui/compact-surface-contract"
import { cn } from "@/lib/utils"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WordlistTagPickerProps {
  t: TranslationFn
  value: string[]
  onChange: (tags: string[]) => void
  density?: "default" | "compact"
}

function normalizeTag(value: string) {
  return value.trim()
}

export function WordlistTagPicker({ t, value, onChange, density = "default" }: WordlistTagPickerProps) {
  const compact = density === "compact"
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
        <Badge size="tag" variant="outline" className="gap-1.5">
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
        size="tag"
        variant="outline"
        className="cursor-pointer gap-1.5 hover:bg-secondary/80"
        render={(<button type="button" onClick={() => setIsAddingTag(true)} />)}
      >
        <Plus className="h-3.5 w-3.5" />
        <span>{t("addTag")}</span>
      </Badge>
    )
  }

  return (
    <div className={compact ? "space-y-2" : "space-y-3"}>
      <div className={compact ? "space-y-1" : "space-y-1.5"}>
        <div className={compact ? "flex flex-wrap items-center gap-1" : "flex flex-wrap items-center gap-1.5"}>
          {value.length ? value.map((tag) => (
            <Badge key={tag} size="tag" variant="secondary" className="gap-1 rounded-md">
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

      <div className={cn(compactSurfaceClassNames.mutedInfo, compact ? "space-y-1.5" : "space-y-2")}>
        <p className={textRole.helperText}>{t("recommendedTags")}</p>
        <div className={compact ? "flex max-h-40 flex-wrap gap-1 overflow-y-auto pr-1" : "flex flex-wrap gap-1.5"}>
          {suggestions.map((tag) => {
            const isSelected = selected.has(tag.displayName)
            return (
              <Badge
                size="tag"
                key={tag.name}
                variant={isSelected ? "secondary" : "outline"}
                className="cursor-pointer gap-1.5 hover:bg-secondary/80 disabled:cursor-default disabled:opacity-60"
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
                <Badge size="micro" variant="count">
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
