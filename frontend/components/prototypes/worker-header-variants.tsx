"use client"

import {
    IconServer,
    IconPlus,
} from "@/components/icons"
import { Button } from "@/components/ui/button"

export function WorkerHeaderVariants() {
    return (
        <div className="flex flex-col gap-16 pb-24">

            {/* Design 1: Current Actual Version */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">01</h2>
                    <h2 className="text-lg font-semibold">主界面版本 (Current Baseline)</h2>
                </div>
                <p className="text-sm text-muted-foreground">当前系统中正在使用的纯净版 LED 灯阵控制台（原方案 18）。自带自适应横向扫描光效底纹。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed">
                    <div className="rounded-xl border bg-card p-6 shadow-sm overflow-hidden relative">
                        {/* Decorative scanline adapted for light/dark mode */}
                        <div className="absolute inset-0 bg-[linear-gradient(transparent_50%,rgba(0,0,0,0.02)_50%)] dark:bg-[linear-gradient(transparent_50%,rgba(255,255,255,0.02)_50%)] bg-[length:100%_4px] pointer-events-none z-0"></div>

                        <div className="relative z-10 flex flex-col md:flex-row justify-between items-start md:items-center gap-8">
                            <div className="flex flex-col gap-2">
                                <h3 className="text-muted-foreground text-xs tracking-[0.2em] uppercase font-mono mb-2">Cluster Matrix_</h3>
                                <div className="flex items-end gap-6">
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-foreground">25</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">TOTAL</span>
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-emerald-600 dark:text-emerald-400 drop-shadow-[0_0_8px_rgba(52,211,153,0.3)]">23</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">ONLINE</span>
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-destructive drop-shadow-[0_0_8px_rgba(239,68,68,0.3)]">02</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">OFFLINE</span>
                                    </div>
                                </div>
                            </div>

                            <div className="flex-1 w-full max-w-sm md:px-6 md:border-l border-border/50">
                                <p className="text-[10px] text-muted-foreground font-mono mb-3 uppercase">Live Nodes Map (25)</p>
                                <div className="flex flex-wrap gap-1.5">
                                    {[...Array(23)].map((_, i) => (
                                        <div key={`tm-${i}`} className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_5px_rgba(16,185,129,0.5)]" />
                                    ))}
                                    {[...Array(2)].map((_, i) => (
                                        <div key={`tme-${i}`} className="w-2 h-2 rounded-full bg-destructive shadow-[0_0_6px_rgba(239,68,68,0.6)] animate-pulse" />
                                    ))}
                                </div>
                            </div>

                            <div className="flex flex-col sm:flex-row md:flex-col gap-3 shrink-0">
                                <Button size="sm" variant="outline" className="font-mono text-xs shadow-sm bg-background h-8">
                                    [ ARCHITECTURE ]
                                </Button>
                                <Button size="sm" className="font-mono text-xs shadow-sm bg-primary/90 hover:bg-primary h-8">
                                    &gt; REG_NODE
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>


            {/* Design 2: Holographic Glassmorphism */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">02</h2>
                    <h2 className="text-lg font-semibold">全息弥散光影 (Holographic Glow)</h2>
                </div>
                <p className="text-sm text-muted-foreground">在面板背后加入极具现代感的绿色/透明渐变光晕，并使背景变得具有毛玻璃半透明效果，数字仿佛漂浮在光源之上。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed relative overflow-hidden">
                    {/* Background glow decoration */}
                    <div className="absolute top-1/2 left-1/4 -translate-y-1/2 w-64 h-64 bg-emerald-500/20 dark:bg-emerald-500/10 rounded-full blur-[80px] pointer-events-none" />
                    <div className="absolute top-1/2 right-1/4 -translate-y-1/2 w-48 h-48 bg-emerald-300/10 dark:bg-emerald-300/5 rounded-full blur-[60px] pointer-events-none" />

                    <div className="rounded-xl border border-white/40 dark:border-white/10 bg-white/60 dark:bg-black/40 backdrop-blur-xl p-6 shadow-[0_8px_32px_rgba(0,0,0,0.05)] relative z-10 transition-all">
                        <div className="relative z-10 flex flex-col md:flex-row justify-between items-start md:items-center gap-8">
                            <div className="flex flex-col gap-2">
                                <h3 className="text-emerald-700 dark:text-emerald-400/80 text-xs tracking-[0.2em] uppercase font-mono mb-2 drop-shadow-sm">Cluster Matrix_</h3>
                                <div className="flex items-end gap-6">
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-foreground">25</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">TOTAL</span>
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-emerald-600 dark:text-emerald-400 drop-shadow-[0_0_15px_rgba(52,211,153,0.5)]">23</span>
                                        <span className="text-[10px] text-emerald-600/70 dark:text-emerald-400/70 font-mono tracking-widest uppercase">ONLINE</span>
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-destructive drop-shadow-[0_0_15px_rgba(239,68,68,0.5)]">02</span>
                                        <span className="text-[10px] text-destructive/70 font-mono tracking-widest uppercase">OFFLINE</span>
                                    </div>
                                </div>
                            </div>

                            <div className="flex-1 w-full max-w-sm md:px-6 md:border-l border-emerald-500/20">
                                <p className="text-[10px] text-muted-foreground font-mono mb-3 uppercase">Live Nodes Map (25)</p>
                                <div className="flex flex-wrap gap-1.5">
                                    {[...Array(23)].map((_, i) => (
                                        <div key={`tm-${i}`} className="w-2.5 h-2.5 rounded-full bg-emerald-500/80 backdrop-blur-md shadow-[inset_0_1px_2px_rgba(255,255,255,0.4),0_0_8px_rgba(16,185,129,0.4)]" />
                                    ))}
                                    {[...Array(2)].map((_, i) => (
                                        <div key={`tme-${i}`} className="w-2.5 h-2.5 rounded-full bg-destructive/80 backdrop-blur-md shadow-[inset_0_1px_2px_rgba(255,255,255,0.4),0_0_8px_rgba(239,68,68,0.4)] animate-pulse" />
                                    ))}
                                </div>
                            </div>

                            <div className="flex flex-col sm:flex-row md:flex-col gap-3 shrink-0">
                                <Button size="sm" variant="outline" className="font-mono text-xs shadow-sm bg-white/50 dark:bg-black/50 backdrop-blur-sm border-white/50 dark:border-white/20 h-8 hover:bg-white/80 dark:hover:bg-white/10">
                                    [ ARCHITECTURE ]
                                </Button>
                                <Button size="sm" className="font-mono text-xs shadow-lg shadow-emerald-500/20 bg-emerald-600 hover:bg-emerald-500 text-white border-0 h-8">
                                    &gt; REG_NODE
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>


            {/* Design 3: Technical Blueprint (Grid Background) */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">03</h2>
                    <h2 className="text-lg font-semibold">科幻网格解析版 (Cyber Grid Blueprint)</h2>
                </div>
                <p className="text-sm text-muted-foreground">面板底图加入强烈的“蓝图网格图”要素。面板整体拥有非常微弱的绿色发光边缘，极大强化“科技分析”的感觉。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed relative">
                    <div className="rounded-xl border border-emerald-500/30 bg-card p-6 shadow-[0_0_15px_rgba(16,185,129,0.05)_inset,0_0_20px_rgba(16,185,129,0.05)] overflow-hidden relative">
                        {/* Blueprint Grid Background Pattern */}
                        <div className="absolute inset-0 bg-[linear-gradient(rgba(16,185,129,0.05)_1px,transparent_1px),linear-gradient(90deg,rgba(16,185,129,0.05)_1px,transparent_1px)] bg-[size:16px_16px] pointer-events-none z-0" />
                        <div className="absolute inset-0 bg-[linear-gradient(rgba(16,185,129,0.02)_1px,transparent_1px),linear-gradient(90deg,rgba(16,185,129,0.02)_1px,transparent_1px)] bg-[size:64px_64px] pointer-events-none z-0" />

                        <div className="relative z-10 flex flex-col md:flex-row justify-between items-start md:items-center gap-8">
                            <div className="flex flex-col gap-2">
                                <div className="flex items-center gap-2 mb-2">
                                    <div className="w-1 h-3 bg-emerald-500" />
                                    <h3 className="text-muted-foreground text-xs tracking-[0.2em] font-mono">SYS_MATRIX / 001</h3>
                                </div>
                                <div className="flex items-end gap-6 pl-3">
                                    <div className="flex flex-col relative">
                                        <div className="absolute -left-3 top-2 w-[1px] h-full bg-border/50" />
                                        <span className="text-4xl font-mono font-light tracking-tighter text-foreground">25</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">TOTAL</span>
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-emerald-600 dark:text-emerald-400 drop-shadow-[0_0_8px_rgba(52,211,153,0.3)]">23</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">ONLINE</span>
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-destructive drop-shadow-[0_0_8px_rgba(239,68,68,0.3)]">02</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">OFFLINE</span>
                                    </div>
                                </div>
                            </div>

                            <div className="flex-1 w-full max-w-sm md:px-6 relative">
                                <div className="absolute top-0 bottom-0 left-0 w-[1px] bg-gradient-to-b from-transparent via-emerald-500/30 to-transparent" />
                                <div className="absolute top-0 bottom-0 right-0 w-[1px] bg-gradient-to-b from-transparent via-emerald-500/30 to-transparent" />
                                <p className="text-[10px] text-muted-foreground font-mono mb-3 uppercase px-2"><span className="text-emerald-500 animate-pulse">●</span> RELAYMAP_L</p>
                                <div className="flex flex-wrap gap-1.5 px-2">
                                    {[...Array(23)].map((_, i) => (
                                        <div key={`tm-${i}`} className="w-2.5 h-2.5 bg-emerald-500 shadow-[0_0_5px_rgba(16,185,129,0.5)]" style={{ clipPath: 'polygon(50% 0%, 100% 25%, 100% 75%, 50% 100%, 0% 75%, 0% 25%)' }} />
                                    ))}
                                    {[...Array(2)].map((_, i) => (
                                        <div key={`tme-${i}`} className="w-2.5 h-2.5 bg-destructive shadow-[0_0_6px_rgba(239,68,68,0.6)] animate-pulse" style={{ clipPath: 'polygon(50% 0%, 100% 25%, 100% 75%, 50% 100%, 0% 75%, 0% 25%)' }} />
                                    ))}
                                </div>
                            </div>

                            <div className="flex flex-col sm:flex-row md:flex-col gap-3 shrink-0">
                                <Button size="sm" variant="outline" className="font-mono text-xs shadow-none bg-transparent border-emerald-500/30 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/10 h-8">
                                    [ ARCHITECTURE ]
                                </Button>
                                <Button size="sm" className="font-mono text-xs shadow-none bg-emerald-600 hover:bg-emerald-500 text-white rounded-none h-8 relative overflow-hidden group">
                                    <div className="absolute inset-0 bg-[linear-gradient(90deg,transparent_0%,rgba(255,255,255,0.2)_50%,transparent_100%)] -translate-x-[150%] group-hover:animate-[shimmer_1.5s_infinite]" />
                                    &gt; REG_NODE
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>


            {/* Design 4: Topo Lines / Dotted Wave */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">04</h2>
                    <h2 className="text-lg font-semibold">浮点波纹等高线 (Dotted Topo Map)</h2>
                </div>
                <p className="text-sm text-muted-foreground">将背景换成科技系统里常出现的圆点阵列 / 等高纹理，让纯净的面板下多了一层数据漂浮的感觉。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed relative">
                    <div className="rounded-xl border border-border/80 bg-card p-6 shadow-md overflow-hidden relative">
                        {/* Dotted background pattern */}
                        <div className="absolute inset-0 opacity-20 dark:opacity-10 pointer-events-none z-0" style={{ backgroundImage: 'radial-gradient(circle at 2px 2px, currentColor 1px, transparent 0)', backgroundSize: '16px 16px' }} />

                        <div className="relative z-10 flex flex-col md:flex-row justify-between items-start md:items-center gap-8">
                            <div className="flex flex-col gap-2">
                                <h3 className="text-foreground font-bold text-xs tracking-[0.2em] uppercase font-mono mb-2">Cluster Matrix_</h3>
                                <div className="flex items-end gap-6 bg-background/80 p-3 rounded-lg border backdrop-blur-sm -ml-3">
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-foreground">25</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">TOTAL</span>
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-emerald-600 dark:text-emerald-400 drop-shadow-[0_0_8px_rgba(52,211,153,0.3)]">23</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">ONLINE</span>
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-4xl font-mono font-light tracking-tighter text-destructive drop-shadow-[0_0_8px_rgba(239,68,68,0.3)]">02</span>
                                        <span className="text-[10px] text-muted-foreground font-mono tracking-widest uppercase">OFFLINE</span>
                                    </div>
                                </div>
                            </div>

                            <div className="flex-1 w-full max-w-sm">
                                <div className="bg-background/80 p-4 rounded-lg border backdrop-blur-sm h-full">
                                    <p className="text-[10px] text-foreground font-bold font-mono mb-3 uppercase border-b pb-2">Live Nodes Map (25)</p>
                                    <div className="flex flex-wrap gap-1.5 mt-2">
                                        {[...Array(23)].map((_, i) => (
                                            <div key={`tm-${i}`} className="w-2 h-2 rounded-[2px] bg-emerald-500 shadow-[0_0_5px_rgba(16,185,129,0.5)]" />
                                        ))}
                                        {[...Array(2)].map((_, i) => (
                                            <div key={`tme-${i}`} className="w-2 h-2 rounded-[2px] bg-destructive shadow-[0_0_6px_rgba(239,68,68,0.6)] animate-pulse" />
                                        ))}
                                    </div>
                                </div>
                            </div>

                            <div className="flex flex-col sm:flex-row md:flex-col gap-3 shrink-0">
                                <Button size="sm" variant="outline" className="font-mono text-xs shadow-sm bg-background/80 backdrop-blur border h-8 font-bold">
                                    [ ARCHITECTURE ]
                                </Button>
                                <Button size="sm" className="font-mono text-xs shadow-sm bg-foreground text-background h-8 font-bold">
                                    &gt; REG_NODE
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Design 5: NieR Automata Style (Theme-Aware) */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">05</h2>
                    <h2 className="text-lg font-semibold">尤尔哈系统 - 融合版 (YoRHa Theme-Aware)</h2>
                </div>
                <p className="text-sm text-muted-foreground">优化了配色的《尼尔：机械纪元》风格。全面重构了底色，直接使用系统原生的卡片面板 (bg-card / border) 与默认文字色 (foreground/muted-foreground)，让排版与您的项目更加浑然天成。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed">
                    {/* YoRHa Wrapper - Native Theme Adapter */}
                    <div className="bg-card border border-border/50 rounded-xl p-6 relative overflow-hidden h-full uppercase font-mono tracking-widest text-foreground shadow-sm">
                        {/* Thin outline border referencing NieR, but using theme border */}
                        <div className="absolute inset-2 border border-border/60 pointer-events-none rounded-lg" />
                        {/* Decorative pattern using muted color */}
                        <div className="absolute top-0 right-0 w-32 h-full bg-[radial-gradient(var(--border)_1px,transparent_1px)] bg-[size:10px_10px] opacity-20" />

                        <div className="relative z-10 flex flex-col md:flex-row justify-between items-start md:items-center gap-8">

                            <div className="flex flex-col gap-0 border-l-4 border-primary pl-4">
                                <span className="text-[10px] font-bold text-muted-foreground">SYSTEM // DATA:</span>
                                <h3 className="text-foreground text-2xl tracking-[0.3em] font-light">CLUSTER_MAP</h3>
                            </div>

                            <div className="flex flex-1 gap-8 md:px-8 md:border-l border-border/60 w-full">
                                <div className="flex flex-col">
                                    <span className="text-[9px] mb-1 font-bold text-muted-foreground">ALL_UNITS</span>
                                    <span className="text-3xl font-light">25</span>
                                </div>
                                <div className="flex flex-col">
                                    <span className="text-[9px] mb-1 font-bold text-muted-foreground">ACTIVE</span>
                                    <span className="text-3xl font-light text-emerald-600 dark:text-emerald-400">23</span>
                                </div>
                                <div className="flex flex-col">
                                    <span className="text-[9px] mb-1 font-bold text-muted-foreground">LOST_SIGNAL</span>
                                    <span className="text-3xl font-light text-destructive animate-pulse">02</span>
                                </div>
                            </div>

                            <div className="flex flex-col gap-2 shrink-0">
                                <button className="bg-primary text-primary-foreground px-6 py-2 text-xs hover:bg-primary/90 transition-colors flex items-center justify-between min-w-[160px]">
                                    <span>INSTALL</span>
                                    <span className="opacity-50 text-[10px]">01</span>
                                </button>
                                <button className="border border-border text-foreground px-6 py-2 text-xs hover:bg-muted transition-colors flex items-center justify-between min-w-[160px]">
                                    <span>TOPOLOGY</span>
                                    <span className="opacity-50 text-[10px]">02</span>
                                </button>
                            </div>
                        </div>

                        {/* Node Bar aligned to bottom */}
                        <div className="mt-8 relative z-10 flex items-center gap-2 border-t border-border/60 pt-4">
                            <span className="text-[10px] font-bold text-muted-foreground">STATUS_ARRAY</span>
                            <div className="flex-1 h-2 flex gap-[2px]">
                                {[...Array(23)].map((_, i) => (
                                    <div key={`yr-${i}`} className="h-full flex-1 bg-emerald-500/80 border-r border-background" />
                                ))}
                                {[...Array(2)].map((_, i) => (
                                    <div key={`yre-${i}`} className="h-full flex-1 bg-destructive/80 border-r border-background" />
                                ))}
                            </div>
                        </div>
                    </div>
                </div>
            </div>


            {/* Design 6: Arknights Endfield / Industrial Brutalism (Theme-Aware) */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">06</h2>
                    <h2 className="text-lg font-semibold">终末地重工 - 融合版 (Endfield Theme-Aware)</h2>
                </div>
                <p className="text-sm text-muted-foreground">《明日方舟：终末地》风格的自适应版。斜切角和重工业排版得以保留，但原本的黄色施工色带和纯黑背景，被替换为系统的主题强调色 (Primary) 和卡片底色 (Card)，与全局UI完美结合不突兀。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed">
                    {/* Endfield Wrapper - Theme Aware */}
                    <div className="bg-card p-6 relative uppercase font-sans tracking-wide text-foreground border-b-4 border-primary shadow-sm overflow-hidden">
                        {/* Angled background decor */}
                        <div className="absolute top-0 right-0 w-[40%] h-full bg-muted/50" style={{ clipPath: 'polygon(20% 0, 100% 0, 100% 100%, 0 100%)' }} />
                        <div className="absolute top-0 right-0 w-2 h-full bg-primary/20" style={{ clipPath: 'polygon(100% 0, 100% 0, 100% 100%, 0 100%)' }} />

                        <div className="relative z-10 flex flex-col md:flex-row justify-between items-start md:items-center gap-8">

                            <div className="flex flex-col gap-1 w-48">
                                <div className="flex items-center gap-2 mb-2">
                                    <div className="w-4 h-4 bg-primary flex items-center justify-center text-primary-foreground font-bold text-[10px]">C</div>
                                    <span className="text-xs text-primary font-bold tracking-widest">CLUSTER.DEV</span>
                                </div>
                                <h3 className="text-3xl font-black italic tracking-tighter">NODE_FACILITY</h3>
                            </div>

                            <div className="flex flex-col gap-3 flex-1">
                                <div className="flex items-end gap-2">
                                    <span className="text-sm text-muted-foreground bg-muted px-2 py-0.5 rounded-sm">RSR_LIMIT</span>
                                    <span className="text-4xl font-black italic text-foreground">25</span>
                                </div>

                                {/* Slanted Progress / Node Bar */}
                                <div className="h-6 w-full max-w-sm bg-muted/50 border border-border flex p-1 gap-1" style={{ transform: 'skewX(-10deg)' }}>
                                    {[...Array(23)].map((_, i) => (
                                        <div key={`ef-${i}`} className="flex-1 bg-primary" />
                                    ))}
                                    {[...Array(2)].map((_, i) => (
                                        <div key={`efe-${i}`} className="flex-1 bg-destructive animate-pulse" />
                                    ))}
                                </div>
                                <div className="flex justify-between w-full max-w-sm text-[10px] text-muted-foreground font-bold mt-1">
                                    <span>23 ONLINE (OK)</span>
                                    <span className="text-destructive">02 ERR</span>
                                </div>
                            </div>

                            <div className="flex flex-col sm:flex-row gap-3 shrink-0">
                                <button className="bg-muted text-foreground border border-border px-6 py-3 font-bold text-xs hover:bg-secondary transition-colors uppercase italic" style={{ transform: 'skewX(-10deg)' }}>
                                    <span className="block" style={{ transform: 'skewX(10deg)' }}>Read_Topo</span>
                                </button>
                                <button className="bg-primary text-primary-foreground px-6 py-3 font-black text-xs hover:bg-primary/90 transition-colors uppercase italic shadow-[3px_3px_0_rgba(0,0,0,0.2)] dark:shadow-none" style={{ transform: 'skewX(-10deg)' }}>
                                    <span className="block" style={{ transform: 'skewX(10deg)' }}>Install // Node</span>
                                </button>
                            </div>
                        </div>

                        {/* Caution Strip - using theme colors */}
                        <div className="absolute top-0 left-0 w-32 h-1 bg-[repeating-linear-gradient(45deg,theme(colors.primary.DEFAULT),theme(colors.primary.DEFAULT)_5px,theme(colors.card.DEFAULT)_5px,theme(colors.card.DEFAULT)_10px)]" />
                    </div>
                </div>
            </div>

            {/* Design 7: CRT Terminal (Theme-Aware) */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">07</h2>
                    <h2 className="text-lg font-semibold">主题指令终端 (Terminal Theme-Aware)</h2>
                </div>
                <p className="text-sm text-muted-foreground">保留纯文字和矩阵转储排版的代码命令行感觉。荧光绿变成了系统对应状态的主题色，背景为纯黑（黑客原汁原味）或跟随卡片颜色。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed">
                    <div className="bg-zinc-950 p-6 rounded-lg font-mono text-primary border-2 border-primary/20 shadow-[inset_0_0_20px_rgba(0,0,0,0.5)] relative">
                        {/* CRT Scanline Overlay */}
                        <div className="absolute inset-0 bg-[linear-gradient(transparent_50%,rgba(0,0,0,0.25)_50%)] bg-[length:100%_4px] pointer-events-none" />

                        <div className="relative z-10">
                            <div className="mb-4">
                                <p className="text-sm mb-1">{`> CONNECTED TO CLUSTER_MAINFRAME`}</p>
                                <p className="text-[10px] opacity-70 mb-4">{`> SECURE CONNECTION ESTABLISHED... FETCHING METRICS...`}</p>
                            </div>

                            <div className="flex flex-col md:flex-row justify-between gap-8 mb-6">
                                <div className="space-y-2">
                                    <div className="flex gap-4">
                                        <span className="w-24 opacity-70">SYS_TOTAL</span>
                                        <span className="font-bold">25</span>
                                    </div>
                                    <div className="flex gap-4">
                                        <span className="w-24 opacity-70">SYS_ONLINE</span>
                                        <span className="font-bold relative">
                                            <span className="absolute inset-0 bg-primary opacity-50 blur-md"></span>
                                            <span className="relative z-10">23</span>
                                        </span>
                                    </div>
                                    <div className="flex gap-4 text-destructive">
                                        <span className="w-24 opacity-70">SYS_ERROR</span>
                                        <span className="font-bold relative animate-pulse">
                                            <span className="absolute inset-0 bg-destructive opacity-50 blur-md"></span>
                                            <span className="relative z-10">02</span>
                                        </span>
                                    </div>
                                </div>

                                <div className="flex-1 w-full max-w-md">
                                    <p className="mb-2 opacity-70">NODE_ARRAY_DUMP:</p>
                                    <div className="flex flex-wrap gap-1">
                                        {[...Array(23)].map((_, i) => (
                                            <div key={`crt-${i}`} className="w-4 h-4 border border-primary/50 bg-primary/20 flex items-center justify-center">
                                                <span className="text-[10px]">1</span>
                                            </div>
                                        ))}
                                        {[...Array(2)].map((_, i) => (
                                            <div key={`crte-${i}`} className="w-4 h-4 border border-destructive/50 bg-destructive/20 text-destructive flex items-center justify-center animate-pulse">
                                                <span className="text-[10px]">0</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            </div>

                            <div className="flex gap-4 text-sm mt-4 pt-4 border-t border-primary/20 border-dashed">
                                <button className="hover:bg-primary hover:text-primary-foreground transition-colors px-2 py-1">
                                    {`[1] REG_NEW_NODE`}
                                </button>
                                <button className="hover:bg-primary hover:text-primary-foreground transition-colors px-2 py-1">
                                    {`[2] RUN_DIAGNOSTICS`}
                                </button>
                                <span className="animate-pulse px-2 py-1">_</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Design 8: Cyberpunk Edgerunner (Theme-Aware) */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">08</h2>
                    <h2 className="text-lg font-semibold">夜之城骇客 (Cyberpunk Edgerunner)</h2>
                </div>
                <p className="text-sm text-muted-foreground">充满力量感的高反差区块，不规则的切角、断裂的线条与强烈的错位阴影。所有颜色都绑定了系统的主题色和对比色，在任何主题下都具备强烈的赛博精神病既视感。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed">
                    <div className="bg-card relative overflow-hidden font-sans uppercase">
                        {/* Background Glitch Elements */}
                        <div className="absolute top-0 right-10 w-32 h-64 bg-primary/20" style={{ clipPath: 'polygon(0% 0%, 100% 0%, 100% 30%, 80% 35%, 100% 40%, 100% 100%, 0% 100%, 0% 80%, 20% 75%, 0% 70%)' }}></div>
                        <div className="absolute top-1/2 left-1/4 w-full h-[2px] bg-primary/50 mix-blend-difference z-20"></div>

                        <div className="relative z-10 flex flex-col md:flex-row shadow-[4px_4px_0_theme(colors.primary.DEFAULT)] border-2 border-foreground bg-background dark:bg-card">

                            {/* Left Box (Title) */}
                            <div className="bg-primary text-primary-foreground p-6 flex flex-col justify-between min-w-[200px] shadow-[inset_-10px_0_0_theme(colors.background)] overflow-hidden relative">
                                <div className="absolute -left-4 top-1/2 w-8 h-8 bg-background rotate-45"></div>
                                <div>
                                    <span className="text-[10px] font-black tracking-widest block mb-1">SYS.OP: 2077</span>
                                    <h3 className="text-3xl font-black italic tracking-tighter mix-blend-overlay">M4TRIX</h3>
                                </div>
                                <div className="mt-8 text-[10px] font-bold">
                                    [ CONNECTION. OK ]
                                </div>
                            </div>

                            {/* Middle Box (Data) */}
                            <div className="p-6 flex-1 flex flex-col justify-center border-l-0 md:border-l-2 border-foreground relative">
                                <div className="flex items-end gap-x-8 gap-y-4 flex-wrap">
                                    <div className="relative group">
                                        <span className="absolute -left-2 -top-1 text-[8px] font-black text-muted-foreground">SYS/TTL</span>
                                        <span className="text-5xl font-black italic text-foreground group-hover:text-primary transition-colors">25</span>
                                    </div>
                                    <div className="relative pl-6 before:absolute before:left-0 before:top-1/4 before:h-1/2 before:w-1 before:bg-primary">
                                        <span className="absolute left-6 -top-1 text-[8px] font-black text-primary">ACTV.NODE</span>
                                        <span className="text-4xl font-black italic text-foreground">23</span>
                                    </div>
                                    <div className="relative pl-6 before:absolute before:left-0 before:top-1/4 before:h-1/2 before:w-1 before:bg-destructive before:animate-bounce">
                                        <span className="absolute left-6 -top-1 text-[8px] font-black text-destructive">ERR.FATAL</span>
                                        <span className="text-4xl font-black italic text-destructive drop-shadow-[2px_2px_0_rgba(255,0,0,0.5)]">02</span>
                                    </div>
                                </div>

                                <div className="mt-6 flex flex-wrap gap-1 w-full max-w-sm">
                                    {[...Array(23)].map((_, i) => (
                                        <div key={`cp-${i}`} className="w-3 h-3 bg-primary" style={{ clipPath: 'polygon(20% 0, 100% 0, 80% 100%, 0% 100%)' }} />
                                    ))}
                                    {[...Array(2)].map((_, i) => (
                                        <div key={`cpe-${i}`} className="w-3 h-3 bg-destructive animate-pulse" style={{ clipPath: 'polygon(20% 0, 100% 0, 80% 100%, 0% 100%)' }} />
                                    ))}
                                </div>
                            </div>

                            {/* Right Box (Actions) */}
                            <div className="p-6 flex flex-col justify-center gap-3 bg-muted/50 border-t-2 md:border-t-0 md:border-l-2 border-foreground min-w-[200px]">
                                <button className="relative w-full bg-transparent border-2 border-foreground text-foreground px-4 py-2 font-black italic text-sm hover:bg-foreground hover:text-background transition-all overflow-hidden group">
                                    <span className="absolute inset-0 bg-primary translate-y-full group-hover:translate-y-0 transition-transform -z-10"></span>
                                    SYS.DIAGNOSTICS
                                </button>
                                <button className="relative w-full bg-primary text-primary-foreground px-4 py-2 font-black italic text-lg shadow-[4px_4px_0_theme(colors.foreground)] hover:translate-x-1 hover:translate-y-1 hover:shadow-none transition-all">
                                    REG_NODE
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Design 9: Persona 5 Phantom Style */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">09</h2>
                    <h2 className="text-lg font-semibold">异闻怪盗 (Phantom Thieves / P5)</h2>
                </div>
                <p className="text-sm text-muted-foreground">《女神异闻录5》撕纸/星星拼贴风格。充满了叛逆的倾斜角度、黑白红（这里的红变为你的系统主题色）的极高色彩对撞，以及类似恐吓信剪报一样的随性张狂。</p>

                <div className="p-4 rounded-xl border bg-card/10 border-dashed overflow-hidden flex items-center justify-center py-12">
                    <div className="relative w-full max-w-4xl rotate-1">
                        {/* Background Star/Jagged blob */}
                        <div className="absolute -inset-4 bg-foreground -rotate-2" style={{ clipPath: 'polygon(2% 10%, 95% 5%, 100% 85%, 5% 95%)' }}></div>
                        <div className="absolute -inset-2 bg-primary rotate-1" style={{ clipPath: 'polygon(5% 5%, 98% 2%, 92% 98%, 2% 90%)' }}></div>

                        <div className="relative z-10 bg-card p-6 flex flex-col md:flex-row justify-between items-center gap-8 shadow-2xl -rotate-1" style={{ clipPath: 'polygon(0 0, 100% 2%, 98% 100%, 1% 98%)' }}>

                            {/* Title */}
                            <div className="relative flex flex-col rotate-3 shrink-0">
                                <span className="bg-foreground text-background font-black px-2 py-1 text-xs self-start -rotate-6 shadow-[2px_2px_0_theme(colors.primary.DEFAULT)]">CLUSTER</span>
                                <h3 className="text-background bg-primary text-4xl font-black italic px-4 py-2 shadow-[-4px_4px_0_theme(colors.foreground)] relative z-10">MATRIX</h3>
                            </div>

                            {/* Data Stars */}
                            <div className="flex gap-4 md:gap-8 flex-1 justify-center relative">
                                {/* Decor */}
                                <div className="absolute top-0 right-0 w-full h-full border-4 border-dashed border-muted-foreground/20 rounded-full scale-150 pointer-events-none -z-10 rotate-12"></div>

                                <div className="flex flex-col items-center bg-background border-4 border-foreground p-3 shadow-[4px_4px_0_theme(colors.foreground)] -rotate-3">
                                    <span className="bg-primary text-primary-foreground text-[10px] font-black px-2 -mt-6 rotate-2">TOTAL_CORES</span>
                                    <span className="text-4xl font-black text-foreground mt-2">25</span>
                                </div>

                                <div className="flex flex-col items-center bg-primary border-4 border-foreground p-3 shadow-[4px_4px_0_theme(colors.foreground)] rotate-2">
                                    <span className="bg-background text-foreground text-[10px] font-black px-2 -mt-6 -rotate-2">ACTIVE</span>
                                    <span className="text-4xl font-black text-primary-foreground mt-2">23</span>
                                </div>

                                <div className="flex flex-col items-center bg-background border-4 border-foreground p-3 shadow-[4px_4px_0_theme(colors.destructive.DEFAULT)] rotate-6">
                                    <span className="bg-destructive text-destructive-foreground text-[10px] font-black px-2 -mt-6 -rotate-6">DANGER</span>
                                    <span className="text-4xl font-black text-destructive mt-2 animate-bounce">02</span>
                                </div>
                            </div>

                            {/* Nodes & Buttons */}
                            <div className="flex flex-col gap-4 shrink-0 -rotate-2">
                                <div className="bg-foreground p-2 flex flex-wrap gap-1 max-w-[150px] justify-center shadow-[2px_2px_0_theme(colors.primary.DEFAULT)]">
                                    {[...Array(23)].map((_, i) => (
                                        <div key={`p5-${i}`} className="w-3 h-3 bg-primary rounded-full" />
                                    ))}
                                    {[...Array(2)].map((_, i) => (
                                        <div key={`p5e-${i}`} className="w-3 h-3 bg-background animate-ping rounded-full" />
                                    ))}
                                </div>

                                <button className="bg-primary text-primary-foreground border-4 border-foreground px-6 py-2 font-black italic text-xl shadow-[4px_4px_0_theme(colors.foreground)] hover:-translate-y-1 hover:shadow-[6px_6px_0_theme(colors.foreground)] transition-all rotate-3">
                                    TAKE_NODE!
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Design 10: VisionOS Spatial Glass */}
            <div className="space-y-4">
                <div className="flex items-center gap-2 border-b pb-2">
                    <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">10</h2>
                    <h2 className="text-lg font-semibold">空间计算毛玻璃 (VisionOS Spatial)</h2>
                </div>
                <p className="text-sm text-muted-foreground">Apple Vision Pro 的设计语言。抛弃了所有的硬边框，使用极端的圆角、极高强度的背景模糊（Blur）、微弱的纯白描边光和优雅的非衬线字体排版，让整个组件仿佛漂浮在现实之上。</p>

                {/* Simulated colorful background to show off glassmorphism */}
                <div className="p-8 rounded-xl border border-dashed relative overflow-hidden bg-gradient-to-br from-indigo-500/20 via-purple-500/20 to-pink-500/20 dark:from-indigo-900/40 dark:via-purple-900/40 dark:to-pink-900/40">
                    {/* Floating Orbs in background */}
                    <div className="absolute top-10 left-10 w-32 h-32 bg-primary/40 rounded-full blur-3xl"></div>
                    <div className="absolute bottom-10 right-10 w-48 h-48 bg-blue-500/30 rounded-full blur-3xl"></div>

                    <div className="rounded-[2.5rem] border border-white/20 dark:border-white/10 bg-white/30 dark:bg-black/30 backdrop-blur-2xl p-8 shadow-2xl relative z-10 font-sans">

                        <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-10">

                            <div className="flex flex-col gap-1">
                                <h3 className="text-foreground/60 text-sm font-medium tracking-wide">Infrastructure Matrix</h3>
                                <div className="text-5xl font-semibold tracking-tight text-foreground flex items-baseline gap-2">
                                    25 <span className="text-xl text-foreground/50 font-medium">Nodes</span>
                                </div>
                            </div>

                            <div className="flex flex-1 gap-6 md:px-10 border-l border-white/10 w-full">
                                <div className="flex flex-col gap-2 p-4 rounded-3xl bg-white/20 dark:bg-black/20 backdrop-blur-md border border-white/10 flex-1 relative overflow-hidden group">
                                    <div className="absolute inset-0 bg-primary/10 opacity-0 group-hover:opacity-100 transition-opacity"></div>
                                    <span className="text-foreground/70 text-xs font-semibold">Online Status</span>
                                    <div className="text-3xl font-bold text-foreground flex items-center gap-2">
                                        23
                                        <div className="w-2 h-2 rounded-full bg-primary shadow-[0_0_8px_theme(colors.primary.DEFAULT)]"></div>
                                    </div>
                                </div>

                                <div className="flex flex-col gap-2 p-4 rounded-3xl bg-white/20 dark:bg-black/20 backdrop-blur-md border border-white/10 flex-1 relative overflow-hidden">
                                    <span className="text-foreground/70 text-xs font-semibold">Offline / Errors</span>
                                    <div className="text-3xl font-bold text-foreground flex items-center gap-2">
                                        02
                                        <div className="w-2 h-2 rounded-full bg-destructive shadow-[0_0_8px_theme(colors.destructive.DEFAULT)] animate-pulse"></div>
                                    </div>
                                </div>
                            </div>

                            <div className="flex flex-col justify-center gap-3 shrink-0">
                                <div className="flex gap-1 mb-2 max-w-[120px] flex-wrap justify-end">
                                    {[...Array(23)].map((_, i) => (
                                        <div key={`vs-${i}`} className="w-2 h-2 rounded-full bg-primary/80" />
                                    ))}
                                    {[...Array(2)].map((_, i) => (
                                        <div key={`vse-${i}`} className="w-2 h-2 rounded-full bg-destructive/80 animate-pulse" />
                                    ))}
                                </div>
                                <button className="bg-white/40 dark:bg-white/10 hover:bg-white/60 dark:hover:bg-white/20 text-foreground px-6 py-3 rounded-full font-semibold text-sm backdrop-blur-md transition-all border border-white/20 shadow-sm flex items-center justify-center gap-2">
                                    <IconPlus className="w-4 h-4" /> Add Node
                                </button>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Design 11: EVA MAGI System (Theme-Aware) */}
                <div className="space-y-4">
                    <div className="flex items-center gap-2 border-b pb-2">
                        <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">11</h2>
                        <h2 className="text-lg font-semibold">三贤人系统 (EVA MAGI Style)</h2>
                    </div>
                    <p className="text-sm text-muted-foreground">《新世纪福音战士》MAGI 控制台风格。巨大的无填充文字、交错的斜纹警示条、高压力的警报排版。全面绑定主题色，随时都能准备启动绝对领域。</p>

                    <div className="p-4 rounded-xl border bg-card/10 border-dashed">
                        <div className="bg-card p-2 relative overflow-hidden font-sans border-4 border-foreground">
                            {/* Background honeycomb / hex pattern */}
                            <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyOCIgaGVpZ2h0PSI0OSIgdmlld0JveD0iMCAwIDI4IDQ5Ij48cGF0aCBkPSJNMzkgMjQuNUwzNC41IDE1SDI1LjVMMjEgMjQuNUwyNS41IDM0SDM0LjVMMzkgMjQuNVoiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzg4OCIgc3Ryb2tlLW9wYWNpdHk9IjAuMiIvPjwvc3ZnPg==')] opacity-10 pointer-events-none mix-blend-overlay"></div>

                            <div className="border border-foreground bg-background/50 p-6 relative">
                                {/* Emergency Borders */}
                                <div className="absolute top-0 left-0 w-full h-2 bg-[repeating-linear-gradient(45deg,theme(colors.destructive.DEFAULT),theme(colors.destructive.DEFAULT)_10px,transparent_10px,transparent_20px)]" />
                                <div className="absolute bottom-0 left-0 w-full h-2 bg-[repeating-linear-gradient(45deg,theme(colors.destructive.DEFAULT),theme(colors.destructive.DEFAULT)_10px,transparent_10px,transparent_20px)]" />

                                <div className="flex flex-col md:flex-row justify-between items-stretch gap-6 pt-4 pb-4">

                                    {/* Title Section */}
                                    <div className="flex flex-col justify-center bg-primary text-primary-foreground p-6 transform -skew-x-12 relative overflow-hidden">
                                        <h3 className="text-5xl font-black italic tracking-tighter mix-blend-overlay opacity-50 absolute -left-2 top-0 pointer-events-none">MELCHIOR</h3>
                                        <div className="relative z-10 transform skew-x-12 flex flex-col items-center">
                                            <span className="text-xl font-black tracking-widest uppercase">Nodes</span>
                                            <span className="text-[10px] font-bold bg-background text-primary px-2 py-0.5 mt-2">CONDITION.GREEN</span>
                                        </div>
                                    </div>

                                    {/* Data Display */}
                                    <div className="flex-1 border-2 border-foreground p-4 flex justify-between items-center relative overflow-hidden">
                                        {/* Outline huge number */}
                                        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 text-[120px] font-black italic text-transparent [-webkit-text-stroke:2px_theme(colors.primary.DEFAULT)] opacity-20 pointer-events-none tracking-tighter">
                                            25
                                        </div>

                                        <div className="relative z-10 text-center flex-1">
                                            <p className="text-[10px] font-black tracking-widest text-muted-foreground">TOTAL_UNITS</p>
                                            <p className="text-4xl font-black">{`[ 25 ]`}</p>
                                        </div>
                                        <div className="w-1 h-12 bg-foreground rotate-12 mx-2"></div>
                                        <div className="relative z-10 text-center flex-1">
                                            <p className="text-[10px] font-black tracking-widest text-primary">SYNC_RATIO</p>
                                            <p className="text-4xl font-black text-primary">92%</p>
                                        </div>
                                        <div className="w-1 h-12 bg-foreground rotate-12 mx-2"></div>
                                        <div className="relative z-10 text-center flex-1">
                                            <p className="text-[10px] font-black tracking-widest text-destructive">REJECTED</p>
                                            <p className="text-4xl font-black text-destructive animate-pulse">{`{ 02 }`}</p>
                                        </div>
                                    </div>

                                    {/* Actions */}
                                    <div className="flex flex-col justify-center gap-2">
                                        <button className="bg-foreground text-background font-black p-3 hover:bg-primary hover:text-primary-foreground transition-all uppercase tracking-widest text-sm border-b-4 border-r-4 border-black/50 dark:border-white/20 active:border-0 active:translate-x-1 active:translate-y-1">
                                            OVERRIDE
                                        </button>
                                    </div>
                                </div>

                                {/* Status Graph */}
                                <div className="mt-4 border-2 border-foreground p-2 flex gap-1 h-8">
                                    {[...Array(23)].map((_, i) => (
                                        <div key={`eva-${i}`} className="flex-1 bg-primary transform -skew-x-12" />
                                    ))}
                                    {[...Array(2)].map((_, i) => (
                                        <div key={`evae-${i}`} className="flex-1 bg-destructive transform -skew-x-12 animate-pulse scale-y-125" />
                                    ))}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Design 12: E-Ink Minimalist */}
                <div className="space-y-4">
                    <div className="flex items-center gap-2 border-b pb-2">
                        <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">12</h2>
                        <h2 className="text-lg font-semibold">电子墨水屏 (E-Ink Reader)</h2>
                    </div>
                    <p className="text-sm text-muted-foreground">抛弃所有色彩和光影的极简美学。一切只由黑线、白底和经典的衬线字体构成，还原类似 Kindle 的物理阅读体验。</p>

                    <div className="p-4 rounded-xl border bg-card/10 border-dashed">
                        <div className="bg-[#f4f4f4] dark:bg-[#1a1a1a] text-zinc-900 dark:text-zinc-100 p-8 font-serif shadow-inner border border-zinc-300 dark:border-zinc-800">
                            <div className="border-b-2 border-current pb-4 mb-8 flex justify-between items-baseline">
                                <div>
                                    <h3 className="text-3xl tracking-tight mb-1">Infrastructure</h3>
                                    <p className="text-xs font-sans tracking-widest uppercase opacity-60">Status Overview . Vol 1</p>
                                </div>
                                <div className="text-sm opacity-80 italic">
                                    Page 1 of 25
                                </div>
                            </div>

                            <div className="flex flex-col md:flex-row gap-12 items-center">
                                <div className="flex gap-8 items-baseline">
                                    <span className="text-8xl">25</span>
                                    <div className="flex flex-col gap-2 opacity-80">
                                        <span className="font-sans uppercase text-sm tracking-widest border-b border-current pb-1">Nodes Registered</span>
                                        <span className="italic">Active capability.</span>
                                    </div>
                                </div>

                                <div className="flex-1 border-l-2 border-current pl-12 flex flex-col gap-6">
                                    <div className="flex justify-between items-center">
                                        <span className="text-lg">Functioning</span>
                                        <span className="text-2xl border border-current px-4 py-1 rounded-full">23</span>
                                    </div>
                                    <div className="flex justify-between items-center text-zinc-500 dark:text-zinc-400">
                                        <span className="text-lg line-through text-zinc-500">Unresponsive</span>
                                        <span className="text-2xl font-bold">02</span>
                                    </div>
                                </div>
                            </div>

                            <div className="mt-12 flex justify-end gap-6 font-sans">
                                <button className="border-b border-transparent hover:border-current pb-1 uppercase tracking-widest text-xs transition-all">
                                    [ Expand Graph ]
                                </button>
                                <button className="border-b border-transparent hover:border-current pb-1 uppercase tracking-widest text-xs transition-all">
                                    [ Add Device ]
                                </button>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Design 13: Pip-Boy Wasteland */}
                <div className="space-y-4">
                    <div className="flex items-center gap-2 border-b pb-2">
                        <h2 className="text-xl font-bold font-mono text-muted-foreground mr-2">13</h2>
                        <h2 className="text-lg font-semibold">废土生存终端 (Pip-Boy Style)</h2>
                    </div>
                    <p className="text-sm text-muted-foreground">《辐射》系列 Pip-Boy 风格。模拟粗糙的圆极管显示器边缘，大锅盖曲面感，以及经典的单色磷光显示体验，不过我们给它做了跟随主色调的热更新！</p>

                    <div className="p-4 rounded-xl border bg-card/10 border-dashed">
                        <div className="bg-[#121212] p-8 rounded-[3rem] border-[12px] border-zinc-800 shadow-[inset_0_0_50px_rgba(0,0,0,0.8),_0_10px_20px_rgba(0,0,0,0.5)] relative overflow-hidden font-mono uppercase text-primary">
                            {/* CRT Screen curve highlights */}
                            <div className="absolute top-0 left-0 w-full h-1/3 bg-gradient-to-b from-white/5 to-transparent rounded-[100%] pointer-events-none transform -translate-y-1/2"></div>
                            <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,transparent_50%,rgba(0,0,0,0.4)_120%)] pointer-events-none"></div>

                            {/* Screen interlacing */}
                            <div className="absolute inset-0 bg-[repeating-linear-gradient(transparent,transparent_2px,rgba(0,0,0,0.3)_3px)] pointer-events-none"></div>

                            <div className="relative z-10 p-4">
                                <div className="flex justify-between border-b-2 border-primary/50 pb-2 mb-6 text-sm tracking-widest opacity-80">
                                    <span>STAT</span>
                                    <span>INV</span>
                                    <span>DATA</span>
                                    <span>MAP</span>
                                    <span>RADIO</span>
                                </div>

                                <div className="flex gap-8">
                                    {/* Fallout Boy Face Placeholder */}
                                    <div className="w-1/3 border-2 border-primary/30 rounded-lg flex items-center justify-center p-4 bg-primary/5 shadow-[0_0_15px_theme(colors.primary.DEFAULT)_inset]">
                                        <div className="flex flex-col items-center gap-2 opacity-80">
                                            <IconServer className="w-16 h-16" />
                                            <span className="text-xs tracking-widest">VAULT.CTL</span>
                                        </div>
                                    </div>

                                    <div className="w-2/3 flex flex-col justify-center gap-6">
                                        <div className="flex justify-between items-end border-b border-primary/20 pb-2">
                                            <span className="tracking-widest opacity-80">TOTAL_WORKERS</span>
                                            <span className="text-5xl font-bold shadow-[0_0_10px_theme(colors.primary.DEFAULT)]">25</span>
                                        </div>

                                        <div className="flex gap-8">
                                            <div className="flex flex-col gap-1 w-full">
                                                <div className="flex justify-between items-center text-xs opacity-80 mb-1">
                                                    <span>HP (ACTIVE)</span>
                                                    <span>23/25</span>
                                                </div>
                                                <div className="h-4 border border-primary/50 p-[2px] w-full shadow-[0_0_5px_theme(colors.primary.DEFAULT)]">
                                                    <div className="h-full bg-primary" style={{ width: '92%' }}></div>
                                                </div>
                                            </div>
                                        </div>

                                        <div className="flex gap-8">
                                            <div className="flex flex-col gap-1 w-full">
                                                <div className="flex justify-between items-center text-xs opacity-80 mb-1">
                                                    <span>RADS (LOST)</span>
                                                    <span>02/25</span>
                                                </div>
                                                <div className="h-4 border border-primary/50 p-[2px] w-full">
                                                    <div className="h-full bg-destructive flex gap-1 animate-pulse" style={{ width: '8%' }}></div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                <div className="mt-8 pt-4 flex gap-4 text-sm justify-between opacity-80">
                                    <span>[E] INSTALL</span>
                                    <span>[R] REPAIR</span>
                                    <span>[TAB] EXIT</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

            </div>
        </div>
    )
}
