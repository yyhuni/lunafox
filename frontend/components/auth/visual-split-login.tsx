"use client"

/* eslint-disable @next/next/no-img-element -- Admin-uploaded media uses an opaque Server route outside Next's image loader. */

import * as React from "react"
import dynamic from "next/dynamic"

import { LunaFoxMark } from "@/components/brand/lunafox-mark"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { usePublicLoginVisual } from "@/hooks/use-login-visual"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { PublicLoginVisual } from "@/types/login-visual.types"

interface VisualSplitLoginProps {
  onLogin: (username: string, password: string) => Promise<void>
  authDone?: boolean
  isPending?: boolean
  onVisualReady?: () => void
  className?: string
  translations: VisualSplitLoginTranslations
  preview?: VisualSplitLoginPreview
}

export interface VisualSplitLoginTranslations {
  ariaLabel: string
  consoleLabel: string
  usernameLabel: string
  usernamePlaceholder: string
  passwordLabel: string
  passwordPlaceholder: string
  submit: string
  accessHelp: string
}

export interface VisualSplitLoginPreview {
  visual: PublicLoginVisual
  visualControl: React.ReactNode
}

const appVersion = process.env.NEXT_PUBLIC_IMAGE_TAG || "dev"
const terminalGridMul: [number, number] = [2, 1]
const minShaderHardwareConcurrency = 4

const FaultyTerminal = dynamic(() => import("@/components/faulty-terminal"), {
  ssr: false,
})

type NetworkInformationLike = EventTarget & {
  saveData?: boolean
  effectiveType?: string
}

type NavigatorWithConnection = Navigator & {
  connection?: NetworkInformationLike
}

type LoginShaderDecision = {
  enabled: boolean
  resolved: boolean
}

function getNetworkInformation() {
  return (navigator as NavigatorWithConnection).connection
}

function canCreateWebGLContext() {
  const canvas = document.createElement("canvas")
  const context = canvas.getContext("webgl2") ?? canvas.getContext("webgl")
  context?.getExtension("WEBGL_lose_context")?.loseContext()
  return Boolean(context)
}

function useLoginShaderEnabled() {
  const [decision, setDecision] = React.useState<LoginShaderDecision>({
    enabled: false,
    resolved: false,
  })

  React.useEffect(() => {
    const reducedMotionQuery = window.matchMedia("(prefers-reduced-motion: reduce)")
    const finePointerQuery = window.matchMedia("(pointer: fine)")
    const desktopQuery = window.matchMedia("(min-width: 768px)")
    const networkInformation = getNetworkInformation()

    const update = () => {
      const hasEnoughCpu =
        !navigator.hardwareConcurrency ||
        navigator.hardwareConcurrency >= minShaderHardwareConcurrency
      const effectiveType = networkInformation?.effectiveType ?? ""
      const slowConnection =
        networkInformation?.saveData === true ||
        effectiveType === "slow-2g" ||
        effectiveType === "2g"

      setDecision({
        enabled:
          !reducedMotionQuery.matches &&
          finePointerQuery.matches &&
          desktopQuery.matches &&
          hasEnoughCpu &&
          !slowConnection &&
          canCreateWebGLContext(),
        resolved: true,
      })
    }

    update()

    reducedMotionQuery.addEventListener("change", update)
    finePointerQuery.addEventListener("change", update)
    desktopQuery.addEventListener("change", update)
    networkInformation?.addEventListener("change", update)

    return () => {
      reducedMotionQuery.removeEventListener("change", update)
      finePointerQuery.removeEventListener("change", update)
      desktopQuery.removeEventListener("change", update)
      networkInformation?.removeEventListener("change", update)
    }
  }, [])

  return decision
}

function useReducedMotion() {
  const [reducedMotion, setReducedMotion] = React.useState(false)

  React.useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)")
    const update = () => setReducedMotion(query.matches)

    update()
    query.addEventListener("change", update)
    return () => query.removeEventListener("change", update)
  }, [])

  return reducedMotion
}

