"use client"

import React from "react"
import { useRouter } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"
import { useQueryClient } from "@tanstack/react-query"
import { TerminalLogin } from "@/components/ui/terminal-login"
import { LoadingState } from "@/components/loading-spinner"
import { useLogin, useAuth } from "@/hooks/use-auth"
import { vulnerabilityKeys } from "@/hooks/use-vulnerabilities"
import { getAssetStatistics, getStatisticsHistory } from "@/services/dashboard.service"
import { getScans } from "@/services/scan.service"
import { VulnerabilityService } from "@/services/vulnerability.service"

export default function LoginPage() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const { data: auth, isLoading: authLoading } = useAuth()
  const { mutateAsync: login, isPending } = useLogin()
  const t = useTranslations("auth.terminal")
  const locale = useLocale()

  const loginStartedRef = React.useRef(false)
  const [loginReady, setLoginReady] = React.useState(false)

  const [isReady, setIsReady] = React.useState(false)
  const [loginProcessing, setLoginProcessing] = React.useState(false)
  const [isExiting, setIsExiting] = React.useState(false)
  const exitStartedRef = React.useRef(false)
  const showLoading = !isReady || loginProcessing
  const showExitOverlay = isExiting

  const withLocale = React.useCallback((path: string) => {
    if (path.startsWith(`/${locale}/`)) return path
    const normalized = path.startsWith("/") ? path : `/${path}`
    return `/${locale}${normalized}`
  }, [locale])

  // Hide the inline boot splash and show login content
  React.useEffect(() => {
    let cancelled = false

    const waitForLoad = new Promise<void>((resolve) => {
      if (typeof document === "undefined") {
        resolve()
        return
      }
      if (document.readyState === "complete") {
        resolve()
        return
      }
      const handleLoad = () => resolve()
      window.addEventListener("load", handleLoad, { once: true })
    })

    const waitForPrefetch = new Promise<void>((resolve) => {
      if (typeof window === "undefined") {
        resolve()
        return
      }
      const w = window as Window & { __lunafoxRoutePrefetchDone?: boolean }
      if (w.__lunafoxRoutePrefetchDone) {
        resolve()
        return
      }
      const handlePrefetchDone = () => resolve()
      window.addEventListener("lunafox:route-prefetch-done", handlePrefetchDone, { once: true })
    })

    const waitForPrefetchOrTimeout = Promise.race([
      waitForPrefetch,
      new Promise<void>((resolve) => setTimeout(resolve, 3000)),
    ])

    Promise.all([waitForLoad, waitForPrefetchOrTimeout]).then(() => {
      if (cancelled) return
      setIsReady(true)
    })

    return () => {
      cancelled = true
    }
  }, [])

  // Extract preloading logic into reusable functions
  const prefetchDashboardData = React.useCallback(async () => {
    const scansParams = { page: 1, pageSize: 10 }
    const vulnsParams = { page: 1, pageSize: 10 }

    return Promise.allSettled([
      queryClient.prefetchQuery({
        queryKey: ["asset", "statistics"],
        queryFn: getAssetStatistics,
      }),
      queryClient.prefetchQuery({
        queryKey: ["asset", "statistics", "history", 7],
        queryFn: () => getStatisticsHistory(7),
      }),
      queryClient.prefetchQuery({
        queryKey: ["scans", scansParams],
        queryFn: () => getScans(scansParams),
      }),
      queryClient.prefetchQuery({
        queryKey: vulnerabilityKeys.list(vulnsParams),
        queryFn: () => VulnerabilityService.getAllVulnerabilities(vulnsParams),
      }),
    ])
  }, [queryClient])

  // Memoize translations object to avoid recreating on every render
  const translations = React.useMemo(() => ({
    title: t("title"),
    subtitle: t("subtitle"),
    usernamePrompt: t("usernamePrompt"),
    passwordPrompt: t("passwordPrompt"),
    authenticating: t("authenticating"),
    processing: t("processing"),
    accessGranted: t("accessGranted"),
    welcomeMessage: t("welcomeMessage"),
    authFailed: t("authFailed"),
    invalidCredentials: t("invalidCredentials"),
    shortcuts: t("shortcuts"),
    submit: t("submit"),
    cancel: t("cancel"),
    clear: t("clear"),
    startEnd: t("startEnd"),
  }), [t])

  // If already logged in, warm up the dashboard, then redirect.
  React.useEffect(() => {
    if (authLoading) return
    if (!auth?.authenticated) return
    if (loginStartedRef.current) return

    let cancelled = false
    let timer: number | undefined

    void (async () => {
      setLoginProcessing(true)
      await prefetchDashboardData()

      if (cancelled) return
      setLoginProcessing(false)
      if (!exitStartedRef.current) {
        exitStartedRef.current = true
        setIsExiting(true)
        timer = window.setTimeout(() => {
          router.replace(withLocale("/dashboard/"))
        }, 300)
      }
    })()

    return () => {
      cancelled = true
      if (timer) window.clearTimeout(timer)
    }
  }, [auth?.authenticated, authLoading, prefetchDashboardData, router, withLocale])

  React.useEffect(() => {
    if (!loginReady) return
    if (exitStartedRef.current) return
    exitStartedRef.current = true
    setIsExiting(true)
    const timer = window.setTimeout(() => {
      router.replace(withLocale("/dashboard/"))
    }, 300)
    return () => window.clearTimeout(timer)
  }, [loginReady, router, withLocale])

  const handleLogin = async (username: string, password: string) => {
    loginStartedRef.current = true
    setLoginReady(false)
    setLoginProcessing(true)

    // Execute independent operations in parallel: login verification + preloading dashboard bundle
    const [loginRes] = await Promise.all([
      login({ username, password }),
      router.prefetch(withLocale("/dashboard/")),
    ])

    // Preload dashboard data
    await prefetchDashboardData()

    // Prime auth cache so AuthLayout doesn't flash a full-screen loading state.
    queryClient.setQueryData(["auth", "me"], {
      authenticated: true,
      user: loginRes.user,
    })

    setLoginProcessing(false)
    setLoginReady(true)
  }

  return (
    <div className="relative flex min-h-svh flex-col bg-background text-foreground">
      {showLoading && !showExitOverlay ? (
        <LoadingState
          active
          message="loading…"
          className="fixed inset-0 z-50 bg-background"
        />
      ) : null}
      {showExitOverlay ? (
        <div className="fixed inset-0 z-50 bg-background" />
      ) : null}
      {/* Circuit Board Animation */}
      <div className={`fixed inset-0 z-0 transition-opacity duration-300 ${isReady ? "opacity-100" : "opacity-0"}`}>
        <div className="circuit-container">
          {/* Grid pattern */}
          <div className="circuit-grid" />
          
          {/* === Main backbone traces === */}
          {/* Horizontal main lines - 6 lines */}
          <div className="trace trace-h" style={{ top: '12%', left: 0, width: '100%' }}>
            <div className="trace-glow" style={{ animationDuration: '6s' }} />
          </div>
          <div className="trace trace-h" style={{ top: '28%', left: 0, width: '100%' }}>
            <div className="trace-glow" style={{ animationDelay: '1s', animationDuration: '5s' }} />
          </div>
          <div className="trace trace-h" style={{ top: '44%', left: 0, width: '100%' }}>
            <div className="trace-glow" style={{ animationDelay: '2s', animationDuration: '5.5s' }} />
          </div>
          <div className="trace trace-h" style={{ top: '60%', left: 0, width: '100%' }}>
            <div className="trace-glow" style={{ animationDelay: '3s', animationDuration: '4.5s' }} />
          </div>
          <div className="trace trace-h" style={{ top: '76%', left: 0, width: '100%' }}>
            <div className="trace-glow" style={{ animationDelay: '4s', animationDuration: '5s' }} />
          </div>
          <div className="trace trace-h" style={{ top: '92%', left: 0, width: '100%' }}>
            <div className="trace-glow" style={{ animationDelay: '5s', animationDuration: '6s' }} />
          </div>
          
          {/* Vertical main lines - 6 lines */}
          <div className="trace trace-v" style={{ left: '8%', top: 0, height: '100%' }}>
            <div className="trace-glow trace-glow-v" style={{ animationDelay: '0.5s', animationDuration: '7s' }} />
          </div>
          <div className="trace trace-v" style={{ left: '24%', top: 0, height: '100%' }}>
            <div className="trace-glow trace-glow-v" style={{ animationDelay: '1.5s', animationDuration: '6s' }} />
          </div>
          <div className="trace trace-v" style={{ left: '40%', top: 0, height: '100%' }}>
            <div className="trace-glow trace-glow-v" style={{ animationDelay: '2.5s', animationDuration: '5.5s' }} />
          </div>
          <div className="trace trace-v" style={{ left: '56%', top: 0, height: '100%' }}>
            <div className="trace-glow trace-glow-v" style={{ animationDelay: '3.5s', animationDuration: '6.5s' }} />
          </div>
          <div className="trace trace-v" style={{ left: '72%', top: 0, height: '100%' }}>
            <div className="trace-glow trace-glow-v" style={{ animationDelay: '4.5s', animationDuration: '5s' }} />
          </div>
          <div className="trace trace-v" style={{ left: '88%', top: 0, height: '100%' }}>
            <div className="trace-glow trace-glow-v" style={{ animationDelay: '5.5s', animationDuration: '6s' }} />
          </div>
          
        </div>
        
        <style jsx>{`
          .circuit-container {
            position: absolute;
            inset: 0;
            background: var(--background);
            overflow: hidden;
            --login-grid: color-mix(in oklch, var(--foreground) 6%, transparent);
            --login-trace: color-mix(in oklch, var(--foreground) 16%, transparent);
            --login-glow: color-mix(in oklch, var(--primary) 65%, transparent);
            --login-glow-muted: color-mix(in oklch, var(--foreground) 45%, transparent);
          }
          
          .circuit-grid {
            position: absolute;
            inset: 0;
            background-image: 
              linear-gradient(var(--login-grid) 1px, transparent 1px),
              linear-gradient(90deg, var(--login-grid) 1px, transparent 1px);
            background-size: 40px 40px;
          }
          
          .trace {
            position: absolute;
            background: var(--login-trace);
            overflow: hidden;
          }
          
          .trace-h {
            height: 2px;
          }
          
          .trace-v {
            width: 2px;
          }
          
          .trace-glow {
            position: absolute;
            top: -2px;
            left: -20%;
            width: 30%;
            height: 6px;
            background: linear-gradient(90deg, transparent, var(--login-glow), var(--login-glow-muted), transparent);
            animation: traceFlow 3s linear infinite;
            filter: blur(2px);
          }
          
          .trace-glow-v {
            top: -20%;
            left: -2px;
            width: 6px;
            height: 30%;
            background: linear-gradient(180deg, transparent, var(--login-glow), var(--login-glow-muted), transparent);
            animation: traceFlowV 3s linear infinite;
          }
          
          @keyframes traceFlow {
            0% { left: -30%; }
            100% { left: 100%; }
          }
          
          @keyframes traceFlowV {
            0% { top: -30%; }
            100% { top: 100%; }
          }
        `}</style>
      </div>

      {/* Fingerprint identifier - for FOFA/Shodan and other search engines to identify */}
      <meta name="generator" content="LunaFox ASM Platform" />

      {/* Main content area */}
      <div
        className={`relative z-10 flex-1 flex items-center justify-center p-6 transition-[opacity,transform] duration-300 ${
          isReady ? "opacity-100 translate-y-0" : "opacity-0 translate-y-2"
        }`}
      >
        <TerminalLogin
          onLogin={handleLogin}
          authDone={loginReady}
          isPending={isPending}
          translations={translations}
          className={`transition-[opacity,transform] duration-300 ${
            isExiting ? "opacity-0 scale-[0.98]" : "opacity-100 scale-100"
          }`}
        />
      </div>

      {/* Version number - fixed at the bottom of the page */}
      <div
        className={`relative z-10 flex-shrink-0 text-center py-4 transition-opacity duration-300 ${
          isReady && !isExiting ? "opacity-100" : "opacity-0"
        }`}
      >
        <p className="text-xs text-muted-foreground">
          {process.env.NEXT_PUBLIC_IMAGE_TAG || "dev"}
        </p>
      </div>
    </div>
  )
}
