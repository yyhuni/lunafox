"use client"

import * as React from "react"
import * as TabsPrimitive from "@radix-ui/react-tabs"

import { cn } from "@/lib/utils"

function Tabs({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      className={cn("flex flex-col gap-2", className)}
      {...props}
    />
  )
}

interface TabsListProps extends React.ComponentProps<typeof TabsPrimitive.List> {
  variant?: "default" | "underline" | "minimal" | "split" | "minimal-tab"
  size?: "sm" | "md"
}

function TabsList({
  className,
  variant = "default",
  size = "md",
  ...props
}: TabsListProps) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      data-variant={variant}
      className={cn(
        "inline-flex w-fit items-center justify-center",
        variant === "default" && "bg-muted text-muted-foreground h-9 rounded-lg p-[3px]",
        variant === "underline" && "gap-3 bg-transparent p-0",
        variant === "underline" && size === "sm" && "h-8",
        variant === "underline" && size === "md" && "h-9",
        variant === "minimal" && "h-8 gap-3 bg-transparent p-0",
        variant === "split" && "h-8 gap-3 bg-transparent p-0",
        variant === "minimal-tab" && "h-8 gap-4 bg-transparent p-0",
        className
      )}
      {...props}
    />
  )
}

interface TabsTriggerProps extends React.ComponentProps<typeof TabsPrimitive.Trigger> {
  variant?: "default" | "underline" | "minimal" | "split" | "minimal-tab"
  size?: "sm" | "md"
}

function TabsTrigger({
  className,
  variant = "default",
  size = "md",
  ...props
}: TabsTriggerProps) {
  return (
    <TabsPrimitive.Trigger
      data-slot="tabs-trigger"
      data-variant={variant}
      className={cn(
        "inline-flex items-center justify-center gap-1.5 text-sm font-medium whitespace-nowrap cursor-pointer transition-[background-color,border-color,color,box-shadow] focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        variant === "default" && "data-[state=active]:bg-background dark:data-[state=active]:text-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:outline-ring dark:data-[state=active]:border-zinc-500 dark:data-[state=active]:bg-input/30 text-foreground dark:text-muted-foreground h-[calc(100%-1px)] flex-1 rounded-md border border-transparent px-2 py-1 focus-visible:ring-[1px] focus-visible:outline-1 data-[state=active]:shadow-sm",
        variant === "underline" && "text-muted-foreground data-[state=active]:text-foreground px-2 border-b-2 border-transparent data-[state=active]:border-primary rounded-none bg-transparent",
        variant === "underline" && size === "sm" && "h-8 py-1 text-xs",
        variant === "underline" && size === "md" && "h-9 py-1.5",
        variant === "minimal" && "text-muted-foreground data-[state=active]:text-foreground px-2 py-1 text-[11px] font-medium border-b-2 border-transparent data-[state=active]:border-primary rounded-none bg-transparent hover:text-foreground hover:border-primary/50 transition-colors",
        variant === "split" && "text-muted-foreground data-[state=active]:text-foreground px-2 py-1 text-[11px] font-medium border-b border-border rounded-none bg-transparent hover:text-foreground transition-colors data-[state=active]:font-semibold focus-visible:ring-0 shadow-none",
        variant === "minimal-tab" && "text-muted-foreground data-[state=active]:text-foreground px-2 py-1 text-[11px] font-medium border-b-2 border-border data-[state=active]:border-primary rounded-none bg-transparent hover:text-foreground hover:border-primary/50 transition-colors focus-visible:ring-0 focus-visible:ring-offset-0",
        className
      )}
      {...props}
    />
  )
}

function TabsContent({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="tabs-content"
      className={cn("flex-1 outline-none", className)}
      {...props}
    />
  )
}

export { Tabs, TabsList, TabsTrigger, TabsContent }
