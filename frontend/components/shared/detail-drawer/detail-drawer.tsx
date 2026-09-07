"use client";
import type { ReactNode } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Sheet, SheetClose, SheetContent, SheetDescription, SheetHeader, SheetTitle, } from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { semanticIcons } from "@/components/icons";
import { EdgePanelHeader } from "@/components/shared/edge-panel-header";
import { detailDrawerContentClassName } from "@/lib/ui/overlay-styles";
import { cn } from "@/lib/utils";
interface DetailDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    title: ReactNode;
    description?: ReactNode;
    titleMeta?: ReactNode;
    headerMeta?: ReactNode;
    children: ReactNode;
    sidecar?: ReactNode;
    onSidecarClose?: () => void;
    className?: string;
    contentClassName?: string;
    sidecarClassName?: string;
    headerClassName?: string;
    titleClassName?: string;
}
export function DetailDrawer({ open, onOpenChange, title, description, titleMeta, headerMeta, children, sidecar, onSidecarClose, className, contentClassName, sidecarClassName, headerClassName, titleClassName, }: DetailDrawerProps) {
    const tActions = useTranslations("common.actions");
    const titleText = typeof title === "string" ? title : undefined;
    const descriptionNode = description ? (<SheetDescription className="sr-only">{description}</SheetDescription>) : null;
    return (<Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" showCloseButton={false} className={cn(detailDrawerContentClassName, className)} onEscapeKeyDown={(event) => {
            if (sidecar && onSidecarClose) {
                event.preventDefault();
                onSidecarClose();
            }
        }}>
        <div className={cn("relative flex h-full min-h-0 flex-col bg-card", contentClassName)}>
          {sidecar ? (<div className={cn("absolute inset-0 z-20 flex min-h-0 bg-card", "lg:inset-y-0 lg:left-auto lg:right-full lg:z-10 lg:w-1/2 lg:max-w-lg lg:border-r lg:shadow-lg", "lg:transform-gpu lg:animate-in lg:fade-in-0 lg:slide-in-from-right-2 lg:duration-[var(--motion-duration-enter)]", sidecarClassName)}>
              {sidecar}
            </div>) : null}

          <div className="flex h-full min-h-0 flex-1 flex-col bg-card">
            <SheetHeader className={cn("border-b px-6 py-4 text-left", headerClassName)}>
              <EdgePanelHeader
                variant="detail"
                title={<SheetTitle className={cn("min-w-0 truncate", titleClassName)} title={titleText}>{title}</SheetTitle>}
                description={descriptionNode}
                titleMeta={titleMeta}
                headerMeta={headerMeta}
                actions={(
                  <SheetClose render={<Button type="button" variant="ghost" size="icon-sm" aria-label={tActions("close")} className="shrink-0"/>}>
                    <semanticIcons.action.cancel />
                  </SheetClose>
                )}
              />
            </SheetHeader>
            {children}
          </div>
        </div>
      </SheetContent>
    </Sheet>);
}
type DetailDrawerTabsProps = React.ComponentProps<typeof Tabs>;
export function DetailDrawerTabs({ className, ...props }: DetailDrawerTabsProps) {
    return (<Tabs className={cn("flex min-h-0 flex-1 flex-col gap-0", className)} {...props}/>);
}
type DetailDrawerTabsListProps = React.ComponentProps<typeof TabsList>;
export function DetailDrawerTabsList({ className, size = "md", variant = "content", ...props }: DetailDrawerTabsListProps) {
    return (<TabsList variant={variant} size={size} className={cn(variant === "content" && "border-border/70", variant === "content" && size === "sm" && "gap-2", variant === "content" && size === "md" && "gap-5", className)} {...props}/>);
}
type DetailDrawerTabsTriggerProps = React.ComponentProps<typeof TabsTrigger>;
export function DetailDrawerTabsTrigger({ className, size = "md", variant = "content", ...props }: DetailDrawerTabsTriggerProps) {
    return (<TabsTrigger variant={variant} size={size} className={cn(variant === "content" && "text-muted-foreground hover:border-border hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/40 data-active:border-foreground data-active:text-foreground", className)} {...props}/>);
}
type DetailDrawerTabsContentProps = React.ComponentProps<typeof TabsContent>;
export function DetailDrawerTabsContent({ className, ...props }: DetailDrawerTabsContentProps) {
    return <TabsContent className={cn("mt-0", className)} {...props}/>;
}
