"use client"

import React, { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { IconBrandGithub, IconStarFilled, IconStar } from "@/components/icons"
import { PageHeader } from "@/components/common/page-header"
import { motion } from "framer-motion"

const PENGUIN_LOGISTICS_BARCODE_HEIGHTS = [2, 4, 3, 5, 2, 4] as const

export default function GithubButtonDemoPage() {
  const [stars, setStars] = useState<number | null>(null)
  
  // Simulate loading of stars, the API will be called in the actual project
  useEffect(() => {
    // Delay 800ms to simulate network requests
    const timer = setTimeout(() => {
      setStars(1286) // simulate a number
    }, 800)
    return () => clearTimeout(timer)
  }, [])

  return (
    <div className="flex h-screen flex-col bg-background">
      <PageHeader
        title="GitHub Star Button Variants"
        breadcrumbItems={[
          { label: "Tools", href: "/tools" },
          { label: "GitHub Buttons", href: "/tools/github-buttons" },
        ]}
      />
      
      <div className="flex-1 overflow-auto p-6 md:p-8">
        <div className="mx-auto grid max-w-4xl gap-8">
          
          <div className="space-y-4">
            <h2 className="text-lg font-semibold">1. Standard Variants (GitHub Style)</h2>
            <p className="text-sm text-muted-foreground">Standard buttons mimicking GitHub&apos;s own UI or slight variations.</p>
            
            <div className="flex flex-wrap items-center gap-6 rounded-lg border border-dashed p-6">
              
              {/* Variant A: Default Outline */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Outline (Current)</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-0 pl-3 pr-0 overflow-hidden"
                  asChild
                >
                  <a href="https://github.com" className="flex items-center">
                    <div className="flex items-center gap-2 mr-3">
                      <IconBrandGithub className="size-4" />
                      <span>Star</span>
                    </div>
                    <div className="flex h-full items-center border-l bg-muted/50 px-3 hover:bg-muted transition-colors">
                      <span className="text-xs tabular-nums">{stars?.toLocaleString() ?? "…"}</span>
                      <IconStarFilled className="ml-1 size-3 text-yellow-500/80" />
                    </div>
                  </a>
                </Button>
              </div>

              {/* Variant B: Dark/Solid */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Solid Dark</span>
                <Button
                  size="sm"
                  className="gap-0 pl-3 pr-0 overflow-hidden bg-zinc-900 text-white hover:bg-zinc-800 dark:bg-zinc-50 dark:text-zinc-900 dark:hover:bg-zinc-200"
                  asChild
                >
                  <a href="https://github.com" className="flex items-center">
                    <div className="flex items-center gap-2 mr-3">
                      <IconBrandGithub className="size-4" />
                      <span>Star</span>
                    </div>
                    <div className="flex h-full items-center border-l border-white/20 bg-white/10 px-3 transition-colors dark:border-black/10 dark:bg-black/5">
                      <span className="text-xs tabular-nums">{stars?.toLocaleString() ?? "…"}</span>
                    </div>
                  </a>
                </Button>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <h2 className="text-lg font-semibold">2. Call-to-Action Emphasis</h2>
            <p className="text-sm text-muted-foreground">Using color and icons to draw more attention.</p>
            
            <div className="flex flex-wrap items-center gap-6 rounded-lg border border-dashed p-6">
              
               {/* Variant C: Yellow Icon Highlight */}
               <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Yellow Star Icon</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-2"
                  asChild
                >
                  <a href="https://github.com">
                     <IconBrandGithub className="size-4" />
                     <span>Star on GitHub</span>
                     <IconStarFilled className="size-3.5 text-yellow-500" />
                  </a>
                </Button>
              </div>

               {/* Variant D: Gradient Border (Simulated) */}
               <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Subtle Highlight</span>
                <Button
                  variant="secondary"
                  size="sm"
                  className="gap-2 border-yellow-500/30 bg-yellow-500/10 hover:bg-yellow-500/20 text-yellow-700 dark:text-yellow-400"
                  asChild
                >
                  <a href="https://github.com">
                     <IconStar className="size-4" />
                     <span>Star us!</span>
                     <span className="ml-1 rounded bg-background/50 px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                      {stars?.toLocaleString() ?? "…"}
                     </span>
                  </a>
                </Button>
              </div>
            </div>
          </div>


          <div className="space-y-4">
            <h2 className="text-lg font-semibold">3. Playful / Social Proof</h2>
            <p className="text-sm text-muted-foreground">More engaging or minimal styles.</p>
            
            <div className="flex flex-wrap items-center gap-6 rounded-lg border border-dashed p-6">
              
              {/* Variant E: Pill Shape */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Pill Shape</span>
                <div className="flex items-center rounded-full border bg-background p-1 shadow-sm">
                  <a href="https://github.com" className="flex items-center gap-1.5 rounded-full bg-zinc-900 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-zinc-800 dark:bg-zinc-50 dark:text-zinc-900">
                    <IconBrandGithub className="size-3.5" />
                    <span>Star</span>
                  </a>
                  <div className="flex items-center px-2 pr-3 text-xs font-medium text-muted-foreground">
                    <IconStarFilled className="mr-1 size-3 text-yellow-500" />
                    {stars?.toLocaleString() ?? "…"}
                  </div>
                </div>
              </div>

              {/* Variant F: Minimal Text */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Minimal Link</span>
                <a href="https://github.com" className="group flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
                   <span className="flex items-center gap-1 rounded-md border bg-background px-2 py-1 transition-colors group-hover:border-foreground/30">
                     <IconStar className="size-3.5" /> 
                     <span className="text-xs font-medium">{stars?.toLocaleString() ?? "…"}</span>
                   </span>
                   <span className="text-xs">stars on GitHub</span>
                </a>
              </div>

            </div>
          </div>
          
          <div className="space-y-4">
            <h2 className="text-lg font-semibold">4. Animated (Framer Motion)</h2>
            <p className="text-sm text-muted-foreground">Interactive animations to delight users.</p>
            
            <div className="flex flex-wrap items-center gap-6 rounded-lg border border-dashed p-6">
              
              {/* Variant G: Bounce on Hover */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Bounce Icon</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-2 group"
                  asChild
                >
                  <a href="https://github.com">
                    <motion.div
                      whileHover={{ scale: 1.2, rotate: 15 }}
                      transition={{ type: "spring", stiffness: 400, damping: 10 }}
                    >
                      <IconStarFilled className="size-4 text-yellow-500" />
                    </motion.div>
                    <span>Star on GitHub</span>
                    <span className="ml-1 text-xs text-muted-foreground tabular-nums">
                      {stars?.toLocaleString() ?? "…"}
                    </span>
                  </a>
                </Button>
              </div>

              {/* Variant H: Shine Effect */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Shine Effect</span>
                <Button
                  size="sm"
                  className="relative overflow-hidden bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90"
                  asChild
                >
                  <a href="https://github.com">
                    <motion.div
                      className="absolute inset-0 -translate-x-full bg-gradient-to-r from-transparent via-white/20 to-transparent dark:via-black/10"
                      initial={{ x: "-100%" }}
                      animate={{ x: "200%" }}
                      transition={{ repeat: Infinity, duration: 2, repeatDelay: 3 }}
                    />
                    <IconBrandGithub className="mr-2 size-4" />
                    <span>Star us</span>
                    <span className="ml-2 rounded bg-white/20 px-1.5 py-0.5 text-xs font-medium text-white dark:bg-black/10 dark:text-black">
                      {stars?.toLocaleString() ?? "…"}
                    </span>
                  </a>
                </Button>
              </div>
              
              {/* Variant I: Expand on Hover */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Expand Count</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-0 pl-3 pr-0 overflow-hidden group"
                  asChild
                >
                  <a href="https://github.com" className="flex items-center">
                    <div className="flex items-center gap-2 mr-3">
                      <IconBrandGithub className="size-4" />
                      <span>Star</span>
                    </div>
                    <motion.div 
                      className="flex h-full items-center border-l bg-muted/50 px-3 overflow-hidden"
                      initial={{ width: 0, opacity: 0, paddingLeft: 0, paddingRight: 0 }}
                      whileHover={{ width: "auto", opacity: 1, paddingLeft: 12, paddingRight: 12 }}
                      animate={{ width: "auto", opacity: 1, paddingLeft: 12, paddingRight: 12 }} 
                    >
                      <span className="text-xs tabular-nums whitespace-nowrap">{stars?.toLocaleString() ?? "…"}</span>
                      <IconStarFilled className="ml-1 size-3 text-yellow-500/80" />
                    </motion.div>
                  </a>
                </Button>
              </div>

            </div>
          </div>

          <div className="space-y-4">
            <h2 className="text-lg font-semibold">6. Experimental Styles (User Request)</h2>
            <p className="text-sm text-muted-foreground">Concept designs: Rounded, Cyberpunk, Arknights, NieR, Deconstructivist.</p>
            
            <div className="flex flex-wrap items-center gap-8 rounded-lg border border-dashed p-6">
              
              {/* Variant: Fully Rounded */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Fully Rounded</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="rounded-full gap-0 pl-4 pr-0 overflow-hidden border-zinc-200 dark:border-zinc-800"
                  asChild
                >
                  <a href="https://github.com" className="group">
                    <div className="flex items-center gap-2 mr-4">
                      <IconBrandGithub className="size-4" />
                      <span>Star</span>
                    </div>
                    <div className="flex h-full items-center border-l bg-zinc-100 px-4 hover:bg-zinc-200 transition-colors dark:bg-zinc-800 dark:hover:bg-zinc-700">
                      <span className="text-xs tabular-nums">{stars?.toLocaleString() ?? "…"}</span>
                      <IconStarFilled className="ml-1 size-3 text-yellow-500" />
                    </div>
                  </a>
                </Button>
              </div>

              {/* Variant: Cyberpunk */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Cyberpunk (Dark)</span>
                <a href="https://github.com" className="group relative inline-flex items-center justify-center p-0.5 overflow-hidden text-sm font-medium rounded-sm bg-gradient-to-br from-[#fcee0a] to-[#00f0ff] hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#00f0ff] focus-visible:ring-offset-2 focus-visible:ring-offset-black">
                  <span className="relative flex items-center px-4 py-1.5 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] ease-in duration-75 bg-black text-[#fcee0a] rounded-sm group-hover:bg-opacity-90 font-mono tracking-wider uppercase border border-[#fcee0a]/30 shadow-[0_0_15px_rgba(252,238,10,0.3)]">
                    <IconBrandGithub className="mr-2 size-4" />
                    <span className="mr-2 font-bold">STAR</span>
                    <span className="border-l border-[#fcee0a]/50 pl-2 text-cyan-300">{stars?.toLocaleString() ?? "ERR"}</span>
                  </span>
                </a>
              </div>

              {/* Variant: Cyberpunk Light */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Cyberpunk (Light)</span>
                <a href="https://github.com" className="group relative inline-flex items-center justify-center overflow-hidden text-sm font-medium rounded-sm bg-[#fcee0a] text-black hover:bg-[#e6d809] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-black focus-visible:ring-offset-2 focus-visible:ring-offset-[#fcee0a] shadow-[4px_4px_0px_#000000]">
                  <span className="relative flex items-center px-5 py-2 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] ease-in duration-75 font-mono tracking-wider uppercase border-2 border-black">
                    <IconBrandGithub className="mr-2 size-4" />
                    <span className="mr-2 font-bold">STAR</span>
                    <span className="border-l-2 border-black pl-2 font-bold">{stars?.toLocaleString() ?? "2077"}</span>
                  </span>
                  {/* Glitch overlay */}
                  <span className="absolute top-0 left-0 w-full h-full bg-[#00f0ff] mix-blend-multiply opacity-0 group-hover:opacity-100 group-hover:translate-x-1 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-75 pointer-events-none" />
                </a>
              </div>

              {/* Variant: Arknights (Light/Rhodes Clean) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Arknights (Light)</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-[#F5F5F5] text-zinc-900 border border-zinc-300 hover:border-[#00C8FF] transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] overflow-hidden min-w-[120px]">
                   {/* Diagonal background element */}
                   <div className="absolute inset-0 bg-white transform -skew-x-12 translate-x-[-150%] group-hover:translate-x-[-20%] transition-transform duration-500 opacity-80" />
                   
                   <div className="relative flex items-center px-3 z-10 w-full justify-between">
                     <div className="flex items-center gap-2">
                       <IconBrandGithub className="size-4 text-zinc-700 group-hover:text-[#00C8FF] transition-colors" />
                       <span className="font-sans font-bold uppercase text-[10px] tracking-widest text-zinc-600 group-hover:text-zinc-900 transition-colors">Star</span>
                     </div>
                     <span className="font-mono text-[10px] text-zinc-500 group-hover:text-[#00C8FF] transition-colors">
                       [{stars?.toLocaleString() ?? "…"}]
                     </span>
                   </div>
                   
                   {/* Corner accents */}
                   <div className="absolute top-0 right-0 w-1 h-1 bg-[#00C8FF] opacity-0 group-hover:opacity-100 transition-opacity" />
                   <div className="absolute bottom-0 left-0 w-1 h-1 bg-[#00C8FF] opacity-0 group-hover:opacity-100 transition-opacity" />
                </a>
              </div>

              {/* Variant: Arknights Light (Industrial) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Arknights (Light)</span>
                <a href="https://github.com" className="group relative flex items-center h-10 bg-white border-l-4 border-l-[#00C8FF] border-y border-r border-y-zinc-200 border-r-zinc-200 min-w-[140px] hover:bg-zinc-50 transition-colors">
                   <div className="flex-1 flex flex-col justify-center px-3">
                      <div className="flex items-center justify-between mb-0.5">
                         <span className="text-[8px] font-bold text-zinc-400 uppercase tracking-widest">Rhodes Island</span>
                         <div className="flex gap-0.5">
                            <div className="w-1 h-1 bg-[#00C8FF]" />
                            <div className="w-1 h-1 bg-zinc-300" />
                            <div className="w-1 h-1 bg-zinc-300" />
                         </div>
                      </div>
                      
                      <div className="flex items-center justify-between">
                         <div className="flex items-center gap-1.5">
                            <IconBrandGithub className="size-3.5 text-zinc-800" />
                            <span className="text-xs font-black text-zinc-800 uppercase tracking-tighter">PRTS.TERM</span>
                         </div>
                         <span className="text-xs font-mono font-bold text-zinc-600 group-hover:text-[#00C8FF] transition-colors">{stars?.toLocaleString()}</span>
                      </div>
                   </div>
                </a>
              </div>

               {/* Variant: Penguin Logistics */}
               <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Penguin Logistics</span>
                <a href="https://github.com" className="group flex items-center h-9 bg-white border-2 border-black relative overflow-hidden min-w-[130px] hover:shadow-[4px_4px_0px_#000] transition-shadow">
                  {/* Barcode Strip */}
                  <div className="absolute left-0 top-0 bottom-0 w-1.5 bg-black flex flex-col justify-between py-0.5">
                    {PENGUIN_LOGISTICS_BARCODE_HEIGHTS.map((height, i) => (
                      <div key={i} className="w-full bg-white" style={{ height }} />
                    ))}
                  </div>

                  <div className="flex items-center w-full pl-4 pr-3 justify-between">
                    <div className="flex flex-col leading-none">
                      <span className="text-[10px] font-bold uppercase tracking-tighter">P.L. Logistics</span>
                      <div className="flex items-center gap-1">
                         <IconBrandGithub className="size-3" />
                         <span className="font-bold text-xs uppercase">STAR</span>
                      </div>
                    </div>
                    <div className="flex items-center justify-center bg-black text-white text-[10px] font-mono px-1 py-0.5 min-w-[40px]">
                      {stars?.toLocaleString() ?? "…"}
                    </div>
                  </div>
                  
                  {/* Sticker Overlay */}
                  <div className="absolute -right-2 -top-2 bg-[#00AEEF] text-white text-[8px] font-bold px-2 py-0.5 rotate-12 shadow-sm border border-black">
                    EXP
                  </div>
                </a>
              </div>

              {/* Variant: Reunion */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Reunion</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-[#222] text-white min-w-[120px] overflow-hidden">
                  {/* Grunge Texture / Stencil */}
                  <div className="absolute inset-0 border-2 border-[#ff5000] opacity-80 group-hover:opacity-100 transition-opacity" style={{ clipPath: 'polygon(0 0, 100% 0, 100% 80%, 90% 100%, 0 100%)' }} />
                  
                  <div className="relative z-10 flex items-center justify-between w-full px-3">
                     <div className="flex items-center gap-2">
                        <IconBrandGithub className="size-4 text-[#ff5000]" />
                        <span className="font-black italic uppercase tracking-wider text-xs">STAR</span>
                     </div>
                     <span className="font-mono text-xs text-[#ff5000] border-b border-[#ff5000]">
                       {stars?.toLocaleString() ?? "err"}
                     </span>
                  </div>
                  
                  {/* Background Scratches */}
                  <div className="absolute inset-0 bg-[repeating-linear-gradient(45deg,transparent,transparent_2px,#000_2px,#000_4px)] opacity-20 pointer-events-none" />
                </a>
              </div>

              {/* Variant: Rhodes Operation */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Rhodes Operation</span>
                <a href="https://github.com" className="group relative flex items-center justify-center h-9 px-6 bg-[#2B2B2B]/90 backdrop-blur-sm border border-[#444] hover:border-[#eee] transition-colors overflow-hidden">
                  {/* Hexagon Pattern */}
                  <div className="absolute inset-0 opacity-10 bg-[url('https://api.iconify.design/mdi:hexagon-outline.svg?color=%23ffffff')] bg-[length:20px_20px]" />
                  
                  {/* Center Line */}
                  <div className="absolute left-0 top-1/2 w-full h-[1px] bg-gradient-to-r from-transparent via-[#ffffff55] to-transparent scale-x-0 group-hover:scale-x-100 transition-transform duration-500" />

                  <div className="relative flex items-center gap-3 z-10">
                    <IconBrandGithub className="size-3.5 text-[#bbb] group-hover:text-white transition-colors" />
                    <span className="text-xs font-medium tracking-[0.2em] text-[#bbb] group-hover:text-white transition-colors uppercase">Deploy</span>
                    <div className="h-3 w-[1px] bg-[#555]" />
                    <span className="font-mono text-[10px] text-[#00C8FF] text-shadow-[0_0_5px_rgba(0,200,255,0.5)]">
                       {stars?.toLocaleString() ?? "…"}
                    </span>
                  </div>
                  
                  {/* Corner Brackets */}
                  <div className="absolute top-1 left-1 w-2 h-2 border-t border-l border-[#888] opacity-50 group-hover:opacity-100 group-hover:w-3 group-hover:h-3 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]" />
                  <div className="absolute bottom-1 right-1 w-2 h-2 border-b border-r border-[#888] opacity-50 group-hover:opacity-100 group-hover:w-3 group-hover:h-3 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]" />
                </a>
              </div>

              {/* Variant: Rhine Lab */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Rhine Lab (Science)</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-white border border-zinc-200 rounded-sm overflow-hidden min-w-[130px] shadow-sm hover:shadow-md transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]">
                  {/* Orange Accent Bar */}
                  <div className="absolute top-0 left-0 w-1 h-full bg-[#FF7D00]" />
                  
                  {/* Green Status Dot */}
                  <div className="absolute top-1.5 right-1.5 w-1.5 h-1.5 rounded-full bg-[#4ADE80] animate-pulse" />

                  <div className="flex flex-col pl-4 pr-3 py-1 w-full">
                    <div className="flex items-center justify-between">
                       <span className="text-[9px] font-bold text-zinc-400 uppercase tracking-widest">Scientific</span>
                       <span className="text-[8px] font-mono text-zinc-300">v.1.0</span>
                    </div>
                    <div className="flex items-center justify-between mt-0.5">
                       <div className="flex items-center gap-1.5">
                          <IconBrandGithub className="size-3 text-zinc-700" />
                          <span className="text-xs font-bold text-zinc-800">STAR</span>
                       </div>
                       <span className="font-mono text-xs text-[#FF7D00] font-bold">
                         {stars?.toLocaleString() ?? "…"}
                       </span>
                    </div>
                  </div>
                </a>
              </div>

              {/* Variant: Laterano / Sankta */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Laterano (Divine)</span>
                <a href="https://github.com" className="group relative flex items-center justify-center h-9 px-5 bg-[#F5F5F0] border border-[#D4AF37] text-[#333] overflow-hidden rounded-[2px] shadow-[0_2px_10px_rgba(212,175,55,0.1)] hover:shadow-[0_4px_15px_rgba(212,175,55,0.25)] transition-shadow">
                  {/* Decorative Cross/Halo */}
                  <div className="absolute inset-0 flex items-center justify-center opacity-10 pointer-events-none">
                     <div className="w-[80%] h-[1px] bg-[#D4AF37]" />
                     <div className="absolute h-[80%] w-[1px] bg-[#D4AF37]" />
                     <div className="absolute w-[60%] h-[60%] border border-[#D4AF37] rounded-full" />
                  </div>

                  <div className="relative z-10 flex items-center gap-2">
                    <IconBrandGithub className="size-3.5 text-[#D4AF37]" />
                    <span className="font-serif italic font-bold text-xs tracking-wide">Sanctify</span>
                    <span className="text-[10px] font-bold bg-[#D4AF37] text-white px-1.5 py-0.5 rounded-sm">
                       {stars?.toLocaleString() ?? "…"}
                    </span>
                  </div>
                </a>
              </div>

               {/* Variant: Kjerag / SilverAsh */}
               <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Kjerag (Silver)</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-slate-50 border border-slate-200 overflow-hidden min-w-[120px]">
                   {/* Ice Gradient */}
                   <div className="absolute inset-0 bg-gradient-to-r from-transparent via-sky-100/30 to-transparent translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000" />
                   
                   {/* Snowflake-like decorative element */}
                   <div className="absolute -left-2 -top-2 w-6 h-6 border border-slate-300 rotate-45 opacity-20" />
                   
                   <div className="relative z-10 flex items-center justify-between w-full px-3">
                     <div className="flex items-center gap-1.5">
                        <IconBrandGithub className="size-3.5 text-slate-600" />
                        <span className="text-xs font-medium text-slate-700 uppercase tracking-widest">Enciodes</span>
                     </div>
                     <div className="h-full border-l border-slate-200 pl-2 ml-2 flex items-center">
                        <span className="font-serif text-xs text-slate-500 group-hover:text-sky-600 transition-colors">
                          {stars?.toLocaleString() ?? "…"}
                        </span>
                     </div>
                   </div>
                </a>
              </div>

               {/* Variant: L.G.D. */}
               <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">L.G.D. (Police)</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-white border border-blue-600 shadow-sm overflow-hidden min-w-[130px]">
                  {/* Left Stripe */}
                  <div className="absolute left-0 top-0 bottom-0 w-1.5 bg-blue-600" />
                  
                  {/* Background Stripes */}
                  <div className="absolute inset-0 bg-[repeating-linear-gradient(45deg,transparent,transparent_10px,rgba(37,99,235,0.05)_10px,rgba(37,99,235,0.05)_20px)]" />

                  <div className="relative z-10 flex items-center w-full pl-4 pr-3 justify-between">
                     <div className="flex flex-col">
                        <span className="text-[9px] font-black text-blue-600 leading-none">L.G.D.</span>
                        <div className="flex items-center gap-1">
                           <IconBrandGithub className="size-3 text-slate-700" />
                           <span className="text-xs font-bold text-slate-800 uppercase">STAR</span>
                        </div>
                     </div>
                     <div className="flex items-center gap-1">
                        <div className="w-1.5 h-1.5 rounded-full bg-red-500 opacity-20 group-hover:animate-ping" />
                        <span className="font-mono text-xs font-bold text-slate-600">
                          {stars?.toLocaleString() ?? "…"}
                        </span>
                     </div>
                  </div>
                </a>
              </div>

               {/* Variant: Kazimierz */}
               <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Kazimierz (Neon)</span>
                <a href="https://github.com" className="group relative flex items-center justify-center h-9 px-4 bg-white border-2 border-transparent bg-clip-padding rounded-sm overflow-hidden">
                   {/* Neon Border Effect */}
                   <div className="absolute inset-0 bg-gradient-to-r from-blue-400 via-purple-400 to-pink-400 opacity-50 group-hover:opacity-100 transition-opacity -z-10" />
                   <div className="absolute inset-[2px] bg-white -z-10 rounded-[1px]" />
                   
                   <div className="flex items-center gap-2">
                      <span className="text-[10px] font-bold text-transparent bg-clip-text bg-gradient-to-r from-blue-600 to-purple-600 uppercase tracking-tighter">
                        Sponsored
                      </span>
                      <div className="w-[1px] h-3 bg-zinc-200" />
                      <div className="flex items-center gap-1 text-zinc-800">
                         <IconBrandGithub className="size-3" />
                         <span className="font-mono text-xs font-bold">{stars?.toLocaleString() ?? "…"}</span>
                      </div>
                   </div>
                </a>
              </div>

               {/* Variant: Blue Archive */}
               <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Blue Archive</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-white min-w-[140px] shadow-sm rounded-tl-xl rounded-br-xl border-t-2 border-b-2 border-[#1289F4] overflow-visible hover:-translate-y-0.5 transition-transform">
                   
                   {/* Halo Effect */}
                   <div className="absolute -top-3 left-1/2 -translate-x-1/2 w-8 h-2 border border-pink-400/50 rounded-[50%] opacity-0 group-hover:opacity-100 group-hover:-top-4 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-500" />

                   <div className="flex items-center w-full px-4 justify-between relative bg-white rounded-tl-[10px] rounded-br-[10px]">
                      <div className="flex items-center gap-1.5">
                         <div className="bg-[#1289F4] rounded-full p-0.5">
                           <IconBrandGithub className="size-3 text-white" />
                         </div>
                         <span className="text-xs font-bold text-[#2B3F56]">Schale</span>
                      </div>
                      
                      <div className="flex flex-col items-end leading-none">
                         <span className="text-[9px] text-[#8899AA] uppercase font-bold tracking-wide">Request</span>
                         <span className="text-sm font-black text-[#2B3F56] font-sans">
                           {stars?.toLocaleString() ?? "…"}
                         </span>
                      </div>
                   </div>
                </a>
              </div>

               {/* Variant: NieR: Automata */}
               <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">NieR: Automata</span>
                <a href="https://github.com" className="group flex items-center gap-3 bg-[#DAD4BB] text-[#4D4D4D] px-4 py-1.5 border border-[#4D4D4D] shadow-[2px_2px_0px_#4D4D4D] hover:translate-x-[1px] hover:translate-y-[1px] hover:shadow-[1px_1px_0px_#4D4D4D] transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] relative overflow-hidden">
                  {/* Scanline effect overlay */}
                  <div className="absolute inset-0 bg-[linear-gradient(rgba(18,16,16,0)_50%,rgba(0,0,0,0.1)_50%),linear-gradient(90deg,rgba(255,0,0,0.06),rgba(0,255,0,0.02),rgba(0,0,255,0.06))] z-0 bg-[length:100%_2px,3px_100%] pointer-events-none" />
                  
                  <div className="relative z-10 flex items-center gap-2">
                    <div className="size-1.5 bg-[#4D4D4D] animate-pulse rounded-full" />
                    <span className="font-serif tracking-widest font-bold text-xs uppercase">Pod</span>
                  </div>
                  <span className="relative z-10 font-mono text-xs border-b border-[#4D4D4D] border-dashed border-opacity-50 group-hover:border-opacity-100">
                    {stars?.toLocaleString() ?? "…"}
                  </span>
                </a>
              </div>

              {/* Variant: Deconstructivist */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Deconstructivist</span>
                <a href="https://github.com" className="group relative h-9 w-28 mt-1">
                   {/* Background Layer */}
                   <div className="absolute inset-0 bg-zinc-900 dark:bg-zinc-100 translate-x-1.5 translate-y-1.5 transition-transform group-hover:translate-x-2 group-hover:translate-y-2" />
                   
                   {/* Foreground Layer */}
                   <div className="absolute inset-0 border border-zinc-900 dark:border-zinc-100 bg-white dark:bg-black flex items-center justify-center hover:-translate-y-0.5 hover:-translate-x-0.5 transition-transform duration-200 z-10">
                      <IconBrandGithub className="mr-1.5 size-3.5" />
                      <span className="font-bold italic text-xs">STAR</span>
                   </div>
                   
                   {/* Floating Tag */}
                   <div className="absolute -top-2 -right-3 bg-red-500 text-white text-[9px] font-bold px-1.5 py-0.5 transform rotate-12 z-20 shadow-sm">
                     {stars?.toLocaleString() ?? "…"}
                   </div>
                </a>
              </div>

            </div>
          </div>

          <div className="space-y-4">
            <h2 className="text-lg font-semibold">8. Industrial / Mecha (Light Theme)</h2>
            <p className="text-sm text-muted-foreground">Functional, heavy-duty aesthetics with an anime touch.</p>
            
            <div className="flex flex-wrap items-center gap-8 rounded-lg border p-6 bg-zinc-100">
              
              {/* Variant: IOP / Logistics */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Logistics (IOP)</span>
                <a href="https://github.com" className="group relative flex items-center h-10 bg-[#F0F2F5] border-2 border-[#8A95A5] text-[#3E4856] min-w-[140px] hover:bg-white transition-colors">
                   {/* Left Accent */}
                   <div className="w-1.5 h-full bg-[#FFD000] border-r border-[#8A95A5]" />
                   
                   <div className="flex-1 flex flex-col justify-center px-3">
                      <div className="flex items-center justify-between border-b border-[#8A95A5]/30 pb-0.5">
                         <span className="text-[9px] font-bold tracking-wider text-[#8A95A5]">T-DOLL</span>
                         <span className="text-[9px] font-mono text-[#FFD000] bg-[#3E4856] px-1 rounded-[1px]">★5</span>
                      </div>
                      <div className="flex items-center justify-between pt-0.5">
                         <div className="flex items-center gap-1.5">
                            <IconBrandGithub className="size-3.5" />
                            <span className="text-xs font-black uppercase">SUPPLY</span>
                         </div>
                         <span className="text-xs font-mono font-bold">{stars?.toLocaleString()}</span>
                      </div>
                   </div>
                </a>
              </div>

              {/* Variant: Mecha Panel (Anaheim) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Mecha Panel (AE)</span>
                <a href="https://github.com" className="group relative flex items-center h-10 bg-white border border-zinc-300 shadow-sm overflow-hidden min-w-[150px] hover:border-red-500 transition-colors">
                   {/* Caution Strip */}
                   <div className="absolute top-0 right-0 p-1">
                      <div className="flex gap-0.5">
                         {[...Array(4)].map((_, i) => (
                           <div key={i} className="w-4 h-1 bg-zinc-200 -skew-x-12 group-hover:bg-red-500/20 transition-colors" />
                         ))}
                      </div>
                   </div>

                   <div className="h-full w-2 bg-zinc-100 border-r border-zinc-200 flex flex-col items-center justify-center gap-1">
                      <div className="w-1 h-1 rounded-full bg-zinc-300 group-hover:bg-green-500 transition-colors" />
                      <div className="w-1 h-1 rounded-full bg-zinc-300" />
                      <div className="w-1 h-1 rounded-full bg-zinc-300" />
                   </div>

                   <div className="flex-1 px-3 flex flex-col justify-center">
                      <span className="text-[8px] font-mono text-zinc-400 uppercase leading-none mb-1">System Status: Normal</span>
                      <div className="flex items-center justify-between">
                         <span className="text-sm font-bold tracking-tighter text-zinc-800 group-hover:text-red-600 transition-colors">RX-78</span>
                         <div className="flex items-center gap-1 bg-zinc-100 px-1.5 py-0.5 rounded border border-zinc-200">
                            <IconBrandGithub className="size-3" />
                            <span className="font-mono text-xs font-bold">{stars?.toLocaleString()}</span>
                         </div>
                      </div>
                   </div>
                </a>
              </div>

              {/* Variant: NERV / Warning */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Monitor (NERV)</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-white border border-red-600 text-red-600 hover:bg-red-50 transition-colors min-w-[130px] shadow-[2px_2px_0px_#dc2626]">
                   {/* Hex pattern overlay */}
                   <div className="absolute inset-0 opacity-5 bg-[url('https://api.iconify.design/mdi:hexagon-multiple.svg?color=%23ff0000')] bg-[length:30px_30px]" />
                   
                   <div className="relative z-10 flex items-center w-full">
                      <div className="h-full bg-red-600 text-white flex items-center px-2 font-black italic text-xs tracking-widest">
                         EMERGENCY
                      </div>
                      <div className="flex-1 flex items-center justify-center gap-2 px-2">
                         <IconBrandGithub className="size-4" />
                         <span className="font-mono text-sm font-bold border-b-2 border-red-600/20 group-hover:border-red-600 transition-colors">
                           {stars?.toLocaleString()}
                         </span>
                      </div>
                   </div>
                </a>
              </div>

              {/* Variant: Hyperion (Borderlands) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Construct (Hyperion)</span>
                <a href="https://github.com" className="group relative flex items-center h-10 bg-white border-2 border-zinc-800 border-t-[4px] border-t-[#FFD700] min-w-[140px] hover:-translate-y-0.5 transition-transform">
                   <div className="flex-1 flex flex-col justify-center px-3 relative">
                      {/* Technical lines */}
                      <div className="absolute top-0 right-0 w-8 h-[1px] bg-zinc-200" />
                      <div className="absolute bottom-0 left-0 w-8 h-[1px] bg-zinc-200" />
                      
                      <div className="flex items-center justify-between mb-0.5">
                         <span className="text-[8px] font-bold text-zinc-400 uppercase tracking-widest">Loader Bot</span>
                         <div className="w-1.5 h-1.5 rounded-full bg-[#FFD700]" />
                      </div>
                      
                      <div className="flex items-center justify-between">
                         <div className="flex items-center gap-1.5">
                            <IconBrandGithub className="size-3.5 text-zinc-800" />
                            <span className="text-xs font-black text-zinc-800 uppercase tracking-tighter">DIGISTRUCT</span>
                         </div>
                         <span className="text-xs font-mono font-bold text-zinc-800">{stars?.toLocaleString()}</span>
                      </div>
                   </div>
                </a>
              </div>

              {/* Variant: Aperture Science (Portal) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Test Chamber (Aperture)</span>
                <a href="https://github.com" className="group relative flex items-center h-10 bg-white border border-zinc-200 min-w-[150px] overflow-hidden hover:shadow-md transition-shadow">
                   {/* Split Background */}
                   <div className="absolute left-0 top-0 bottom-0 w-1/2 bg-white border-r border-zinc-100" />
                   
                   {/* Portal Accents */}
                   <div className="absolute left-0 top-0 bottom-0 w-1 bg-[#FFA500] opacity-0 group-hover:opacity-100 transition-opacity" />
                   <div className="absolute right-0 top-0 bottom-0 w-1 bg-[#00AEEF] opacity-0 group-hover:opacity-100 transition-opacity" />

                   <div className="relative z-10 flex items-center w-full px-3 justify-between">
                      <div className="flex items-center gap-2">
                         <div className="flex flex-col items-center justify-center border border-zinc-800 rounded-full w-5 h-5 p-0.5">
                            {/* Aperture Logo Sim */}
                            <svg viewBox="0 0 100 100" className="w-full h-full fill-current text-zinc-800 animate-spin-slow">
                               <path d="M50 0 L60 20 L40 20 Z" transform="rotate(0 50 50)" />
                               <path d="M50 0 L60 20 L40 20 Z" transform="rotate(45 50 50)" />
                               <path d="M50 0 L60 20 L40 20 Z" transform="rotate(90 50 50)" />
                               <path d="M50 0 L60 20 L40 20 Z" transform="rotate(135 50 50)" />
                               <path d="M50 0 L60 20 L40 20 Z" transform="rotate(180 50 50)" />
                               <path d="M50 0 L60 20 L40 20 Z" transform="rotate(225 50 50)" />
                               <path d="M50 0 L60 20 L40 20 Z" transform="rotate(270 50 50)" />
                               <path d="M50 0 L60 20 L40 20 Z" transform="rotate(315 50 50)" />
                            </svg>
                         </div>
                         <span className="text-[10px] font-bold text-zinc-600 uppercase tracking-widest group-hover:text-zinc-900">Subject</span>
                      </div>
                      <span className="font-mono text-sm font-bold text-zinc-800">{stars?.toLocaleString()}</span>
                   </div>
                </a>
              </div>

              {/* Variant: E.F.S.F. (Gundam) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Federation (E.F.S.F.)</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-[#F5F5DC] border border-[#8B8B83] text-[#4A4A4A] min-w-[140px] hover:bg-[#EEEEE0] transition-colors">
                   {/* Rank Stripe */}
                   <div className="absolute left-2 top-1/2 -translate-y-1/2 w-8 h-1.5 flex gap-0.5">
                      <div className="flex-1 bg-[#D32F2F]" />
                      <div className="flex-1 bg-[#D32F2F]" />
                      <div className="flex-1 bg-[#FBC02D]" />
                   </div>
                   
                   <div className="flex-1 flex items-center justify-end px-3 gap-3">
                      <div className="flex flex-col items-end">
                         <span className="text-[7px] font-bold leading-none tracking-tighter opacity-70">U.C.0079</span>
                         <span className="text-[9px] font-black leading-none tracking-wide">E.F.S.F.</span>
                      </div>
                      <div className="h-4 w-[1px] bg-[#8B8B83]/30" />
                      <div className="flex items-center gap-1">
                         <IconBrandGithub className="size-3.5" />
                         <span className="font-mono text-xs font-bold">{stars?.toLocaleString()}</span>
                      </div>
                   </div>
                </a>
              </div>

              {/* Variant: Section 9 (GitS) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Section 9 (Cyber)</span>
                <a href="https://github.com" className="group relative flex items-center h-10 bg-white/80 backdrop-blur border border-emerald-500/30 text-emerald-700 min-w-[140px] overflow-hidden hover:border-emerald-500 hover:text-emerald-600 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]">
                   {/* Digital Noise Background */}
                   <div className="absolute inset-0 bg-[linear-gradient(0deg,transparent_24%,rgba(16,185,129,0.05)_25%,rgba(16,185,129,0.05)_26%,transparent_27%,transparent_74%,rgba(16,185,129,0.05)_75%,rgba(16,185,129,0.05)_76%,transparent_77%,transparent),linear-gradient(90deg,transparent_24%,rgba(16,185,129,0.05)_25%,rgba(16,185,129,0.05)_26%,transparent_27%,transparent_74%,rgba(16,185,129,0.05)_75%,rgba(16,185,129,0.05)_76%,transparent_77%,transparent)] bg-[length:20px_20px]" />
                   
                   <div className="relative z-10 flex items-center justify-between w-full px-3">
                      <div className="flex items-center gap-2">
                         <div className="w-2 h-2 border border-emerald-500 rounded-full flex items-center justify-center">
                            <div className="w-0.5 h-0.5 bg-emerald-500 rounded-full animate-pulse" />
                         </div>
                         <span className="text-[10px] font-mono uppercase tracking-widest">Net</span>
                      </div>
                      
                      <div className="flex items-center gap-1.5">
                         <span className="text-xs font-mono">{stars?.toLocaleString()}</span>
                         <IconBrandGithub className="size-3.5" />
                      </div>
                   </div>
                   
                   {/* Scanning Line */}
                   <div className="absolute top-0 w-full h-[1px] bg-emerald-400 opacity-50 group-hover:animate-[scan_2s_linear_infinite]" />
                </a>
              </div>

              {/* Variant: Atlas (Titanfall) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Atlas (Mech)</span>
                <a href="https://github.com" className="group relative flex items-center h-10 bg-white border-l-4 border-l-[#F85A3E] border-y border-r border-y-zinc-200 border-r-zinc-200 min-w-[140px] hover:bg-zinc-50 transition-colors">
                   <div className="flex-1 flex flex-col justify-center px-3">
                      <div className="flex items-center justify-between mb-0.5">
                         <span className="text-[8px] font-bold text-zinc-400 uppercase">IMC Mfg.</span>
                         <div className="flex gap-0.5">
                            <div className="w-3 h-0.5 bg-[#F85A3E]" />
                            <div className="w-1 h-0.5 bg-zinc-300" />
                         </div>
                      </div>
                      
                      <div className="flex items-center justify-between">
                         <div className="flex items-center gap-1.5">
                            <IconBrandGithub className="size-3.5 text-zinc-700" />
                            <span className="text-xs font-black text-zinc-800 uppercase tracking-tighter">TITAN</span>
                         </div>
                         <span className="text-xs font-mono font-bold text-zinc-600 group-hover:text-[#F85A3E] transition-colors">{stars?.toLocaleString()}</span>
                      </div>
                   </div>
                </a>
              </div>

            </div>
          </div>

          <div className="space-y-4">
            <h2 className="text-lg font-semibold">9. Minimalist / Clean (Light Theme)</h2>
            <p className="text-sm text-muted-foreground">Subtle, airy designs that blend into modern interfaces.</p>
            
            <div className="flex flex-wrap items-center gap-8 rounded-lg border p-6 bg-white">
              
              {/* Variant: Paper / Shadow */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Paper Lift</span>
                <a href="https://github.com" className="group relative flex items-center h-9 px-4 bg-white border border-zinc-100 rounded-sm shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:shadow-[0_4px_12px_rgba(0,0,0,0.08)] hover:-translate-y-0.5 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300">
                   <div className="flex items-center gap-2">
                      <IconBrandGithub className="size-3.5 text-zinc-400 group-hover:text-zinc-800 transition-colors" />
                      <div className="h-3 w-[1px] bg-zinc-100" />
                      <span className="text-xs font-medium text-zinc-600 group-hover:text-zinc-900 transition-colors">{stars?.toLocaleString()}</span>
                   </div>
                </a>
              </div>

              {/* Variant: Glass / Frost */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Frosted Glass</span>
                <div className="relative p-[1px] rounded-full bg-gradient-to-br from-zinc-200 to-zinc-100">
                   <a href="https://github.com" className="flex items-center gap-2 h-8 px-4 bg-white/60 backdrop-blur-md rounded-full hover:bg-white/80 transition-colors">
                      <IconBrandGithub className="size-3.5 text-zinc-600" />
                      <span className="text-xs font-medium text-zinc-700">{stars?.toLocaleString()}</span>
                   </a>
                </div>
              </div>

              {/* Variant: Tag / Badge */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Tag Style</span>
                <a href="https://github.com" className="group flex items-center">
                   <div className="flex items-center justify-center h-8 w-8 bg-zinc-100 rounded-l-md border border-zinc-200 border-r-0 group-hover:bg-zinc-200 transition-colors">
                      <IconBrandGithub className="size-4 text-zinc-600" />
                   </div>
                   <div className="flex items-center h-8 px-3 bg-white border border-zinc-200 rounded-r-md">
                      <span className="text-xs font-mono font-medium text-zinc-600">{stars?.toLocaleString()}</span>
                   </div>
                </a>
              </div>

              {/* Variant: Underline / Link */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Underline</span>
                <a href="https://github.com" className="group flex items-center gap-2 py-1">
                   <span className="text-sm font-medium text-zinc-800">Star on GitHub</span>
                   <span className="flex items-center justify-center min-w-[24px] h-5 bg-zinc-100 text-[10px] font-bold text-zinc-600 rounded-full group-hover:bg-black group-hover:text-white transition-colors">
                      {stars?.toLocaleString()}
                   </span>
                   <div className="absolute bottom-0 left-0 w-full h-[1px] bg-zinc-200 group-hover:bg-black transition-colors" />
                </a>
              </div>

              {/* Variant: Dot / Status */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Status Dot</span>
                <a href="https://github.com" className="flex items-center gap-2 h-8 px-3 rounded-md hover:bg-zinc-50 transition-colors">
                   <div className="relative flex items-center justify-center w-2 h-2">
                      <div className="absolute w-2 h-2 bg-green-500 rounded-full opacity-20 animate-ping" />
                      <div className="relative w-1.5 h-1.5 bg-green-500 rounded-full" />
                   </div>
                   <span className="text-xs font-medium text-zinc-500 hover:text-zinc-900 transition-colors">
                      {stars?.toLocaleString()} stars
                   </span>
                </a>
              </div>

            </div>
          </div>

          <div className="space-y-4">
            <h2 className="text-lg font-semibold">10. ACG / Kawaii (Light Theme)</h2>
            <p className="text-sm text-muted-foreground">Playful, colorful, and character-inspired designs.</p>
            
            <div className="flex flex-wrap items-center gap-8 rounded-lg border p-6 bg-rose-50/30">
              
              {/* Variant: Magical Girl */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Mahou Shoujo</span>
                <a href="https://github.com" className="group relative flex items-center h-10 px-1 bg-white border-2 border-pink-300 rounded-full shadow-[4px_4px_0px_#f9a8d4] hover:translate-x-[2px] hover:translate-y-[2px] hover:shadow-[2px_2px_0px_#f9a8d4] transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]">
                   <div className="w-8 h-8 flex items-center justify-center bg-pink-100 rounded-full border border-pink-200">
                      <IconStarFilled className="size-4 text-pink-500 animate-pulse" />
                   </div>
                   <div className="px-3 flex flex-col items-start leading-none gap-0.5">
                      <span className="text-[10px] font-bold text-pink-400 uppercase tracking-wide">Star Me!</span>
                      <span className="text-sm font-black text-pink-600 font-mono">{stars?.toLocaleString()}</span>
                   </div>
                   {/* Sparkles decoration */}
                   <div className="absolute -top-1 -right-1 text-yellow-400 text-xs">✦</div>
                   <div className="absolute -bottom-1 left-2 text-blue-300 text-[10px]">✧</div>
                </a>
              </div>

              {/* Variant: Manga Panel */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Manga Panel</span>
                <a href="https://github.com" className="group relative flex items-center justify-center w-32 h-10 bg-white border-2 border-black overflow-hidden hover:bg-zinc-50 transition-colors">
                   {/* Speed lines background */}
                   <div className="absolute inset-0 opacity-10 bg-[repeating-linear-gradient(90deg,transparent,transparent_2px,#000_2px,#000_4px)]" style={{ transform: 'skewX(-20deg)' }} />
                   
                   <div className="relative z-10 flex items-center gap-2">
                      <div className="bg-black text-white px-1.5 py-0.5 text-[10px] font-black italic transform -rotate-6 shadow-sm border border-white">
                         GIT!
                      </div>
                      <span className="font-black italic text-xl tracking-tighter">{stars?.toLocaleString()}</span>
                   </div>
                   
                   {/* Speech bubble tail */}
                   <div className="absolute -bottom-1 right-4 w-2 h-2 bg-white border-r-2 border-b-2 border-black transform rotate-45" />
                </a>
              </div>

              {/* Variant: Virtual Streamer (Live) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">VTuber Live</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-white rounded-md border border-purple-200 shadow-sm overflow-hidden min-w-[130px] hover:border-purple-400 transition-colors">
                   <div className="flex items-center px-2 bg-red-500 text-white h-full text-[10px] font-bold uppercase gap-1 animate-pulse">
                      <div className="w-1.5 h-1.5 bg-white rounded-full" />
                      LIVE
                   </div>
                   <div className="flex-1 flex items-center justify-between px-3">
                      <span className="text-xs font-medium text-purple-900">Fans</span>
                      <span className="text-xs font-mono font-bold text-purple-600">{stars?.toLocaleString()}</span>
                   </div>
                   {/* Chat overlay effect */}
                   <div className="absolute inset-0 bg-gradient-to-t from-purple-500/5 to-transparent pointer-events-none" />
                </a>
              </div>

              {/* Variant: TCG Card */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">TCG Rarity</span>
                <a href="https://github.com" className="group relative flex items-center h-10 w-32 bg-amber-50 border-2 border-[#D4AF37] rounded-sm shadow-sm hover:-translate-y-1 transition-transform">
                   {/* Holographic foil effect simulation */}
                   <div className="absolute inset-0 bg-gradient-to-tr from-transparent via-white/40 to-transparent opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none" />
                   
                   <div className="absolute -top-2.5 left-1/2 -translate-x-1/2 bg-[#D4AF37] text-white text-[8px] font-bold px-2 rounded-full border border-amber-100">
                      LEGENDARY
                   </div>
                   
                   <div className="flex w-full items-center justify-center gap-2">
                      <IconStarFilled className="size-4 text-amber-500 drop-shadow-sm" />
                      <span className="font-serif font-bold text-amber-900 text-sm">{stars?.toLocaleString()}</span>
                   </div>
                   
                   {/* Card corners */}
                   <div className="absolute bottom-0.5 right-0.5 text-[6px] font-mono text-amber-800/50 px-1">#001</div>
                </a>
              </div>

              {/* Variant: Pixel RPG UI */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">RPG Menu</span>
                <a href="https://github.com" className="group relative flex items-center h-9 bg-[#E6D6BF] border-2 border-[#5C4033] shadow-[2px_2px_0px_#5C4033] active:translate-y-[2px] active:shadow-none active:translate-x-[2px] transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] rounded-[2px] min-w-[130px]">
                   <div className="absolute inset-0 border border-[#C0A080] m-[2px]" />
                   
                   <div className="relative z-10 flex w-full items-center justify-between px-3">
                      <span className="text-xs font-bold text-[#5C4033] font-serif">EXP</span>
                      <div className="flex items-center gap-1 bg-[#5C4033] px-1.5 py-0.5 rounded-[1px]">
                         <IconStarFilled className="size-2 text-yellow-400" />
                         <span className="text-[10px] font-mono text-white">{stars?.toLocaleString()}</span>
                      </div>
                   </div>
                </a>
              </div>

            </div>
          </div>
          <div className="space-y-4">
            <h2 className="text-lg font-semibold">11. Final Recommendations (Refined & Clear)</h2>
            <p className="text-sm text-muted-foreground">
              Based on your feedback: Balanced, low-noise, clear &quot;Star&quot; label, consistent with the app theme.
            </p>
            
            <div className="flex flex-wrap items-center gap-8 rounded-lg border border-dashed p-6">
              
              {/* Recommendation A: The "Modern SaaS" (Clean White) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Modern SaaS</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 gap-0 p-0 overflow-hidden bg-white hover:bg-zinc-50 border-zinc-200 text-zinc-800 shadow-sm"
                  asChild
                >
                  <a href="https://github.com" className="flex items-center group">
                    <div className="flex items-center gap-2 px-3 py-1">
                      <IconBrandGithub className="size-3.5 text-zinc-500 group-hover:text-zinc-900 transition-colors" />
                      <span className="text-xs font-medium text-zinc-600 group-hover:text-zinc-900 transition-colors">Star</span>
                    </div>
                    <div className="flex h-full items-center border-l border-zinc-100 bg-zinc-50 px-2.5">
                      <span className="text-xs font-mono tabular-nums text-zinc-500 group-hover:text-zinc-900 transition-colors">
                        {stars?.toLocaleString() ?? "…"}
                      </span>
                    </div>
                  </a>
                </Button>
              </div>

              {/* Recommendation B: The "Dev Tool" (Light Gray/Tech) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Dev Tool (Light)</span>
                <a href="https://github.com" className="group flex items-center gap-2 h-8 px-3 rounded-md border border-zinc-200 bg-white hover:border-zinc-300 hover:shadow-sm transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]">
                   <IconBrandGithub className="size-4 text-zinc-500 group-hover:text-zinc-900 transition-colors" />
                   <span className="text-xs font-medium text-zinc-600 group-hover:text-zinc-900 transition-colors">Star on GitHub</span>
                   <div className="flex items-center gap-1 ml-1 pl-2 border-l border-zinc-100">
                      <IconStarFilled className="size-3 text-zinc-300 group-hover:text-amber-400 transition-colors" />
                      <span className="text-xs font-mono text-zinc-500 group-hover:text-zinc-900 transition-colors">
                        {stars?.toLocaleString()}
                      </span>
                   </div>
                </a>
              </div>

              {/* Recommendation C: The "Focus" (Bright Accent) - FIXED from Dark */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Accent Focus</span>
                <a href="https://github.com" className="group relative flex items-center h-8 bg-white border border-zinc-200 text-zinc-800 rounded-md shadow-sm overflow-hidden pr-3 hover:border-zinc-300 hover:shadow-md transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300">
                   <div className="flex items-center justify-center w-8 h-full bg-zinc-50 border-r border-zinc-100 group-hover:bg-zinc-100 transition-colors">
                      <IconBrandGithub className="size-4 text-zinc-600 group-hover:text-black" />
                   </div>
                   <div className="flex items-center gap-2 px-2">
                      <span className="text-xs font-semibold text-zinc-700 group-hover:text-black">Star</span>
                      <div className="w-[1px] h-3 bg-zinc-200" />
                      <div className="flex items-center gap-1">
                         <span className="text-xs font-mono font-medium text-zinc-600 group-hover:text-black">{stars?.toLocaleString()}</span>
                         {/* Hidden star that appears on hover */}
                         <IconStarFilled className="w-0 overflow-hidden group-hover:w-3 size-3 text-amber-400 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-300" />
                      </div>
                   </div>
                </a>
              </div>

              {/* Recommendation D: The "Minimal Integrated" (Pure White) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Integrated</span>
                <div className="flex items-center p-0.5 rounded-full border border-zinc-200 bg-zinc-50/50 shadow-sm">
                   <a href="https://github.com" className="flex items-center gap-2 px-3 py-1 rounded-full bg-white border border-zinc-100 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] duration-200">
                      <span className="text-xs font-semibold text-zinc-700">Lunafox</span>
                      <span className="text-zinc-300">/</span>
                      <div className="flex items-center gap-1.5">
                         <IconBrandGithub className="size-3.5 text-zinc-500" />
                         <span className="text-xs font-medium text-zinc-800">Star</span>
                      </div>
                   </a>
                   <div className="px-2.5 py-1 text-xs font-mono text-zinc-500">
                      {stars?.toLocaleString()}
                   </div>
                </div>
              </div>

              {/* New Recommendation E: Soft Industrial (Light) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Soft Industrial</span>
                <a href="https://github.com" className="group flex items-center h-8 bg-[#F4F4F5] border border-[#E4E4E7] rounded-sm hover:bg-white hover:border-[#D4D4D8] transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]">
                   <div className="flex items-center justify-center h-full px-2.5 bg-white border-r border-[#E4E4E7] group-hover:border-[#D4D4D8]">
                      <IconBrandGithub className="size-3.5 text-zinc-400 group-hover:text-zinc-800 transition-colors" />
                   </div>
                   <div className="flex items-center px-3 gap-2">
                      <span className="text-xs font-bold text-zinc-500 uppercase tracking-wider group-hover:text-zinc-800 transition-colors">Star</span>
                      <span className="text-[10px] font-mono text-zinc-400 bg-zinc-100 px-1 rounded-sm group-hover:bg-zinc-200 group-hover:text-zinc-600 transition-colors">
                        {stars?.toLocaleString()}
                      </span>
                   </div>
                </a>
              </div>

            </div>
          </div>

          <div className="space-y-4">
            <h2 className="text-lg font-semibold">12. Optimization Candidates (Refined Current Style)</h2>
            <p className="text-sm text-muted-foreground">
              Variations based on the current implementation but addressing the &quot;subtly off&quot; feeling.
            </p>
            
            <div className="flex flex-wrap items-center gap-8 rounded-lg border border-dashed p-6">
              
              {/* Option 1: Cleaned up borders & proportions */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Better Proportions</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 gap-0 p-0 overflow-hidden bg-background hover:bg-muted/50 transition-colors"
                  asChild
                >
                  <a href="https://github.com" className="flex items-center group">
                    <div className="flex items-center gap-2 px-3">
                      <IconBrandGithub className="size-4 text-foreground/80" />
                      <span className="text-xs font-medium text-foreground/80">Star</span>
                    </div>
                    {/* Removed bg-muted/50 to make it cleaner, just a border separator */}
                    <div className="flex h-full items-center border-l px-3 bg-muted/20 group-hover:bg-muted/40 transition-colors">
                      <span className="text-xs font-mono tabular-nums text-foreground/70 group-hover:text-foreground transition-colors">
                        {stars?.toLocaleString() ?? "…"}
                      </span>
                      {/* Changed star to be more subtle/integrated */}
                      <IconStarFilled className="ml-1.5 size-3 text-amber-400 opacity-80 group-hover:opacity-100 group-hover:scale-110 transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top]" />
                    </div>
                  </a>
                </Button>
              </div>

              {/* Option 2: Softer pill shape (Modern standard) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Softer / Pill</span>
                <a href="https://github.com" className="group flex items-center h-8 rounded-full border bg-background hover:border-foreground/20 hover:shadow-sm transition-[background-color,border-color,color,opacity,box-shadow,transform,width,height,top] overflow-hidden">
                   <div className="flex items-center gap-2 px-3 py-1">
                      <IconBrandGithub className="size-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                      <span className="text-xs font-medium text-muted-foreground group-hover:text-foreground transition-colors">Star</span>
                   </div>
                   <div className="h-4 w-[1px] bg-border" />
                   <div className="flex items-center gap-1.5 px-3 py-1 bg-muted/30 h-full group-hover:bg-muted/50 transition-colors">
                      <span className="text-xs font-mono text-muted-foreground group-hover:text-foreground transition-colors">{stars?.toLocaleString()}</span>
                      <IconStarFilled className="size-3 text-amber-400/90" />
                   </div>
                </a>
              </div>

              {/* Option 3: Unified Surface (No internal divider background) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Unified Surface</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 gap-2 px-3 bg-background hover:bg-accent hover:text-accent-foreground border-input shadow-sm"
                  asChild
                >
                  <a href="https://github.com" className="flex items-center">
                    <IconBrandGithub className="size-4 text-muted-foreground" />
                    <span className="text-xs font-medium">Star</span>
                    <div className="flex items-center gap-1 ml-1 text-muted-foreground">
                       <span>•</span>
                       <span className="font-mono text-xs tabular-nums">{stars?.toLocaleString()}</span>
                    </div>
                  </a>
                </Button>
              </div>

              {/* Option 4: "Badge" Style (Star count as badge) */}
              <div className="flex flex-col items-center gap-2">
                <span className="text-xs text-muted-foreground">Count Badge</span>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 gap-2 px-2 hover:bg-muted/60"
                  asChild
                >
                  <a href="https://github.com" className="flex items-center">
                    <div className="flex items-center justify-center p-1 rounded-md bg-white border shadow-sm">
                       <IconBrandGithub className="size-3.5" />
                    </div>
                    <span className="text-xs font-medium">Star</span>
                    <span className="flex items-center justify-center h-5 min-w-5 px-1.5 ml-0.5 rounded-full bg-muted text-[10px] font-mono font-medium text-muted-foreground">
                       {stars?.toLocaleString()}
                    </span>
                  </a>
                </Button>
              </div>

            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
