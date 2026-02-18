"use client"

import { Loader2Icon } from "@/components/icons"
import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

function Spinner({ className, ...props }: React.ComponentPropsWithoutRef<typeof Loader2Icon>) {
  const t = useTranslations("common.ui")
  
  return (
    <Loader2Icon
      role="status"
      aria-label={t("loading")}
      className={cn("size-4 animate-spin", className)}
      {...props}
    />
  )
}

export { Spinner }
