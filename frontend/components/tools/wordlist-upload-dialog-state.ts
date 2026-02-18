import React from "react"
import { useUploadWordlist } from "@/hooks/use-wordlists"

type UseWordlistUploadDialogStateProps = {
  initialOpen?: boolean
}

const normalizeNameFromFile = (fileName: string) => fileName.replace(/\.[^/.]+$/, "")

export function useWordlistUploadDialogState({
  initialOpen = false,
}: UseWordlistUploadDialogStateProps = {}) {
  const [open, setOpen] = React.useState(initialOpen)
  const [name, setName] = React.useState("")
  const [description, setDescription] = React.useState("")
  const [file, setFile] = React.useState<File | null>(null)
  const [isDragActive, setIsDragActive] = React.useState(false)

  const uploadMutation = useUploadWordlist()

  const resetForm = React.useCallback(() => {
    setName("")
    setDescription("")
    setFile(null)
    setIsDragActive(false)
  }, [])

  const handleOpenChange = React.useCallback((nextOpen: boolean) => {
    setOpen(nextOpen)
    if (!nextOpen) {
      resetForm()
    }
  }, [resetForm])

  const handleFileSelectInternal = React.useCallback((selectedFile: File) => {
    setFile(selectedFile)
    setName((prev) => prev || normalizeNameFromFile(selectedFile.name))
  }, [])

  const handleDragOver = React.useCallback((event: React.DragEvent) => {
    event.preventDefault()
    setIsDragActive(true)
  }, [])

  const handleDragLeave = React.useCallback((event: React.DragEvent) => {
    event.preventDefault()
    setIsDragActive(false)
  }, [])

  const handleDrop = React.useCallback((event: React.DragEvent) => {
    event.preventDefault()
    setIsDragActive(false)
    const droppedFile = event.dataTransfer.files[0]
    if (droppedFile && droppedFile.name.endsWith(".txt")) {
      handleFileSelectInternal(droppedFile)
    }
  }, [handleFileSelectInternal])

  const handleFileSelect = React.useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = event.target.files?.[0]
    if (selectedFile) {
      handleFileSelectInternal(selectedFile)
    }
  }, [handleFileSelectInternal])

  const handleSubmit = React.useCallback((event: React.FormEvent) => {
    event.preventDefault()
    if (!name || !file) return

    uploadMutation.mutate(
      { name, description: description || undefined, file },
      {
        onSuccess: () => {
          resetForm()
          setOpen(false)
        },
      }
    )
  }, [description, file, name, resetForm, uploadMutation])

  const removeFile = React.useCallback(() => {
    setFile(null)
  }, [])

  const formatFileSize = React.useCallback((bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }, [])

  return {
    open,
    name,
    description,
    file,
    isDragActive,
    isPending: uploadMutation.isPending,
    setName,
    setDescription,
    handleOpenChange,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleFileSelect,
    handleSubmit,
    removeFile,
    formatFileSize,
  }
}
