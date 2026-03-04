
"use client"

import React, { useState } from "react"
import { MOCK_ENGINES, FEATURE_LIST } from "../data"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Search, Plus, IconDotsVertical, Clock, Activity, CheckCircle2, AlertTriangle } from "@/components/icons"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { ScrollArea } from "@/components/ui/scroll-area"
import Link from "next/link"

export default function CardGridDemo() {
  const [search, setSearch] = useState("")

  const filteredEngines = MOCK_ENGINES.filter(e => 
    e.name.toLowerCase().includes(search.toLowerCase()) || 
    e.description?.toLowerCase().includes(search.toLowerCase())
  )

  const getFeatureIcon = (key: string) => {
    return FEATURE_LIST.find(f => f.key === key)?.icon || "•"
  }

  return (
    <div className="flex flex-col h-full bg-muted/20">
      {/* Header */}
      <div className="border-b bg-background px-6 py-4 flex items-center justify-between sticky top-0 z-10">
        <div className="flex items-center gap-4">
            <Link href="../demo" className="text-muted-foreground hover:text-foreground">← Back</Link>
          <h1 className="text-xl font-semibold">Scan Workflows (Grid View)</h1>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative w-64">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input 
              type="search"
              name="engineSearch"
              autoComplete="off"
              aria-label="Search workflows"
              placeholder="Search workflows…"
              className="pl-9"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            New Workflow
          </Button>
        </div>
      </div>

      <ScrollArea className="flex-1 p-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6 max-w-[1600px] mx-auto">
          {filteredEngines.map((engine) => (
            <Card key={engine.id} className="group hover:shadow-md transition-[border-color,box-shadow] duration-200 border-muted hover:border-primary/50 flex flex-col">
              <CardHeader className="pb-3">
                <div className="flex justify-between items-start">
                  <div className="space-y-1">
                    <CardTitle className="text-lg flex items-center gap-2">
                      {engine.name}
                      {engine.type === 'preset' && (
                        <Badge variant="secondary" className="text-[10px] h-5 px-1.5">PRESET</Badge>
                      )}
                    </CardTitle>
                    <CardDescription className="line-clamp-2 min-h-[40px]">
                      {engine.description}
                    </CardDescription>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 -mr-2 opacity-0 group-hover:opacity-100 transition-opacity"
                        aria-label="More actions"
                      >
                        <IconDotsVertical className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem>Edit Configuration</DropdownMenuItem>
                      <DropdownMenuItem>Duplicate</DropdownMenuItem>
                      <DropdownMenuItem className="text-destructive">Delete</DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </CardHeader>
              
              <CardContent className="pb-3 flex-1">
                <div className="flex flex-wrap gap-1.5 mb-4">
                  {engine.features.map(f => (
                    <Badge key={f} variant="outline" className="bg-background/50 font-normal">
                      <span className="mr-1.5">{getFeatureIcon(f)}</span>
                      {FEATURE_LIST.find(item => item.key === f)?.label || f}
                    </Badge>
                  ))}
                  {engine.features.length === 0 && (
                    <span className="text-sm text-muted-foreground italic">No features enabled</span>
                  )}
                </div>
              </CardContent>

              <CardFooter className="pt-3 border-t bg-muted/10 flex items-center justify-between text-xs text-muted-foreground">
                <div className="flex items-center gap-3">
                    <div className="flex items-center gap-1" title="Last Updated">
                        <Clock className="h-3.5 w-3.5" />
                        <span>{new Date(engine.updatedAt).toLocaleDateString()}</span>
                    </div>
                    {engine.isValid ? (
                        <div className="flex items-center gap-1 text-green-600 dark:text-green-500">
                            <CheckCircle2 className="h-3.5 w-3.5" />
                            <span>Valid</span>
                        </div>
                    ) : (
                        <div className="flex items-center gap-1 text-amber-600 dark:text-amber-500">
                            <AlertTriangle className="h-3.5 w-3.5" />
                            <span>Invalid</span>
                        </div>
                    )}
                </div>
                {engine.type === 'user' && (
                    <div className="flex items-center gap-1" title="Usage Stats">
                        <Activity className="h-3.5 w-3.5" />
                        <span>{engine.stats.runs} runs</span>
                    </div>
                )}
              </CardFooter>
            </Card>
          ))}
          
          {/* Add New Card Placeholder */}
          <button
            type="button"
            className="border-2 border-dashed rounded-xl p-6 flex flex-col items-center justify-center text-muted-foreground hover:text-primary hover:border-primary/50 hover:bg-primary/5 transition-[background-color,border-color,color] min-h-[250px] gap-3"
          >
            <div className="h-12 w-12 rounded-full bg-muted flex items-center justify-center">
              <Plus className="h-6 w-6" />
            </div>
            <span className="font-medium">Create New Workflow</span>
          </button>
        </div>
      </ScrollArea>
    </div>
  )
}
