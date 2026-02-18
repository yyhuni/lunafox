import React from "react"
import { toast } from "sonner"
import {
  useImportARLFingerprints,
  useImportEholeFingerprints,
  useImportFingerPrintHubFingerprints,
  useImportFingersFingerprints,
  useImportGobyFingerprints,
  useImportWappalyzerFingerprints,
} from "@/hooks/use-fingerprints"
import { getErrorMessage } from "@/lib/error-utils"
import {
  buildFingerprintConfig,
  getAcceptConfig,
  type FingerprintType,
} from "@/components/fingerprints/import-fingerprint-dialog-utils"

type UseImportFingerprintDialogStateProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  fingerprintType: FingerprintType
  acceptedFileTypes?: string
  t: ((key: string, params?: Record<string, string | number | Date>) => string) & {
    raw: (key: string) => string
  }
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useImportFingerprintDialogState({
  open,
  onOpenChange,
  onSuccess,
  fingerprintType,
  acceptedFileTypes,
  t,
  tToast,
}: UseImportFingerprintDialogStateProps) {
  const [files, setFiles] = React.useState<File[]>([])

  const eholeImportMutation = useImportEholeFingerprints()
  const gobyImportMutation = useImportGobyFingerprints()
  const wappalyzerImportMutation = useImportWappalyzerFingerprints()
  const fingersImportMutation = useImportFingersFingerprints()
  const fingerprinthubImportMutation = useImportFingerPrintHubFingerprints()
  const arlImportMutation = useImportARLFingerprints()

  const configMap = React.useMemo(() => buildFingerprintConfig(t), [t])
  const config = configMap[fingerprintType]

  const importMutation = {
    ehole: eholeImportMutation,
    goby: gobyImportMutation,
    wappalyzer: wappalyzerImportMutation,
    fingers: fingersImportMutation,
    fingerprinthub: fingerprinthubImportMutation,
    arl: arlImportMutation,
  }[fingerprintType]

  const acceptConfig = React.useMemo(
    () => getAcceptConfig(fingerprintType, acceptedFileTypes),
    [acceptedFileTypes, fingerprintType]
  )

  const handleDrop = React.useCallback((acceptedFiles: File[]) => {
    setFiles(acceptedFiles)
  }, [])

  const handleImport = React.useCallback(async () => {
    if (files.length === 0) {
      toast.error(tToast("selectFileFirst"))
      return
    }

    const file = files[0]
    const isYamlFile = file.name.endsWith(".yaml") || file.name.endsWith(".yml")

    if (!isYamlFile) {
      try {
        const text = await file.text()
        let json: unknown

        try {
          json = JSON.parse(text)
        } catch {
          if (fingerprintType === "goby") {
            const lines = text.trim().split("\n").filter((line) => line.trim())
            if (lines.length === 0) {
              toast.error(t("import.emptyData"))
              return
            }
            json = lines.map((line, index) => {
              try {
                return JSON.parse(line)
              } catch {
                throw new Error(`Line ${index + 1}: Invalid JSON`)
              }
            })
          } else {
            throw new Error("Invalid JSON")
          }
        }

        const validation = config.validate(json)
        if (!validation.valid) {
          toast.error(validation.error)
          return
        }
      } catch (error) {
        toast.error(getErrorMessage(error) || tToast("invalidJsonFile"))
        return
      }
    }

    try {
      const result = await importMutation.mutateAsync(file)
      toast.success(t("import.importSuccessDetail", { created: result.created, failed: result.failed }))
      setFiles([])
      onOpenChange(false)
      onSuccess?.()
    } catch (error) {
      toast.error(getErrorMessage(error) || tToast("importFailed"))
    }
  }, [config, files, fingerprintType, importMutation, onOpenChange, onSuccess, t, tToast])

  const handleClose = React.useCallback((nextOpen: boolean) => {
    if (!nextOpen) {
      setFiles([])
    }
    onOpenChange(nextOpen)
  }, [onOpenChange])

  return {
    files,
    setFiles,
    config,
    importMutation,
    acceptConfig,
    handleDrop,
    handleImport,
    handleClose,
    open,
  }
}
