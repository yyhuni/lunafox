import React from "react"
import { Building2, Check, ChevronsUpDown, Loader2, Plus } from "@/components/icons"
import { IconChevronLeft, IconChevronRight, IconChevronsLeft, IconChevronsRight } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { DialogFooter } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { LoadingSpinner } from "@/components/loading-spinner"
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"
import type { PaginationInfo } from "@/types/common.types"
import type { Organization } from "@/types/organization.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface InvalidTargetInfo {
  index: number
  originalTarget: string
  error: string
}

interface AddTargetInputSectionProps {
  t: TranslationFn
  formTargets: string
  targetCount: number
  invalidTargets: InvalidTargetInfo[]
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  onInputChange: (value: string) => void
  onScroll: (event: React.UIEvent<HTMLTextAreaElement>) => void
  isPending: boolean
}

export function AddTargetInputSection({
  t,
  formTargets,
  targetCount,
  invalidTargets,
  lineNumbersRef,
  textareaRef,
  onInputChange,
  onScroll,
  isPending,
}: AddTargetInputSectionProps) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="targets">
        {t("targetList")} <span className="text-destructive">*</span>
      </Label>
      <div className="flex border rounded-md overflow-hidden h-[180px]">
        <div className="flex-shrink-0 w-12 border-r bg-muted/50">
          <div
            ref={lineNumbersRef}
            className="py-3 px-2 text-right font-mono text-xs text-muted-foreground leading-[1.4] h-full overflow-y-auto scrollbar-hide"
          >
            {Array.from({ length: Math.max(formTargets.split("\n").length, 8) }, (_, i) => (
              <div key={i + 1} className="h-[20px]">
                {i + 1}
              </div>
            ))}
          </div>
        </div>
        <div className="flex-1 overflow-hidden">
          <Textarea
            ref={textareaRef}
            id="targets"
            name="targets"
            autoComplete="off"
            value={formTargets}
            onChange={(event) => onInputChange(event.target.value)}
            onScroll={onScroll}
            placeholder={t("targetPlaceholder")}
            disabled={isPending}
            className="font-mono h-full overflow-y-auto resize-none border-0 focus-visible:ring-0 focus-visible:ring-offset-0 leading-[1.4] text-sm py-3"
            style={{ lineHeight: "20px" }}
          />
        </div>
      </div>
      <div className="text-xs text-muted-foreground">
        {t("targetCount", { count: targetCount })}
      </div>
      {invalidTargets.length > 0 && (
        <div className="text-xs text-destructive">
          {t("invalidCount", {
            count: invalidTargets.length,
            line: invalidTargets[0].index + 1,
            target: invalidTargets[0].originalTarget,
            error: invalidTargets[0].error,
          })}
        </div>
      )}
    </div>
  )
}

interface AddTargetOrganizationPickerProps {
  t: TranslationFn
  tPagination: TranslationFn
  isLoading: boolean
  orgPickerOpen: boolean
  handleOrgPickerOpenChange: (open: boolean) => void
  formOrganizationId?: string
  selectedOrgName: string
  organizations: Organization[]
  organizationPaginationInfo?: PaginationInfo
  orgSearchQuery: string
  setOrgSearchQuery: (value: string) => void
  orgPage: number
  setOrgPage: React.Dispatch<React.SetStateAction<number>>
  orgPageSize: number
  setOrgPageSize: React.Dispatch<React.SetStateAction<number>>
  pageSizeOptions: number[]
  onSelectOrganization: (id: string, name: string) => void
  isPending: boolean
  onOpenPicker: () => void
}

