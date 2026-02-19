"use client"

import * as React from "react"
import dynamic from "next/dynamic"
import { cn } from "@/lib/utils"
import type { ShuffleRef } from "@/components/shuffle"
import type { LoginStep, TerminalLine, TerminalLoginTranslations } from "@/components/ui/terminal-login-types"

const Shuffle = dynamic(() => import("@/components/shuffle"), { ssr: false })

type BootLine = {
  text: string
  className?: string
}

const AUTH_STEP_DELAYS_MS = [400, 400, 400, 400]

export function AuthBootLog({
  authenticatingLabel,
  processingLabel,
  done = false,
  className,
}: {
  authenticatingLabel: string
  processingLabel: string
  done?: boolean
  className?: string
}) {
  const [visible, setVisible] = React.useState(0)

  const authLines = React.useMemo<BootLine[]>(
    () => [
      { text: `> ${authenticatingLabel}`, className: "text-foreground/90" },
      { text: "> initializing secure channel…", className: "text-muted-foreground" },
      { text: "> validating credentials…", className: "text-muted-foreground" },
      { text: "> checking session…", className: "text-foreground/90" },
    ],
    [authenticatingLabel]
  )

  React.useEffect(() => {
    setVisible(0)

    const timers: Array<ReturnType<typeof setTimeout>> = []
    let acc = 0

    for (let i = 0; i < authLines.length; i++) {
      acc += AUTH_STEP_DELAYS_MS[i] ?? 220
      timers.push(
        setTimeout(() => {
          setVisible((prev) => Math.max(prev, i + 1))
        }, acc)
      )
    }

    return () => {
      timers.forEach(clearTimeout)
    }
  }, [authLines])

  React.useEffect(() => {
    if (!done) return
    setVisible(authLines.length)
  }, [authLines.length, done])

  const rawProgress = Math.round((Math.min(visible, authLines.length) / authLines.length) * 100)
  const progress = done ? 100 : Math.min(rawProgress, 99)

  return (
    <div className={className}>
      <div className="space-y-1">
        {authLines.slice(0, visible).map((line, idx) => (
          <div key={idx} className={cn("animate-typing", line.className)}>
            {line.text}
          </div>
        ))}

        <div className="text-foreground/80">
          <span className="inline-block h-4 w-2 align-middle bg-foreground/80 animate-pulse" />
        </div>
      </div>

      <div className="mt-6">
        <div className="h-1.5 w-full rounded bg-foreground/10 overflow-hidden">
          <div
            className="h-full bg-gradient-to-r from-primary/40 to-primary transition-[width] duration-300"
            style={{
              width: `${progress}%`,
              boxShadow:
                "0 0 10px hsl(var(--primary) / 0.35), 0 0 18px hsl(var(--primary) / 0.2)",
            }}
          />
        </div>
        <div className="mt-2 text-xs text-muted-foreground">{processingLabel}</div>
      </div>
    </div>
  )
}

export function TerminalLoginHeader({ title }: { title: string }) {
  return (
    <div className="border-border flex items-center gap-x-2 border-b px-4 py-3">
      <div className="flex flex-row gap-x-2">
        <div className="h-3 w-3 rounded-full bg-[var(--error)]"></div>
        <div className="h-3 w-3 rounded-full bg-[var(--warning)]"></div>
        <div className="h-3 w-3 rounded-full bg-[var(--success)]"></div>
      </div>
      <span className="ml-2 text-xs text-muted-foreground font-mono">{title}</span>
    </div>
  )
}

export function TerminalLoginBanner({
  subtitle,
  shuffleRef,
}: {
  subtitle: string
  shuffleRef: React.RefObject<ShuffleRef | null>
}) {
  return (
    <div className="mb-6 text-center">
      <Shuffle
        ref={shuffleRef}
        text="LUNAFOX"
        className="!text-4xl sm:!text-5xl md:!text-6xl !font-bold text-primary"
        shuffleDirection="up"
        duration={0.5}
        stagger={0.04}
        shuffleTimes={2}
        triggerOnHover={true}
        triggerOnce={false}
        autoPlay={true}
      />
      <div className="mt-3 flex items-center gap-3 text-muted-foreground text-sm">
        <span className="h-px flex-1 bg-border" />
        <span className="whitespace-nowrap">{subtitle}</span>
        <span className="h-px flex-1 bg-border" />
      </div>
    </div>
  )
}

interface TerminalMobilePanelProps {
  step: LoginStep
  username: string
  password: string
  isInputDisabled: boolean
  authDone: boolean
  onUsernameChange: (value: string) => void
  onPasswordChange: (value: string) => void
  onSubmit: () => void
  t: TerminalLoginTranslations
}

