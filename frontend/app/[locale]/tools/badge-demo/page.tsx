"use client"

import { PageHeader } from "@/components/common/page-header"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { 
  CheckCircle2, Clock, Loader2, 
  XCircle, Activity, Server, Database,
  Terminal, Hash, Tag
} from "@/components/icons"

export default function BadgeDemoPage() {
  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6 animate-in fade-in zoom-in duration-500">
      <PageHeader
        code="UI-BADGE"
        title="Industrial Status Badges"
        description="Collection of status indicators and data tags."
      />

      <div className="grid gap-6 md:grid-cols-2 px-4 lg:px-6">
        
        {/* Style A: Bauhaus Side-Border (Default) with Animations - Borderless Variant */}
        <Card className="border-l-4 border-l-slate-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Tag className="w-5 h-5" />
              Style A: Bauhaus Animated (Borderless)
            </CardTitle>
            <CardDescription>
              Micro-interactions applied to the full badge, no side borders.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex flex-col gap-6 items-center justify-center bg-secondary/20">
              
              {/* Variant 1: Full Slide In */}
              <div className="flex gap-4 items-center w-full justify-center">
                <span className="text-xs text-muted-foreground w-20">Slide In:</span>
                <span className="group inline-flex items-center gap-1.5 px-2.5 py-1 text-[10px] font-mono font-medium uppercase tracking-wider border border-border bg-background text-success overflow-hidden relative">
                  <span className="absolute inset-0 bg-success/10 transform -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></span>
                  <CheckCircle2 className="w-3 h-3 relative z-10" />
                  <span className="relative z-10">Success</span>
                </span>
              </div>

              {/* Variant 2: Full Pulse Glow */}
              <div className="flex gap-4 items-center w-full justify-center">
                <span className="text-xs text-muted-foreground w-20">Pulse Glow:</span>
                <span className="inline-flex items-center gap-1.5 px-2.5 py-1 text-[10px] font-mono font-medium uppercase tracking-wider border border-warning/50 bg-background text-warning relative shadow-[0_0_8px_rgba(245,158,11,0.2)] animate-pulse">
                  <Loader2 className="w-3 h-3 animate-spin" />
                  Running
                </span>
              </div>

              {/* Variant 3: Text Reveal (Inverted) */}
              <div className="flex gap-4 items-center w-full justify-center">
                <span className="text-xs text-muted-foreground w-20">Reveal:</span>
                <span className="group inline-flex items-center gap-1.5 px-2.5 py-1 text-[10px] font-mono font-medium uppercase tracking-wider border border-border bg-background text-error relative overflow-hidden transition-colors hover:text-white hover:border-error">
                  <span className="absolute inset-0 bg-error transform -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></span>
                  <XCircle className="w-3 h-3 relative z-10" />
                  <span className="relative z-10">Failed</span>
                </span>
              </div>

              {/* Variant 4: Border Travel */}
              <div className="flex gap-4 items-center w-full justify-center">
                <span className="text-xs text-muted-foreground w-20">Travel:</span>
                <div className="relative group p-[1px] overflow-hidden rounded-sm">
                  <span className="absolute inset-0 bg-gradient-to-r from-info via-transparent to-info opacity-0 group-hover:opacity-100 transition-opacity duration-500 animate-[spin_2s_linear_infinite]" style={{backgroundSize: '200% 200%'}}></span>
                  <span className="relative inline-flex items-center gap-1.5 px-2.5 py-1 text-[10px] font-mono font-medium uppercase tracking-wider bg-background text-info border border-transparent">
                    <Clock className="w-3 h-3" />
                    Pending
                  </span>
                </div>
              </div>

            </div>
          </CardContent>
        </Card>

        {/* Style B: Signal Dot */}
        <Card className="border-l-4 border-l-emerald-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="w-5 h-5 text-emerald-500" />
              Style B: Signal Dot (Borderless)
            </CardTitle>
            <CardDescription>
              Live indicators with pulsing dots. Clean, no borders.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex flex-col gap-6 items-center justify-center bg-zinc-950">
              
              {/* Variant 1: Pure Dot */}
              <div className="flex gap-4 items-center w-full justify-center">
                <span className="text-xs text-zinc-500 w-24">Pure Dot:</span>
                <div className="flex items-center gap-2 text-emerald-500">
                  <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-500 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                  </span>
                  <span className="text-xs font-medium tracking-wide">ONLINE</span>
                </div>
              </div>

              {/* Variant 2: Soft Background */}
              <div className="flex gap-4 items-center w-full justify-center">
                <span className="text-xs text-zinc-500 w-24">Soft BG:</span>
                <div className="flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-500">
                  <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-500 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                  </span>
                  <span className="text-xs font-medium tracking-wide">CONNECTED</span>
                </div>
              </div>

              {/* Variant 3: Glowing Text */}
              <div className="flex gap-4 items-center w-full justify-center">
                <span className="text-xs text-zinc-500 w-24">Glowing:</span>
                <div className="flex items-center gap-2 text-amber-500 drop-shadow-[0_0_8px_rgba(245,158,11,0.5)]">
                   <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-500 opacity-75 duration-1000"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
                  </span>
                  <span className="text-xs font-medium tracking-wide">PROCESSING</span>
                </div>
              </div>

               {/* Variant 4: Square Dot */}
               <div className="flex gap-4 items-center w-full justify-center">
                <span className="text-xs text-zinc-500 w-24">Square Dot:</span>
                <div className="flex items-center gap-2 text-zinc-400">
                  <span className="h-2 w-2 bg-zinc-600 animate-pulse"></span>
                  <span className="text-xs font-medium tracking-wide">OFFLINE</span>
                </div>
              </div>

            </div>
          </CardContent>
        </Card>

        {/* Style C: Neon Terminal */}
        <Card className="border-l-4 border-l-violet-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Terminal className="w-5 h-5 text-violet-500" />
              Style C: Neon Terminal
            </CardTitle>
            <CardDescription>
              High contrast, glowing text style. Retro-futuristic.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-8 border border-dashed rounded-lg flex flex-wrap gap-4 items-center justify-center bg-black">
              
              <span className="px-2 py-0.5 bg-violet-500/10 border border-violet-500/50 text-violet-400 text-xs font-mono shadow-[0_0_10px_rgba(139,92,246,0.3)] hover:shadow-[0_0_15px_rgba(139,92,246,0.5)] transition-shadow cursor-default">
                [SYSTEM_READY]
              </span>

              <span className="px-2 py-0.5 bg-cyan-500/10 border border-cyan-500/50 text-cyan-400 text-xs font-mono shadow-[0_0_10px_rgba(6,182,212,0.3)] hover:shadow-[0_0_15px_rgba(6,182,212,0.5)] transition-shadow cursor-default">
                [DATA_FLOW]
              </span>

               <span className="px-2 py-0.5 bg-pink-500/10 border border-pink-500/50 text-pink-400 text-xs font-mono shadow-[0_0_10px_rgba(236,72,153,0.3)] hover:shadow-[0_0_15px_rgba(236,72,153,0.5)] transition-shadow cursor-default animate-pulse">
                [CRITICAL]
              </span>

            </div>
          </CardContent>
        </Card>

        {/* Style D: Mechanical Plate */}
        <Card className="border-l-4 border-l-slate-400/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="w-5 h-5" />
              Style D: Mechanical Plate
            </CardTitle>
            <CardDescription>
              Solid, riveted look. Good for asset tags or hardware status.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-8 border border-dashed rounded-lg flex flex-wrap gap-4 items-center justify-center bg-secondary/20">
              
              {/* Plate 1 */}
              <div className="relative px-3 py-1 bg-zinc-200 dark:bg-zinc-800 border-2 border-zinc-400 dark:border-zinc-600 text-zinc-700 dark:text-zinc-300 text-[10px] font-bold tracking-widest uppercase shadow-sm">
                <div className="absolute top-0.5 left-0.5 w-1 h-1 rounded-full bg-zinc-400/50 shadow-[inset_0_1px_1px_rgba(0,0,0,0.5)]"></div>
                <div className="absolute top-0.5 right-0.5 w-1 h-1 rounded-full bg-zinc-400/50 shadow-[inset_0_1px_1px_rgba(0,0,0,0.5)]"></div>
                <div className="absolute bottom-0.5 left-0.5 w-1 h-1 rounded-full bg-zinc-400/50 shadow-[inset_0_1px_1px_rgba(0,0,0,0.5)]"></div>
                <div className="absolute bottom-0.5 right-0.5 w-1 h-1 rounded-full bg-zinc-400/50 shadow-[inset_0_1px_1px_rgba(0,0,0,0.5)]"></div>
                ASSET-09
              </div>

               {/* Plate 2 (Dark) */}
              <div className="relative px-3 py-1 bg-zinc-900 border-2 border-zinc-700 text-zinc-400 text-[10px] font-bold tracking-widest uppercase shadow-sm">
                 <div className="absolute top-0.5 left-0.5 w-1 h-1 rounded-full bg-zinc-800 shadow-[inset_0_1px_1px_rgba(0,0,0,1)]"></div>
                <div className="absolute top-0.5 right-0.5 w-1 h-1 rounded-full bg-zinc-800 shadow-[inset_0_1px_1px_rgba(0,0,0,1)]"></div>
                <div className="absolute bottom-0.5 left-0.5 w-1 h-1 rounded-full bg-zinc-800 shadow-[inset_0_1px_1px_rgba(0,0,0,1)]"></div>
                <div className="absolute bottom-0.5 right-0.5 w-1 h-1 rounded-full bg-zinc-800 shadow-[inset_0_1px_1px_rgba(0,0,0,1)]"></div>
                PROD-ENV
              </div>

            </div>
          </CardContent>
        </Card>

         {/* Style E: Scan Line */}
         <Card className="border-l-4 border-l-blue-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Database className="w-5 h-5 text-blue-500" />
              Style E: Scan Line
            </CardTitle>
            <CardDescription>
              Background scanning animation for active processes.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-8 border border-dashed rounded-lg flex flex-wrap gap-4 items-center justify-center bg-secondary/20">
              
              <div className="relative overflow-hidden px-3 py-1 bg-blue-500/10 border border-blue-500/30 text-blue-600 dark:text-blue-400 text-xs font-mono font-medium rounded-sm">
                <div className="absolute inset-0 bg-gradient-to-r from-transparent via-blue-500/20 to-transparent -translate-x-full animate-[shimmer_2s_infinite]"></div>
                INDEXING...
              </div>

               <div className="relative overflow-hidden px-3 py-1 bg-purple-500/10 border border-purple-500/30 text-purple-600 dark:text-purple-400 text-xs font-mono font-medium rounded-sm">
                <div className="absolute inset-0 bg-gradient-to-r from-transparent via-purple-500/20 to-transparent -translate-x-full animate-[shimmer_1.5s_infinite]"></div>
                ANALYZING
              </div>

            </div>
          </CardContent>
        </Card>

        {/* Style F: Interactive Chip */}
        <Card className="border-l-4 border-l-orange-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Hash className="w-5 h-5 text-orange-500" />
              Style F: Interactive Chip
            </CardTitle>
            <CardDescription>
              Micro-interactions on hover. Reveals delete/action button.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-8 border border-dashed rounded-lg flex flex-wrap gap-4 items-center justify-center bg-secondary/20">
              
              <div className="group flex items-center bg-secondary hover:bg-destructive/10 border border-border hover:border-destructive/30 rounded-full pl-3 pr-1 py-1 transition-colors duration-200 cursor-pointer">
                <span className="text-xs font-medium text-foreground group-hover:text-destructive transition-colors mr-2">tag:vulnerability</span>
                <div className="h-4 w-4 rounded-full bg-background flex items-center justify-center text-muted-foreground group-hover:bg-destructive group-hover:text-white transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]">
                  <span className="text-[10px]">✕</span>
                </div>
              </div>

               <div className="group relative overflow-hidden bg-background border border-border px-3 py-1 rounded cursor-pointer hover:border-primary transition-colors">
                 <span className="relative z-10 text-xs font-medium text-muted-foreground group-hover:text-primary transition-colors">domain:example.com</span>
                 <div className="absolute inset-0 bg-primary/5 translate-y-full group-hover:translate-y-0 transition-transform duration-200"></div>
               </div>

            </div>
          </CardContent>
        </Card>

        {/* Style G: Ghost Frame */}
        <Card className="border-l-4 border-l-slate-200/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Tag className="w-5 h-5 text-slate-400" />
              Style G: Ghost Frame
            </CardTitle>
            <CardDescription>
              Ultra-light borders with transparent backgrounds. Subtle and clean.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-8 border border-dashed rounded-lg flex flex-wrap gap-4 items-center justify-center bg-background">
              
              <span className="px-2.5 py-1 text-[10px] font-mono font-medium uppercase tracking-wider border border-border/60 text-muted-foreground">
                Default
              </span>

              <span className="px-2.5 py-1 text-[10px] font-mono font-medium uppercase tracking-wider border border-dashed border-border text-muted-foreground">
                Draft
              </span>
              
              <span className="px-2.5 py-1 text-[10px] font-mono font-medium uppercase tracking-wider border border-primary/20 text-primary/80 bg-primary/5">
                Active
              </span>

            </div>
          </CardContent>
        </Card>

        {/* Style H: Minimal Tab */}
        <Card className="border-l-4 border-l-slate-300/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Tag className="w-5 h-5 text-slate-500" />
              Style H: Minimal Tab
            </CardTitle>
            <CardDescription>
              Bottom-border only indicators. Like file folder tabs.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-8 border border-dashed rounded-lg flex flex-wrap gap-4 items-center justify-center bg-background">
              
              <span className="px-2 py-1 text-[11px] font-medium text-foreground border-b-2 border-primary">
                Production
              </span>

              <span className="px-2 py-1 text-[11px] font-medium text-muted-foreground border-b-2 border-border hover:text-foreground hover:border-primary/50 transition-colors cursor-pointer">
                Staging
              </span>

              <span className="px-2 py-1 text-[11px] font-medium text-muted-foreground border-b-2 border-transparent hover:border-border transition-colors cursor-pointer">
                Development
              </span>

            </div>
          </CardContent>
        </Card>

        {/* Style I: Data Field */}
        <Card className="border-l-4 border-l-slate-400/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Hash className="w-5 h-5 text-slate-600" />
              Style I: Data Field
            </CardTitle>
            <CardDescription>
              Label-value pairs with light background. Good for metadata.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-8 border border-dashed rounded-lg flex flex-wrap gap-4 items-center justify-center bg-background">
              
              <div className="flex items-center text-[10px] font-mono border border-border bg-secondary/30">
                <span className="px-2 py-1 text-muted-foreground border-r border-border/50 bg-secondary/50">
                  VERSION
                </span>
                <span className="px-2 py-1 text-foreground font-medium">
                  v2.0.4
                </span>
              </div>

               <div className="flex items-center text-[10px] font-mono border border-border bg-secondary/30">
                <span className="px-2 py-1 text-muted-foreground border-r border-border/50 bg-secondary/50">
                  REGION
                </span>
                <span className="px-2 py-1 text-foreground font-medium flex items-center gap-1">
                  <span className="w-1.5 h-1.5 bg-green-500 rounded-full"></span>
                  US-EAST
                </span>
              </div>

            </div>
          </CardContent>
        </Card>

        {/* Style J: Status Dot (Minimal) */}
        <Card className="border-l-4 border-l-slate-200/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="w-5 h-5 text-slate-400" />
              Style J: Status Dot (Minimal)
            </CardTitle>
            <CardDescription>
              Cleanest possible indicator. Square dot + Text.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-8 border border-dashed rounded-lg flex flex-wrap gap-6 items-center justify-center bg-background">
              
              <div className="flex items-center gap-2 text-xs font-medium text-foreground">
                <span className="w-2 h-2 bg-success"></span>
                Operational
              </div>

              <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                <span className="w-2 h-2 bg-warning animate-pulse"></span>
                Degraded
              </div>

               <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                <span className="w-2 h-2 border border-foreground/30"></span>
                Stopped
              </div>

            </div>
          </CardContent>
        </Card>

      </div>
    </div>
  )
}
