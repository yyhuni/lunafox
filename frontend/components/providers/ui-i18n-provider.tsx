"use client"

import { useTranslations } from "next-intl"
import { ExpandableI18nProvider } from "@/components/ui/data-table/expandable-cell"

/**
 * UI components i18n provider
 * Provides translations for common UI components like expandable cells
 */
export function UiI18nProvider({ children }: { children: React.ReactNode }) {
  const t = useTranslations("tooltips")

  return (
    <ExpandableI18nProvider expand={t("expand")} collapse={t("collapse")}>
      {children}
    </ExpandableI18nProvider>
  )
}
