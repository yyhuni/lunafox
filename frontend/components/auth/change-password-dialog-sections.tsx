"use client"

import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface ChangePasswordDialogHeaderProps {
  t: TranslationFn
}

export function ChangePasswordDialogHeader({ t }: ChangePasswordDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{t("title")}</DialogTitle>
      <DialogDescription>{t("desc")}</DialogDescription>
    </DialogHeader>
  )
}

interface ChangePasswordFormFieldsProps {
  t: TranslationFn
  oldPassword: string
  newPassword: string
  confirmPassword: string
  onOldPasswordChange: (value: string) => void
  onNewPasswordChange: (value: string) => void
  onConfirmPasswordChange: (value: string) => void
}

export function ChangePasswordFormFields({
  t,
  oldPassword,
  newPassword,
  confirmPassword,
  onOldPasswordChange,
  onNewPasswordChange,
  onConfirmPasswordChange,
}: ChangePasswordFormFieldsProps) {
  return (
    <div className="grid gap-4 py-4">
      <div className="grid gap-2">
        <Label htmlFor="oldPassword">{t("currentPassword")}</Label>
        <Input
          id="oldPassword"
          name="currentPassword"
          type="password"
          autoComplete="current-password"
          value={oldPassword}
          onChange={(event) => onOldPasswordChange(event.target.value)}
          required
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="newPassword">{t("newPassword")}</Label>
        <Input
          id="newPassword"
          name="newPassword"
          type="password"
          autoComplete="new-password"
          value={newPassword}
          onChange={(event) => onNewPasswordChange(event.target.value)}
          required
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="confirmPassword">{t("confirmPassword")}</Label>
        <Input
          id="confirmPassword"
          name="confirmPassword"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(event) => onConfirmPasswordChange(event.target.value)}
          required
        />
      </div>
    </div>
  )
}

interface ChangePasswordErrorProps {
  error: string
}

export function ChangePasswordError({ error }: ChangePasswordErrorProps) {
  if (!error) return null

  return (
    <p className="text-sm text-destructive">{error}</p>
  )
}

interface ChangePasswordDialogFooterProps {
  t: TranslationFn
  isPending: boolean
  onCancel: () => void
}

export function ChangePasswordDialogFooter({
  t,
  isPending,
  onCancel,
}: ChangePasswordDialogFooterProps) {
  return (
    <DialogFooter>
      <Button type="button" variant="outline" onClick={onCancel}>
        {t("cancel")}
      </Button>
      <Button type="submit" disabled={isPending}>
        {isPending ? t("saving") : t("save")}
      </Button>
    </DialogFooter>
  )
}
