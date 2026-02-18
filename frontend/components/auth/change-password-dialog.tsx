"use client"

import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import { useChangePasswordDialogState } from "@/components/auth/change-password-dialog-state"
import {
  ChangePasswordDialogHeader,
  ChangePasswordFormFields,
  ChangePasswordError,
  ChangePasswordDialogFooter,
} from "@/components/auth/change-password-dialog-sections"

interface ChangePasswordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ChangePasswordDialog({ open, onOpenChange }: ChangePasswordDialogProps) {
  const t = useTranslations("auth.changePassword")

  const {
    oldPassword,
    newPassword,
    confirmPassword,
    error,
    isPending,
    setOldPassword,
    setNewPassword,
    setConfirmPassword,
    handleSubmit,
    handleOpenChange,
  } = useChangePasswordDialogState({
    onOpenChange,
    t,
  })

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[400px]">
        <ChangePasswordDialogHeader t={t} />
        <form onSubmit={handleSubmit}>
          <ChangePasswordFormFields
            t={t}
            oldPassword={oldPassword}
            newPassword={newPassword}
            confirmPassword={confirmPassword}
            onOldPasswordChange={setOldPassword}
            onNewPasswordChange={setNewPassword}
            onConfirmPasswordChange={setConfirmPassword}
          />
          <ChangePasswordError error={error} />
          <ChangePasswordDialogFooter
            t={t}
            isPending={isPending}
            onCancel={() => handleOpenChange(false)}
          />
        </form>
      </DialogContent>
    </Dialog>
  )
}
