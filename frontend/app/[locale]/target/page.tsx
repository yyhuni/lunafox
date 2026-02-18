"use client"

import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { PageHeader } from "@/components/common/page-header"
import { useTranslations } from "next-intl"

const AllTargetsDetailView = dynamic(
  () => import("@/components/target/all-targets-detail-view").then((mod) => mod.AllTargetsDetailView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton rows={6} columns={5} withPadding />,
  }
)

export default function AllTargetsPage() {
  const t = useTranslations("pages.target")

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="TGT-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Target list */}
      <div className="px-4 lg:px-6">
        <AllTargetsDetailView />
      </div>
    </div>
  )
}
