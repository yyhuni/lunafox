"use client"

import type { Control } from "react-hook-form"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import type { AgentConfigFormValues } from "@/components/settings/workers/worker-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface AgentConfigDialogHeaderProps {
  t: TranslationFn
}

export function AgentConfigDialogHeader({ t }: AgentConfigDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{t("config.title")}</DialogTitle>
      <DialogDescription>{t("config.desc")}</DialogDescription>
    </DialogHeader>
  )
}

interface AgentConfigFormFieldsProps {
  t: TranslationFn
  control: Control<AgentConfigFormValues>
}

export function AgentConfigFormFields({ t, control }: AgentConfigFormFieldsProps) {
  return (
    <>
      <FormField
        control={control}
        name="maxTasks"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("config.maxTasks")}</FormLabel>
            <FormControl>
              <Input type="number" min={1} max={20} inputMode="numeric" autoComplete="off" step={1} {...field} />
            </FormControl>
            <FormDescription>{t("config.maxTasksDesc")}</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
      <div className="grid grid-cols-3 gap-3">
        <FormField
          control={control}
          name="cpuThreshold"
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t("config.cpuThreshold")}</FormLabel>
              <FormControl>
                <Input type="number" min={1} max={100} inputMode="numeric" autoComplete="off" step={1} {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name="memThreshold"
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t("config.memThreshold")}</FormLabel>
              <FormControl>
                <Input type="number" min={1} max={100} inputMode="numeric" autoComplete="off" step={1} {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name="diskThreshold"
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t("config.diskThreshold")}</FormLabel>
              <FormControl>
                <Input type="number" min={1} max={100} inputMode="numeric" autoComplete="off" step={1} {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
    </>
  )
}

interface AgentConfigDialogFooterProps {
  t: TranslationFn
  tCommon: TranslationFn
  isPending: boolean
  onCancel: () => void
}

export function AgentConfigDialogFooter({
  t,
  tCommon,
  isPending,
  onCancel,
}: AgentConfigDialogFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isPending}
      >
        {tCommon("cancel")}
      </Button>
      <Button type="submit" disabled={isPending}>
        {isPending ? t("config.saving") : t("config.save")}
      </Button>
    </DialogFooter>
  )
}
