import React from "react"
import { useTranslations } from "next-intl"

export type ArchitectureLabels = {
  location: string
  comms: string
  responsibilities: string
}

export type ArchitectureRoleDetail = {
  id: string
  title: string
  location: string
  comms: string
  responsibilities: string[]
}

export function useArchitectureDialogState() {
  const t = useTranslations("pages.workers")
  const [open, setOpen] = React.useState(false)

  const labels = React.useMemo<ArchitectureLabels>(() => ({
    location: t("flowTableLocation"),
    comms: t("flowTableComms"),
    responsibilities: t("flowTableResponsibilities"),
  }), [t])

  const roleDetails = React.useMemo<ArchitectureRoleDetail[]>(() => [
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
  ], [t])

  const flowSteps = React.useMemo(() => [
    t("flowStep1"),
    t("flowStep2"),
    t("flowStep3"),
  ], [t])

  return {
    t,
    open,
    setOpen,
    labels,
    roleDetails,
    flowSteps,
  }
}
