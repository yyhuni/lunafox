"use client"

import React, { useState, useEffect } from "react"
import { useTranslations } from "next-intl"
import { AlertTriangle, Loader2, Ban } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useGlobalBlacklist, useUpdateGlobalBlacklist } from "@/hooks/use-global-blacklist"
import { PageHeader } from "@/components/common/page-header"

/**
 * Global blacklist settings page
 */
export default function GlobalBlacklistPage() {
  const t = useTranslations("pages.settings.blacklist")
  
  const [blacklistText, setBlacklistText] = useState("")
  const [hasChanges, setHasChanges] = useState(false)

  const { data, isLoading, error } = useGlobalBlacklist()
  const updateBlacklist = useUpdateGlobalBlacklist()

  // Initialize text when data loads
  useEffect(() => {
    if (data?.patterns) {
      setBlacklistText(data.patterns.join("\n"))
      setHasChanges(false)
    }
  }, [data])

  // Handle text change
  const handleTextChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setBlacklistText(e.target.value)
    setHasChanges(true)
  }

  // Handle save
  const handleSave = () => {
    const patterns = blacklistText
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0)

    updateBlacklist.mutate(
      { patterns },
      {
        onSuccess: () => {
          setHasChanges(false)
        },
      }
    )
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <PageHeader
          code="BLK-01"
          title={t("title")}
          description={t("description")}
        />
        <div className="px-4 lg:px-6 space-y-4">
          <div className="space-y-2">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-96" />
          </div>
          <Skeleton className="h-[400px] w-full" />
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center py-12">
        <AlertTriangle className="h-10 w-10 text-destructive mb-4" />
        <p className="text-muted-foreground">{t("loadError")}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="BLK-01"
        title={t("title")}
        description={t("description")}
      />

      <div className="px-4 lg:px-6">
        {/* Blacklist card */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Ban className="h-5 w-5 text-muted-foreground" />
              <CardTitle>{t("card.title")}</CardTitle>
            </div>
            <CardDescription>{t("card.description")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Rules hint */}
            <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-muted-foreground">
              <span className="font-medium text-foreground">{t("rules.title")}:</span>
              <span><code className="bg-muted px-1.5 py-0.5 rounded text-xs">*.gov</code> {t("rules.domain")}</span>
              <span><code className="bg-muted px-1.5 py-0.5 rounded text-xs">*cdn*</code> {t("rules.keyword")}</span>
              <span><code className="bg-muted px-1.5 py-0.5 rounded text-xs">192.168.1.1</code> {t("rules.ip")}</span>
              <span><code className="bg-muted px-1.5 py-0.5 rounded text-xs">10.0.0.0/8</code> {t("rules.cidr")}</span>
            </div>

            {/* Scope hint */}
            <div className="rounded-lg border bg-muted/50 p-3 text-sm">
              <p className="text-muted-foreground">{t("scopeHint")}</p>
            </div>

            {/* Input */}
            <Textarea
              name="blacklistRules"
              autoComplete="off"
              value={blacklistText}
              onChange={handleTextChange}
              placeholder={t("placeholder")}
              className="min-h-[320px] font-mono text-sm"
            />

            {/* Save button */}
            <div className="flex justify-end">
              <Button
                onClick={handleSave}
                disabled={!hasChanges || updateBlacklist.isPending}
              >
                {updateBlacklist.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                {t("save")}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
