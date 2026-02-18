"use client"

import React from "react"

type BadgeType = 'info' | 'success' | 'warning' | 'error'
type BadgeProps = { label: string; value: string | number; type: BadgeType }

export default function BadgeVariantsPage() {
  const stats: BadgeProps[] = [
    { label: "SUBDOMAIN", value: "156", type: "info" },
    { label: "WEBSITE", value: "89", type: "success" },
    { label: "IP", value: "45", type: "warning" },
    { label: "VULN", value: "23", type: "error" },
  ]

  return (
    <div className="flex flex-col gap-12 p-8 max-w-5xl mx-auto">
      <div>
        <h1 className="text-2xl font-bold mb-2">数据徽章 (Badge) 样式方案</h1>
        <p className="text-muted-foreground">用于表格中展示统计数据的微型组件设计对比</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-12">
        
        {/* Option A: Minimalist Industry (current) */}
        <div className="space-y-4">
          <div className="border-b pb-2">
            <h3 className="font-medium">方案 A：极简工业 (当前)</h3>
            <p className="text-xs text-muted-foreground">淡色背景 + 细边框 + 直角，冷静客观</p>
          </div>
          <div className="p-6 bg-zinc-50 border rounded-lg">
             <div className="bg-white border rounded shadow-sm p-4">
               <div className="text-xs text-muted-foreground mb-2">模拟表格行</div>
               <div className="flex flex-wrap gap-2">
                  {stats.map((s, i) => (
                    <BadgeA key={i} {...s} />
                  ))}
               </div>
             </div>
          </div>
        </div>

        {/* Option B: solid capsule */}
        <div className="space-y-4">
          <div className="border-b pb-2">
            <h3 className="font-medium">方案 B：胶囊实心</h3>
            <p className="text-xs text-muted-foreground">完全圆角，色彩饱和度稍高，视觉权重强</p>
          </div>
          <div className="p-6 bg-zinc-50 border rounded-lg">
             <div className="bg-white border rounded shadow-sm p-4">
               <div className="text-xs text-muted-foreground mb-2">模拟表格行</div>
               <div className="flex flex-wrap gap-2">
                  {stats.map((s, i) => (
                    <BadgeB key={i} {...s} />
                  ))}
               </div>
             </div>
          </div>
        </div>

        {/* Option C: Dotted with dots */}
        <div className="space-y-4">
          <div className="border-b pb-2">
            <h3 className="font-medium">方案 C：点缀圆点</h3>
            <p className="text-xs text-muted-foreground">无背景，靠圆点区分，最干净，适合极高密度</p>
          </div>
          <div className="p-6 bg-zinc-50 border rounded-lg">
             <div className="bg-white border rounded shadow-sm p-4">
               <div className="text-xs text-muted-foreground mb-2">模拟表格行</div>
               <div className="flex flex-wrap gap-3">
                  {stats.map((s, i) => (
                    <BadgeC key={i} {...s} />
                  ))}
               </div>
             </div>
          </div>
        </div>

        {/* Option D: Technological Stroke */}
        <div className="space-y-4">
          <div className="border-b pb-2">
            <h3 className="font-medium">方案 D：科技描边</h3>
            <p className="text-xs text-muted-foreground">透明背景，强描边，数字加粗，强调数据感</p>
          </div>
          <div className="p-6 bg-zinc-50 border rounded-lg">
             <div className="bg-white border rounded shadow-sm p-4">
               <div className="text-xs text-muted-foreground mb-2">模拟表格行</div>
               <div className="flex flex-wrap gap-2">
                  {stats.map((s, i) => (
                    <BadgeD key={i} {...s} />
                  ))}
               </div>
             </div>
          </div>
        </div>

        {/* Option E: Microcard */}
        <div className="space-y-4">
          <div className="border-b pb-2">
            <h3 className="font-medium">方案 E：微型卡片</h3>
            <p className="text-xs text-muted-foreground">带阴影，左侧色条，像便利贴，有质感</p>
          </div>
          <div className="p-6 bg-zinc-50 border rounded-lg">
             <div className="bg-white border rounded shadow-sm p-4">
               <div className="text-xs text-muted-foreground mb-2">模拟表格行</div>
               <div className="flex flex-wrap gap-2">
                  {stats.map((s, i) => (
                    <BadgeE key={i} {...s} />
                  ))}
               </div>
             </div>
          </div>
        </div>

        {/* Option F: Color-block splice (Bauhaus-style) */}
        <div className="space-y-4">
          <div className="border-b pb-2">
            <h3 className="font-medium">方案 F：色块拼接 (Bauhaus)</h3>
            <p className="text-xs text-muted-foreground">几何分割，左侧数字色块，右侧说明，强结构感</p>
          </div>
          <div className="p-6 bg-zinc-50 border rounded-lg">
             <div className="bg-white border rounded shadow-sm p-4">
               <div className="text-xs text-muted-foreground mb-2">模拟表格行</div>
               <div className="flex flex-wrap gap-3">
                  {stats.map((s, i) => (
                    <BadgeF key={i} {...s} />
                  ))}
               </div>
             </div>
          </div>
        </div>

        {/* Solution G: thick underline */}
        <div className="space-y-4">
          <div className="border-b pb-2">
            <h3 className="font-medium">方案 G：粗下划线</h3>
            <p className="text-xs text-muted-foreground">无背景，底部粗线强调，极度透气</p>
          </div>
          <div className="p-6 bg-zinc-50 border rounded-lg">
             <div className="bg-white border rounded shadow-sm p-4">
               <div className="text-xs text-muted-foreground mb-2">模拟表格行</div>
               <div className="flex flex-wrap gap-4">
                  {stats.map((s, i) => (
                    <BadgeG key={i} {...s} />
                  ))}
               </div>
             </div>
          </div>
        </div>

        {/* Scheme H: Industrial nameplate (black and white) */}
        <div className="space-y-4">
          <div className="border-b pb-2">
            <h3 className="font-medium">方案 H：工业铭牌</h3>
            <p className="text-xs text-muted-foreground">深色底白字，左侧彩色状态条，硬核质感</p>
          </div>
          <div className="p-6 bg-zinc-50 border rounded-lg">
             <div className="bg-white border rounded shadow-sm p-4">
               <div className="text-xs text-muted-foreground mb-2">模拟表格行</div>
               <div className="flex flex-wrap gap-2">
                  {stats.map((s, i) => (
                    <BadgeH key={i} {...s} />
                  ))}
               </div>
             </div>
          </div>
        </div>

      </div>
    </div>
  )
}

