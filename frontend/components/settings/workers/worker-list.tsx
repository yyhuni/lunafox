"use client"

import { useMemo, useState } from "react"
import { useTranslations } from "next-intl"
import {
  IconServer,
  IconCloud,
  IconCloudOff,
  IconHeartbeat,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  Card,
} from "@/components/ui/card"
import {
  Dialog,
  DialogTrigger,
} from "@/components/ui/dialog"
import { ConfirmDialog } from "@/components/ui/confirm-dialog"
import { Skeleton } from "@/components/ui/skeleton"
import {
  useAgents,
  useCreateRegistrationToken,
  useDeleteAgent,
} from "@/hooks/use-agents"
import type { Agent, RegistrationTokenResponse } from "@/types/agent.types"
import { AgentConfigDialog } from "./worker-dialog"
import { AgentCardCompact } from "./agent-card-compact"
import { AgentInstallDialog } from "./agent-install-dialog"
import { ArchitectureDialog } from "./architecture-dialog"
import { AgentLogDrawer } from "./agent-log-drawer"

function EmptyState({ onOpenInstall }: { onOpenInstall: () => void }) {
  const t = useTranslations("settings.workers")

  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="mb-4 rounded-full bg-muted p-4">
        <IconServer className="h-12 w-12 text-muted-foreground" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{t("empty.title")}</h3>
      <p className="text-sm text-muted-foreground mb-6 max-w-md">{t("empty.desc")}</p>
      <Button onClick={onOpenInstall}>{t("empty.cta")}</Button>
    </div>
  )
}

export function AgentList() {
  const t = useTranslations("settings.workers")
  const [page] = useState(1)
  const [pageSize] = useState(100)
  const [installOpen, setInstallOpen] = useState(false)
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null)
  const [configDialogOpen, setConfigDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [agentToDelete, setAgentToDelete] = useState<Agent | null>(null)
  const [token, setToken] = useState<RegistrationTokenResponse | null>(null)
  const [logDrawerOpen, setLogDrawerOpen] = useState(false)
  const [logAgent, setLogAgent] = useState<Agent | null>(null)

  const { data, isLoading } = useAgents(page, pageSize)
  const createToken = useCreateRegistrationToken()
  const deleteAgent = useDeleteAgent()

  const agents = useMemo(() => data?.results || [], [data?.results])
  const hasAgents = agents.length > 0

  const stats = useMemo(() => {
    const total = data?.total ?? agents.length
    const online = agents.filter((agent) => agent.status === "online").length
    const offline = agents.filter((agent) => agent.status === "offline").length
    const unhealthy = agents.filter((agent) => {
      const state = agent.health?.state?.toLowerCase()
      return state && state !== "ok"
    }).length

    return [
      { label: t("stats.total"), value: total, icon: IconServer },
      { label: t("stats.online"), value: online, icon: IconCloud },
      { label: t("stats.offline"), value: offline, icon: IconCloudOff },
      { label: t("stats.unhealthy"), value: unhealthy, icon: IconHeartbeat },
    ]
  }, [agents, data?.total, t])

  const handleGenerateToken = async () => {
    try {
      const response = await createToken.mutateAsync()
      setToken(response)
    } catch {
      // handled by hook
    }
  }

  const handleConfigure = (agent: Agent) => {
    setSelectedAgent(agent)
    setConfigDialogOpen(true)
  }


  const handleDelete = (agent: Agent) => {
    setAgentToDelete(agent)
    setDeleteDialogOpen(true)
  }

  const handleOpenLogs = (agent: Agent) => {
    setLogAgent(agent)
    setLogDrawerOpen(true)
  }

  const confirmDelete = async () => {
    if (!agentToDelete) return
    try {
      await deleteAgent.mutateAsync(agentToDelete.id)
      setDeleteDialogOpen(false)
      setAgentToDelete(null)
    } catch {
      // handled by hook
    }
  }


  return (
    <div className="space-y-6">
      {hasAgents && (
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          {stats.map((stat) => (
            <Card key={stat.label} className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg border bg-muted/40 p-2">
                  <stat.icon className="h-5 w-5 text-muted-foreground" />
                </div>
                <div>
                  <p className="text-2xl font-semibold tracking-tight tabular-nums">{stat.value}</p>
                  <p className="text-xs text-muted-foreground">{stat.label}</p>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      <div className="space-y-4">
        <div className="flex justify-end">
          <div className="flex items-center gap-2">
            <ArchitectureDialog />
            <Dialog open={installOpen} onOpenChange={setInstallOpen}>
              <DialogTrigger asChild>
                <Button size="sm">
                  <IconHeartbeat className="h-4 w-4 mr-2" />
                  {t("install.openDialog")}
                </Button>
              </DialogTrigger>
              <AgentInstallDialog
                open={installOpen}
                token={token}
                isGenerating={createToken.isPending}
                onGenerate={handleGenerateToken}
              />

            </Dialog>
          </div>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-52 w-full rounded-lg" />
            ))}
          </div>
        ) : !hasAgents ? (
          <EmptyState onOpenInstall={() => setInstallOpen(true)} />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {agents.map((agent) => (
              <AgentCardCompact
                key={agent.id}
                agent={agent}
                onConfig={handleConfigure}
                onDelete={handleDelete}
                onLogs={handleOpenLogs}
              />
            ))}
          </div>
        )}
      </div>

      <AgentConfigDialog
        open={configDialogOpen}
        onOpenChange={setConfigDialogOpen}
        agent={selectedAgent}
      />

      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title={t("actions.deleteTitle")}
        description={t("actions.deleteDesc", { name: agentToDelete?.name ?? "" })}
        onConfirm={confirmDelete}
        variant="destructive"
        loading={deleteAgent.isPending}
      />

      <AgentLogDrawer
        open={logDrawerOpen}
        onOpenChange={setLogDrawerOpen}
        agent={logAgent}
      />

    </div>
  )
}
