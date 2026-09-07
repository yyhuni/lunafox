"use client"

import * as React from "react"
import { IconRefresh, Loader2Icon } from "@/components/icons"
import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

const Spinner = React.memo(function Spinner({ className, ...props }: React.ComponentPropsWithoutRef<typeof Loader2Icon>) {
  const t = useTranslations("common.ui")
  
  return (
    <Loader2Icon
      data-slot="spinner"
      role="status"
      aria-live="polite"
      aria-label={t("loading")}
      className={cn("loading-spinner size-4", className)}
      {...props}
    />
  )
})

const RefreshSpinner = React.memo(function RefreshSpinner({ className, ...props }: React.ComponentPropsWithoutRef<typeof IconRefresh>) {
  return (
    <IconRefresh
      aria-hidden="true"
      data-slot="refresh-spinner"
      className={cn("loading-spinner size-4", className)}
      {...props}
    />
  )
})

export { Spinner, RefreshSpinner }
