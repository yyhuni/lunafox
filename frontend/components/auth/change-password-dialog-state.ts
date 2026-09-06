import React from "react"
import { useChangePassword } from "@/hooks/use-auth"
import { getErrorMessage } from "@/lib/api-client"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type UseChangePasswordDialogStateProps = {
  onOpenChange: (open: boolean) => void
  t: TranslationFn
}

export function useChangePasswordDialogState({
  onOpenChange,
  t,
}: UseChangePasswordDialogStateProps) {
  const [oldPassword, setOldPassword] = React.useState("")
  const [newPassword, setNewPassword] = React.useState("")
  const [confirmPassword, setConfirmPassword] = React.useState("")
  const [error, setError] = React.useState("")

  const { mutate: changePassword, isPending } = useChangePassword()

  const resetForm = React.useCallback(() => {
    setOldPassword("")
    setNewPassword("")
    setConfirmPassword("")
    setError("")
  }, [])

  const handleOpenChange = React.useCallback((nextOpen: boolean) => {
    if (!nextOpen) {
      resetForm()
    }
    onOpenChange(nextOpen)
  }, [onOpenChange, resetForm])

  const handleSubmit = React.useCallback((event: React.FormEvent) => {
    event.preventDefault()
    setError("")

    if (newPassword !== confirmPassword) {
      setError(t("passwordMismatch"))
      return
    }

    if (newPassword.length < 6) {
      setError(t("passwordTooShort", { min: 6 }))
      return
    }

    changePassword(
      { oldPassword, newPassword },
      {
        onSuccess: () => {
          handleOpenChange(false)
        },
        onError: (err: unknown) => {
          setError(getErrorMessage(err))
        },
      }
    )
  }, [changePassword, confirmPassword, handleOpenChange, newPassword, oldPassword, t])

  return {
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
  }
}
