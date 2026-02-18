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
import { ArchitectureFlow } from "./architecture-flow"

export function AgentArchitectureCard() {
  const t = useTranslations("pages.workers")
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
      id: "worker",
      title: t("flowWorkerTitle"),
      location: t("flowWorkerLocation"),
      comms: t("flowWorkerComms"),
      responsibilities: [
        t("flowWorkerItem1"),
        t("flowWorkerItem2"),
        t("flowWorkerItem3"),
      ],
    },
  ]

  return (
    <Card className="shadow-none">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">{t("flowTitle")}</CardTitle>
        <CardDescription className="text-xs">{t("flowDesc")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="space-y-2">
            <div>
              <p className="text-xs font-medium">{t("flowDiagramTitle")}</p>
              <p className="text-xs text-muted-foreground">
                {t("flowDiagramDesc")}
              </p>
            </div>
            <ArchitectureFlow />
          </div>
          <div className="space-y-2">
            <div>
              <p className="text-xs font-medium">{t("flowRolesTitle")}</p>
              <p className="text-xs text-muted-foreground">
                {t("flowRolesDesc")}
              </p>
            </div>
            <div className="grid gap-3 md:grid-cols-3">
              {roleDetails.map((role) => (
                <div
                  key={role.id}
                  className="rounded-md border bg-muted/10 p-3"
                >
                  <p className="text-xs font-medium">{role.title}</p>
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
              <p className="text-xs font-medium">{t("flowStepsTitle")}</p>
              <p className="text-xs text-muted-foreground">
                {t("flowStepsDesc")}
              </p>
            </div>
            <ol className="space-y-1 text-xs text-muted-foreground">
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
