"use client"

import { useEffect, useRef, useState } from "react"

import { semanticIcons } from "@/components/icons"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"
import { getStatusToneSurfaceClass, getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { narrowFormDialogContentClassName } from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"
import { ACTIVATION_SAMPLE_KEYS, sampleLicense, type ActivationPreviewLicense } from "./agent-activation-prototype-state"

type ActivationDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentLimit: number
  renewing: boolean
  onActivate: (license: ActivationPreviewLicense) => void
}

export function AgentActivationPrototypeDialog({ open, onOpenChange, currentLimit, renewing, onActivate }: ActivationDialogProps) {
  const [key, setKey] = useState("")
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<ActivationPreviewLicense | null>(null)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const preview = sampleLicense(key)

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current)
  }, [])

  useEffect(() => {
    if (error) inputRef.current?.focus()
  }, [error])

  function changeOpen(next: boolean) {
    if (pending) return
    onOpenChange(next)
    if (!next) {
      setKey("")
      setError(null)
      setSuccess(null)
    }
  }

  function activate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (pending || !key.trim()) return
    setPending(true)
    setError(null)
    // Simulates feedback timing only; no key verification or authorization happens here.
    timer.current = setTimeout(() => {
      setPending(false)
      const license = sampleLicense(key)
      if (!license) {
        setError(key.trim().toUpperCase() === "LF-DEMO-EXPIRED"
          ? "这个 Key 已过期，请联系作者获取新的 Key 后重试。当前授权未改变。"
          : "未能识别这个 Key，请检查是否完整复制，或联系作者获取新的 Key。当前授权未改变。")
        return
      }
      onActivate(license)
      setSuccess(license)
    }, 900)
  }

  return (
    <Dialog open={open} onOpenChange={changeOpen}>
      <DialogContent className={cn(narrowFormDialogContentClassName, "max-h-dvh overflow-y-auto")} initialFocus={inputRef}>
        <DialogHeader className="pr-8 text-left">
          <DialogTitle>{success ? "扩容已激活" : renewing ? "更新扩容授权" : "解锁更多 Agent"}</DialogTitle>
          <DialogDescription>
            {success ? "额度已更新，现在可以继续添加 Agent。" : "粘贴你的授权 Key，激活后即可增加节点。"}
          </DialogDescription>
        </DialogHeader>

        {success ? (
          <div className="space-y-5 py-2">
            <div role="status" className="flex items-center gap-3">
              <span className={cn("radius-round flex size-10 shrink-0 items-center justify-center", getStatusToneSurfaceClass("success"))}>
                <semanticIcons.status.success aria-hidden="true" className={cn("size-5", getStatusToneTextClass("success"))} />
              </span>
              <div>
                <p className={textRole.sectionTitle}>已解锁 10 个 Agent 额度</p>
                <p className={cn("mt-1", textRole.helperText)}>有效期 {success.months} 个月 · 至 {success.expiresAt}</p>
              </div>
            </div>
            <dl className={cn("grid grid-cols-2 gap-3 border-y py-4", textRole.metadataLabel)}>
              <dt>Agent 节点额度</dt><dd className={cn("text-right", textRole.metadataValueStrong)}>10 个</dd>
              <dt>授权编号</dt><dd className={cn("break-all text-right", textRole.code)}>{success.licenseId}</dd>
              <dt>单节点任务配置</dt><dd className={cn("text-right", textRole.metadataValue)}>保持原配置</dd>
            </dl>
            <DialogFooter>
              <Button onClick={() => changeOpen(false)} className="w-full sm:w-auto">返回添加 Agent</Button>
            </DialogFooter>
          </div>
        ) : (
          <form onSubmit={activate} className="space-y-4 pt-2" aria-busy={pending}>
            <div className="space-y-2">
              <Label htmlFor="activation-key">授权 Key</Label>
              <Textarea
                ref={inputRef}
                id="activation-key"
                name="activationKey"
                value={key}
                onChange={(event) => { setKey(event.target.value); setError(null) }}
                placeholder="粘贴完整的授权 Key"
                className={cn("min-h-24 resize-none", textRole.code)}
                spellCheck={false}
                autoCapitalize="none"
                disabled={pending}
                aria-invalid={Boolean(error)}
                aria-describedby={error ? "activation-key-error" : "activation-key-help"}
              />
              <p id="activation-key-help" className={textRole.helperText}>授权包含节点额度与有效期，无需为每个 Agent 单独激活。</p>
              {error && <p id="activation-key-error" role="alert" className={cn(textRole.helperText, getStatusToneTextClass("error"))}>{error}</p>}
            </div>

            {preview && !error ? (
              <div aria-live="polite" className="radius-control space-y-2 border bg-muted/20 p-3">
                <div className="flex items-center justify-between gap-2">
                  <span className={textRole.bodyStrong}>扩容授权</span><Badge size="compact" variant="outline">{preview.months} 个月</Badge>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className={textRole.metricValueDisplay}>{currentLimit}</span>
                  <span aria-hidden="true" className={textRole.bodySubtle}>→</span>
                  <span className={textRole.metricValueDisplay}>10</span>
                  <span className={textRole.helperText}>个 Agent 额度</span>
                </div>
                <p className={textRole.helperText}>有效期至 {preview.expiresAt}</p>
              </div>
            ) : null}

            <Collapsible>
              <CollapsibleTrigger render={<Button type="button" variant="link" size="sm" />}>
                <semanticIcons.navigation.support aria-hidden="true" />如何获取 Key？
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div className={cn("space-y-2 pb-1 pt-2", textRole.bodySubtle)}>
                  <p className={textRole.bodyStrong}>联系作者获取 Key</p>
                  <p>获取后复制完整 Key，回到这里粘贴激活。</p>
                </div>
              </CollapsibleContent>
            </Collapsible>

            <Separator />
            <div className="space-y-2">
              <p className={textRole.helperText}>仅供预览 · 点击填入示例 Key</p>
              <div className="flex flex-wrap gap-2">
                {ACTIVATION_SAMPLE_KEYS.map((sample, index) => (
                  <Button key={sample} type="button" variant="outline" size="xs" disabled={pending} onClick={() => { setKey(sample); setError(null); inputRef.current?.focus() }}>{index + 1} 个月</Button>
                ))}
                <Button type="button" variant="ghost" size="xs" disabled={pending} onClick={() => { setKey("LF-DEMO-EXPIRED"); setError(null) }}>过期 Key</Button>
              </div>
            </div>

            {pending && <Alert role="status"><semanticIcons.concept.apiKey aria-hidden="true" /><AlertDescription>正在验证授权，请稍候…</AlertDescription></Alert>}
            <DialogFooter className="pt-1">
              <Button type="button" variant="outline" disabled={pending} onClick={() => changeOpen(false)}>取消</Button>
              <Button type="submit" disabled={!key.trim()} loading={pending} loadingLabel="正在激活">激活扩容</Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
