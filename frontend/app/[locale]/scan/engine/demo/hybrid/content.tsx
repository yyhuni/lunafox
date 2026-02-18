
"use client"

import React, { useState } from "react"
import { MOCK_ENGINES, FEATURE_LIST } from "../data"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { 
    Plus, IconDotsVertical, Clock, 
    CheckCircle2, AlertTriangle, IconSearch, IconSettings,
    IconChevronRight, IconDatabase, IconServer
} from "@/components/icons"
import { 
    DropdownMenu, 
    DropdownMenuContent, 
    DropdownMenuItem, 
    DropdownMenuTrigger 
} from "@/components/ui/dropdown-menu"
import Link from "next/link"
import { cn } from "@/lib/utils"

export default function HybridDemo() {
  const [selectedId, setSelectedId] = useState<number>(MOCK_ENGINES[0].id)
  const [search, setSearch] = useState("")

  const filteredEngines = MOCK_ENGINES.filter(e => 
    e.name.toLowerCase().includes(search.toLowerCase())
  )

  const selectedEngine = MOCK_ENGINES.find(e => e.id === selectedId) || MOCK_ENGINES[0]

  return (
    <div className="flex flex-col h-full bg-background overflow-hidden">
        {/* Top Navigation / Header */}
        <header className="border-b h-14 flex items-center justify-between px-4 shrink-0 bg-muted/5">
            <div className="flex items-center gap-4">
                <Link href="../demo" className="text-muted-foreground hover:text-foreground text-sm flex items-center gap-1">
                    <IconChevronRight className="rotate-180 h-4 w-4" /> Back
                </Link>
                <Separator orientation="vertical" className="h-6" />
                <h1 className="font-semibold text-sm">Scan Engine Management</h1>
            </div>
            <div className="flex items-center gap-2">
                <Button variant="default" size="sm" className="h-8">
                    <Plus className="h-3.5 w-3.5 mr-1.5" /> New Engine
                </Button>
            </div>
        </header>

        <div className="flex flex-1 overflow-hidden">
            {/* Left Sidebar / Menu List */}
            <aside className="w-[300px] flex flex-col border-r bg-muted/10">
                <div className="p-3">
                    <div className="relative">
                        <IconSearch className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                        <Input 
                            type="search"
                            name="engineSearch"
                            autoComplete="off"
                            aria-label="Search engines"
                            placeholder="Search engines…"
                            className="pl-8 h-9 text-sm"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                </div>
                
                <ScrollArea className="flex-1">
                    <div className="p-3 space-y-1">
                        <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                            Presets
                        </div>
                        {filteredEngines.filter(e => e.type === 'preset').map(engine => (
                            <button
                                key={engine.id}
                                type="button"
                                onClick={() => setSelectedId(engine.id)}
                                aria-pressed={selectedId === engine.id}
                                className={cn(
                                    "w-full text-left px-3 py-2 rounded-md text-sm transition-colors flex items-center gap-3",
                                    selectedId === engine.id 
                                        ? "bg-primary/10 text-primary font-medium" 
                                        : "hover:bg-muted text-muted-foreground hover:text-foreground"
                                )}
                            >
                                <IconDatabase className="h-4 w-4 opacity-70" />
                                <span className="truncate flex-1">{engine.name}</span>
                                {selectedId === engine.id && <div className="w-1 h-1 rounded-full bg-primary" />}
                            </button>
                        ))}

                        <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider mt-4">
                            My Engines
                        </div>
                        {filteredEngines.filter(e => e.type === 'user').map(engine => (
                            <button
                                key={engine.id}
                                type="button"
                                onClick={() => setSelectedId(engine.id)}
                                aria-pressed={selectedId === engine.id}
                                className={cn(
                                    "w-full text-left px-3 py-2 rounded-md text-sm transition-colors flex items-center gap-3",
                                    selectedId === engine.id 
                                        ? "bg-primary/10 text-primary font-medium" 
                                        : "hover:bg-muted text-muted-foreground hover:text-foreground"
                                )}
                            >
                                <IconServer className="h-4 w-4 opacity-70" />
                                <span className="truncate flex-1">{engine.name}</span>
                                {!engine.isValid && <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />}
                            </button>
                        ))}
                    </div>
                </ScrollArea>
                
                <div className="p-3 border-t bg-muted/5">
                    <div className="flex items-center justify-between text-xs text-muted-foreground px-2">
                        <span>{filteredEngines.length} engines total</span>
                        <IconSettings className="h-3.5 w-3.5" aria-hidden="true" />
                    </div>
                </div>
            </aside>

            {/* Right Content / Detail View */}
            <main className="flex-1 flex flex-col min-w-0 bg-background">
                {selectedEngine ? (
                    <>
                        <div className="px-8 py-6 border-b">
                            <div className="flex items-start justify-between">
                                <div>
                                    <div className="flex items-center gap-3 mb-2">
                                        <h2 className="text-2xl font-bold tracking-tight">{selectedEngine.name}</h2>
                                        {selectedEngine.type === 'preset' ? (
                                            <Badge variant="secondary">System Preset</Badge>
                                        ) : (
                                            <Badge variant="outline">Custom Engine</Badge>
                                        )}
                                        {selectedEngine.isValid ? (
                                            <Badge className="bg-green-500/15 text-green-600 hover:bg-green-500/25 border-green-200 shadow-none">
                                                <CheckCircle2 className="h-3 w-3 mr-1" /> Ready
                                            </Badge>
                                        ) : (
                                            <Badge variant="destructive" className="bg-amber-500/15 text-amber-600 hover:bg-amber-500/25 border-amber-200 shadow-none">
                                                <AlertTriangle className="h-3 w-3 mr-1" /> Config Error
                                            </Badge>
                                        )}
                                    </div>
                                    <p className="text-muted-foreground max-w-2xl">
                                        {selectedEngine.description}
                                    </p>
                                </div>
                                <div className="flex items-center gap-2">
                                        <DropdownMenu>
                                            <DropdownMenuTrigger asChild>
                                            <Button variant="outline" size="icon" aria-label="More actions">
                                                <IconDotsVertical className="h-4 w-4" />
                                            </Button>
                                        </DropdownMenuTrigger>
                                        <DropdownMenuContent align="end">
                                            <DropdownMenuItem>Duplicate</DropdownMenuItem>
                                            <DropdownMenuItem>Export Configuration</DropdownMenuItem>
                                            <DropdownMenuItem className="text-destructive">Delete</DropdownMenuItem>
                                        </DropdownMenuContent>
                                    </DropdownMenu>
                                    <Button>Edit Configuration</Button>
                                </div>
                            </div>
                            
                            <div className="flex items-center gap-6 mt-6 text-sm text-muted-foreground">
                                <div className="flex items-center gap-2">
                                    <Clock className="h-4 w-4" />
                                    <span>Updated {new Date(selectedEngine.updatedAt).toLocaleDateString()}</span>
                                </div>
                                <div className="flex items-center gap-2">
                                    <IconSettings className="h-4 w-4" />
                                    <span>v1.2.0</span>
                                </div>
                            </div>
                        </div>

                        <div className="flex-1 min-h-0">
                            <Tabs defaultValue="overview" className="h-full flex flex-col">
                                <div className="px-8 pt-4 border-b">
                                    <TabsList>
                                        <TabsTrigger value="overview">Overview</TabsTrigger>
                                        <TabsTrigger value="configuration">Configuration (YAML)</TabsTrigger>
                                    </TabsList>
                                </div>
                                
                                <TabsContent value="overview" className="flex-1 min-h-0 m-0">
                                    <ScrollArea className="h-full">
                                        <div className="p-8 max-w-5xl space-y-8">
                                            <section>
                                                <h3 className="text-lg font-semibold mb-4">Enabled Capabilities</h3>
                                                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                                                    {selectedEngine.features.map(featureKey => {
                                                        const feature = FEATURE_LIST.find(f => f.key === featureKey)
                                                        return (
                                                            <div key={featureKey} className="flex items-start gap-3 p-4 rounded-lg border bg-card text-card-foreground shadow-sm">
                                                                <div className="text-2xl">{feature?.icon}</div>
                                                                <div>
                                                                    <div className="font-medium">{feature?.label}</div>
                                                                    <div className="text-xs text-muted-foreground mt-1">
                                                                        Active module enabled
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        )
                                                    })}
                                                </div>
                                            </section>

                                            <Separator />

                                            <section>
                                                <h3 className="text-lg font-semibold mb-4">Description & Metadata</h3>
                                                <div className="bg-muted/30 rounded-lg p-6 border">
                                                    <h4 className="text-sm font-medium mb-2 text-muted-foreground">Detailed Description</h4>
                                                    <p className="text-sm leading-relaxed max-w-3xl mb-6">
                                                        {selectedEngine.description || "No description provided."}
                                                        This engine is configured to perform specific security checks based on the enabled modules above.
                                                    </p>
                                                    
                                                    <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
                                                        <div>
                                                            <div className="text-xs font-medium text-muted-foreground mb-1">Type</div>
                                                            <div className="text-sm">{selectedEngine.type === 'preset' ? 'System Preset' : 'User Custom'}</div>
                                                        </div>
                                                        <div>
                                                            <div className="text-xs font-medium text-muted-foreground mb-1">Created At</div>
                                                            <div className="text-sm">2023-10-01</div>
                                                        </div>
                                                        <div>
                                                            <div className="text-xs font-medium text-muted-foreground mb-1">Last Modifier</div>
                                                            <div className="text-sm">System Admin</div>
                                                        </div>
                                                        <div>
                                                            <div className="text-xs font-medium text-muted-foreground mb-1">Version</div>
                                                            <div className="text-sm">1.2.0</div>
                                                        </div>
                                                    </div>
                                                </div>
                                            </section>
                                        </div>
                                    </ScrollArea>
                                </TabsContent>
                                
                                <TabsContent value="configuration" className="flex-1 min-h-0 m-0">
                                    <div className="h-full bg-muted/20 p-0 relative group">
                                        <div className="absolute top-4 right-4 z-10 opacity-0 group-hover:opacity-100 transition-opacity">
                                            <Button variant="secondary" size="sm">Copy YAML</Button>
                                        </div>
                                        <ScrollArea className="h-full">
                                            <div className="p-6">
                                                <pre className="font-mono text-sm bg-background p-6 rounded-lg border shadow-sm">
{`# ${selectedEngine.name} Configuration
version: 1.0.0
enabled_features:
${selectedEngine.features.map(f => `  - ${f}`).join('\n')}

# Advanced Settings
concurrency: 5
timeout: 30s
retries: 3
user_agent: "LunaFox-Scanner/1.0"
exclude_paths:
  - "/logout"
  - "/admin"
`}</pre>
                                            </div>
                                        </ScrollArea>
                                    </div>
                                </TabsContent>
                            </Tabs>
                        </div>
                    </>
                ) : (
                    <div className="flex-1 flex items-center justify-center text-muted-foreground">
                        Select an engine to view details
                    </div>
                )}
            </main>
        </div>
    </div>
  )
}