export function TerminalMobilePanel({
  step,
  username,
  password,
  isInputDisabled,
  authDone,
  onUsernameChange,
  onPasswordChange,
  onSubmit,
  t,
}: TerminalMobilePanelProps) {
  return (
    <div className="sm:hidden">
      {(step === "username" || step === "password" || step === "error") && (
        <form
          onSubmit={(event) => {
            event.preventDefault()
            onSubmit()
          }}
          className="space-y-4"
        >
          <div>
            <label htmlFor="terminal-mobile-username" className="text-primary text-xs mb-1 block">{t.usernamePrompt}</label>
            <input
              id="terminal-mobile-username"
              name="username"
              type="text"
              aria-label={t.usernamePrompt}
              value={username}
              onChange={(event) => onUsernameChange(event.target.value)}
              disabled={isInputDisabled}
              className="w-full bg-background/60 border border-border rounded px-3 py-2 text-foreground outline-none focus:border-primary focus-visible:ring-2 focus-visible:ring-primary/40 font-mono text-sm"
              autoComplete="username"
              inputMode="text"
              spellCheck={false}
            />
          </div>
          <div>
            <label htmlFor="terminal-mobile-password" className="text-primary text-xs mb-1 block">{t.passwordPrompt}</label>
            <input
              id="terminal-mobile-password"
              name="password"
              type="password"
              aria-label={t.passwordPrompt}
              value={password}
              onChange={(event) => onPasswordChange(event.target.value)}
              disabled={isInputDisabled}
              className="w-full bg-background/60 border border-border rounded px-3 py-2 text-foreground outline-none focus:border-primary focus-visible:ring-2 focus-visible:ring-primary/40 font-mono text-sm"
              autoComplete="current-password"
            />
          </div>
          {step === "error" && (
            <p className="text-destructive text-sm">{t.invalidCredentials}</p>
          )}
          <button
            type="submit"
            disabled={isInputDisabled}
            className="w-full py-2 px-4 bg-primary hover:bg-primary/90 disabled:opacity-50 text-primary-foreground font-mono text-sm rounded transition-colors"
          >
            {t.submit}
          </button>
        </form>
      )}
      {step === "authenticating" && (
        <div className="py-4">
          <AuthBootLog authenticatingLabel={t.authenticating} processingLabel={t.processing} done={authDone} />
        </div>
      )}
      {step === "success" && (
        <div className="text-primary text-center py-4">
          {t.accessGranted}
        </div>
      )}
    </div>
  )
}

interface TerminalDesktopPanelProps {
  step: LoginStep
  lines: TerminalLine[]
  t: TerminalLoginTranslations
  prompt: string
  renderInput: () => React.ReactNode
  inputProps: {
    value: string
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void
    onKeyDown: (event: React.KeyboardEvent<HTMLInputElement>) => void
    onSelect: (event: React.SyntheticEvent<HTMLInputElement>) => void
    onFocus: () => void
    onBlur: () => void
    disabled: boolean
    type: "password" | "text"
    autoComplete: string
  }
  inputRef: React.RefObject<HTMLInputElement | null>
  authDone: boolean
}

export function TerminalDesktopPanel({
  step,
  lines,
  t,
  prompt,
  renderInput,
  inputProps,
  inputRef,
  authDone,
}: TerminalDesktopPanelProps) {
  return (
    <div className="hidden sm:block">
      {lines.map((line, index) => (
        <span
          key={index}
          className={cn(
            "whitespace-pre-wrap",
            line.type === "prompt" && "text-primary",
            line.type === "input" && "text-foreground",
            line.type === "info" && "text-muted-foreground",
            line.type === "success" && "text-primary",
            line.type === "error" && "text-destructive",
            line.type === "warning" && "text-amber-500"
          )}
        >
          {line.text}
          {(line.type === "prompt" || line.text === "") ? "" : "\n"}
        </span>
      ))}

      {(step === "username" || step === "password") && (
        <div className="flex items-center">
          <span className="text-primary">{prompt}</span>
          {renderInput()}
          <input
            ref={inputRef}
            className="absolute opacity-0 pointer-events-none"
            name={step === "password" ? "password" : "username"}
            aria-label={step === "password" ? t.passwordPrompt : t.usernamePrompt}
            spellCheck={false}
            {...inputProps}
          />
        </div>
      )}

      {step === "authenticating" && (
        <div className="mt-2">
          <AuthBootLog authenticatingLabel={t.authenticating} processingLabel={t.processing} done={authDone} />
        </div>
      )}

      {(step === "username" || step === "password") && (
        <div className="mt-6 text-xs text-muted-foreground">
          <span className="text-muted-foreground">{t.shortcuts}:</span>{" "}
          <span className="text-primary">Enter</span> {t.submit}{" "}
          <span className="text-primary">Ctrl+C</span> {t.cancel}{" "}
          <span className="text-primary">Ctrl+U</span> {t.clear}{" "}
          <span className="text-primary">Ctrl+A/E</span> {t.startEnd}
        </div>
      )}
    </div>
  )
}
