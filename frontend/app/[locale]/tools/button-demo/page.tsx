"use client"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/common/page-header"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { 
  Scan, ShieldAlert, Radio, Terminal, Loader2, 
  Database, Activity, Lock, Cpu, Search, Zap
} from "@/components/icons"

export default function ButtonDemoPage() {
  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6 animate-in fade-in zoom-in duration-500">
      <PageHeader
        code="UI-DEMO"
        title="Industrial UI Kit"
        description="Collection of industrial minimal micro-interactions and components."
      />

      {/* Section 1: Buttons */}
      <div className="px-4 lg:px-6">
        <h2 className="text-lg font-semibold tracking-tight">Interactive Buttons</h2>
      </div>
      <div className="grid gap-6 md:grid-cols-2 px-4 lg:px-6">
        
        {/* Style A: Scanner Sweep */}
        <Card className="border-l-4 border-l-primary/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Scan className="w-5 h-5" />
              Style A: Scanner Sweep
            </CardTitle>
            <CardDescription>
              Simulates a laser scanner sweeping across the button. Best for &quot;Scan&quot;, &quot;Search&quot; or &quot;Analyze&quot; actions.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex items-center justify-center gap-4 bg-secondary/20">
              <Button 
                variant="outline" 
                className="
                  relative overflow-hidden group border-primary/20 
                  hover:border-primary hover:text-primary 
                  transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300
                  bg-background/50 backdrop-blur-sm
                  min-w-[140px]
                "
              >
                <div className="
                  absolute inset-0 
                  translate-x-[-100%] group-hover:translate-x-[100%] 
                  bg-gradient-to-r from-transparent via-primary/10 to-transparent 
                  transition-transform duration-1000 ease-in-out
                  skew-x-12
                " />
                <Scan className="mr-2 h-4 w-4 transition-transform group-hover:scale-110 group-hover:rotate-90 duration-500" />
                <span className="relative z-10">Quick Scan</span>
                <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-primary group-hover:w-full transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 ease-out" />
              </Button>

              <Button 
                className="
                  relative overflow-hidden group 
                  bg-primary text-primary-foreground
                  hover:bg-primary/90
                  min-w-[140px]
                "
              >
                <div className="
                  absolute inset-0 
                  translate-x-[-100%] group-hover:translate-x-[100%] 
                  bg-gradient-to-r from-transparent via-white/20 to-transparent 
                  transition-transform duration-700 ease-in-out
                  skew-x-12
                " />
                <Scan className="mr-2 h-4 w-4" />
                <span className="relative z-10">Full Scan</span>
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Style B: Mechanical Construct */}
        <Card className="border-l-4 border-l-orange-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Terminal className="w-5 h-5 text-orange-500" />
              Style B: Mechanical Construct
            </CardTitle>
            <CardDescription>
              Borders construct themselves on hover. Precise, engineered feel.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex items-center justify-center gap-4 bg-secondary/20">
              <Button 
                variant="ghost" 
                className="
                  relative group 
                  hover:bg-transparent
                  text-foreground/70 hover:text-foreground
                  min-w-[140px]
                "
              >
                <span className="absolute top-0 left-0 w-[2px] h-0 bg-orange-500 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 group-hover:h-full" />
                <span className="absolute bottom-0 right-0 w-[2px] h-0 bg-orange-500 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 group-hover:h-full" />
                <span className="absolute top-0 right-0 w-0 h-[2px] bg-orange-500 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 delay-100 group-hover:w-full" />
                <span className="absolute bottom-0 left-0 w-0 h-[2px] bg-orange-500 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 delay-100 group-hover:w-full" />
                
                <Terminal className="mr-2 h-4 w-4" />
                Execute
              </Button>

              <Button 
                variant="outline"
                className="
                  relative group border-0
                  bg-secondary/50
                  min-w-[140px] overflow-hidden
                "
              >
                <div className="absolute inset-0 border border-orange-500/0 group-hover:border-orange-500/100 transition-colors duration-300 scale-95 group-hover:scale-100" />
                <div className="absolute top-0 left-0 w-1 h-1 bg-orange-500 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                <div className="absolute top-0 right-0 w-1 h-1 bg-orange-500 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                <div className="absolute bottom-0 left-0 w-1 h-1 bg-orange-500 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                <div className="absolute bottom-0 right-0 w-1 h-1 bg-orange-500 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                
                <span className="mr-2 text-xs font-mono text-orange-500 opacity-0 group-hover:opacity-100 absolute left-2 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 -translate-x-2 group-hover:translate-x-0">{">"}</span>
                <span className="transition-transform duration-300 group-hover:translate-x-2">Compile</span>
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Style C: Signal Pulse */}
        <Card className="border-l-4 border-l-emerald-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Radio className="w-5 h-5 text-emerald-500" />
              Style C: Signal Pulse
            </CardTitle>
            <CardDescription>
              Square pulse indicator. Good for &quot;Live&quot;, &quot;Connect&quot; or showing active state.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex items-center justify-center gap-4 bg-secondary/20">
              <Button 
                variant="outline" 
                className="
                  relative gap-2 border-primary/20 pl-3
                  hover:bg-secondary/80
                  min-w-[140px]
                "
              >
                <span className="relative flex h-2 w-2">
                  <span className="animate-ping absolute inline-flex h-full w-full bg-emerald-500 opacity-75 duration-1000 rounded-none"></span>
                  <span className="relative inline-flex h-2 w-2 bg-emerald-500 rounded-none"></span>
                </span>
                Live Feed
              </Button>

               <Button 
                className="
                  relative gap-2 bg-emerald-900/10 text-emerald-600 border border-emerald-500/20
                  hover:bg-emerald-900/20 hover:text-emerald-500
                  min-w-[140px]
                "
              >
                <span className="relative flex h-2 w-2">
                  <span className="animate-[ping_2s_cubic-bezier(0,0,0.2,1)_infinite] absolute inline-flex h-full w-full bg-emerald-500 opacity-75 rounded-none"></span>
                </span>
                <span className="text-xs font-mono tracking-widest">CONNECTING</span>
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Style D: Warning Lock */}
        <Card className="border-l-4 border-l-destructive/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldAlert className="w-5 h-5 text-destructive" />
              Style D: Warning / Lock
            </CardTitle>
            <CardDescription>
              Glitchy, high-contrast alerts. For destructive actions or critical alerts.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex items-center justify-center gap-4 bg-secondary/20">
              <Button 
                variant="outline" 
                className="
                  relative overflow-hidden group border-destructive/30 text-destructive
                  hover:bg-destructive hover:text-destructive-foreground
                  transition-colors duration-200
                  min-w-[140px]
                  px-3
                "
              >
                <div className="
                  absolute inset-0 bg-[repeating-linear-gradient(45deg,transparent,transparent_10px,#00000010_10px,#00000010_20px)]
                  opacity-0 group-hover:opacity-100 transition-opacity duration-300
                " />
                <span className="relative z-10 flex items-center gap-2">
                  <ShieldAlert className="h-4 w-4 transition-transform duration-200 group-hover:rotate-12" />
                  <span className="font-mono font-bold tracking-tight">PURGE</span>
                </span>
              </Button>

              <Button 
                className="
                  relative overflow-hidden group 
                  bg-background border border-destructive/50 text-destructive
                  hover:border-destructive
                  min-w-[140px]
                  px-3
                "
              >
                <span className="absolute inset-y-0 left-0 w-[2px] bg-destructive transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 group-hover:w-full opacity-10" />
                <span className="absolute bottom-0 right-0 h-[2px] w-0 bg-destructive transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 group-hover:w-full" />
                <span className="relative z-10 flex items-center gap-2">
                    <Loader2 className="h-4 w-4 animate-spin opacity-0 group-hover:opacity-100 transition-opacity duration-200" />
                    <span className="inline-grid">
                      <span className="col-start-1 row-start-1 transition-opacity duration-200 group-hover:opacity-0">Stop Service</span>
                      <span className="col-start-1 row-start-1 opacity-0 transition-opacity duration-200 group-hover:opacity-100">Stopping...</span>
                    </span>
                </span>
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Style J: Data Shutter */}
        <Card className="border-l-4 border-l-slate-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Database className="w-5 h-5 text-slate-500" />
              Style J: Data Shutter
            </CardTitle>
            <CardDescription>
              Aperture-style closing animation. Mechanical feel for saving or archiving.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex items-center justify-center gap-4 bg-secondary/20">
              <Button 
                variant="outline" 
                className="
                  relative overflow-hidden group 
                  bg-transparent border-slate-500/50 text-slate-500
                  hover:text-white
                  min-w-[140px]
                "
              >
                {/* Top Shutter */}
                <div className="absolute top-0 left-0 w-full h-1/2 bg-slate-700 -translate-y-full group-hover:translate-y-0 transition-transform duration-300 ease-out z-0" />
                {/* Bottom Shutter */}
                <div className="absolute bottom-0 left-0 w-full h-1/2 bg-slate-700 translate-y-full group-hover:translate-y-0 transition-transform duration-300 ease-out z-0" />
                
                <span className="relative z-10 flex items-center gap-2">
                  <Database className="w-4 h-4" />
                  ARCHIVE
                </span>
              </Button>

              <Button 
                className="
                  relative overflow-hidden group 
                  bg-slate-900 border border-slate-700 text-slate-400
                  hover:text-white
                  min-w-[140px]
                "
              >
                 {/* Left Shutter */}
                 <div className="absolute top-0 left-0 h-full w-1/2 bg-slate-600 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-in-out z-0 mix-blend-overlay" />
                 {/* Right Shutter */}
                 <div className="absolute top-0 right-0 h-full w-1/2 bg-slate-600 translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-in-out z-0 mix-blend-overlay" />
                 
                 <span className="relative z-10 flex items-center gap-2 font-mono text-xs tracking-wider">
                   SECURE_SAVE
                 </span>
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Style K: Precision Target */}
        <Card className="border-l-4 border-l-red-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Radio className="w-5 h-5 text-red-500" />
              Style K: Precision Target
            </CardTitle>
            <CardDescription>
              Reticle and corner markers for high-precision actions.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex items-center justify-center gap-4 bg-secondary/20">
              <Button 
                variant="ghost"
                className="
                  relative group 
                  text-red-500 hover:text-red-400 hover:bg-red-950/20
                  min-w-[140px]
                "
              >
                {/* Corner Markers */}
                <span className="absolute top-0 left-0 w-2 h-2 border-l-2 border-t-2 border-red-500 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 group-hover:w-full group-hover:h-full opacity-50 group-hover:opacity-100" />
                <span className="absolute bottom-0 right-0 w-2 h-2 border-r-2 border-b-2 border-red-500 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300 group-hover:w-full group-hover:h-full opacity-50 group-hover:opacity-100" />
                
                {/* Crosshair (Center) */}
                <span className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-20 pointer-events-none">
                   <div className="w-full h-[1px] bg-red-500"></div>
                   <div className="absolute h-full w-[1px] bg-red-500"></div>
                </span>

                <span className="relative z-10 font-mono tracking-widest">LOCATE</span>
              </Button>

              <Button 
                className="
                   relative overflow-hidden group
                   bg-transparent border border-red-900 text-red-700
                   hover:text-red-500 hover:border-red-500
                   min-w-[140px]
                "
              >
                <div className="absolute inset-0 flex items-center justify-center">
                   <div className="w-[120%] h-[1px] bg-red-500/20 rotate-45 transform scale-0 group-hover:scale-100 transition-transform duration-500"></div>
                   <div className="absolute w-[120%] h-[1px] bg-red-500/20 -rotate-45 transform scale-0 group-hover:scale-100 transition-transform duration-500"></div>
                </div>
                <span className="relative z-10 flex items-center gap-2">
                   <div className="w-1.5 h-1.5 bg-current rounded-full animate-pulse"></div>
                   TARGET_LOCKED
                </span>
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Style L: Neon Circuit */}
        <Card className="border-l-4 border-l-violet-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Zap className="w-5 h-5 text-violet-500" />
              Style L: Neon Circuit
            </CardTitle>
            <CardDescription>
              Glowing paths and text effects for energy or connection.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex items-center justify-center gap-4 bg-zinc-950">
               <Button 
                className="
                  relative group overflow-visible
                  bg-zinc-900 border border-zinc-800 text-violet-400
                  hover:border-violet-500/50 hover:text-violet-300 hover:shadow-[0_0_15px_rgba(139,92,246,0.3)]
                  transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300
                  min-w-[140px]
                "
              >
                <span className="relative z-10 flex items-center gap-2">
                  <Zap className="w-4 h-4" />
                  ENERGIZE
                </span>
                
                {/* Bottom Glow Bar */}
                <div className="absolute -bottom-[1px] left-1/2 -translate-x-1/2 w-1/2 h-[2px] bg-violet-500 shadow-[0_0_10px_rgba(139,92,246,1)] opacity-0 group-hover:opacity-100 group-hover:w-full transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300"></div>
              </Button>
            </div>
          </CardContent>
        </Card>

      </div>

      {/* Section 2: Inputs & Cards */}
      <h2 className="px-6 text-lg font-semibold tracking-tight mt-6">Inputs & Data Display</h2>
      <div className="grid gap-6 md:grid-cols-2 px-4 lg:px-6">

        {/* Style E: Cyber Input */}
        <Card className="border-l-4 border-l-blue-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Search className="w-5 h-5 text-blue-500" />
              Style E: Cyber Input
            </CardTitle>
            <CardDescription>
              Inputs with focus animations and status indicators.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex flex-col items-center justify-center gap-4 bg-secondary/20">
              
              <div className="relative w-full max-w-sm group">
                <Input 
                  type="search"
                  name="searchQueryDemo"
                  autoComplete="off"
                  placeholder="SEARCH_QUERY…"
                  className="
                    bg-background/50 border-primary/20 
                    focus-visible:ring-0 focus-visible:border-primary
                    transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300
                    pl-10 font-mono text-sm
                  "
                />
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground group-focus-within:text-primary transition-colors duration-300" />
                
                {/* Corner Accents */}
                <div className="absolute top-0 left-0 w-2 h-2 border-l-2 border-t-2 border-primary opacity-0 group-focus-within:opacity-100 transition-opacity duration-300" />
                <div className="absolute bottom-0 right-0 w-2 h-2 border-r-2 border-b-2 border-primary opacity-0 group-focus-within:opacity-100 transition-opacity duration-300" />
              </div>

              <div className="relative w-full max-w-sm group">
                <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-500 to-purple-600 rounded opacity-0 group-focus-within:opacity-30 blur transition duration-500"></div>
                <Input 
                  type="text"
                  name="commandInputDemo"
                  autoComplete="off"
                  placeholder="COMMAND_INPUT" 
                  className="
                    relative bg-background border-zinc-800
                    focus-visible:ring-0 focus-visible:border-blue-500
                    font-mono
                  "
                />
                <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
                   <div className="w-1.5 h-1.5 bg-blue-500 animate-pulse"></div>
                </div>
              </div>

            </div>
          </CardContent>
        </Card>

        {/* Style F: Data Card */}
        <Card className="border-l-4 border-l-purple-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Database className="w-5 h-5 text-purple-500" />
              Style F: Active Data Card
            </CardTitle>
            <CardDescription>
              Cards that reveal more data or structure on hover.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg bg-secondary/20">
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                {/* Variant 1: Grid Sweep */}
                <div
                  className="
                    group relative w-full h-28 
                    bg-card border border-border p-4
                    hover:border-purple-500 transition-colors duration-300
                    cursor-pointer overflow-hidden
                  "
                >
                  <div className="
                    absolute inset-0 
                    bg-[linear-gradient(rgba(168,85,247,0.05)_1px,transparent_1px),linear-gradient(90deg,rgba(168,85,247,0.05)_1px,transparent_1px)] 
                    bg-[size:10px_10px]
                    opacity-0 group-hover:opacity-100 transition-opacity duration-500
                  " />
                  <div className="relative z-10 flex flex-col justify-between h-full">
                    <div className="flex justify-between items-start">
                      <Database className="w-5 h-5 text-muted-foreground group-hover:text-purple-500 transition-colors" />
                      <span className="text-[10px] font-mono text-muted-foreground group-hover:text-purple-500">DB-01</span>
                    </div>
                    <div>
                      <div className="text-2xl font-bold font-mono">98.2%</div>
                      <div className="text-[10px] text-muted-foreground uppercase tracking-wider">Uptime</div>
                    </div>
                  </div>
                  <div className="
                    absolute top-0 left-0 w-full h-[1px] bg-purple-500/50
                    -translate-y-full group-hover:translate-y-[7rem]
                    transition-transform duration-1000 ease-linear
                  " />
                </div>

                {/* Variant 2: Access Layer */}
                <div
                  className="
                    group relative w-full h-28 
                    bg-card border border-border p-4
                    hover:border-blue-500/60 transition-colors duration-300
                    cursor-pointer overflow-hidden
                  "
                >
                  <div className="
                    absolute inset-0 
                    bg-[linear-gradient(120deg,transparent_0%,rgba(59,130,246,0.12)_45%,transparent_70%)]
                    opacity-0 group-hover:opacity-100 transition-opacity duration-500
                  " />
                  <div className="absolute inset-x-0 bottom-0 h-[2px] bg-blue-500/60 scale-x-0 group-hover:scale-x-100 origin-left transition-transform duration-500" />
                  <div className="relative z-10 flex flex-col justify-between h-full">
                    <div className="flex justify-between items-start">
                      <Lock className="w-5 h-5 text-muted-foreground group-hover:text-blue-500 transition-colors" />
                      <span className="text-[10px] font-mono text-muted-foreground group-hover:text-blue-500">AUTH-07</span>
                    </div>
                    <div>
                      <div className="text-2xl font-bold font-mono">99.6%</div>
                      <div className="text-[10px] text-muted-foreground uppercase tracking-wider">Access Pass</div>
                    </div>
                  </div>
                </div>

                {/* Variant 3: Processor Rail */}
                <div
                  className="
                    group relative w-full h-28 
                    bg-card border border-border p-4
                    hover:border-emerald-500/60 transition-colors duration-300
                    cursor-pointer overflow-hidden
                  "
                >
                  <div className="absolute inset-y-0 left-0 w-[3px] bg-emerald-500/60 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                  <div className="relative z-10 flex flex-col justify-between h-full">
                    <div className="flex justify-between items-start">
                      <Cpu className="w-5 h-5 text-muted-foreground group-hover:text-emerald-500 transition-colors" />
                      <span className="text-[10px] font-mono text-muted-foreground group-hover:text-emerald-500">CPU-11</span>
                    </div>
                    <div>
                      <div className="text-2xl font-bold font-mono">62%</div>
                      <div className="text-[10px] text-muted-foreground uppercase tracking-wider">Load</div>
                    </div>
                  </div>
                  <div className="absolute bottom-3 left-4 right-4 grid grid-cols-8 gap-1">
                    {Array.from({ length: 8 }).map((_, i) => (
                      <span
                        key={i}
                        className={
                          i < 6
                            ? "h-1 rounded-sm bg-emerald-500/50 group-hover:bg-emerald-500/80 transition-colors"
                            : "h-1 rounded-sm bg-border/70 transition-colors"
                        }
                      />
                    ))}
                  </div>
                </div>

                {/* Variant 4: Alert Core */}
                <div
                  className="
                    group relative w-full h-28 
                    bg-card border border-border p-4
                    hover:border-amber-500/60 transition-colors duration-300
                    cursor-pointer overflow-hidden
                  "
                >
                  <div className="
                    absolute inset-0 
                    bg-[radial-gradient(circle_at_20%_20%,rgba(245,158,11,0.18),transparent_60%)]
                    opacity-0 group-hover:opacity-100 transition-opacity duration-500
                  " />
                  <div className="relative z-10 flex flex-col justify-between h-full">
                    <div className="flex justify-between items-start">
                      <Zap className="w-5 h-5 text-muted-foreground group-hover:text-amber-500 transition-colors" />
                      <span className="text-[10px] font-mono text-muted-foreground group-hover:text-amber-500">ALRT-3</span>
                    </div>
                    <div>
                      <div className="text-2xl font-bold font-mono">14</div>
                      <div className="text-[10px] text-muted-foreground uppercase tracking-wider">Active Signals</div>
                    </div>
                  </div>
                  <div className="absolute top-3 right-3 h-2 w-2 rounded-full bg-amber-500/80 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Style G: Status Indicators */}
        <Card className="border-l-4 border-l-cyan-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="w-5 h-5 text-cyan-500" />
              Style G: Status Bars
            </CardTitle>
            <CardDescription>
              Industrial progress bars and loading states.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex flex-col items-center justify-center gap-6 bg-secondary/20">
              
              {/* Striped Progress */}
              <div className="w-full space-y-2">
                <div className="flex justify-between text-xs uppercase text-muted-foreground">
                  <span>Processing</span>
                  <span className="font-mono text-cyan-500">72%</span>
                </div>
                <div className="h-2 w-full bg-secondary overflow-hidden relative">
                  <div className="absolute inset-0 w-[72%] bg-cyan-500"></div>
                  {/* Stripes Overlay */}
                  <div className="absolute inset-0 w-[72%] bg-[linear-gradient(45deg,rgba(0,0,0,0.1)_25%,transparent_25%,transparent_50%,rgba(0,0,0,0.1)_50%,rgba(0,0,0,0.1)_75%,transparent_75%,transparent)] bg-[length:10px_10px] animate-[progress-stripes_1s_linear_infinite]"></div>
                </div>
              </div>

              {/* Segmented Loader */}
              <div className="flex items-center gap-1">
                 {[1,2,3,4,5].map((i) => (
                   <div key={i} className={`w-3 h-8 bg-cyan-500/20 animate-pulse`} style={{ animationDelay: `${i * 100}ms` }}></div>
                 ))}
                 <span className="ml-2 font-mono text-xs text-cyan-500 animate-pulse">LOADING_MODULES</span>
              </div>

            </div>
          </CardContent>
        </Card>

         {/* Style H: Glitch Typography */}
         <Card className="border-l-4 border-l-pink-500/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Cpu className="w-5 h-5 text-pink-500" />
              Style H: Glitch Typography
            </CardTitle>
            <CardDescription>
              Using global `.lunafox-glitch-text` class for high-impact headers.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="p-8 border border-dashed rounded-lg flex flex-col items-center justify-center gap-6 bg-zinc-950 overflow-hidden relative">
              
              <div className="lunafox-glitch-text text-4xl font-black tracking-tighter text-white" data-text="SYSTEM FAILURE">
                SYSTEM FAILURE
              </div>

              <div className="font-mono text-sm text-pink-500 flex items-center gap-2">
                 <span className="inline-block w-2 h-4 bg-pink-500 animate-ping"></span>
                 CRITICAL_ERROR_0x892
              </div>

            </div>
          </CardContent>
        </Card>
        
        {/* Style I: Splash Screen Glitch */}
        <Card className="border-l-4 border-l-indigo-500/50 md:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Zap className="w-5 h-5 text-indigo-500" />
              Style I: Splash Glitch Effect
            </CardTitle>
            <CardDescription>
              Full-container glitch effect typically used for login/loading screens.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="h-64 border border-dashed rounded-lg relative overflow-hidden bg-black flex items-center justify-center group">
              
              {/* This container has the glitch effect applied */}
              <div className="absolute inset-0 lunafox-splash-glitch opacity-50 group-hover:opacity-100 transition-opacity duration-500"></div>
              
              <div className="relative z-30 flex flex-col items-center gap-4">
                 <div className="h-16 w-16 bg-white rounded-full flex items-center justify-center">
                    <div className="h-12 w-12 bg-black rounded-full"></div>
                 </div>
                 <h2 className="text-2xl font-bold text-white tracking-widest">LUNAFOX</h2>
                 <p className="text-xs font-mono text-indigo-400 animate-pulse">INITIALIZING NEURAL LINK...</p>
              </div>

            </div>
          </CardContent>
        </Card>

      </div>
    </div>
  )
}