export function AddTargetOrganizationPicker({
  t,
  tPagination,
  isLoading,
  orgPickerOpen,
  handleOrgPickerOpenChange,
  formOrganizationId,
  selectedOrgName,
  organizations,
  organizationPaginationInfo,
  orgSearchQuery,
  setOrgSearchQuery,
  orgPage,
  setOrgPage,
  orgPageSize,
  setOrgPageSize,
  pageSizeOptions,
  onSelectOrganization,
  isPending,
  onOpenPicker,
}: AddTargetOrganizationPickerProps) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="organization">
        {t("linkOrganization")}
      </Label>
      <Button
        type="button"
        variant="outline"
        role="combobox"
        className="w-full justify-between"
        onClick={onOpenPicker}
        disabled={isPending || isLoading}
      >
        {isLoading ? (
          <span className="flex items-center gap-2">
            <Loader2 className="h-4 w-4 animate-spin" />
            {t("loading")}
          </span>
        ) : formOrganizationId ? (
          <span className="flex items-center gap-2">
            <Building2 className="h-4 w-4" />
            <span className="truncate">{selectedOrgName}</span>
          </span>
        ) : (
          t("selectOrganization")
        )}
        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
      </Button>
      <CommandDialog
        open={orgPickerOpen}
        onOpenChange={handleOrgPickerOpenChange}
      >
        <CommandInput
          placeholder={t("searchOrganization")}
          value={orgSearchQuery}
          onValueChange={(value) => setOrgSearchQuery(value)}
        />
        <CommandList className="max-h-[300px] overflow-y-auto overscroll-contain">
          {isLoading ? (
            <div className="py-6 text-center text-sm">
              <Loader2 className="mx-auto h-4 w-4 animate-spin" />
            </div>
          ) : organizations.length === 0 ? (
            <CommandEmpty>{t("noOrganization")}</CommandEmpty>
          ) : (
            <CommandGroup>
              <div className="grid grid-cols-2 gap-1 p-1">
                {organizations.map((org) => (
                  <CommandItem
                    key={org.id}
                    value={org.id.toString()}
                    onSelect={() => onSelectOrganization(org.id.toString(), org.name)}
                    className="cursor-pointer"
                  >
                    <Check
                      className={cn(
                        "mr-1 h-3.5 w-3.5 flex-shrink-0",
                        formOrganizationId === org.id.toString()
                          ? "opacity-100"
                          : "opacity-0"
                      )}
                    />
                    <Building2 className="mr-1 h-3.5 w-3.5 flex-shrink-0" />
                    <span className="font-medium text-sm truncate">{org.name}</span>
                  </CommandItem>
                ))}
              </div>
            </CommandGroup>
          )}
        </CommandList>
        {organizationPaginationInfo && (
          <div className="flex items-center justify-between border-t p-2 bg-muted/50">
            <div className="text-xs text-muted-foreground">
              {t("orgPagination", {
                total: organizationPaginationInfo.total,
                page: organizationPaginationInfo.page,
                totalPages: organizationPaginationInfo.totalPages,
              })}
            </div>
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1">
                <span className="text-xs text-muted-foreground">{t("perPage")}</span>
                <Select
                  value={orgPageSize.toString()}
                  onValueChange={(value) => {
                    setOrgPageSize(Number(value))
                    setOrgPage(1)
                  }}
                >
                  <SelectTrigger className="h-7 w-16 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {pageSizeOptions.map((size) => (
                      <SelectItem key={size} value={size.toString()}>
                        {size}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex items-center space-x-2">
                <Button
                  type="button"
                  variant="outline"
                  className="hidden h-8 w-8 p-0 lg:flex"
                  onClick={() => setOrgPage(1)}
                  disabled={orgPage === 1 || isLoading}
                >
                  <span className="sr-only">{tPagination("first")}</span>
                  <IconChevronsLeft />
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="h-8 w-8 p-0"
                  onClick={() => setOrgPage((prev) => Math.max(1, prev - 1))}
                  disabled={orgPage === 1 || isLoading}
                >
                  <span className="sr-only">{tPagination("previous")}</span>
                  <IconChevronLeft />
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="h-8 w-8 p-0"
                  onClick={() => setOrgPage((prev) => Math.min(organizationPaginationInfo.totalPages, prev + 1))}
                  disabled={orgPage === organizationPaginationInfo.totalPages || isLoading}
                >
                  <span className="sr-only">{tPagination("next")}</span>
                  <IconChevronRight />
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="hidden h-8 w-8 p-0 lg:flex"
                  onClick={() => setOrgPage(organizationPaginationInfo.totalPages)}
                  disabled={orgPage === organizationPaginationInfo.totalPages || isLoading}
                >
                  <span className="sr-only">{tPagination("last")}</span>
                  <IconChevronsRight />
                </Button>
              </div>
            </div>
          </div>
        )}
      </CommandDialog>
    </div>
  )
}

interface AddTargetDialogFooterProps {
  t: TranslationFn
  isPending: boolean
  isFormValid: boolean
  onCancel: () => void
}

export function AddTargetDialogFooter({
  t,
  isPending,
  isFormValid,
  onCancel,
}: AddTargetDialogFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isPending}
      >
        {t("cancel")}
      </Button>
      <Button
        type="submit"
        disabled={isPending || !isFormValid}
      >
        {isPending ? (
          <>
            <LoadingSpinner />
            {t("creating")}
          </>
        ) : (
          <>
            <Plus />
            {t("addTarget")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
