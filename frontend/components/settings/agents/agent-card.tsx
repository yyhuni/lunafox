"use client";
import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { IconSettings, IconTrash, IconChevronDown, IconChevronUp, IconAlertTriangle, IconActivity, } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, } from "@/components/ui/card";
import { DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, } from "@/components/ui/dropdown-menu";
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners";
import { Collapsible, CollapsibleContent, CollapsibleTrigger, } from "@/components/ui/collapsible";
import { Status, StatusIndicator, StatusLabel } from "@/components/shared/feedback/status";
import { useFormatNumber, useFormatHeartbeatTime } from "@/lib/i18n-format";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
import type { Agent } from "@/types/agent.types";
import { MetricProgress } from "./metric-progress";
import { getAgentHealthBadgeVariant, getAgentHealthClassName, getAgentRuntimeBorderClass, getAgentRuntimeShellClass, getAgentHeartbeatTextClass, getAgentRuntimeStatus, getAgentRuntimeTextClass, } from "./agent-status";
import { getAgentConnectionIpDisplay } from "./agent-connection-ip";
function formatUptime(seconds?: number | null) {
    if (seconds === null || seconds === undefined)
        return "-";
    const total = Math.max(0, Math.floor(seconds));
    const minutes = Math.floor(total / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    if (days > 0)
        return `${days}d ${hours % 24}h`;
    if (hours > 0)
        return `${hours}h ${minutes % 60}m`;
    return `${minutes}m`;
}
interface AgentCardProps {
    agentNode: Agent;
    onConfig: (agentNode: Agent) => void;
    onDelete: (agentNode: Agent) => void;
}
export function AgentCard({ agentNode, onConfig, onDelete, }: AgentCardProps) {
    const t = useTranslations("settings.agents");
    const formatHeartbeatTime = useFormatHeartbeatTime();
    const formatNumber = useFormatNumber();
    const connectionIpDisplay = getAgentConnectionIpDisplay(agentNode, {
        offline: t("connectionIp.offline"),
        unobserved: t("connectionIp.unobserved"),
    });
    const [isExpanded, setIsExpanded] = useState(false);
    const statusLabel = useMemo(() => {
        if (agentNode.status === "online")
            return t("status.online");
        if (agentNode.status === "offline")
            return t("status.offline");
        return t("status.unknown");
    }, [agentNode.status, t]);
    const healthState = (agentNode.health?.state || "unknown").toLowerCase();
    const healthLabel = useMemo(() => {
        if (healthState === "ok")
            return t("health.ok");
        if (healthState === "warning" || healthState === "warn")
            return t("health.warning");
        if (healthState === "error" || healthState === "critical")
            return t("health.warning");
        return t("health.unknown");
    }, [healthState, t]);
    const heartbeat = agentNode.heartbeat;
    // Check if any metric exceeds the threshold
    const hasWarnings = useMemo(() => {
        if (!heartbeat)
            return false;
        return (heartbeat.cpu >= agentNode.cpuThreshold ||
            heartbeat.mem >= agentNode.memThreshold ||
            heartbeat.disk >= agentNode.diskThreshold);
    }, [heartbeat, agentNode]);
    // Calculate last heartbeat time difference (seconds)
    const lastHeartbeatSeconds = useMemo(() => {
        if (!agentNode.lastHeartbeat)
            return null;
        const now = Date.now();
        const lastHeartbeat = new Date(agentNode.lastHeartbeat).getTime();
        return Math.floor((now - lastHeartbeat) / 1000);
    }, [agentNode.lastHeartbeat]);
    // Determine whether the heartbeat has expired (more than 30 seconds)
    const isHeartbeatStale = lastHeartbeatSeconds !== null && lastHeartbeatSeconds > 30;
    return (<Card className={cn("transition-[background-color,border-color,box-shadow,opacity] duration-200 hover:shadow-md", getAgentRuntimeBorderClass(agentNode.status), getAgentRuntimeShellClass(agentNode.status))}>
      <CardHeader className="pb-3">
        <div className="flex gap-3 items-start justify-between">
          <div className="flex-1 min-w-0 space-y-2">
            <div className="flex flex-wrap gap-2 items-center">
              <CardTitle className={cn(textRole.bodyStrong, "truncate")}>{agentNode.displayName}</CardTitle>
              {agentNode.agentVersion && (<Badge variant="secondary" className="shrink-0 text-[10px]">
                  {agentNode.agentVersion}
                </Badge>)}
            </div>

            <div className="flex flex-wrap gap-2 items-center">
              <Status status={getAgentRuntimeStatus(agentNode.status)}>
                <StatusIndicator />
                <StatusLabel>{statusLabel}</StatusLabel>
              </Status>
              <Badge variant={getAgentHealthBadgeVariant(healthState)} className={getAgentHealthClassName(healthState)}>
                {healthLabel}
              </Badge>
              {hasWarnings && (<Badge variant="warning">
                  <IconAlertTriangle className="h-3 mr-1 w-3"/>
                  {t("card.warning")}
                </Badge>)}
            </div>

            <div className="flex gap-1 items-center text-muted-foreground text-xs">
              <span className="truncate">
                {agentNode.observedHostname || t("unknownHost")} · {connectionIpDisplay}
              </span>
            </div>
          </div>

          <DenseRowActionMenu ariaLabel={t("actions.title")} ownerClassName="shrink-0">
            <DropdownMenuLabel>{t("actions.title")}</DropdownMenuLabel>
            <DropdownMenuItem onClick={() => onConfig(agentNode)}>
              <IconSettings className="h-4 w-4"/>
              {t("actions.config")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={() => onDelete(agentNode)}>
              <IconTrash className="h-4 w-4"/>
              {t("actions.delete")}
            </DropdownMenuItem>
          </DenseRowActionMenu>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Basic information - always shown */}
        <div className="gap-2 grid grid-cols-2 text-xs">
          <div className="bg-muted/40 border p-2.5 rounded-lg">
            <p className="mb-1 text-muted-foreground">{t("metrics.lastHeartbeat")}</p>
            <div className="flex gap-1 items-center">
              <p className={cn("font-medium", getAgentHeartbeatTextClass())} title={formatHeartbeatTime(agentNode.lastHeartbeat).title}>
                {formatHeartbeatTime(agentNode.lastHeartbeat).display}
              </p>
              {agentNode.status === "online" && !isHeartbeatStale && (<IconActivity className={cn("h-3 w-3", getAgentRuntimeTextClass("online"))}/>)}
            </div>
          </div>
          <div className="bg-muted/40 border p-2.5 rounded-lg">
            <p className="mb-1 text-muted-foreground">{t("metrics.connectedAt")}</p>
            <p className="font-medium" title={formatHeartbeatTime(agentNode.connectedAt).title}>
              {formatHeartbeatTime(agentNode.connectedAt).display}
            </p>
          </div>
        </div>

        {/* Collapsible details */}
        {heartbeat && (<Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
            <CollapsibleTrigger render={<Button variant="ghost" size="sm" layout="between"/>}>
                <span>{isExpanded ? t("card.hideDetails") : t("card.showDetails")}</span>
                {isExpanded ? (<IconChevronUp className="h-4 w-4"/>) : (<IconChevronDown className="h-4 w-4"/>)}
              </CollapsibleTrigger>

            <CollapsibleContent className="pt-3 space-y-3">
              {/* System Metrics - with progress bar and threshold warnings */}
              <div className="bg-muted/20 border p-3 rounded-lg space-y-2.5">
                <p className={cn("mb-2 text-muted-foreground", textRole.compactPrimary)}>
                  {t("card.systemMetrics")}
                </p>
                <MetricProgress label={t("metrics.cpu")} value={heartbeat.cpu} threshold={agentNode.cpuThreshold}/>
                <MetricProgress label={t("metrics.mem")} value={heartbeat.mem} threshold={agentNode.memThreshold}/>
                <MetricProgress label={t("metrics.disk")} value={heartbeat.disk} threshold={agentNode.diskThreshold}/>
              </div>

              {/* Execution load and run time */}
              <div className="gap-2 grid grid-cols-3 text-xs">
                <div className="bg-muted/30 border p-2.5 rounded-lg">
                  <p className="mb-1 text-muted-foreground">{t("metrics.runningTasks")}</p>
                  <p className={textRole.metricValueDisplay}>{formatNumber.formatInteger(heartbeat.runningTasks)}</p>
                </div>
                <div className="bg-muted/30 border p-2.5 rounded-lg">
                  <p className="mb-1 text-muted-foreground">{t("metrics.usedTaskSlots")}</p>
                  <p className={textRole.metricValueDisplay}>
                    {formatNumber.formatInteger(heartbeat.taskSlotsUsed)}
                    <span className="ml-1 text-muted-foreground text-xs">/ {agentNode.maxTasks}</span>
                  </p>
                </div>
                <div className="bg-muted/30 border p-2.5 rounded-lg">
                  <p className="mb-1 text-muted-foreground">{t("metrics.uptime")}</p>
                  <p className={textRole.metricValueDisplay}>{formatUptime(heartbeat.uptime)}</p>
                </div>
              </div>

              {/* Configuration information */}
              <div className="bg-muted/20 border p-3 rounded-lg">
                <p className={cn("mb-2 text-muted-foreground", textRole.compactPrimary)}>
                  {t("card.configuration")}
                </p>
                <div className="flex flex-wrap gap-1.5">
                  <Badge variant="outline" className="text-[10px]">
                    {t("metrics.maxTasks", { value: agentNode.maxTasks })}
                  </Badge>
                  <Badge variant="outline" className="text-[10px]">
                    CPU: {agentNode.cpuThreshold}%
                  </Badge>
                  <Badge variant="outline" className="text-[10px]">
                    Mem: {agentNode.memThreshold}%
                  </Badge>
                  <Badge variant="outline" className="text-[10px]">
                    Disk: {agentNode.diskThreshold}%
                  </Badge>
                </div>
              </div>

              {/* Last updated */}
              {heartbeat.updatedAt && (<div className="pt-1 text-[10px] text-center text-muted-foreground">
                  {t("metrics.updatedAt", { value: formatHeartbeatTime(heartbeat.updatedAt).display })}
                </div>)}
            </CollapsibleContent>
          </Collapsible>)}

        {/* Prompt when there is no heartbeat data */}
        {!heartbeat && (<div className="bg-muted/20 border border-dashed p-4 rounded-lg text-center">
            <p className="text-muted-foreground text-xs">
              {t("card.waitingForHeartbeat")}
            </p>
          </div>)}
      </CardContent>
    </Card>);
}
