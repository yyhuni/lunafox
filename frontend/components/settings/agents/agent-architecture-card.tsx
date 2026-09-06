"use client"

import { useTranslations } from "next-intl"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { textRole } from "@/lib/typography"
import { ArchitectureFlow } from "./architecture-flow"

export function AgentArchitectureCard() {
  const t = useTranslations("pages.agents")
  const labels = {
    location: t("flowTableLocation"),
    comms: t("flowTableComms"),
    responsibilities: t("flowTableResponsibilities"),
  }
  const roleDetails = [
    {
      id: "server",
      title: t("flowServerTitle"),
      location: t("flowServerLocation"),
      comms: t("flowServerComms"),
      responsibilities: [
        t("flowServerItem1"),
        t("flowServerItem2"),
        t("flowServerItem3"),
      ],
    },
    {
      id: "agent",
      title: t("flowAgentTitle"),
      location: t("flowAgentLocation"),
      comms: t("flowAgentComms"),
      responsibilities: [
        t("flowAgentItem1"),
        t("flowAgentItem2"),
        t("flowAgentItem3"),
      ],
    },
    {
      id: "engine",
      title: t("flowEngineRoleTitle"),
      location: t("flowEngineLocation"),
      comms: t("flowEngineComms"),
      responsibilities: [
        t("flowEngineItem1"),
        t("flowEngineItem2"),
        t("flowEngineItem3"),
      ],
    },
  ]

  return (
    <Card className="shadow-none">
      <CardHeader className="pb-3">
        <CardTitle className={textRole.sectionTitle}>{t("flowTitle")}</CardTitle>
        <CardDescription className={textRole.helperText}>{t("flowDesc")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="space-y-2">
            <div>
              <p className={textRole.compactSectionTitle}>{t("flowDiagramTitle")}</p>
              <p className={textRole.helperText}>
                {t("flowDiagramDesc")}
              </p>
            </div>
            <ArchitectureFlow />
          </div>
          <div className="space-y-2">
            <div>
              <p className={textRole.compactSectionTitle}>{t("flowRolesTitle")}</p>
              <p className={textRole.helperText}>
                {t("flowRolesDesc")}
              </p>
            </div>
            <div className="gap-3 grid md:grid-cols-3">
              {roleDetails.map((role) => (
                <div
                  key={role.id}
                  className="bg-muted/10 border p-3 rounded-md"
                >
                  <p className={textRole.compactPrimary}>{role.title}</p>
                  <div className="mt-2 space-y-1 text-[11px] text-muted-foreground">
                    <p>
                      <span className="text-foreground">
                        {labels.location}:
                      </span>
                      {role.location}
                    </p>
                    <p>
                      <span className="text-foreground">
                        {labels.comms}:
                      </span>
                      {role.comms}
                    </p>
                    <p>
                      <span className="text-foreground">
                        {labels.responsibilities}:
                      </span>
                      {role.responsibilities.join(" / ")}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <Separator />
          <div className="space-y-2">
            <div>
              <p className={textRole.compactSectionTitle}>{t("flowStepsTitle")}</p>
              <p className={textRole.helperText}>
                {t("flowStepsDesc")}
              </p>
            </div>
            <ol className="space-y-1 text-muted-foreground text-xs">
              <li className="flex gap-2">
                <span className="font-medium text-foreground">1.</span>
                <span>{t("flowStep1")}</span>
              </li>
              <li className="flex gap-2">
                <span className="font-medium text-foreground">2.</span>
                <span>{t("flowStep2")}</span>
              </li>
              <li className="flex gap-2">
                <span className="font-medium text-foreground">3.</span>
                <span>{t("flowStep3")}</span>
              </li>
            </ol>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
