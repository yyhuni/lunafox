"use client"

import React from "react"
import { 
  Globe, 
  Network, 
  Server, 
  Link2, 
  FolderOpen, 
  MoreHorizontal, 
  ArrowUpRight, 
  ArrowDownRight,
  Activity,
  Scan
} from "@/components/icons"
import { Button } from "@/components/ui/button"

const assetCards = [
  { label: "Websites", value: 1284, icon: Globe, trend: 12, status: "healthy" },
  { label: "Subdomains", value: 3921, icon: Network, trend: 5, status: "warning" },
  { label: "IPs", value: 264, icon: Server, trend: -2, status: "critical" },
  { label: "URLs", value: 8421, icon: Link2, trend: 8, status: "healthy" },
  { label: "Directories", value: 1560, icon: FolderOpen, trend: 0, status: "healthy" },
]

function SectionHeader({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <div className="border-b bg-muted/40 px-4 py-3">
      <div className="text-sm font-semibold">{title}</div>
      <div className="text-xs text-muted-foreground mt-1">{description}</div>
    </div>
  )
}

function Grid({ children }: { children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
      {children}
    </div>
  )
}

export default function AssetCardVariantsPage() {
  return (
    <div className="asset-card-variants flex flex-col gap-8 p-8 max-w-6xl mx-auto">
      <div>
        <h1 className="text-2xl font-bold mb-2">目标详情卡片样式方案</h1>
        <p className="text-muted-foreground">同一组数据的多种卡片视觉风格</p>
      </div>

      {/* Plan A */}
      <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
        <SectionHeader
          title="方案 A：Bauhaus Index"
          description="强调编号与工业感，适合与仪表盘风格统一"
        />
        <div className="p-4 bg-muted/10">
          <Grid>
            {assetCards.map((card, index) => (
              <div
                key={card.label}
                className="relative border border-border bg-card pt-8 pb-4 px-3 hover:border-primary/40 transition-colors cursor-pointer motion-lift"
              >
                <span className="absolute top-2 left-3 text-[10px] font-mono text-muted-foreground">
                  {String(index + 1).padStart(2, "0")}{" //"}
                </span>
                <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-muted-foreground">
                  <span>{card.label}</span>
                  <card.icon className="h-4 w-4" />
                </div>
                <div className="mt-3 text-2xl font-semibold tabular-nums">
                  {card.value.toLocaleString()}
                </div>
                <div className="mt-3 h-1.5 w-full bg-border/60">
                  <div className="h-full w-[55%] bg-primary/60" />
                </div>
              </div>
            ))}
          </Grid>
        </div>
      </div>

      {/* Plan B */}
      <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
        <SectionHeader
          title="方案 B：Rail Tag"
          description="左侧警戒线 + 结构化信息块，强调可扫描的工业节奏"
        />
        <div className="p-4 bg-muted/10">
          <Grid>
            {assetCards.map((card) => (
              <div
                key={card.label}
                className="relative border border-border bg-card px-3 py-4 hover:border-primary/40 transition-colors cursor-pointer motion-scan"
              >
                <span className="absolute inset-y-0 left-0 w-[3px] bg-primary/70" />
                <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-muted-foreground">
                  <span>{card.label}</span>
                  <card.icon className="h-4 w-4" />
                </div>
                <div className="mt-3 text-2xl font-semibold tabular-nums">
                  {card.value.toLocaleString()}
                </div>
                <div className="mt-2 text-xs text-muted-foreground">Last 24h</div>
              </div>
            ))}
          </Grid>
        </div>
      </div>

      {/* Plan C */}
      <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
        <SectionHeader
          title="方案 C：Split Panel"
          description="右侧功能舱，清晰区隔信息与图标"
        />
        <div className="p-4 bg-muted/10" data-motion="stagger">
          <Grid>
            {assetCards.map((card, index) => (
              <div
                key={card.label}
                className="flex border border-border bg-card hover:border-primary/40 transition-colors cursor-pointer motion-rise"
                style={{ ["--delay" as string]: `${index * 70}ms` } as React.CSSProperties}
              >
                <div className="flex flex-1 flex-col gap-1 p-3">
                  <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
                    {card.label}
                  </div>
                  <div className="text-2xl font-semibold tabular-nums">
                    {card.value.toLocaleString()}
                  </div>
                  <div className="text-[11px] text-muted-foreground">Assets</div>
                </div>
                <div className="w-12 border-l border-border bg-secondary/50 flex items-center justify-center">
                  <card.icon className="h-4 w-4 text-muted-foreground" />
                </div>
              </div>
            ))}
          </Grid>
        </div>
      </div>

      {/* Scheme D Series - Tech Schematic Variations */}
      <div className="space-y-6">
        <h2 className="text-xl font-bold">方案 D 扩展系列：Tech Schematic (技术图纸风格)</h2>

        {/* D1: Standard Schematic (Base) */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <SectionHeader
            title="D1: Standard Schematic"
            description="基础版：修复了边框颜色，保留虚线与角标，强调静态的工业图纸感"
          />
          <div className="p-4 bg-muted/10">
            <Grid>
              {assetCards.map((card) => (
                <div
                  key={card.label}
                  className="group relative bg-card p-4 hover:bg-accent/5 transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300 cursor-pointer"
                >
                  {/* Schematic Borders - Use border-border/40 to reduce visual weight */}
                  <div className="absolute inset-0 border border-border/40 border-t-0 group-hover:border-primary/30 transition-colors" />
                  
                  {/* Corners */}
                  <div className="absolute top-0 right-0 h-2 w-2 border-r border-t border-primary/50" />
                  <div className="absolute bottom-0 left-0 h-2 w-2 border-l border-b border-primary/50" />
                  
                  <div className="flex justify-between items-start mb-2">
                    <div className="text-[10px] font-mono text-muted-foreground bg-muted px-1.5 py-0.5 rounded-sm">
                      DAT-{card.label.substring(0, 3).toUpperCase()}
                    </div>
                    <card.icon className="h-4 w-4 text-muted-foreground/70 group-hover:text-primary transition-colors" />
                  </div>
                  
                  <div className="text-3xl font-light tracking-tight text-foreground group-hover:translate-x-1 transition-transform duration-300">
                    {card.value.toLocaleString()}
                  </div>
                  
                  <div className="mt-2 flex items-center gap-2">
                     <div className="h-px flex-1 bg-border border-t border-dashed border-muted-foreground/20" />
                     <span className="text-[10px] text-muted-foreground font-mono uppercase tracking-wider">{card.label}</span>
                  </div>
                </div>
              ))}
            </Grid>
          </div>
        </div>

        {/* D2: Active Scan (Animated) */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <SectionHeader
            title="D2: Active Scan"
            description="动态版：增加扫描线光效，模拟实时监控状态"
          />
          <div className="p-4 bg-muted/10">
            <Grid>
              {assetCards.map((card) => (
                <div
                  key={card.label}
                  className="group relative bg-card p-4 overflow-hidden cursor-pointer hover:shadow-lg transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300"
                >
                  {/* Base Border */}
                  <div className="absolute inset-0 border border-border/40 border-t-0" />
                  
                  {/* Animated Scan Line - Only visible on hover or active */}
                  <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none">
                     <div className="absolute inset-0 animate-scan-y bg-gradient-to-b from-transparent via-primary/5 to-transparent" />
                     <div className="absolute top-0 left-0 w-full h-[1px] bg-primary/20 shadow-[0_0_10px_rgba(59,130,246,0.5)] animate-scan-line" />
                  </div>

                  {/* Corners that glow on hover */}
                  <div className="absolute top-0 right-0 h-2 w-2 border-r border-t border-primary/30 group-hover:border-primary group-hover:shadow-[0_0_8px_rgba(59,130,246,0.6)] transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300" />
                  <div className="absolute bottom-0 left-0 h-2 w-2 border-l border-b border-primary/30 group-hover:border-primary group-hover:shadow-[0_0_8px_rgba(59,130,246,0.6)] transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300" />
                  
                  <div className="relative z-10">
                    <div className="flex justify-between items-start mb-2">
                      <div className="text-[10px] font-mono text-muted-foreground group-hover:text-primary/80 transition-colors">
                        S-{card.label.substring(0, 1)}00{card.value % 10}
                      </div>
                      <card.icon className="h-4 w-4 text-muted-foreground/70 group-hover:text-primary transition-colors" />
                    </div>
                    
                    <div className="text-3xl font-light tracking-tight text-foreground">
                      {card.value.toLocaleString()}
                    </div>
                    
                    <div className="mt-2 flex items-center justify-between border-t border-dashed border-border/50 pt-2">
                       <span className="text-[10px] text-muted-foreground font-mono uppercase tracking-wider">{card.label}</span>
                       <div className="h-1.5 w-1.5 rounded-full bg-emerald-500/50 animate-pulse" />
                    </div>
                  </div>
                </div>
              ))}
            </Grid>
          </div>
        </div>

        {/* D3: Corner Pulse (Focus) */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <SectionHeader
            title="D3: Corner Pulse"
            description="聚焦版：Hover 时边角向内收缩并高亮，强调选中感"
          />
          <div className="p-4 bg-muted/10">
            <Grid>
              {assetCards.map((card) => (
                <div
                  key={card.label}
                  className="group relative bg-card p-4 cursor-pointer"
                >
                  {/* Background Grid Pattern (Subtle) */}
                  <div className="absolute inset-0 bg-[linear-gradient(90deg,rgba(0,0,0,0.02)_1px,transparent_1px)] bg-[size:20px_20px] [mask-image:linear-gradient(to_bottom,white,transparent)]" />

                  {/* Dynamic Corners */}
                  <div className="absolute top-0 left-0 h-3 w-3 border-l-2 border-t-2 border-border group-hover:border-primary group-hover:w-full group-hover:h-full transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300 ease-out" />
                  <div className="absolute bottom-0 right-0 h-3 w-3 border-r-2 border-b-2 border-border group-hover:border-primary group-hover:w-full group-hover:h-full transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300 ease-out" />
                  
                  <div className="relative z-10 flex flex-col h-full justify-between">
                    <div className="flex justify-between items-start">
                       <span className="text-xs font-semibold uppercase tracking-widest text-muted-foreground group-hover:text-primary transition-colors">{card.label}</span>
                       <card.icon className="h-4 w-4 text-muted-foreground group-hover:scale-110 transition-transform" />
                    </div>

                    <div className="mt-4">
                      <div className="text-3xl font-bold text-foreground">
                        {card.value.toLocaleString()}
                      </div>
                      <div className="text-[10px] text-muted-foreground mt-1 font-mono">
                         UPDATED: NOW
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </Grid>
          </div>
        </div>

        {/* D4: Data Stream (Glitch) */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <SectionHeader
            title="D4: Data Stream"
            description="数据流版：背景带有微弱的数字流噪点，Hover 时产生轻微故障效果"
          />
          <div className="p-4 bg-muted/10">
            <Grid>
              {assetCards.map((card) => (
                <div
                  key={card.label}
                  className="group relative bg-card p-4 overflow-hidden border border-border/40 border-t-0 cursor-pointer"
                >
                  {/* Glitch Overlay */}
                  <div className="absolute inset-0 bg-primary/5 translate-x-[-100%] group-hover:translate-x-0 transition-transform duration-300 skew-x-12" />
                  
                  {/* Decorative Side Bar */}
                  <div className="absolute left-0 top-0 bottom-0 w-[2px] bg-border group-hover:bg-primary transition-colors" />

                  <div className="relative z-10 pl-3">
                    <div className="flex items-center gap-2 mb-3">
                      <div className="p-1 rounded bg-muted/50 group-hover:bg-background/80 transition-colors">
                        <card.icon className="h-3.5 w-3.5 text-muted-foreground" />
                      </div>
                      <div className="h-px flex-1 bg-border/60" />
                      <div className="text-[9px] font-mono text-muted-foreground/60">
                        0x{card.value.toString(16).toUpperCase()}
                      </div>
                    </div>

                    <div className="text-3xl font-light tabular-nums tracking-tighter group-hover:text-primary transition-colors">
                      {card.value.toLocaleString()}
                    </div>

                    <div className="mt-2 text-xs text-muted-foreground font-medium uppercase tracking-wider group-hover:translate-x-1 transition-transform">
                      {card.label}
                    </div>
                  </div>
                </div>
              ))}
            </Grid>
          </div>
        </div>
      </div>

      {/* Solution E: Status Indicator */}
      <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
        <SectionHeader
          title="方案 E：Status Context"
          description="带有状态颜色指示和趋势数据"
        />
        <div className="p-4 bg-muted/10">
          <Grid>
            {assetCards.map((card) => {
              const statusColor = 
                card.status === 'critical' ? 'bg-red-500' : 
                card.status === 'warning' ? 'bg-amber-500' : 
                'bg-emerald-500';
              
              return (
                <div
                  key={card.label}
                  className="relative overflow-hidden bg-card rounded-lg border shadow-sm hover:shadow-md transition-[color,background-color,border-color,opacity,transform,box-shadow] cursor-pointer group"
                >
                  <div className={`absolute top-0 left-0 w-full h-1 ${statusColor} opacity-80`} />
                  
                  <div className="p-4">
                    <div className="flex justify-between items-start">
                      <div className="p-2 rounded-md bg-secondary/50 group-hover:bg-primary/10 transition-colors">
                        <card.icon className="h-4 w-4 text-foreground/70 group-hover:text-primary" />
                      </div>
                      {card.trend !== 0 && (
                        <div className={`flex items-center text-xs font-medium ${card.trend > 0 ? 'text-green-600' : 'text-red-600'}`}>
                          {card.trend > 0 ? <ArrowUpRight className="h-3 w-3 mr-0.5" /> : <ArrowDownRight className="h-3 w-3 mr-0.5" />}
                          {Math.abs(card.trend)}%
                        </div>
                      )}
                    </div>
                    
                    <div className="mt-3">
                      <div className="text-2xl font-bold">{card.value.toLocaleString()}</div>
                      <div className="text-xs text-muted-foreground mt-0.5 font-medium">{card.label}</div>
                    </div>
                  </div>
                </div>
              )
            })}
          </Grid>
        </div>
      </div>

      {/* Solution F: Hover Actions */}
      <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
        <SectionHeader
          title="方案 F：Hover Actions"
          description="悬停时显示操作按钮，强调交互性"
        />
        <div className="p-4 bg-muted/10">
          <Grid>
            {assetCards.map((card) => (
              <div
                key={card.label}
                className="group relative bg-card rounded-lg border p-4 transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:border-primary/50"
              >
                <div className="flex flex-col h-full justify-between gap-4">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold uppercase text-muted-foreground tracking-wider">{card.label}</span>
                    <card.icon className="h-4 w-4 text-muted-foreground" />
                  </div>
                  
                  <div>
                    <div className="text-3xl font-bold tracking-tight">{card.value.toLocaleString()}</div>
                    <div className="flex items-center gap-1.5 mt-2">
                       <Activity className="h-3 w-3 text-emerald-500" />
                       <span className="text-xs text-muted-foreground">Active monitoring</span>
                    </div>
                  </div>

                  {/* Hover Overlay Actions */}
                  <div className="absolute inset-0 bg-card/95 backdrop-blur-[1px] opacity-0 group-hover:opacity-100 transition-opacity flex flex-col items-center justify-center gap-2 p-2">
                    <Button size="sm" variant="outline" className="w-full h-8 text-xs gap-1.5">
                      <Scan className="h-3.5 w-3.5" />
                      Quick Scan
                    </Button>
                    <Button size="sm" variant="ghost" className="w-full h-8 text-xs gap-1.5">
                      <MoreHorizontal className="h-3.5 w-3.5" />
                      Details
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </Grid>
        </div>
      </div>

      {/* Solution G: Cyber ​​HUD */}
      <div className="border rounded-xl overflow-hidden shadow-sm bg-black text-white">
        <SectionHeader
          title="方案 G：Cyber HUD (Dark Mode Exclusive)"
          description="深色模式专用的高对比度、荧光风格"
        />
        <div className="p-4 bg-neutral-950">
          <Grid>
            {assetCards.map((card) => (
              <div
                key={card.label}
                className="relative bg-neutral-900 border border-neutral-800 p-4 overflow-hidden group hover:border-cyan-500/50 transition-colors"
              >
                {/* Glow effect */}
                <div className="absolute -top-10 -right-10 w-20 h-20 bg-cyan-500/10 blur-xl rounded-full group-hover:bg-cyan-500/20 transition-[color,background-color,border-color,opacity,transform,box-shadow]" />
                
                <div className="relative z-10">
                  <div className="flex justify-between items-start mb-3">
                    <card.icon className="h-5 w-5 text-neutral-500 group-hover:text-cyan-400 transition-colors" />
                    <div className="h-1.5 w-1.5 bg-neutral-700 rounded-full group-hover:bg-cyan-400 group-hover:shadow-[0_0_8px_rgba(34,211,238,0.8)] transition-[color,background-color,border-color,opacity,transform,box-shadow]" />
                  </div>
                  
                  <div className="text-3xl font-mono text-neutral-100 tabular-nums">
                    {card.value.toLocaleString()}
                  </div>
                  
                  <div className="mt-2 text-[10px] uppercase tracking-[0.2em] text-neutral-500 group-hover:text-cyan-200/70 transition-colors">
                    {card.label}
                  </div>
                </div>
                
                {/* Decorative corner */}
                <div className="absolute bottom-0 right-0 p-1">
                   <div className="w-2 h-2 border-r border-b border-neutral-700 group-hover:border-cyan-500/50" />
                </div>
              </div>
            ))}
          </Grid>
        </div>
      </div>

      <style jsx>{`
        :global([data-theme="bauhaus"] .asset-card-variants .bg-card) {
          border-top: 1px solid var(--border);
        }

        .motion-lift {
          transition: transform 0.2s ease, box-shadow 0.2s ease;
        }

        .motion-lift:hover {
          transform: translateY(-4px);
          box-shadow: 0 10px 20px rgba(15, 23, 42, 0.08);
        }

        .motion-scan {
          overflow: hidden;
        }

        .motion-scan::after {
          content: "";
          position: absolute;
          inset: 0;
          background: linear-gradient(120deg, transparent 0%, rgba(59, 130, 246, 0.12) 45%, transparent 70%);
          transform: translateX(-120%);
          transition: transform 0.6s ease;
          pointer-events: none;
        }

        .motion-scan:hover::after {
          transform: translateX(120%);
        }

        .animate-scan-line {
          animation: scanLine 2s linear infinite;
        }
        
        @keyframes scanLine {
          0% { transform: translateY(-100%); opacity: 0; }
          10% { opacity: 1; }
          90% { opacity: 1; }
          100% { transform: translateY(400%); opacity: 0; }
        }

        .animate-scan-y {
           animation: scanY 3s ease-in-out infinite;
        }

        @keyframes scanY {
          0%, 100% { transform: translateY(-10%); opacity: 0; }
          50% { transform: translateY(10%); opacity: 0.2; }
        }


        [data-motion="stagger"] .motion-rise {
          animation: cardRise 0.45s ease forwards;
          opacity: 0;
          transform: translateY(6px);
          animation-delay: var(--delay, 0ms);
        }

        @keyframes cardRise {
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @media (prefers-reduced-motion: reduce) {
          .motion-lift,
          .motion-scan,
          [data-motion="stagger"] .motion-rise {
            transition: none;
            animation: none;
          }

          .motion-scan::after {
            display: none;
          }
        }
      `}</style>
    </div>
  )
}
