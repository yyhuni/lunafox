"use client"

import * as React from "react"
import { useTranslations } from "next-intl"

import {
  Check,
  ChevronDown,
  Eye,
  IconEyeOff as EyeOff,
  Lock,
  Trash2,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { compactControlTier } from "@/lib/ui/compact-control-tier"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

interface HeaderEntryItem {
  id: string
  name: string
  value: string
  masked?: boolean
}

export interface EngineHttpHeadersPopoverProps {
  fieldId: string
  labelId?: string
  value: unknown
  disabled?: boolean
  onChange: (value: string[]) => void
  describedBy?: string
  invalid?: boolean
}

function parseHeaderStrings(raw: unknown): HeaderEntryItem[] {
  if (!Array.isArray(raw)) return []
  return raw
    .filter((item): item is string => typeof item === "string" && item.trim().length > 0)
    .map((str) => {
      const idx = str.indexOf(":")
      const name = idx > 0 ? str.slice(0, idx).trim() : "Custom-Header"
      const val = idx > 0 ? str.slice(idx + 1).trim() : str.trim()
      return {
        id: "hdr_" + Math.random().toString(36).slice(2, 9),
        name,
        value: val,
        masked: true,
      }
    })
}

// A serialized row needs both a header name and a value. Incomplete rows
// (no name, or an entirely empty value) never reach engineConfig; anything
// the user typed is committed verbatim so the trigger summary always shows
// exactly what the engine will send.
function serializeHeaderItems(items: HeaderEntryItem[]): string[] {
  return items
    .filter((item) => item.name.trim().length > 0 && item.value.trim().length > 0)
    .map((item) => `${item.name.trim()}: ${item.value.trim()}`)
}

export function EngineHttpHeadersPopover({
  fieldId,
  labelId,
  value,
  disabled = false,
  onChange,
  describedBy,
  invalid = false,
}: EngineHttpHeadersPopoverProps) {
  const t = useTranslations("scan.initiate.engineConfigForm")
  const [open, setOpen] = React.useState(false)
  const currentArray = React.useMemo(() => (Array.isArray(value) ? (value as string[]) : []), [value])
  const [drafts, setDrafts] = React.useState<HeaderEntryItem[]>(() => parseHeaderStrings(value))

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setDrafts(parseHeaderStrings(value))
    }
    setOpen(nextOpen)
  }

  const handleAddPreset = (preset: "cookie" | "bearer") => {
    const newItem: HeaderEntryItem = {
      id: "hdr_" + Math.random().toString(36).slice(2, 9),
      name: preset === "cookie" ? "Cookie" : "Authorization",
      // Presets only pre-fill the name (plus the Bearer scheme prefix); the
      // credential itself is always entered by the user, never templated.
      value: preset === "bearer" ? "Bearer " : "",
      masked: true,
    }
    setDrafts((prev) => [...prev, newItem])
  }

  const handleAddCustom = () => {
    setDrafts((prev) => [
      ...prev,
      {
        id: "hdr_" + Math.random().toString(36).slice(2, 9),
        name: "",
        value: "",
        masked: true,
      },
    ])
  }

  const handleSave = () => {
    onChange(serializeHeaderItems(drafts))
    setOpen(false)
  }

  // Clear only resets the in-popover draft; the committed value stays
  // unchanged until the user confirms with the Done action.
  const handleClearDrafts = () => {
    setDrafts([])
  }

  const count = currentArray.length
  const headerKeysSummary = currentArray
    .map((str) => str.split(":")[0]?.trim())
    .filter(Boolean)
    .join(", ")

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            id={fieldId}
            type="button"
            variant="outline"
            size="sm"
            layout="between"
            disabled={disabled}
            aria-labelledby={labelId}
            aria-describedby={describedBy}
            aria-invalid={invalid || undefined}
            className={cn(
              "h-8 w-full justify-between text-xs font-normal transition-colors",
              count > 0
                ? "border-primary/50 bg-primary/5 hover:bg-primary/10 text-foreground"
                : "border-input bg-background hover:bg-accent text-muted-foreground"
            )}
          />
        }
      >
        <span className="flex min-w-0 items-center gap-1.5 truncate">
          <Lock aria-hidden className={cn("size-3.5 shrink-0", count > 0 ? "text-primary" : "text-muted-foreground")} />
          <span className="truncate">
            {count > 0
              ? `${t("headersConfigured", { count })} (${headerKeysSummary})`
              : t("noHeadersConfigured")}
          </span>
        </span>
        <ChevronDown aria-hidden className="size-3.5 shrink-0 text-muted-foreground" />
      </PopoverTrigger>

      <PopoverContent
        align="start"
        minWidth="content"
        positionMethod="fixed"
        collisionPadding={8}
        className="w-96 p-0"
      >
        {/* Toolbar band: same muted-band pattern as EngineDurationInput. */}
        <div className="flex items-center justify-between gap-2 border-b border-border/70 bg-muted/30 px-3 py-1.5">
          <p className={textRole.navLabel}>{t("configureHeaders")}</p>

          <div className="flex items-center gap-1">
            <Button
              type="button"
              variant="outline"
              size="sm"
              className={cn(compactControlTier, "px-2")}
              onClick={() => handleAddPreset("cookie")}
            >
              + Cookie
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className={cn(compactControlTier, "px-2")}
              onClick={() => handleAddPreset("bearer")}
            >
              + Token
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className={cn(compactControlTier, "px-2 text-muted-foreground hover:text-foreground")}
              title={t("addCustomHeader")}
              onClick={handleAddCustom}
            >
              {t("customHeaderButton")}
            </Button>
          </div>
        </div>

        {/* Entries area */}
        <div className="bg-background px-2.5 py-2">
          {drafts.length === 0 ? (
            <div className="space-y-1 rounded border border-dashed border-border py-4 text-center">
              <p className={textRole.helperText}>{t("emptyTitle")}</p>
              <p className={cn("opacity-70", textRole.helperText)}>{t("emptyHint")}</p>
            </div>
          ) : (
            // p-1 keeps the inputs' 3px focus rings visible: an overflow-y-auto
            // box also clips the other axis, so the ring would otherwise be
            // shaved off at the scroll container edges.
            <div className="max-h-52 space-y-1.5 overflow-y-auto p-1">
              {drafts.map((d) => (
                <div key={d.id} className="grid grid-cols-12 items-center gap-1.5">
                  <div className="col-span-4">
                    <Input
                      value={d.name}
                      onChange={(e) =>
                        setDrafts((prev) =>
                          prev.map((item) => (item.id === d.id ? { ...item, name: e.target.value } : item))
                        )
                      }
                      placeholder={t("headerNamePlaceholder")}
                      aria-label={t("headerNamePlaceholder")}
                      size="sm"
                      className={cn(compactControlTier, "font-mono")}
                    />
                  </div>
                  <div className="relative col-span-7 flex items-center">
                    <Input
                      type={d.masked ? "password" : "text"}
                      value={d.value}
                      onChange={(e) =>
                        setDrafts((prev) =>
                          prev.map((item) => (item.id === d.id ? { ...item, value: e.target.value } : item))
                        )
                      }
                      placeholder={t("headerValuePlaceholder")}
                      aria-label={t("headerValuePlaceholder")}
                      size="sm"
                      className={cn(compactControlTier, "pr-6 font-mono")}
                    />
                    <button
                      type="button"
                      onClick={() =>
                        setDrafts((prev) =>
                          prev.map((item) => (item.id === d.id ? { ...item, masked: !item.masked } : item))
                        )
                      }
                      className="absolute right-1.5 text-muted-foreground hover:text-foreground"
                      aria-label={d.masked ? t("showPlaintext") : t("hidePlaintext")}
                    >
                      {d.masked ? <Eye aria-hidden className="size-3" /> : <EyeOff aria-hidden className="size-3" />}
                    </button>
                  </div>
                  <div className="col-span-1 flex justify-end">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      className="size-7 text-muted-foreground hover:text-destructive"
                      aria-label={t("removeHeader")}
                      onClick={() => setDrafts((prev) => prev.filter((item) => item.id !== d.id))}
                    >
                      <Trash2 aria-hidden className="size-3" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer band: dismissing the popover (outside click or Escape)
            discards drafts, so only the committing action needs a button. */}
        <div className="flex items-center justify-between gap-2 border-t border-border/70 bg-muted/20 px-3 py-1.5">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className={cn(compactControlTier, "text-muted-foreground hover:text-destructive")}
            onClick={handleClearDrafts}
          >
            {t("clearAllHeaders")}
          </Button>

          <Button
            type="button"
            variant="primary"
            size="sm"
            className={cn(compactControlTier, "gap-1")}
            onClick={handleSave}
          >
            <Check aria-hidden className="size-3" />
            {t("done")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