// Helper function
const getColor = (type: BadgeType) => {
  switch (type) {
    case 'info': return 'text-sky-600 bg-sky-50 border-sky-200';
    case 'success': return 'text-emerald-600 bg-emerald-50 border-emerald-200';
    case 'warning': return 'text-amber-600 bg-amber-50 border-amber-200';
    case 'error': return 'text-rose-600 bg-rose-50 border-rose-200';
    default: return 'text-zinc-600 bg-zinc-50 border-zinc-200';
  }
}

const getSolidBg = (type: BadgeType) => {
  switch (type) {
    case 'info': return 'bg-sky-600';
    case 'success': return 'bg-emerald-600';
    case 'warning': return 'bg-amber-500';
    case 'error': return 'bg-rose-600';
    default: return 'bg-zinc-600';
  }
}

const getDotColor = (type: BadgeType) => {
  switch (type) {
    case 'info': return 'bg-sky-500';
    case 'success': return 'bg-emerald-500';
    case 'warning': return 'bg-amber-500';
    case 'error': return 'bg-rose-500';
    default: return 'bg-zinc-500';
  }
}

const getLineColor = (type: BadgeType) => {
  switch (type) {
    case 'info': return 'border-sky-500';
    case 'success': return 'border-emerald-500';
    case 'warning': return 'border-amber-500';
    case 'error': return 'border-rose-500';
    default: return 'border-zinc-500';
  }
}

const getBorderColor = (type: BadgeType) => {
  switch (type) {
    case 'info': return 'border-sky-500 text-sky-600';
    case 'success': return 'border-emerald-500 text-emerald-600';
    case 'warning': return 'border-amber-500 text-amber-600';
    case 'error': return 'border-rose-500 text-rose-600';
    default: return 'border-zinc-500 text-zinc-600';
  }
}

