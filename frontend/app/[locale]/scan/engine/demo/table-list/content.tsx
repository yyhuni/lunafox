
"use client"

import React, { useState } from "react"
import { MOCK_ENGINES, FEATURE_LIST } from "../data"
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { 
    Search, Plus, Filter, MoreHorizontal, 
    Download, Upload, Check, AlertTriangle 
} from "@/components/icons"
import { 
    DropdownMenu, 
    DropdownMenuContent, 
    DropdownMenuItem, 
    DropdownMenuLabel, 
    DropdownMenuSeparator, 
    DropdownMenuTrigger 
} from "@/components/ui/dropdown-menu"
import Link from "next/link"

export default function TableListDemo() {
  const [search, setSearch] = useState("")
  
  const filteredEngines = MOCK_ENGINES.filter(e => 
    e.name.toLowerCase().includes(search.toLowerCase()) || 
    e.type.includes(search.toLowerCase())
  )

  return (
    <div className="flex flex-col h-full bg-background">
        <div className="border-b px-6 py-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
                <Link href="../demo" className="text-muted-foreground hover:text-foreground">← Back</Link>
                <h1 className="text-xl font-semibold">Scan Engines (Table View)</h1>
            </div>
            <div className="flex items-center gap-2">
                <Button variant="outline" size="sm">
                    <Upload className="h-4 w-4 mr-2" />
                    Import
                </Button>
                <Button variant="outline" size="sm">
                    <Download className="h-4 w-4 mr-2" />
                    Export
                </Button>
                <Button size="sm">
                    <Plus className="h-4 w-4 mr-2" />
                    Add Engine
                </Button>
            </div>
        </div>

        <div className="p-6 space-y-4">
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-2 flex-1 max-w-sm">
                    <div className="relative w-full">
                        <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                        <Input 
                            type="search"
                            name="engineFilter"
                            autoComplete="off"
                            aria-label="Filter engines"
                            placeholder="Filter engines…"
                            className="pl-9 h-9"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" className="h-9 border-dashed">
                        <Filter className="h-4 w-4 mr-2" />
                        Type
                    </Button>
                    <Button variant="outline" size="sm" className="h-9 border-dashed">
                        <Filter className="h-4 w-4 mr-2" />
                        Status
                    </Button>
                </div>
            </div>

            <div className="rounded-md border">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-[300px]">Engine Name</TableHead>
                            <TableHead>Features</TableHead>
                            <TableHead className="w-[100px]">Type</TableHead>
                            <TableHead className="w-[100px]">Status</TableHead>
                            <TableHead className="w-[150px]">Last Updated</TableHead>
                            <TableHead className="w-[50px]"></TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {filteredEngines.map((engine) => (
                            <TableRow key={engine.id}>
                                <TableCell className="font-medium">
                                    <div className="flex flex-col">
                                        <span>{engine.name}</span>
                                        <span className="text-xs text-muted-foreground truncate max-w-[280px]">
                                            {engine.description}
                                        </span>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <div className="flex items-center gap-1 flex-wrap">
                                        {engine.features.slice(0, 3).map(f => (
                                            <Badge key={f} variant="secondary" className="text-[10px] h-5 px-1 font-normal">
                                                {FEATURE_LIST.find(item => item.key === f)?.label || f}
                                            </Badge>
                                        ))}
                                        {engine.features.length > 3 && (
                                            <Badge variant="outline" className="text-[10px] h-5 px-1 text-muted-foreground">
                                                +{engine.features.length - 3}
                                            </Badge>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    {engine.type === 'preset' ? (
                                        <Badge variant="secondary" className="font-normal">Preset</Badge>
                                    ) : (
                                        <Badge variant="outline" className="font-normal">Custom</Badge>
                                    )}
                                </TableCell>
                                <TableCell>
                                    {engine.isValid ? (
                                        <div className="flex items-center text-xs text-green-600 dark:text-green-500 font-medium">
                                            <Check className="h-3.5 w-3.5 mr-1" />
                                            Valid
                                        </div>
                                    ) : (
                                        <div className="flex items-center text-xs text-amber-600 dark:text-amber-500 font-medium">
                                            <AlertTriangle className="h-3.5 w-3.5 mr-1" />
                                            Issue
                                        </div>
                                    )}
                                </TableCell>
                                <TableCell className="text-muted-foreground text-xs">
                                    {new Date(engine.updatedAt).toLocaleDateString()}
                                </TableCell>
                                <TableCell>
                                    <DropdownMenu>
                                        <DropdownMenuTrigger asChild>
                                            <Button variant="ghost" className="h-8 w-8 p-0">
                                                <span className="sr-only">Open menu</span>
                                                <MoreHorizontal className="h-4 w-4" />
                                            </Button>
                                        </DropdownMenuTrigger>
                                        <DropdownMenuContent align="end">
                                            <DropdownMenuLabel>Actions</DropdownMenuLabel>
                                            <DropdownMenuItem>Edit engine</DropdownMenuItem>
                                            <DropdownMenuItem>View details</DropdownMenuItem>
                                            <DropdownMenuSeparator />
                                            <DropdownMenuItem className="text-destructive">Delete engine</DropdownMenuItem>
                                        </DropdownMenuContent>
                                    </DropdownMenu>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>
            
            <div className="text-xs text-muted-foreground text-center">
                Showing {filteredEngines.length} engines
            </div>
        </div>
    </div>
  )
}
