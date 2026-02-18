import React from "react"
import { AlertTriangle, IconPlus, IconX } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { DialogFooter } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { LoadingSpinner } from "@/components/loading-spinner"
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import type { UseFormReturn } from "react-hook-form"
import type { AddToolFormValues } from "@/components/tools/config/add-tool-dialog-state"
import { CategoryNameMap } from "@/types/tool.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface ToolCategorySelectorProps {
  t: TranslationFn
  availableCategories: string[]
  selectedCategories: string[]
  disabled: boolean
  onToggle: (name: string) => void
  onRemove: (name: string) => void
}

function ToolCategorySelector({
  t,
  availableCategories,
  selectedCategories,
  disabled,
  onToggle,
  onRemove,
}: ToolCategorySelectorProps) {
  return (
    <div className="grid gap-2">
      <FormLabel>{t("categoryTags")}</FormLabel>

      {selectedCategories.length > 0 && (
        <div className="flex flex-wrap gap-2 p-3 border rounded-md bg-muted/50">
          {selectedCategories.map((categoryName) => (
            <Badge
              key={categoryName}
              variant="default"
              className="flex items-center gap-1 px-2 py-1"
            >
              {CategoryNameMap[categoryName] || categoryName}
              <button
                type="button"
                onClick={() => onRemove(categoryName)}
                disabled={disabled}
                className="ml-1 hover:bg-primary/20 rounded-full p-0.5"
              >
                <IconX className="h-3 w-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}

      <div className="flex flex-wrap gap-2 p-3 border rounded-md">
        {availableCategories.length > 0 ? (
          availableCategories.map((categoryName) => {
            const isSelected = selectedCategories.includes(categoryName)
            return (
              <Badge
                asChild
                key={categoryName}
                variant={isSelected ? "secondary" : "outline"}
                className="hover:bg-secondary/80 transition-colors"
              >
                <button
                  type="button"
                  onClick={() => onToggle(categoryName)}
                >
                  {CategoryNameMap[categoryName] || categoryName}
                </button>
              </Badge>
            )
          })
        ) : (
          <p className="text-sm text-muted-foreground">{t("noCategories")}</p>
        )}
      </div>
    </div>
  )
}

interface AddToolBasicInfoSectionProps {
  t: TranslationFn
  form: UseFormReturn<AddToolFormValues>
  isPending: boolean
  availableCategories: string[]
  selectedCategories: string[]
  onToggleCategory: (name: string) => void
  onRemoveCategory: (name: string) => void
}

export function AddToolBasicInfoSection({
  t,
  form,
  isPending,
  availableCategories,
  selectedCategories,
  onToggleCategory,
  onRemoveCategory,
}: AddToolBasicInfoSectionProps) {
  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-muted-foreground">{t("basicInfo")}</h3>

      <FormField
        control={form.control}
        name="name"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("toolName")} <span className="text-destructive">*</span></FormLabel>
            <FormControl>
              <Input
                placeholder={t("toolNamePlaceholder")}
                disabled={isPending}
                maxLength={255}
                autoComplete="off"
                {...field}
              />
            </FormControl>
            <FormDescription>{t("characters", { count: field.value.length, max: 255 })}</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="repoUrl"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("repoUrl")}</FormLabel>
            <FormControl>
              <Input
                type="url"
                placeholder={t("repoUrlPlaceholder")}
                disabled={isPending}
                maxLength={512}
                autoComplete="url"
                inputMode="url"
                {...field}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="version"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("currentVersion")}</FormLabel>
            <FormControl>
              <Input
                placeholder={t("versionPlaceholder")}
                disabled={isPending}
                maxLength={100}
                autoComplete="off"
                {...field}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="description"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("toolDesc")}</FormLabel>
            <FormControl>
              <Textarea
                placeholder={t("toolDescPlaceholder")}
                disabled={isPending}
                rows={3}
                maxLength={1000}
                autoComplete="off"
                {...field}
              />
            </FormControl>
            <FormDescription>{t("characters", { count: (field.value || "").length, max: 1000 })}</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <ToolCategorySelector
        t={t}
        availableCategories={availableCategories}
        selectedCategories={selectedCategories}
        disabled={isPending}
        onToggle={onToggleCategory}
        onRemove={onRemoveCategory}
      />
    </div>
  )
}

