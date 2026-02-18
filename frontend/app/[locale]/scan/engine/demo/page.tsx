
"use client"

import Link from "next/link"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ArrowRight, LayoutGrid, IconListDetails, GitBranch, IconLayoutColumns } from "@/components/icons"

export default function DemoIndexPage() {
  const demos = [
    {
      title: "Design A: Card Grid",
      description: "Visual-first layout focusing on quick identification and status overview. Best for small to medium number of engines.",
      path: "scan/engine/demo/card-grid",
      icon: <LayoutGrid className="h-8 w-8 text-primary" />,
      color: "bg-blue-500/10"
    },
    {
      title: "Design B: Data Table",
      description: "Information-dense layout for managing many engines with sorting and filtering capabilities. Best for power users.",
      path: "scan/engine/demo/table-list",
      icon: <IconListDetails className="h-8 w-8 text-primary" />,
      color: "bg-green-500/10"
    },
    {
      title: "Design C: Hybrid View",
      description: "Balanced approach with a side menu for navigation and detailed view for configuration. Offers best context retention.",
      path: "scan/engine/demo/hybrid",
      icon: <IconLayoutColumns className="h-8 w-8 text-primary" />, 
      color: "bg-orange-500/10"
    },
    {
      title: "Design D: Feature Flow",
      description: "Conceptual layout visualizing engines as processing pipelines. Best for understanding complex scan logic.",
      path: "scan/engine/demo/feature-flow",
      icon: <GitBranch className="h-8 w-8 text-primary" />,
      color: "bg-purple-500/10"
    }
  ]

  return (
    <div className="container mx-auto py-10 max-w-5xl">
      <div className="mb-10 text-center">
        <h1 className="text-3xl font-bold tracking-tight mb-3">Scan Engine Design Concepts</h1>
        <p className="text-muted-foreground max-w-2xl mx-auto">
          Exploring different UX patterns for managing scan engine configurations. 
          Select a demo below to interact with the design prototype.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {demos.map((demo) => (
          <Link href={`/${demo.path}`} key={demo.path} className="block group h-full">
            <Card className="h-full transition-[border-color,box-shadow,transform] duration-300 hover:shadow-lg hover:-translate-y-1 border-muted hover:border-primary/50">
              <CardHeader>
                <div className={`w-14 h-14 rounded-2xl flex items-center justify-center mb-4 ${demo.color}`}>
                  {demo.icon}
                </div>
                <CardTitle className="group-hover:text-primary transition-colors">
                  {demo.title}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <CardDescription className="mb-6 min-h-[80px]">
                  {demo.description}
                </CardDescription>
                <div className="flex items-center text-sm font-medium text-primary">
                  View Demo <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
                </div>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  )
}
