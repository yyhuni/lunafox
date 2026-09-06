"use client";
import * as React from "react";
import { IconDotsVertical } from "@/components/icons";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger, } from "@/components/ui/dropdown-menu";
import { SidebarMenuButton, useSidebar } from "@/components/ui/sidebar";
import { shellOverlaySideOffsets } from "@/lib/ui/overlay-styles";
type DropdownAlign = "start" | "center" | "end";
type DropdownSide = "top" | "right" | "bottom" | "left";
type ButtonSize = React.ComponentProps<typeof Button>["size"];
type ButtonVariant = React.ComponentProps<typeof Button>["variant"];
export interface HeaderIconActionMenuProps {
    ariaLabel: string;
    icon: React.ReactNode;
    children: React.ReactNode;
    align?: DropdownAlign;
    side?: DropdownSide;
    sideOffset?: number;
    buttonSize?: ButtonSize;
    buttonVariant?: ButtonVariant;
    contentClassName?: string;
    disabled?: boolean;
}
export function HeaderIconActionMenu({ ariaLabel, icon, children, align = "end", side, sideOffset = shellOverlaySideOffsets.header, buttonSize = "icon-sm", buttonVariant = "ghost", contentClassName, disabled, }: HeaderIconActionMenuProps) {
    return (<DropdownMenu>
      <DropdownMenuTrigger render={<Button variant={buttonVariant} size={buttonSize} aria-label={ariaLabel} disabled={disabled}/>}> 
          {icon}
          <span className="sr-only">{ariaLabel}</span>
        </DropdownMenuTrigger>
      <DropdownMenuContent align={align} side={side} sideOffset={sideOffset} width="content-fit" className={contentClassName}>
        {children}
      </DropdownMenuContent>
    </DropdownMenu>);
}
export interface SidebarUserMenuProps {
    userName: React.ReactNode;
    userSubline?: React.ReactNode;
    avatarSrc?: string;
    avatarAlt: string;
    avatarFallback: React.ReactNode;
    children: React.ReactNode;
    contentClassName?: string;
    sideOffset?: number;
    triggerClassName?: string;
}
function SidebarUserSummary({ userName, userSubline, avatarSrc, avatarAlt, avatarFallback, avatarClassName, }: Omit<SidebarUserMenuProps, "children" | "contentClassName" | "sideOffset" | "triggerClassName"> & {
    avatarClassName?: string;
}) {
    return (<>
      <Avatar className={cn("relative z-10 h-8 w-8 rounded-lg", avatarClassName)}>
        {avatarSrc ? <AvatarImage src={avatarSrc} alt={avatarAlt}/> : null}
        <AvatarFallback className="rounded-lg">{avatarFallback}</AvatarFallback>
      </Avatar>
      <div className="relative z-10 grid flex-1 text-left leading-tight">
        <span data-slot="sidebar-user-name" className={cn(textRole.navLabel, "truncate text-sidebar-foreground group-hover/sidebar-user:text-sidebar-accent-foreground group-data-[popup-open]/sidebar-user:text-sidebar-accent-foreground")}>
          {userName}
        </span>
        {userSubline ? (<span data-slot="sidebar-user-subline" className={cn(textRole.helperText, "truncate text-sidebar-foreground/75 group-hover/sidebar-user:text-sidebar-accent-foreground group-data-[popup-open]/sidebar-user:text-sidebar-accent-foreground")}>
            {userSubline}
          </span>) : null}
      </div>
    </>);
}
export function SidebarUserMenu({ userName, userSubline, avatarSrc, avatarAlt, avatarFallback, children, contentClassName, sideOffset, triggerClassName, }: SidebarUserMenuProps) {
    const { isMobile } = useSidebar();
    // The desktop trigger is inset by the sidebar footer; the generic 4px
    // popup offset would otherwise place the account menu against the shell edge.
    const resolvedSideOffset = sideOffset ?? (isMobile ? undefined : shellOverlaySideOffsets.sidebarDesktop);
    return (<DropdownMenu>
      <DropdownMenuTrigger render={<SidebarMenuButton size="lg" className={cn("group/sidebar-user", "data-[popup-open]:bg-sidebar-accent data-[popup-open]:text-sidebar-accent-foreground", triggerClassName)}/>}>
          <SidebarUserSummary userName={userName} userSubline={userSubline} avatarSrc={avatarSrc} avatarAlt={avatarAlt} avatarFallback={avatarFallback} avatarClassName={avatarSrc ? "grayscale" : undefined}/>
          <IconDotsVertical className="relative z-10 ml-auto size-4 text-sidebar-foreground/75 group-hover/sidebar-user:text-sidebar-accent-foreground group-data-[popup-open]/sidebar-user:text-sidebar-accent-foreground"/>
        </DropdownMenuTrigger>
      <DropdownMenuContent side={isMobile ? "bottom" : "right"} align="end" sideOffset={resolvedSideOffset} className={cn("rounded-lg", contentClassName)}>
        <DropdownMenuLabel className="p-0 font-normal">
          <div className="flex items-center gap-2 px-1 py-1.5 text-left">
            <SidebarUserSummary userName={userName} userSubline={userSubline} avatarSrc={avatarSrc} avatarAlt={avatarAlt} avatarFallback={avatarFallback}/>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {children}
      </DropdownMenuContent>
    </DropdownMenu>);
}

export function SidebarUserMenuSkeleton() {
    return (<SidebarMenuButton size="lg" className="group/sidebar-user" aria-disabled="true" tabIndex={-1}>
      <SidebarUserSummary userName={<span className="loading-skeleton inline-block h-3.5 w-20 rounded-full align-middle" aria-hidden="true"/>} userSubline={<span className="loading-skeleton inline-block h-3 w-24 rounded-full align-middle" aria-hidden="true"/>} avatarAlt="" avatarFallback={<span className="loading-skeleton block size-full rounded-lg" aria-hidden="true"/>}/>
      <IconDotsVertical className="relative z-10 ml-auto size-4 text-sidebar-foreground/35" aria-hidden="true"/>
    </SidebarMenuButton>);
}