interface AddToolCommandSectionProps {
  t: TranslationFn
  form: UseFormReturn<AddToolFormValues>
  isPending: boolean
}

export function AddToolCommandSection({
  t,
  form,
  isPending,
}: AddToolCommandSectionProps) {
  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-muted-foreground">{t("commandConfig")}</h3>

      <FormField
        control={form.control}
        name="installCommand"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("installCommand")} <span className="text-destructive">*</span></FormLabel>
            <FormControl>
              <Textarea
                placeholder={t("installCommandPlaceholder")}
                disabled={isPending}
                rows={3}
                className="font-mono text-sm"
                autoComplete="off"
                {...field}
              />
            </FormControl>
            <FormDescription className="space-y-1">
              <span className="block"><strong>{t("installCommandHint")}</strong></span>
              <span className="block">• {t("installCommandGit")} <code className="bg-muted px-1 py-0.5 rounded">git clone https://github.com/user/tool</code></span>
              <span className="block">• {t("installCommandGo")} <code className="bg-muted px-1 py-0.5 rounded">go install -v github.com/tool@v1.2.3</code></span>
              <span className="flex items-center gap-1 text-amber-600">
                <AlertTriangle className="h-3.5 w-3.5" />
                {t("installCommandNote")}
              </span>
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="updateCommand"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("updateCommand")} <span className="text-destructive">*</span></FormLabel>
            <FormControl>
              <Textarea
                placeholder={t("updateCommandPlaceholder")}
                disabled={isPending}
                rows={2}
                className="font-mono text-sm"
                autoComplete="off"
                {...field}
              />
            </FormControl>
            <FormDescription className="space-y-1">
              <span className="block">• {t("updateCommandGitHint")} <code className="bg-muted px-1 py-0.5 rounded">git pull</code></span>
              <span className="block">• {t("updateCommandGoHint")}</span>
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="versionCommand"
        render={({ field }) => (
          <FormItem>
            <FormLabel>
              {t("versionCommand")} <span className="text-destructive">*</span>
              {field.value && (
                <span className="ml-2 text-xs text-muted-foreground font-normal">
                  {t("versionCommandAutoGenerated")}
                </span>
              )}
            </FormLabel>
            <FormControl>
              <Input
                placeholder={t("versionCommandPlaceholder")}
                disabled={isPending}
                maxLength={500}
                className="font-mono text-sm"
                autoComplete="off"
                {...field}
              />
            </FormControl>
            <FormDescription className="space-y-1">
              <span className="block">{t("versionCommandHint")}</span>
              <span className="block">• <code className="bg-muted px-1 py-0.5 rounded">toolname -v</code></span>
              <span className="block">• <code className="bg-muted px-1 py-0.5 rounded">toolname -V</code></span>
              <span className="block">• <code className="bg-muted px-1 py-0.5 rounded">toolname --version</code></span>
              <span className="block">• <code className="bg-muted px-1 py-0.5 rounded">python tool_name.py -v</code></span>
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  )
}

interface AddToolDialogFooterProps {
  t: TranslationFn
  isEditMode: boolean
  isPending: boolean
  isFormValid: boolean
  onCancel: () => void
}

export function AddToolDialogFooter({
  t,
  isEditMode,
  isPending,
  isFormValid,
  onCancel,
}: AddToolDialogFooterProps) {
  return (
    <DialogFooter>
      <Button type="button" variant="outline" onClick={onCancel} disabled={isPending}>
        {t("cancel")}
      </Button>
      <Button type="submit" disabled={isPending || !isFormValid}>
        {isPending ? (
          <>
            <LoadingSpinner />
            {isEditMode ? t("saving") : t("creating")}
          </>
        ) : (
          <>
            <IconPlus className="h-5 w-5" />
            {isEditMode ? t("saveChanges") : t("createTool")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
