"use client"

import React from "react"
import { toast } from "sonner"
import { DemoCard, DemoSection } from "@/components/demo/component-gallery-sections/shared"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { ConfirmDialog } from "@/components/ui/confirm-dialog"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer"
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { CardGridSkeleton } from "@/components/ui/card-grid-skeleton"
import { MasterDetailSkeleton } from "@/components/ui/master-detail-skeleton"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { Spinner } from "@/components/ui/spinner"
import { ShieldLoader } from "@/components/ui/shield-loader"
import { CopyablePopoverContent } from "@/components/ui/copyable-popover-content"
import { AlertTriangle } from "@/components/icons"
import { WaveGrid } from "@/components/ui/wave-grid"

export function UiOverlaysSection() {
  const [confirmOpen, setConfirmOpen] = React.useState(false)
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [drawerOpen, setDrawerOpen] = React.useState(false)
  const [sheetOpen, setSheetOpen] = React.useState(false)

  return (
    <DemoSection
      id="ui-overlays"
      title="反馈与浮层"
      description="弹窗、提示、反馈与状态组件。"
    >
      <div className="grid gap-4 px-4 lg:px-6 md:grid-cols-2 xl:grid-cols-3">
        <DemoCard title="Alert" description="信息提示">
          <Alert>
            <AlertTriangle className="size-4" />
            <AlertTitle>扫描警告</AlertTitle>
            <AlertDescription>目标存在高危端口，建议开启深度扫描。</AlertDescription>
          </Alert>
        </DemoCard>

        <DemoCard title="AlertDialog" description="确认型弹窗">
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="outline">打开 AlertDialog</Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>确定执行删除？</AlertDialogTitle>
                <AlertDialogDescription>该操作不可撤销。</AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction>确认</AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </DemoCard>

        <DemoCard title="ConfirmDialog" description="业务确认弹窗">
          <div className="space-y-2">
            <Button onClick={() => setConfirmOpen(true)}>打开 ConfirmDialog</Button>
            <ConfirmDialog
              open={confirmOpen}
              onOpenChange={setConfirmOpen}
              title="提交扫描任务"
              description="提交后将进入队列。"
              onConfirm={() => setConfirmOpen(false)}
            />
          </div>
        </DemoCard>

        <DemoCard title="Dialog" description="通用对话框">
          <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
            <DialogTrigger asChild>
              <Button variant="outline">打开 Dialog</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>新建目标</DialogTitle>
                <DialogDescription>快速添加一个新的资产目标。</DialogDescription>
              </DialogHeader>
              <Input autoComplete="off" name="demoInput" placeholder="example.com" />
              <DialogFooter>
                <Button onClick={() => setDialogOpen(false)}>保存</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </DemoCard>

        <DemoCard title="Sheet" description="侧边面板">
          <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
            <SheetTrigger asChild>
              <Button variant="outline">打开 Sheet</Button>
            </SheetTrigger>
            <SheetContent side="right">
              <SheetHeader>
                <SheetTitle>配置面板</SheetTitle>
                <SheetDescription>快速调整扫描策略。</SheetDescription>
              </SheetHeader>
              <div className="mt-4 space-y-2">
                <Label>策略名称</Label>
                <Input autoComplete="off" name="demoInput" placeholder="Default" />
              </div>
            </SheetContent>
          </Sheet>
        </DemoCard>

        <DemoCard title="Drawer" description="抽屉式交互">
          <Drawer open={drawerOpen} onOpenChange={setDrawerOpen}>
            <DrawerTrigger asChild>
              <Button variant="outline">打开 Drawer</Button>
            </DrawerTrigger>
            <DrawerContent>
              <DrawerHeader>
                <DrawerTitle>快速执行</DrawerTitle>
                <DrawerDescription>启动一个快速扫描任务。</DrawerDescription>
              </DrawerHeader>
              <DrawerFooter className="pb-6">
                <Button onClick={() => setDrawerOpen(false)}>启动</Button>
              </DrawerFooter>
            </DrawerContent>
          </Drawer>
        </DemoCard>

        <DemoCard title="Popover" description="浮层内容">
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="outline">打开 Popover</Button>
            </PopoverTrigger>
            <PopoverContent className="w-64">
              <CopyablePopoverContent value="https://example.com/asset/alpha" />
            </PopoverContent>
          </Popover>
        </DemoCard>

        <DemoCard title="HoverCard" description="悬浮卡片">
          <HoverCard>
            <HoverCardTrigger asChild>
              <Button variant="ghost">Hover 预览</Button>
            </HoverCardTrigger>
            <HoverCardContent className="w-64">
              <div className="space-y-1">
                <p className="text-sm font-medium">资产状态</p>
                <p className="text-xs text-muted-foreground">最近一次扫描：3 分钟前</p>
              </div>
            </HoverCardContent>
          </HoverCard>
        </DemoCard>

        <DemoCard title="Tooltip" description="轻提示">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="outline">提示</Button>
              </TooltipTrigger>
              <TooltipContent>这是一个 Tooltip</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </DemoCard>

        <DemoCard title="DropdownMenu" description="下拉菜单">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline">操作</Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuLabel>快速操作</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem>开始扫描</DropdownMenuItem>
              <DropdownMenuItem>查看详情</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </DemoCard>

        <DemoCard title="Toast (Sonner)" description="轻量通知">
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => toast.success("任务已提交")}
            >
              Success
            </Button>
            <Button
              variant="outline"
              onClick={() => toast.error("请求失败")}
            >
              Error
            </Button>
          </div>
        </DemoCard>

        <DemoCard title="Progress + Spinner" description="进度指示">
          <div className="space-y-3">
            <Progress value={65} />
            <div className="flex items-center gap-2">
              <Spinner />
              <span className="text-xs text-muted-foreground">处理中…</span>
            </div>
          </div>
        </DemoCard>

        <DemoCard title="Skeletons" description="骨架屏">
          <div className="space-y-3">
            <Skeleton className="h-6 w-1/2" />
            <CardGridSkeleton />
          </div>
        </DemoCard>

        <DemoCard title="Master Detail Skeleton" description="主从骨架">
          <MasterDetailSkeleton />
        </DemoCard>

        <DemoCard title="Data Table Skeleton" description="表格骨架">
          <DataTableSkeleton />
        </DemoCard>

        <DemoCard title="Shield Loader" description="工业风加载">
          <div className="flex justify-center">
            <ShieldLoader />
          </div>
        </DemoCard>

        <DemoCard title="Wave Grid" description="网格波纹">
          <WaveGrid />
        </DemoCard>
      </div>
    </DemoSection>
  )
}