// Scenario components
function BadgeA({ label, value, type }: BadgeProps) {
  const colors = getColor(type);
  return (
    <span className={`inline-flex items-center rounded-none border px-2 py-1 text-[10px] font-mono font-medium tracking-wider uppercase ${colors}`}>
      {value} {label}
    </span>
  )
}

function BadgeB({ label, value, type }: BadgeProps) {
  let customClass = "";
  if (type === 'info') customClass = "bg-sky-100 text-sky-700 border-transparent";
  if (type === 'success') customClass = "bg-emerald-100 text-emerald-700 border-transparent";
  if (type === 'warning') customClass = "bg-amber-100 text-amber-700 border-transparent";
  if (type === 'error') customClass = "bg-rose-100 text-rose-700 border-transparent";

  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-[10px] font-bold tracking-wide uppercase ${customClass}`}>
      {value} {label}
    </span>
  )
}

function BadgeC({ label, value, type }: BadgeProps) {
  const dotColor = getDotColor(type);
  return (
    <span className="inline-flex items-center text-[11px] font-medium text-zinc-600">
      <span className={`w-1.5 h-1.5 rounded-full mr-1.5 ${dotColor}`}></span>
      <span className="font-mono font-bold mr-1 text-zinc-900">{value}</span>
      <span className="text-[9px] text-zinc-400 uppercase tracking-wider">{label}</span>
    </span>
  )
}

function BadgeD({ label, value, type }: BadgeProps) {
  const colors = getBorderColor(type);
  return (
    <span className={`inline-flex items-center rounded-sm border px-1.5 py-0.5 text-[10px] font-mono tracking-wider uppercase bg-white ${colors}`}>
      <span className="font-bold mr-1">{value}</span>
      <span className="opacity-70 text-[9px]">{label}</span>
    </span>
  )
}

function BadgeE({ label, value, type }: BadgeProps) {
  const dotColor = getDotColor(type);
  return (
    <span className="inline-flex items-center rounded border border-zinc-200 bg-white shadow-sm px-2 py-0.5 text-[10px] overflow-hidden relative pl-2.5">
       {/* Left color bar */}
      <span className={`absolute left-0 top-0 bottom-0 w-1 ${dotColor}`}></span>
      <span className="font-mono font-bold text-zinc-800 mr-1.5">{value}</span>
      <span className="text-zinc-500 font-medium tracking-wide uppercase text-[9px]">{label}</span>
    </span>
  )
}

function BadgeF({ label, value, type }: BadgeProps) {
  const solidBg = getSolidBg(type);
  
  // For F, we need the border color to match the solid background
  let borderClass = "border-zinc-200";
  if (type === 'info') borderClass = "border-sky-600";
  if (type === 'success') borderClass = "border-emerald-600";
  if (type === 'warning') borderClass = "border-amber-500";
  if (type === 'error') borderClass = "border-rose-600";

  return (
    <span className={`inline-flex items-stretch border ${borderClass} text-[10px] uppercase font-mono tracking-wider overflow-hidden rounded-none`}>
      <span className={`px-1.5 py-0.5 text-white font-bold flex items-center justify-center ${solidBg}`}>
        {value}
      </span>
      <span className="px-1.5 py-0.5 bg-white text-zinc-700 flex items-center font-medium">
        {label}
      </span>
    </span>
  )
}

function BadgeG({ label, value, type }: BadgeProps) {
  const lineColor = getLineColor(type);
  return (
    <span className={`inline-flex items-baseline gap-1.5 text-[10px] uppercase tracking-wider border-b-2 ${lineColor} pb-0.5 px-0.5`}>
      <span className="font-mono font-bold text-base leading-none text-zinc-900">{value}</span>
      <span className="font-medium text-zinc-500">{label}</span>
    </span>
  )
}

function BadgeH({ label, value, type }: BadgeProps) {
  const solidBg = getSolidBg(type);
  return (
    <span className="inline-flex items-center bg-zinc-900 text-white text-[10px] font-mono tracking-wider uppercase rounded-sm overflow-hidden pr-2">
      <span className={`w-1 self-stretch mr-2 ${solidBg}`}></span>
      <span className="font-bold mr-1.5">{value}</span>
      <span className="opacity-60 font-medium">{label}</span>
    </span>
  )
}
