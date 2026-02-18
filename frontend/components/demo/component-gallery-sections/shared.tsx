"use client"

import type { ReactNode } from "react"
import { cn } from "@/lib/utils"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

type DemoCardProps = {
  title: string
  description?: string
  children: ReactNode
  className?: string
}

export const DemoCard = ({ title, description, children, className }: DemoCardProps) => (
  <Card className={cn("border-border/70", className)}>
    <CardHeader className="gap-1">
      <CardTitle className="text-base">{title}</CardTitle>
      {description ? (
        <CardDescription className="text-xs">{description}</CardDescription>
      ) : null}
    </CardHeader>
    <CardContent className="space-y-3">{children}</CardContent>
  </Card>
)

export const DemoSection = ({
  id,
  title,
  description,
  children,
}: {
  id?: string
  title: string
  description?: string
  children: ReactNode
}) => (
  <section id={id} className="space-y-4">
    <div className="px-4 lg:px-6">
      <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
      {description ? (
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      ) : null}
    </div>
    {children}
  </section>
)