function PublishedLoginVisual({
  visual,
  onReady,
}: {
  visual: PublicLoginVisual | undefined
  onReady: () => void
}) {
  const [ready, setReady] = React.useState(false)
  const [failed, setFailed] = React.useState(false)
  const reducedMotion = useReducedMotion()
  const source = visual?.kind === "video" && reducedMotion ? visual.posterUrl : visual?.mediaUrl

  React.useEffect(() => {
    setReady(false)
    setFailed(false)
  }, [source])

  if (!visual || visual.kind === "builtIn" || !source || failed) return null

  const handleReady = () => {
    setReady(true)
    onReady()
  }
  const handleFailure = () => setFailed(true)
  const className = cn(
    "absolute inset-0 z-20 size-full object-cover transition-opacity duration-200",
    ready ? "opacity-100" : "opacity-0"
  )

  if (visual.kind === "image" || reducedMotion) {
    return <img alt="" className={className} data-login-visual-kind={visual.kind} onError={handleFailure} onLoad={handleReady} src={source} />
  }

  return <video aria-hidden="true" autoPlay className={className} data-login-visual-kind="video" loop muted onCanPlay={handleReady} onError={handleFailure} playsInline src={source} />
}

export function VisualSplitLogin({
  onLogin,
  authDone = false,
  isPending = false,
  onVisualReady,
  className,
  translations: t,
  preview,
}: VisualSplitLoginProps) {
  const [email, setEmail] = React.useState("")
  const [password, setPassword] = React.useState("")
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [shaderReady, setShaderReady] = React.useState(false)
  const visualReadyRef = React.useRef(false)
  const { enabled: shaderEnabled, resolved: shaderDecisionResolved } = useLoginShaderEnabled()
  const publicVisual = usePublicLoginVisual()

  const submitting = isSubmitting || isPending || authDone
  const formDisabled = submitting || Boolean(preview)
  const visual = preview?.visual ?? publicVisual.data

  React.useEffect(() => {
    setShaderReady(false)
    visualReadyRef.current = false
  }, [shaderEnabled])

  const markVisualReady = React.useCallback(() => {
    if (visualReadyRef.current) return
    visualReadyRef.current = true
    onVisualReady?.()
  }, [onVisualReady])

  React.useEffect(() => {
    if (!shaderDecisionResolved || shaderEnabled) return
    markVisualReady()
  }, [markVisualReady, shaderDecisionResolved, shaderEnabled])

  const handleShaderFirstFrame = React.useCallback(() => {
    setShaderReady(true)
    markVisualReady()
  }, [markVisualReady])

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (preview) return
    if (!email.trim() || !password.trim()) return

    setIsSubmitting(true)

    try {
      await onLogin(email.trim(), password)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <section
      aria-label={t.ariaLabel}
      className={cn(
        "auth-visual-split bg-background relative flex w-full flex-col overflow-hidden selection:bg-primary selection:text-primary-foreground",
        preview ? "h-full min-h-0" : "min-h-svh",
        className
      )}
    >
      <div className="flex flex-1 flex-col">
        <div className="mx-auto w-full max-w-7xl flex-1 border-x border-border" />

        <div className="w-full">
          <div className="relative mx-auto max-w-7xl border-x border-b border-border before:absolute before:-bottom-px before:right-full before:h-px before:w-screen before:bg-border after:absolute after:-bottom-px after:left-full after:h-px after:w-screen after:bg-border">
            <div className="flex flex-col gap-4 px-8 py-16">
              <div className="flex items-center gap-1.5">
                <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground" />
                <p className={textRole.authHeroEyebrow}>Welcome to LunaFox</p>
              </div>
              <p className={textRole.authHeroTitle}>Sign In.</p>
            </div>
          </div>
        </div>

        <div className="w-full">
          <div className="relative mx-auto max-w-7xl border-x border-b border-border before:absolute before:-bottom-px before:right-full before:h-px before:w-screen before:bg-border after:absolute after:-bottom-px after:left-full after:h-px after:w-screen after:bg-border">
            <div className={cn("relative grid overflow-hidden", preview ? "grid-cols-12 min-h-128" : "grid-cols-1 md:grid-cols-12 md:min-h-128")}>
              <div className={cn("relative overflow-hidden bg-background", preview ? "col-span-7 block" : "hidden md:col-span-7 md:block")}>
                <div
                  className={cn(
                    "auth-shader-fallback pointer-events-none absolute inset-0 z-10 transition-opacity duration-200",
                    shaderReady ? "opacity-0" : "opacity-100"
                  )}
                  data-shader-ready={shaderReady ? "true" : "false"}
                  aria-hidden="true"
                />
                {shaderEnabled ? (
                  <FaultyTerminal
                    scale={1.5}
                    gridMul={terminalGridMul}
                    digitSize={1.2}
                    timeScale={0.5}
                    pause={false}
                    scanlineIntensity={0.5}
                    glitchAmount={1}
                    flickerAmount={1}
                    noiseAmp={1}
                    chromaticAberration={0}
                    dither={0}
                    curvature={0.1}
                    tint="var(--primary)"
                    backgroundColor="var(--background)"
                    mouseReact={false}
                    pageLoadAnimation
                    brightness={0.6}
                    dpr={1}
                    maxFps={30}
                    onFirstFrame={handleShaderFirstFrame}
                    className="absolute inset-0 z-0"
                    aria-hidden="true"
                  />
                ) : null}
                <PublishedLoginVisual onReady={markVisualReady} visual={visual} />
                {preview?.visualControl}
              </div>

              <div className={cn("flex flex-col justify-center px-8 py-12", preview ? "col-span-5" : "md:col-span-5")}>
                <div className="mx-auto flex w-full max-w-sm flex-col gap-6">
                  <div className="flex items-center gap-2">
                    <LunaFoxMark className="size-5" decorative />
                    <span className={textRole.bodyStrong}>{t.consoleLabel}</span>
                  </div>

                  <form className="space-y-6" onSubmit={handleSubmit}>
                    <div className="space-y-4">
                      <div className="space-y-1.5">
                        <Label htmlFor={preview ? "login-visual-preview-email" : "email"} className={textRole.authFormLabel}>
                          {t.usernameLabel}
                        </Label>
                        <Input
                          id={preview ? "login-visual-preview-email" : "email"}
                          type="text"
                          autoComplete="username"
                          placeholder={t.usernamePlaceholder}
                          required
                          value={email}
                          onChange={(event) => setEmail(event.target.value)}
                          disabled={formDisabled}
                          className="rounded-md border-border bg-background px-2.5 text-foreground shadow-xs dark:bg-background"
                        />
                      </div>

                      <div className="space-y-1.5">
                        <Label htmlFor={preview ? "login-visual-preview-password" : "password"} className={textRole.authFormLabel}>
                          {t.passwordLabel}
                        </Label>
                        <Input
                          id={preview ? "login-visual-preview-password" : "password"}
                          type="password"
                          autoComplete="current-password"
                          placeholder={t.passwordPlaceholder}
                          required
                          value={password}
                          onChange={(event) => setPassword(event.target.value)}
                          disabled={formDisabled}
                          className="rounded-md border-border bg-background px-2.5 text-foreground shadow-xs dark:bg-background"
                        />
                      </div>
                    </div>

                    <Button
                      type="submit"
                      size="lg"
                      disabled={formDisabled}
                      loading={!preview && submitting}
                      loadingLabel={t.submit}
                      className="h-10 w-full rounded-lg hover:cursor-pointer"
                    >
                      {t.submit}
                    </Button>
                  </form>

                  <p className={cn(textRole.authFormText, "text-center")}>
                    {t.accessHelp}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="mx-auto h-32 w-full max-w-7xl grow border-x border-border" />
      </div>
      <p className={cn(textRole.helperText, "pointer-events-none absolute bottom-6 left-1/2 -translate-x-1/2 font-mono text-muted-foreground/70")}>
        {appVersion}
      </p>
    </section>
  )
}
