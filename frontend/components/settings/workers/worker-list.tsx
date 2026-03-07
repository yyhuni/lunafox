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
      {/* YoRHa Wrapper - Native Theme Adapter */}
      <div className="bg-card border border-border/50 rounded-xl p-6 relative overflow-hidden uppercase font-mono tracking-widest text-foreground shadow-sm mb-6">
        {/* Thin outline border referencing NieR, but using theme border */}
        <div className="absolute inset-2 border border-border/60 pointer-events-none rounded-lg z-0" />
        {/* Decorative pattern using muted color */}
        <div className="absolute top-0 right-0 w-32 h-full bg-[radial-gradient(var(--border)_1px,transparent_1px)] bg-[size:10px_10px] opacity-20 pointer-events-none z-0" />

        <div className="relative z-10 flex flex-col xl:flex-row justify-between items-start xl:items-center gap-8">

          <div className="flex flex-col gap-0 border-l-4 border-primary pl-4 shrink-0">
            <span className="text-[10px] font-bold text-muted-foreground">{t("title")}</span>
            <h3 className="text-foreground text-2xl tracking-[0.3em] font-light">CLUSTER_MAP</h3>
          </div>

          <div className="flex flex-1 gap-6 xl:gap-8 xl:px-8 xl:border-l border-border/60 w-full overflow-hidden flex-wrap">
            <div className="flex flex-col">
              <span className="text-[9px] mb-1 font-bold text-muted-foreground">{stats[0]?.label}</span>
              <span className="text-3xl font-light">{stats[0]?.value || 0}</span>
            </div>
            <div className="flex flex-col">
              <span className="text-[9px] mb-1 font-bold text-muted-foreground">{stats[1]?.label}</span>
              <span className="text-3xl font-light text-emerald-600 dark:text-emerald-400">{stats[1]?.value || 0}</span>
            </div>
            <div className="flex flex-col">
              <span className="text-[9px] mb-1 font-bold text-muted-foreground">{stats[2]?.label}</span>
              <span className={`text-3xl font-light ${Number(stats[2]?.value) > 0 ? "text-destructive" : "text-foreground"}`}>{stats[2]?.value || 0}</span>
            </div>
            <div className="flex flex-col">
              <span className="text-[9px] mb-1 font-bold text-muted-foreground">{stats[3]?.label}</span>
              <span className={`text-3xl font-light ${Number(stats[3]?.value) > 0 ? "text-amber-500" : "text-foreground"}`}>{stats[3]?.value || 0}</span>
            </div>
          </div>

          <div className="flex flex-col sm:flex-row xl:flex-col gap-2 shrink-0">
            <Dialog open={installOpen} onOpenChange={setInstallOpen}>
              <DialogTrigger asChild>
                <button className="bg-primary text-primary-foreground px-4 py-2 text-xs hover:bg-primary/90 transition-colors font-mono flex items-center justify-between gap-6">
                  <span>{t("install.openDialog")}</span>
                  <span className="opacity-50 text-[10px]">01</span>
                </button>
              </DialogTrigger>
              <AgentInstallDialog
                open={installOpen}
                token={token}
                isGenerating={createToken.isPending}
                onGenerate={handleGenerateToken}
              />
            </Dialog>

            <ArchitectureDialog trigger={
              <button className="border border-border text-foreground px-4 py-2 text-xs hover:bg-muted transition-colors font-mono flex items-center justify-between gap-6">
                <span>{t("viewArchitecture")}</span>
                <span className="opacity-50 text-[10px]">02</span>
              </button>
            } />
          </div>
        </div>

        {/* Node Bar */}
        <div className="mt-8 relative z-10 flex items-center gap-2 border-t border-border/60 pt-4 max-w-full overflow-hidden">
          <span className="text-[10px] font-bold text-muted-foreground shrink-0">STATUS</span>
          <div className="flex-1 h-2 flex gap-[2px]">
            {hasAgents ? [...agents].sort((a, b) => {
              const priority = (agent: typeof a) => {
                const isOnline = agent.status === "online"
                if (!isOnline) return 2 // red
                const isHealthy = agent.health?.state?.toLowerCase() === "ok" || !agent.health?.state
                return isHealthy ? 0 : 1 // green : amber
              }
              return priority(a) - priority(b)
            }).map((agent) => {
              const isHealthy = agent.health?.state?.toLowerCase() === "ok" || !agent.health?.state
              const isOnline = agent.status === "online"
              return (
                <div
                  key={agent.id}
                  className={`h-full flex-1 border-r border-background ${!isOnline ? "bg-destructive/80" : isHealthy ? "bg-emerald-500/80" : "bg-amber-500/80"}`}
                  title={`${agent.name} - ${agent.status}`}
                />
              )
            }) : (
              <span className="text-[10px] text-muted-foreground/50 font-mono flex-1">—</span>
            )}
          </div>
        </div>
      </div>

      <div className="space-y-4">

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
