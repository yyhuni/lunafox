"use client"

import React, { useState, useEffect, useRef } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { 
  Eye, 
  Trash2, 
  MoreHorizontal, 
  IconCircleX,
  IconCheck,
  IconX
} from "@/components/icons"

export default function ScanHistoryActionsDemo() {
  // State for Inline Confirmation Demo
  const [isDeleting, setIsDeleting] = useState(false)

  // State for Custom Context Menu Demo
  const [contextMenu, setContextMenu] = useState<{
    x: number
    y: number
    isOpen: boolean
  }>({ x: 0, y: 0, isOpen: false })
  const contextMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (contextMenu.isOpen && contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
        setContextMenu(prev => ({ ...prev, isOpen: false }))
      }
    }
    document.addEventListener("click", handleClick)
    return () => document.removeEventListener("click", handleClick)
  }, [contextMenu.isOpen])

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault()
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      isOpen: true,
    })
  }

  const handleContextMenuKeyDown = (e: React.KeyboardEvent<HTMLElement>) => {
    if (e.key === "ContextMenu" || (e.shiftKey && e.key === "F10")) {
      e.preventDefault()
      const rect = e.currentTarget.getBoundingClientRect()
      setContextMenu({
        x: rect.left + rect.width / 2,
        y: rect.top + rect.height / 2,
        isOpen: true,
      })
    }
  }

  return (
    <div className="container mx-auto py-10 space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight mb-2">Scan History Action Variants</h1>
        <p className="text-muted-foreground">
          Exploration of different interaction patterns for table actions.
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        
        {/* Variant 1: Dropdown Menu */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 1: Dropdown Menu</CardTitle>
            <CardDescription>
              Cleanest. Reduces visual clutter.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border p-4">
              <div className="flex items-center justify-between py-2 border-b last:border-0">
                <span className="text-sm font-medium">Scan #1024 (Running)</span>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="h-8 w-8 p-0">
                      <span className="sr-only">Open menu</span>
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem>
                      <Eye className="mr-2 h-4 w-4" />
                      View Details
                    </DropdownMenuItem>
                    <DropdownMenuItem>
                      <IconCircleX className="mr-2 h-4 w-4 text-orange-500" />
                      Stop Scan
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem className="text-destructive focus:text-destructive">
                      <Trash2 className="mr-2 h-4 w-4" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Variant 2: Hybrid */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 2: Hybrid (View + Menu)</CardTitle>
            <CardDescription>
              Prioritizes &quot;View&quot;. Other actions tucked away.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border p-4">
              <div className="flex items-center justify-between py-2 border-b last:border-0">
                <span className="text-sm font-medium">Scan #1024 (Running)</span>
                <div className="flex items-center gap-1">
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="View details">
                          <Eye className="h-4 w-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>View Details</TooltipContent>
                    </Tooltip>
                  </TooltipProvider>

                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" className="h-8 w-8 p-0">
                        <span className="sr-only">Open menu</span>
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem>
                        <IconCircleX className="mr-2 h-4 w-4 text-orange-500" />
                        Stop Scan
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem className="text-destructive focus:text-destructive">
                        <Trash2 className="mr-2 h-4 w-4" />
                        Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Variant 3: Inline Refined */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 3: Inline (Colored)</CardTitle>
            <CardDescription>
              High visibility. Uses semantic colors.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border p-4">
              <div className="flex items-center justify-between py-2 border-b last:border-0">
                <span className="text-sm font-medium">Scan #1024 (Running)</span>
                <div className="flex items-center gap-1">
                  <Button variant="ghost" size="icon" className="h-8 w-8 text-primary hover:bg-primary/10" aria-label="View details">
                    <Eye className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" className="h-8 w-8 text-amber-500 hover:text-amber-600 hover:bg-amber-50" aria-label="Stop scan">
                    <IconCircleX className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:bg-destructive/10" aria-label="Delete scan">
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Variant 4: Hover Reveal */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 4: Hover Reveal</CardTitle>
            <CardDescription>
              Minimal noise. Actions appear on row hover.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border p-4">
              <div className="group flex items-center justify-between py-2 border-b last:border-0 relative cursor-default hover:bg-muted/50 -mx-2 px-2 transition-colors rounded-sm">
                <span className="text-sm font-medium">Scan #1024 (Running)</span>
                
                {/* Placeholder to keep height/layout stable if needed, or just absolutely position */}
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-200">
                  <Button variant="ghost" size="icon" className="h-8 w-8 bg-background shadow-sm border" aria-label="View details">
                    <Eye className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" className="h-8 w-8 bg-background shadow-sm border text-destructive" aria-label="Delete scan">
                    <Trash2 className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" className="h-8 w-8 bg-background shadow-sm border" aria-label="More actions">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Variant 5: Dynamic / Smart Action */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 5: Smart Action</CardTitle>
            <CardDescription>
              Context-aware. Primary action changes based on status.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border p-4 space-y-4">
              {/* Scenario A: Running */}
              <div className="flex items-center justify-between py-2 border-b">
                <span className="text-sm font-medium">Scan #1024 (Running)</span>
                <div className="flex items-center gap-2">
                  <Button size="sm" variant="outline" className="h-8 border-orange-200 text-orange-700 hover:bg-orange-50 hover:text-orange-800">
                    <IconCircleX className="mr-2 h-3.5 w-3.5" />
                    Stop
                  </Button>
                  <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="More actions">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </div>
              </div>
              
              {/* Scenario B: Completed */}
              <div className="flex items-center justify-between py-2">
                <span className="text-sm font-medium">Scan #1023 (Completed)</span>
                <div className="flex items-center gap-2">
                  <Button size="sm" variant="outline" className="h-8">
                    <Eye className="mr-2 h-3.5 w-3.5" />
                    View
                  </Button>
                  <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="More actions">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Variant 6: Split Button / Minimal */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 6: Split Action</CardTitle>
            <CardDescription>
              One click for main action, dropdown for rest.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border p-4">
              <div className="flex items-center justify-between py-2 border-b last:border-0">
                <span className="text-sm font-medium">Scan #1024 (Running)</span>
                <div className="flex items-center -space-x-px">
                  <Button 
                    variant="outline" 
                    className="h-8 rounded-r-none border-r-0 px-3 hover:bg-muted"
                  >
                    View
                  </Button>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="outline" className="h-8 w-8 rounded-l-none px-0" aria-label="More actions">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem>
                        <IconCircleX className="mr-2 h-4 w-4" />
                        Stop
                      </DropdownMenuItem>
                      <DropdownMenuItem className="text-destructive">
                        <Trash2 className="mr-2 h-4 w-4" />
                        Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Variant 7: Inline Confirmation */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 7: Inline Confirmation</CardTitle>
            <CardDescription>
              Smoother workflow. Replaces &quot;Are you sure?&quot; modal with in-place confirmation.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border p-4">
              <div className="flex items-center justify-between py-2 border-b last:border-0">
                <span className="text-sm font-medium">Scan #1024 (Completed)</span>
                <div className="flex items-center gap-1">
                  <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="View details">
                    <Eye className="h-4 w-4" />
                  </Button>
                  
                  {isDeleting ? (
                    <div className="flex items-center bg-destructive/10 rounded-md animate-in fade-in slide-in-from-right-4 duration-200">
                      <span className="text-[10px] font-medium text-destructive px-2">Sure?</span>
                      <Button 
                        variant="ghost" 
                        size="icon" 
                        aria-label="Confirm delete"
                        className="h-8 w-8 text-destructive hover:bg-destructive hover:text-white rounded-l-none"
                        onClick={() => {
                          setIsDeleting(false)
                          // Perform delete
                        }}
                      >
                        <IconCheck className="h-3.5 w-3.5" />
                      </Button>
                      <Button 
                        variant="ghost" 
                        size="icon" 
                        aria-label="Cancel delete"
                        className="h-8 w-8 text-muted-foreground hover:text-foreground rounded-r-none"
                        onClick={() => setIsDeleting(false)}
                      >
                        <IconX className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  ) : (
                    <Button 
                      variant="ghost" 
                      size="icon" 
                      aria-label="Delete scan"
                      className="h-8 w-8 text-destructive hover:bg-destructive/10"
                      onClick={() => setIsDeleting(true)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Variant 8: Badge Integrated */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 8: Badge Integrated</CardTitle>
            <CardDescription>
              Actions attached to the status. Highly contextual.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border p-4 space-y-4">
              <div className="flex items-center justify-between py-2 border-b">
                <span className="text-sm font-medium">Scan #1024</span>
                <div className="flex items-center gap-4">
                  <div className="flex items-center rounded-full border bg-secondary/50 pr-1 pl-3 py-0.5 gap-2">
                    <div className="flex items-center gap-1.5">
                      <span className="relative flex h-2 w-2">
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
                        <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
                      </span>
                      <span className="text-xs font-medium">Running</span>
                    </div>
                    <div className="h-4 w-px bg-border mx-1" />
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <button className="rounded-full hover:bg-background p-1 text-muted-foreground hover:text-orange-600 transition-colors" aria-label="Stop scan">
                            <IconCircleX className="h-3.5 w-3.5" />
                          </button>
                        </TooltipTrigger>
                        <TooltipContent>Stop Scan</TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </div>
                  
                  <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="More actions">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <div className="flex items-center justify-between py-2">
                <span className="text-sm font-medium">Scan #1023</span>
                <div className="flex items-center gap-4">
                  <Badge variant="outline" className="bg-green-500/10 text-green-700 border-green-200">
                    Completed
                  </Badge>
                  <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="More actions">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Variant 9: Context Menu (Right Click) */}
        <Card>
          <CardHeader>
            <CardTitle>Variant 9: Context Menu</CardTitle>
            <CardDescription>
              Pro-user interface. Right-click the row to see actions.
              (Try right-clicking the row below)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <button
              type="button"
              onContextMenu={handleContextMenu}
              onKeyDown={handleContextMenuKeyDown}
              aria-label="Open scan actions context menu"
              className="w-full rounded-md border p-4 bg-muted/10 hover:bg-muted/30 transition-colors cursor-context-menu select-none text-left"
            >
              <div className="flex items-center justify-between py-2">
                <span className="text-sm font-medium">Scan #1024 (Running) - Right Click Me</span>
                <span className="text-xs text-muted-foreground">Right click for actions</span>
              </div>
            </button>
            
            {/* Custom Context Menu Portal/Overlay */}
            {contextMenu.isOpen && (
              <div 
                ref={contextMenuRef}
                style={{ 
                  position: 'fixed', 
                  top: contextMenu.y, 
                  left: contextMenu.x,
                  zIndex: 50
                }}
                className="min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md animate-in fade-in zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2"
              >
                <div className="relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50">
                  <Eye className="mr-2 h-4 w-4" />
                  View Details
                </div>
                <div className="relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50">
                  <IconCircleX className="mr-2 h-4 w-4 text-orange-500" />
                  Stop Scan
                </div>
                <div className="-mx-1 my-1 h-px bg-muted" />
                <div className="relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 text-destructive focus:text-destructive">
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
