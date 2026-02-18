"use client"

import React from "react"
import { IconBug, IconRadar, IconTool } from "@/components/icons"

export default function SidebarVariantsPage() {
  return (
    <div className="flex flex-col gap-8 p-8 max-w-6xl mx-auto">
      <div>
        <h1 className="text-2xl font-bold mb-2">侧边栏子菜单样式方案</h1>
        <p className="text-muted-foreground">对比三种不同的选中状态设计风格</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
        
        {/* Option A: Classical Geometry */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 A：经典几何 (推荐)
            <div className="text-xs text-muted-foreground font-normal mt-1">
              加粗左侧线条 (3px)，移除圆点，极简工业风
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
            {/* Simulate sidebar */}
            <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="pl-0 space-y-1 mt-1">
                   {/* Simulate expanded submenu */}
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4 border-l-2 border-transparent">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option A */}
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm font-medium bg-zinc-100 text-foreground ml-0 pl-[26px] border-l-[4px] border-[#FF4C00] relative">
                    <span>扫描历史</span>
                    {/* No right dot */}
                  </div>

                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4 border-l-2 border-transparent">
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option B: Front square point */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 B：前置方点
            <div className="text-xs text-muted-foreground font-normal mt-1">
              左侧无边框，文字前加方块，数据列表感
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="pl-0 space-y-1 mt-1">
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-9">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option B */}
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm font-medium bg-zinc-100 text-foreground ml-4 rounded-md">
                    <div className="w-1.5 h-1.5 bg-[#FF4C00] shrink-0" />
                    <span>扫描历史</span>
                  </div>

                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-9">
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option C: Highlight Color Block */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 C：高亮色块
            <div className="text-xs text-muted-foreground font-normal mt-1">
              深色背景反白文字，强对比，类似选中行
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="pl-0 space-y-1 mt-1">
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option C */}
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm font-medium bg-zinc-800 text-white ml-4 rounded-md shadow-sm">
                    <span>扫描历史</span>
                  </div>

                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option D: Right indicator bar */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 D：右侧指示条
            <div className="text-xs text-muted-foreground font-normal mt-1">
              淡色背景，指示条在最右侧，平衡视觉
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="pl-0 space-y-1 mt-1">
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option D */}
                  <div className="flex items-center justify-between px-2 py-1.5 text-sm font-medium bg-zinc-50 text-foreground ml-4 rounded-md relative overflow-hidden">
                    <span>扫描历史</span>
                    <div className="absolute right-0 top-0 bottom-0 w-1 bg-[#FF4C00]"></div>
                  </div>

                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option E: Text discoloration */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 E：文字变色
            <div className="text-xs text-muted-foreground font-normal mt-1">
              无背景，文字橙色高亮，极致简约
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="pl-0 space-y-1 mt-1">
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option E */}
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm font-semibold text-[#FF4C00] ml-4 rounded-md">
                    <span>扫描历史</span>
                  </div>

                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option F: Orange Capsules */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 F：橙色胶囊
            <div className="text-xs text-muted-foreground font-normal mt-1">
              橙色填充背景，白色文字，强烈的按钮感
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="pl-0 space-y-1 mt-1">
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option F */}
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm font-medium bg-[#FF4C00] text-white ml-4 rounded-full shadow-sm">
                    <span>扫描历史</span>
                  </div>

                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option G: Tree connection line */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 G: 树形连接线
            <div className="text-xs text-muted-foreground font-normal mt-1">
              垂直连线指示层级，极具结构感
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                {/* tree structure container */}
                <div className="relative ml-4 pl-3 space-y-1 mt-1">
                  {/* vertical connection line */}
                  <div className="absolute left-0 top-0 bottom-2 w-px bg-zinc-200"></div>

                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md">
                     {/* horizontal connecting line */}
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200"></div>
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option G */}
                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm font-medium text-[#FF4C00] bg-zinc-50 rounded-md">
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200"></div>
                    <div className="w-1.5 h-1.5 rounded-full bg-[#FF4C00] mr-1"></div>
                    <span>扫描历史</span>
                  </div>

                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md">
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200"></div>
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option H: Timeline Node */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 H: 时间轴节点
            <div className="text-xs text-muted-foreground font-normal mt-1">
              左侧贯穿线 + 节点圆点，类似步骤条
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="relative ml-5 space-y-4 mt-2 mb-2">
                  {/* through line */}
                  <div className="absolute left-0 top-0 bottom-0 w-px bg-zinc-200 transform -translate-x-1/2"></div>

                  <div className="relative flex items-center gap-2 pl-4 text-sm text-muted-foreground">
                    {/* Node is not selected */}
                    <div className="absolute left-0 top-1/2 w-1.5 h-1.5 bg-zinc-300 rounded-full transform -translate-x-1/2 -translate-y-1/2 border border-white ring-2 ring-white"></div>
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option H */}
                  <div className="relative flex items-center gap-2 pl-4 text-sm font-semibold text-foreground">
                    {/* Select node */}
                    <div className="absolute left-0 top-1/2 w-2.5 h-2.5 bg-[#FF4C00] rounded-full transform -translate-x-1/2 -translate-y-1/2 ring-4 ring-zinc-50"></div>
                    <span>扫描历史</span>
                  </div>

                  <div className="relative flex items-center gap-2 pl-4 text-sm text-muted-foreground">
                    <div className="absolute left-0 top-1/2 w-1.5 h-1.5 bg-zinc-300 rounded-full transform -translate-x-1/2 -translate-y-1/2 border border-white ring-2 ring-white"></div>
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option I: Ghost indentation lines */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 I: 幽灵缩进线
            <div className="text-xs text-muted-foreground font-normal mt-1">
              仅在Hover/选中时显示的左侧细条，极简
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="space-y-1 mt-1">
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-50 hover:text-foreground ml-4 border-l border-transparent hover:border-zinc-300 transition-colors pl-3">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Items - Option I */}
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm font-medium text-foreground ml-4 border-l border-[#FF4C00] pl-3 bg-zinc-50/50">
                    <span>扫描历史</span>
                  </div>

                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-50 hover:text-foreground ml-4 border-l border-transparent hover:border-zinc-300 transition-colors pl-3">
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option J: Block Level Cables */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 J: 块级连接线
            <div className="text-xs text-muted-foreground font-normal mt-1">
              粗线条连接，更有分量感，类似GitHub文件树
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="relative ml-2 space-y-0.5 mt-1">
                  {/* thick gray line background */}
                  <div className="absolute left-2 top-0 bottom-2 w-0.5 bg-zinc-100"></div>

                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option J */}
                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm font-medium bg-zinc-100 text-foreground ml-4 rounded-md">
                     {/* When selected, an orange line covers the left side */}
                    <div className="absolute left-[-10px] top-1/2 -translate-y-1/2 h-full w-0.5 bg-[#FF4C00]"></div>
                    <span>扫描历史</span>
                  </div>

                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md ml-4">
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Solution K: Sliding background animation */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 K: 滑动背景动画
            <div className="text-xs text-muted-foreground font-normal mt-1">
              背景色块从左侧滑入，文字位移 (Hover查看效果)
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="space-y-1 mt-1 pl-4">
                  <div className="group relative flex items-center gap-2 px-3 py-1.5 text-sm text-muted-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <span className="relative transition-transform duration-300 group-hover:translate-x-1">扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option K */}
                  <div className="group relative flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute inset-0 bg-zinc-100 border-l-2 border-[#FF4C00]"></div>
                    <span className="relative translate-x-1">扫描历史</span>
                  </div>

                  <div className="group relative flex items-center gap-2 px-3 py-1.5 text-sm text-muted-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <span className="relative transition-transform duration-300 group-hover:translate-x-1">定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Plan L: Neon Glow (Glow) */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 L: 霓虹辉光
            <div className="text-xs text-muted-foreground font-normal mt-1">
              左侧发光条 + 柔和阴影，科技感强
            </div>
          </div>
          <div className="p-4 bg-zinc-900 min-h-[300px]">
             {/* Works better in dark mode */}
             <div className="w-64 bg-zinc-950 border border-zinc-800 rounded-lg shadow-sm overflow-hidden mx-auto text-zinc-400">
              <div className="p-2 space-y-1">
                <div className="flex items-center gap-2 px-2 py-2 text-sm rounded-md bg-zinc-900 text-zinc-100">
                  <IconRadar className="w-4 h-4" />
                  <span>扫描管理</span>
                </div>
                
                <div className="space-y-1 mt-1 pl-4">
                  <div className="flex items-center gap-2 px-3 py-1.5 text-sm hover:text-zinc-200 transition-colors cursor-pointer">
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option L */}
                  <div className="relative flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-white rounded-r-md cursor-pointer">
                    {/* Luminous strip */}
                    <div className="absolute left-0 top-1 bottom-1 w-0.5 bg-[#FF4C00] shadow-[0_0_8px_rgba(255,76,0,0.8)] rounded-full"></div>
                    {/* background glow */}
                    <div className="absolute inset-0 bg-gradient-to-r from-[#FF4C00]/10 to-transparent opacity-50"></div>
                    <span className="relative">扫描历史</span>
                  </div>

                  <div className="flex items-center gap-2 px-3 py-1.5 text-sm hover:text-zinc-200 transition-colors cursor-pointer">
                    <span>定时任务</span>
                  </div>
                </div>

                <div className="flex items-center gap-2 px-2 py-2 text-sm rounded-md hover:bg-zinc-900 transition-colors cursor-pointer">
                  <IconBug className="w-4 h-4" />
                  <span>漏洞管理</span>
                </div>
                 <div className="flex items-center gap-2 px-2 py-2 text-sm rounded-md hover:bg-zinc-900 transition-colors cursor-pointer">
                  <IconTool className="w-4 h-4" />
                  <span>工具箱</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Plan M: Pulse Dots */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 M: 脉冲圆点
            <div className="text-xs text-muted-foreground font-normal mt-1">
              活跃状态的呼吸灯效果，生命力感
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="relative ml-4 pl-3 space-y-1 mt-1">
                  <div className="absolute left-0 top-0 bottom-2 w-px bg-zinc-200"></div>

                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md group cursor-pointer">
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200 group-hover:bg-zinc-400 transition-colors"></div>
                    <span>扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option M */}
                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm font-medium text-foreground bg-zinc-50 rounded-md cursor-pointer">
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-[#FF4C00]"></div>
                    
                    {/* pulse dot structure */}
                    <div className="relative flex items-center justify-center w-2 h-2 mr-1">
                      <span className="absolute inline-flex h-full w-full rounded-full bg-[#FF4C00] opacity-75 animate-ping"></span>
                      <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-[#FF4C00]"></span>
                    </div>
                    
                    <span>扫描历史</span>
                  </div>

                  <div className="relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground hover:bg-zinc-100 rounded-md group cursor-pointer">
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200 group-hover:bg-zinc-400 transition-colors"></div>
                    <span>定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Plan N: K(sliding) + J(thick line) fusion */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 N: 滑动块级线 (K+J)
            <div className="text-xs text-muted-foreground font-normal mt-1">
              左侧粗线结构 + 细腻的滑入背景交互
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="relative ml-2 space-y-0.5 mt-1">
                  {/* J's background line */}
                  <div className="absolute left-2 top-0 bottom-2 w-0.5 bg-zinc-100"></div>

                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground overflow-hidden rounded-md ml-4 cursor-pointer">
                    {/* K's sliding background */}
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <span className="relative transition-transform duration-300 group-hover:translate-x-1">扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option N */}
                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm font-medium text-foreground overflow-hidden rounded-md ml-4 cursor-pointer">
                    {/* Persistent background */}
                    <div className="absolute inset-0 bg-zinc-100"></div>
                    {/* thick orange line of J */}
                    <div className="absolute left-[-10px] top-1/2 -translate-y-1/2 h-full w-0.5 bg-[#FF4C00]"></div>
                    <span className="relative translate-x-1">扫描历史</span>
                  </div>

                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground overflow-hidden rounded-md ml-4 cursor-pointer">
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <span className="relative transition-transform duration-300 group-hover:translate-x-1">定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Scheme O: K(sliding) + M(pulse) fusion */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 O: 滑动脉冲 (K+M)
            <div className="text-xs text-muted-foreground font-normal mt-1">
              背景滑入交互 + 呼吸灯焦点，动感十足
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="relative ml-4 pl-3 space-y-1 mt-1">
                   {/* Thin line of M */}
                  <div className="absolute left-0 top-0 bottom-2 w-px bg-zinc-200"></div>

                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground overflow-hidden rounded-md cursor-pointer">
                    {/* M's horizontal line */}
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200 z-10"></div>
                    {/* K's sliding background */}
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <span className="relative transition-transform duration-300 group-hover:translate-x-1">扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option O */}
                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm font-medium text-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute inset-0 bg-zinc-50"></div>
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-[#FF4C00] z-10"></div>
                    
                    {/* M pulse dot */}
                    <div className="relative flex items-center justify-center w-2 h-2 mr-1 z-10">
                      <span className="absolute inline-flex h-full w-full rounded-full bg-[#FF4C00] opacity-75 animate-ping"></span>
                      <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-[#FF4C00]"></span>
                    </div>
                    
                    <span className="relative z-10">扫描历史</span>
                  </div>

                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200 z-10"></div>
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <span className="relative transition-transform duration-300 group-hover:translate-x-1">定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Plan P: H (time axis) + K (sliding) fusion */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 P: 时间轴滑块 (H+K)
            <div className="text-xs text-muted-foreground font-normal mt-1">
              时间轴节点 + 背景滑入，流程确认感最强
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="relative ml-5 space-y-1 mt-2 mb-2">
                   {/* H's through line */}
                  <div className="absolute left-0 top-0 bottom-0 w-px bg-zinc-200 transform -translate-x-1/2"></div>

                  <div className="group relative flex items-center gap-2 pl-4 py-1.5 text-sm text-muted-foreground overflow-hidden rounded-md cursor-pointer">
                     {/* K's sliding background (note that there is no need to cover the node here, so the margin-left is adjusted) */}
                    <div className="absolute inset-y-0 right-0 left-2 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out rounded-md"></div>
                    
                    {/* Node of H */}
                    <div className="absolute left-0 top-1/2 w-1.5 h-1.5 bg-zinc-300 rounded-full transform -translate-x-1/2 -translate-y-1/2 border border-white ring-2 ring-white z-10"></div>
                    <span className="relative z-10 transition-transform duration-300 group-hover:translate-x-1">扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option P */}
                  <div className="group relative flex items-center gap-2 pl-4 py-1.5 text-sm font-semibold text-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute inset-y-0 right-0 left-2 bg-zinc-100 rounded-md"></div>
                    
                    {/* Selected node of H */}
                    <div className="absolute left-0 top-1/2 w-2.5 h-2.5 bg-[#FF4C00] rounded-full transform -translate-x-1/2 -translate-y-1/2 ring-4 ring-zinc-50 z-10"></div>
                    <span className="relative z-10">扫描历史</span>
                  </div>

                  <div className="group relative flex items-center gap-2 pl-4 py-1.5 text-sm text-muted-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute inset-y-0 right-0 left-2 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out rounded-md"></div>
                    <div className="absolute left-0 top-1/2 w-1.5 h-1.5 bg-zinc-300 rounded-full transform -translate-x-1/2 -translate-y-1/2 border border-white ring-2 ring-white z-10"></div>
                    <span className="relative z-10 transition-transform duration-300 group-hover:translate-x-1">定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Plan N: Dynamic Fusion (J+K+M) */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 N: 动感融合 (J+K+M)
            <div className="text-xs text-muted-foreground font-normal mt-1">
              滑入背景 + 粗线条 + 呼吸光效，旗舰级体验
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="relative space-y-1 mt-1">
                  {/* Background thick gray line */}
                  <div className="absolute left-3 top-0 bottom-2 w-0.5 bg-zinc-100"></div>

                  {/* Normal item: K's slide-in effect */}
                  <div className="group relative flex items-center gap-2 px-3 py-1.5 ml-2 text-sm text-muted-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <span className="relative transition-transform duration-300 group-hover:translate-x-1">扫描概览</span>
                  </div>
                  
                  {/* Selected item: J+M combination */}
                  <div className="group relative flex items-center gap-2 px-3 py-1.5 ml-2 text-sm font-medium text-foreground bg-zinc-50/50 overflow-hidden rounded-md cursor-pointer">
                    {/* K: static background */}
                    <div className="absolute inset-0 bg-zinc-100 opacity-50"></div>
                    
                    {/* J+M: thick line on the left + breathing halo */}
                    <div className="absolute left-0 top-0 bottom-0 w-1 bg-[#FF4C00]">
                       {/* Simulate the flow of light effects */}
                       <div className="absolute inset-0 bg-white/30 animate-pulse"></div>
                    </div>
                    
                    <span className="relative pl-1">扫描历史</span>
                  </div>

                  <div className="group relative flex items-center gap-2 px-3 py-1.5 ml-2 text-sm text-muted-foreground overflow-hidden rounded-md cursor-pointer">
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <span className="relative transition-transform duration-300 group-hover:translate-x-1">定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

        {/* Option O: Magnetic Slider + Active Point (K+M) */}
        <div className="border rounded-xl overflow-hidden shadow-sm bg-background">
          <div className="bg-muted/50 p-3 border-b text-center font-medium">
            方案 O: 磁性滑块 + 活跃点
            <div className="text-xs text-muted-foreground font-normal mt-1">
              滑入背景承载脉冲点，动静结合
            </div>
          </div>
          <div className="p-4 bg-zinc-50/50 min-h-[300px]">
             <div className="w-64 bg-white border rounded-lg shadow-sm overflow-hidden mx-auto">
              <div className="p-2 space-y-1">
                <MenuButton icon={IconRadar} label="扫描管理" active />
                
                <div className="relative ml-4 pl-3 space-y-1 mt-1">
                  <div className="absolute left-0 top-0 bottom-2 w-px bg-zinc-200"></div>

                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground rounded-md cursor-pointer overflow-hidden">
                    {/* K: slide into background */}
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    {/* G: connecting line */}
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200 group-hover:bg-zinc-300 transition-colors z-10"></div>
                    <span className="relative z-10 transition-transform duration-300 group-hover:translate-x-1">扫描概览</span>
                  </div>
                  
                  {/* Selected Item - Option O */}
                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm font-medium text-foreground rounded-md cursor-pointer overflow-hidden">
                    {/* Persistent background */}
                    <div className="absolute inset-0 bg-zinc-100"></div>
                    
                    {/* Cable discoloration */}
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-[#FF4C00] z-10"></div>
                    
                    {/* M: pulse point */}
                    <div className="relative z-10 flex items-center justify-center w-2 h-2 mr-1">
                      <span className="absolute inline-flex h-full w-full rounded-full bg-[#FF4C00] opacity-75 animate-ping"></span>
                      <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-[#FF4C00]"></span>
                    </div>
                    
                    <span className="relative z-10">扫描历史</span>
                  </div>

                  <div className="group relative flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground rounded-md cursor-pointer overflow-hidden">
                    <div className="absolute inset-0 bg-zinc-100 -translate-x-full group-hover:translate-x-0 transition-transform duration-300 ease-out"></div>
                    <div className="absolute left-[-12px] top-1/2 w-3 h-px bg-zinc-200 group-hover:bg-zinc-300 transition-colors z-10"></div>
                    <span className="relative z-10 transition-transform duration-300 group-hover:translate-x-1">定时任务</span>
                  </div>
                </div>

                <MenuButton icon={IconBug} label="漏洞管理" />
                <MenuButton icon={IconTool} label="工具箱" />
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  )
}

function MenuButton({ icon: Icon, label, active }: { icon: React.ComponentType<{ className?: string }>, label: string, active?: boolean }) {
  return (
    <div className={`flex items-center gap-2 px-2 py-2 text-sm rounded-md ${active ? 'bg-zinc-100 font-medium' : 'text-zinc-600 hover:bg-zinc-50'}`}>
      <Icon className="w-4 h-4" />
      <span>{label}</span>
    </div>
  )
}
