"use client"

import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"
import { PageHeader } from "@/components/common/page-header"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const AgentList = dynamic(
  () => import("@/components/settings/workers/worker-list").then((mod) => mod.AgentList),
  {
    ssr: false,
    loading: () => <DataTableSkeleton rows={6} columns={5} withPadding />,
  }
)

export default function WorkersPage() {
  const t = useTranslations("pages.workers")

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="WRK-01"
        title={t("title")}
        description={t("description")}
      />
      <div className="px-4 lg:px-6">
        <AgentList />
      </div>
    </div>
  )
}
