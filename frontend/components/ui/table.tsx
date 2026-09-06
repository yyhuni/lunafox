"use client"

import * as React from "react"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const TABLE_HEADER_RHYTHM_CLASS = "h-10 px-2"
const TABLE_HEADER_RHYTHM_HEIGHT_PX = 40
// Measured boxes include row borders; visual reservations use the rhythm values.
const TABLE_HEADER_ESTIMATED_HEIGHT_PX = 41
const TABLE_CELL_RHYTHM_CLASS = "p-2"
const TABLE_DENSE_ROW_CLASS = "h-12"
const TABLE_DENSE_CELL_RHYTHM_CLASS = "px-2 py-1"
const TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX = 48
const TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX = 49
const TABLE_COMFORTABLE_ROW_CLASS = "h-16"
const TABLE_COMFORTABLE_CELL_RHYTHM_CLASS = "px-2 py-2"
const TABLE_COMFORTABLE_ROW_RHYTHM_HEIGHT_PX = 64
const TABLE_COMFORTABLE_ROW_ESTIMATED_HEIGHT_PX = 65

function Table({ className, ...props }: React.ComponentProps<"table">) {
  return (
    <div
      data-slot="table-container"
      className="overflow-x-auto relative w-full"
    >
      <table
        data-slot="table"
        className={cn("w-full caption-bottom text-sm", className)}
        {...props}
      />
    </div>
  )
}

function TableHeader({ className, ...props }: React.ComponentProps<"thead">) {
  return (
    <thead
      data-slot="table-header"
      className={cn("bg-secondary [&_tr]:border-b [&_tr]:border-border [&_tr]:bg-secondary", className)}
      {...props}
    />
  )
}

function TableBody({ className, ...props }: React.ComponentProps<"tbody">) {
  return (
    <tbody
      data-slot="table-body"
      className={cn("[&_tr:last-child]:border-0", className)}
      {...props}
    />
  )
}

function TableFooter({ className, ...props }: React.ComponentProps<"tfoot">) {
  return (
    <tfoot
      data-slot="table-footer"
      className={cn(
        "bg-secondary/70 border-t border-border [&>tr]:last:border-b-0",
        textRole.tableCellPrimary,
        className
      )}
      {...props}
    />
  )
}

function TableRow({ className, ...props }: React.ComponentProps<"tr">) {
  return (
    <tr
      data-slot="table-row"
      className={cn(
        "border-b border-border bg-card transition-colors hover:bg-secondary data-[state=selected]:bg-secondary/70",
        className
      )}
      {...props}
    />
  )
}

function TableHead({ className, ...props }: React.ComponentProps<"th">) {
  return (
    <th
      data-slot="table-head"
      className={cn(
        TABLE_HEADER_RHYTHM_CLASS,
        "text-left align-middle whitespace-nowrap [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:align-middle",
        textRole.tableHeader,
        className
      )}
      {...props}
    />
  )
}

function TableCell({ className, ...props }: React.ComponentProps<"td">) {
  const usesSharedTableRhythm = typeof className === "string" && (
    className.includes(TABLE_DENSE_CELL_RHYTHM_CLASS) ||
    className.includes(TABLE_COMFORTABLE_CELL_RHYTHM_CLASS)
  )

  return (
    <td
      data-slot="table-cell"
      className={cn(
        !usesSharedTableRhythm && TABLE_CELL_RHYTHM_CLASS,
        "align-middle overflow-hidden [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:align-middle",
        className
      )}
      {...props}
    />
  )
}

function TableCaption({
  className,
  ...props
}: React.ComponentProps<"caption">) {
  return (
    <caption
      data-slot="table-caption"
      className={cn("mt-4", textRole.bodySubtle, className)}
      {...props}
    />
  )
}

export {
  TABLE_CELL_RHYTHM_CLASS,
  TABLE_COMFORTABLE_CELL_RHYTHM_CLASS,
  TABLE_COMFORTABLE_ROW_CLASS,
  TABLE_COMFORTABLE_ROW_ESTIMATED_HEIGHT_PX,
  TABLE_COMFORTABLE_ROW_RHYTHM_HEIGHT_PX,
  TABLE_DENSE_CELL_RHYTHM_CLASS,
  TABLE_DENSE_ROW_CLASS,
  TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX,
  TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX,
  TABLE_HEADER_ESTIMATED_HEIGHT_PX,
  TABLE_HEADER_RHYTHM_CLASS,
  TABLE_HEADER_RHYTHM_HEIGHT_PX,
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
}
