"use client"

import { useTranslations } from "next-intl"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"
import { ScanHistoryList } from "@/components/scan/history/scan-history-list"
import { semanticIcons } from "@/components/icons"

const VulnerabilityIcon = semanticIcons.concept.vulnerability
const ScanIcon = semanticIcons.concept.scan

export function OverviewActivityTabs() {
  const t = useTranslations("overview.activityTabs")

  return (
    <Tabs defaultValue="vulnerabilities" className="w-full">
      <TabsList className="mb-4">
        <TabsTrigger value="vulnerabilities" className="gap-1.5">
          <VulnerabilityIcon className="h-4 w-4" />
          {t("vulnerabilities")}
        </TabsTrigger>
        <TabsTrigger value="scans" className="gap-1.5">
          <ScanIcon className="h-4 w-4" />
          {t("scanHistory")}
        </TabsTrigger>
      </TabsList>

      <TabsContent value="vulnerabilities" className="mt-0">
        <VulnerabilitiesDetailView hideToolbar />
      </TabsContent>
      <TabsContent value="scans" className="mt-0">
        <ScanHistoryList hideToolbar />
      </TabsContent>
    </Tabs>
  )
}
