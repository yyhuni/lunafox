"use client"

import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { PageHeader } from "@/components/common/page-header"
import { useTranslations } from "next-intl"

const OrganizationList = dynamic(
  () => import("@/components/organization/organization-list").then((mod) => mod.OrganizationList),
  {
    ssr: false,
    loading: () => <DataTableSkeleton rows={6} columns={4} withPadding />,
  }
)

/**
 * Organization management page
 * Sub-page under asset management that displays organization list and related operations
 */
export default function OrganizationPage() {
  const t = useTranslations("pages.organization")

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="ORG-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Organization list component */}
      <div className="px-4 lg:px-6">
        <OrganizationList />
      </div>
    </div>
  )
}
