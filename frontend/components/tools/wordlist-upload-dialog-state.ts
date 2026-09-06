import React from "react"
import { useUploadWordlist } from "@/hooks/use-wordlists"

type UseWordlistUploadDialogStateProps = {
  initialOpen?: boolean
}

export function useWordlistUploadDialogState({
  initialOpen = false,
}: UseWordlistUploadDialogStateProps = {}) {
  const [open, setOpen] = React.useState(initialOpen)
  const [description, setDescription] = React.useState("")
  const [tags, setTags] = React.useState<string[]>([])
  const [file, setFile] = React.useState<File | null>(null)
  const [isDragActive, setIsDragActive] = React.useState(false)

  const uploadMutation = useUploadWordlist()

  const resetForm = React.useCallback(() => {
    setDescription("")
    setTags([])
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
    if (!file) return
    uploadMutation.mutate(
      { description: description || undefined, tags, file },
      {
        onSuccess: () => {
          resetForm()
          setOpen(false)
        },
      }
    )
  }, [description, file, resetForm, tags, uploadMutation])

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
    description,
    tags,
    file,
    isDragActive,
    isPending: uploadMutation.isPending,
    setDescription,
    setTags,
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
