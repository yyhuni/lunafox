"use client"

import { useTranslations } from "next-intl"

import { ChevronRight, FileText } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, cardVariants } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Wordlist } from "@/types/wordlist.types"

import { formatWordlistFileSize, formatWordlistUpdatedAt } from "./wordlist-formatters"

const WORDLIST_CARD_SURFACE_CLASS = cn(
  cardVariants({ variant: "shell" }),
  "min-h-48 w-full min-w-0 items-stretch justify-start gap-0 overflow-hidden py-0 text-left whitespace-normal"
)
const WORDLIST_CARD_HEADER_CLASS = "flex min-w-0 gap-2.5 px-3.5 pt-3 pb-2"
const WORDLIST_CARD_TAGS_CLASS = "flex min-h-8 min-w-0 flex-wrap content-start gap-1 px-3.5 py-2"
const WORDLIST_CARD_METRICS_CLASS = "grid grid-cols-2 gap-3 px-3.5 py-2"
const WORDLIST_CARD_FOOTER_CLASS = "mt-auto flex min-w-0 items-center justify-between gap-2 px-3.5 pt-2 pb-3"

interface WordlistCatalogCardProps {
  wordlist: Wordlist
  locale: string
  onSelect: (wordlist: Wordlist) => void
}

export function WordlistCatalogCard({ wordlist, locale, onSelect }: WordlistCatalogCardProps) {
  const t = useTranslations("pages.wordlists")
  const visibleTags = wordlist.tags.slice(0, 2)
  const hiddenTagCount = wordlist.tags.length - visibleTags.length

  return (
    <Button
      type="button"
      variant="outline"
      size="content"
      className={cn(
        WORDLIST_CARD_SURFACE_CLASS,
        "group hover:border-foreground/30 hover:bg-card focus-visible:border-ring focus-visible:ring-ring/40"
      )}
      aria-label={`${t("detailTitle")}: ${wordlist.fileName}`}
      onClick={() => onSelect(wordlist)}
    >
      <span className={WORDLIST_CARD_HEADER_CLASS}>
        <span className="flex size-10 shrink-0 items-center justify-center text-muted-foreground transition-colors group-hover:text-highlight group-focus-visible:text-highlight">
          <FileText aria-hidden="true" className="size-5" />
        </span>
        <span className="min-w-0 space-y-1">
          <span className={cn("block truncate", textRole.sectionTitle)}>{wordlist.fileName}</span>
          <span className={cn("block line-clamp-1 min-h-4", textRole.compactCaption)}>
            {wordlist.description || "-"}
          </span>
        </span>
      </span>

      <span className={WORDLIST_CARD_TAGS_CLASS}>
        {visibleTags.length === 0 ? (
          <span className={textRole.caption}>{t("tagEmpty")}</span>
        ) : (
          visibleTags.map((tag) => <Badge key={tag} variant="secondary">{tag}</Badge>)
        )}
        {hiddenTagCount > 0 ? <Badge variant="secondary">+{hiddenTagCount}</Badge> : null}
      </span>

      <span className={WORDLIST_CARD_METRICS_CLASS}>
        <WordlistMetric label={t("rows")} value={wordlist.lineCount?.toLocaleString() ?? "-"} />
        <WordlistMetric label={t("size")} value={formatWordlistFileSize(wordlist.fileSize)} />
      </span>

      <span className={WORDLIST_CARD_FOOTER_CLASS}>
        <span className={cn("min-w-0 truncate", textRole.caption)} title={formatWordlistUpdatedAt(wordlist.updatedAt, locale)}>
          {t("updatedAt")}: <span className="text-foreground">{formatWordlistUpdatedAt(wordlist.updatedAt, locale)}</span>
        </span>
        <ChevronRight aria-hidden="true" className="size-4 shrink-0 text-muted-foreground transition-colors group-hover:text-foreground" />
      </span>
    </Button>
  )
}

function WordlistMetric({ label, value }: { label: string; value: string }) {
  return (
    <span className="min-w-0">
      <span className={cn("block", textRole.helperText)}>{label}</span>
      <span className={cn("mt-0.5 block truncate tabular-nums", textRole.compactPrimary)}>{value}</span>
    </span>
  )
}

export function WordlistCatalogCardLoadingState() {
  return (
    <Card data-slot="wordlist-card-loading-state" className={WORDLIST_CARD_SURFACE_CLASS}>
      <div className={WORDLIST_CARD_HEADER_CLASS}>
        <div className="flex size-10 shrink-0 items-center justify-center">
          <FileText aria-hidden="true" className="size-5 text-muted-foreground" />
        </div>
        <div className="min-w-0 flex-1 space-y-1.5">
          <Skeleton className="h-4 w-3/5 radius-control" />
          <Skeleton className="h-3 w-full radius-control" />
        </div>
      </div>
      <div className={WORDLIST_CARD_TAGS_CLASS}>
        <Skeleton className="h-6 w-16 radius-badge" />
        <Skeleton className="h-6 w-20 radius-badge" />
        <Skeleton className="h-6 w-8 radius-badge" />
      </div>
      <div className={WORDLIST_CARD_METRICS_CLASS}>
        <div className="space-y-2">
          <Skeleton className="h-3 w-10 radius-control" />
          <Skeleton className="h-4 w-16 radius-control" />
        </div>
        <div className="space-y-2">
          <Skeleton className="h-3 w-10 radius-control" />
          <Skeleton className="h-4 w-14 radius-control" />
        </div>
      </div>
      <div className={WORDLIST_CARD_FOOTER_CLASS}>
        <Skeleton className="h-3 w-28 radius-control" />
        <Skeleton className="size-3.5 radius-control" />
      </div>
    </Card>
  )
}
