"use client"

import { IconPlus, semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS } from "./agent-layout-contract"

const AgentIcon = semanticIcons.concept.agent

type AgentExpansionSlotProps = {
  title: string
  description: string
  actionLabel: string
  onOpenInstall: () => void
}

export function AgentExpansionSlot({
  title,
  description,
  actionLabel,
  onOpenInstall,
}: AgentExpansionSlotProps) {
  return (
    <Button
      type="button"
      variant="outline"
      size="content"
      onClick={onOpenInstall}
      className={cn(
        AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS,
        "group w-full flex-col gap-4 rounded-lg border-dashed bg-muted/10 px-6 py-8 text-center shadow-none hover:border-primary/40 hover:bg-primary/[0.03]"
      )}
      aria-label={title}
    >
      <span className="relative flex size-14 items-center justify-center rounded-full border border-dashed border-border bg-background text-muted-foreground transition-[color,border-color,background-color] group-hover:border-primary/40 group-hover:bg-primary/5 group-hover:text-primary">
        <AgentIcon className="size-6" aria-hidden="true" />
        <IconPlus className="absolute -right-1 -bottom-1 size-4 rounded-full bg-primary p-0.5 text-primary-foreground" aria-hidden="true" />
      </span>
      <span className="space-y-1.5 whitespace-normal">
        <span className={cn("block text-foreground", textRole.panelTitle)}>{title}</span>
        <span className={cn("block", textRole.bodySubtle)}>{description}</span>
      </span>
      <span className={cn("text-primary", textRole.bodyStrong)}>{actionLabel}</span>
    </Button>
  )
}
