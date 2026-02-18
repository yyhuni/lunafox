"use client"

import dynamic from "next/dynamic"
import { Skeleton } from "@/components/ui/skeleton"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const DashboardStatCards = dynamic(
  () => import("@/components/dashboard/dashboard-stat-cards").then((mod) => mod.DashboardStatCards),
  {
    ssr: false,
    loading: () => (
      <div className="grid gap-4 px-4 lg:px-6 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-24" />
        ))}
      </div>
    ),
  }
)

const AssetTrendChart = dynamic(
  () => import("@/components/dashboard/asset-trend-chart").then((mod) => mod.AssetTrendChart),
  {
    ssr: false,
    loading: () => <Skeleton className="h-72 w-full" />,
  }
)

const VulnSeverityChart = dynamic(
  () => import("@/components/dashboard/vuln-severity-chart").then((mod) => mod.VulnSeverityChart),
  {
    ssr: false,
    loading: () => <Skeleton className="h-72 w-full" />,
  }
)

const DashboardDataTable = dynamic(
  () => import("@/components/dashboard/dashboard-data-table").then((mod) => mod.DashboardDataTable),
  {
    ssr: false,
    loading: () => <DataTableSkeleton rows={6} columns={5} withPadding />,
  }
)

export function DashboardLazySections() {
  return (
    <>
      <DashboardStatCards />

      <div className="grid gap-4 px-4 lg:px-6 @xl/main:grid-cols-2">
        <AssetTrendChart />
        <VulnSeverityChart />
      </div>

      <div className="px-4 lg:px-6">
        <DashboardDataTable />
      </div>
    </>
  )
}
