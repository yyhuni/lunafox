"use client";
import * as React from "react";
import type { Table } from "@tanstack/react-table";
import { IconChevronDown, IconLayoutColumns, MoreHorizontal, } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger, } from "@/components/ui/dropdown-menu";
import { DenseRowActionOwner } from "./row-actions";
type DropdownAlign = "start" | "center" | "end";
type DropdownWidth = "default" | "content-fit";
type ButtonSize = React.ComponentProps<typeof Button>["size"];
type ButtonVariant = React.ComponentProps<typeof Button>["variant"];
export interface ToolbarActionMenuProps {
    label: React.ReactNode;
    children: React.ReactNode;
    icon?: React.ReactNode;
    ariaLabel?: string;
    align?: DropdownAlign;
    buttonSize?: ButtonSize;
    buttonVariant?: ButtonVariant;
    menuClassName?: string;
    menuWidth?: DropdownWidth;
}
export function ToolbarActionMenu({ label, children, icon, ariaLabel, align = "end", buttonSize = "sm", buttonVariant = "outline", menuClassName, menuWidth = "content-fit", }: ToolbarActionMenuProps) {
    return (<DropdownMenu>
      <DropdownMenuTrigger render={<Button variant={buttonVariant} size={buttonSize} aria-label={ariaLabel}/>}> 
          {icon}
          {label}
          <IconChevronDown className="h-4 w-4"/>
        </DropdownMenuTrigger>
      <DropdownMenuContent align={align} width={menuWidth} className={menuClassName}>
        {children}
      </DropdownMenuContent>
    </DropdownMenu>);
}
export interface DenseRowActionMenuProps {
    ariaLabel: string;
    children: React.ReactNode;
    leadingActions?: React.ReactNode;
    align?: DropdownAlign;
    menuClassName?: string;
    ownerClassName?: string;
    triggerSize?: ButtonSize;
}
export function DenseRowActionMenu({ ariaLabel, children, leadingActions, align = "end", menuClassName, ownerClassName, triggerSize = "icon-sm", }: DenseRowActionMenuProps) {
    return (<DenseRowActionOwner className={ownerClassName}>
      {leadingActions}
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size={triggerSize} aria-label={ariaLabel}/>}> 
            <MoreHorizontal className="h-4 w-4"/>
            <span className="sr-only">{ariaLabel}</span>
          </DropdownMenuTrigger>
        <DropdownMenuContent align={align} width="compact" className={menuClassName}>
          {children}
        </DropdownMenuContent>
      </DropdownMenu>
    </DenseRowActionOwner>);
}
export interface ColumnVisibilityMenuProps<TData> {
    table: Table<TData>;
    label: React.ReactNode;
    align?: DropdownAlign;
    buttonSize?: ButtonSize;
    buttonVariant?: ButtonVariant;
    getColumnLabel?: (column: {
        id: string;
        columnDef: {
            meta?: {
                title?: string;
            };
        };
    }) => React.ReactNode;
}
function defaultColumnLabel(column: {
    id: string;
    columnDef: {
        meta?: {
            title?: string;
        };
    };
}) {
    return column.columnDef.meta?.title ?? column.id;
}
export function ColumnVisibilityMenu<TData>({ table, label, align = "end", buttonSize = "sm", buttonVariant = "outline", getColumnLabel = defaultColumnLabel, }: ColumnVisibilityMenuProps<TData>) {
    return (<DropdownMenu>
      <DropdownMenuTrigger render={<Button variant={buttonVariant} size={buttonSize}/>}> 
          <IconLayoutColumns className="h-4 w-4"/>
          {label}
          <IconChevronDown className="h-4 w-4"/>
        </DropdownMenuTrigger>
      <DropdownMenuContent align={align} width="content-fit">
        {table
            .getAllColumns()
            .filter((column) => typeof column.accessorFn !== "undefined" && column.getCanHide())
            .map((column) => (<DropdownMenuCheckboxItem key={column.id} className="capitalize" checked={column.getIsVisible()} onCheckedChange={(value) => column.toggleVisibility(!!value)}>
              {getColumnLabel(column)}
            </DropdownMenuCheckboxItem>))}
      </DropdownMenuContent>
    </DropdownMenu>);
}
