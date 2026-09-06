"use client"

import React from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { useQueryClient } from "@tanstack/react-query"
import { VisualSplitLogin } from "@/components/auth/visual-split-login"
import { replaceWithRouteProgress } from "@/components/route-progress"
import { ContentReveal } from "@/components/shared/loading/content-reveal"
import { usePrefetchOverviewData } from "@/hooks/use-overview-prefetch"
import { useLogin, useAuth } from "@/hooks/use-auth"
import { DEFAULT_AUTH_RETURN_TO, resolveSafeReturnTo } from "@/lib/auth-runtime.mjs"

export default function LoginPage() {
  const defaultReturnTo = DEFAULT_AUTH_RETURN_TO
  const router = useRouter()
  const searchParams = useSearchParams()
  const queryClient = useQueryClient()
  const prefetchOverviewData = usePrefetchOverviewData()
  const { data: auth, isLoading: authLoading } = useAuth()
  const { mutateAsync: login, isPending } = useLogin()
  const tVisualLogin = useTranslations("auth.visualLogin")
  const safeReturnTo = React.useMemo(
    () => resolveSafeReturnTo(searchParams.get("returnTo")),
    [searchParams]
  )
  const nextRoute = safeReturnTo ?? defaultReturnTo

  const loginStartedRef = React.useRef(false)
  const [loginReady, setLoginReady] = React.useState(false)
  const [loginVisualReady, setLoginVisualReady] = React.useState(false)
  const [isExiting, setIsExiting] = React.useState(false)
  const exitStartedRef = React.useRef(false)
  const showExitOverlay = isExiting
  const shouldKeepLoginSurfaceForHydration = authLoading || auth?.authenticated !== true
  const showLoginSurface =
    !showExitOverlay &&
    (shouldKeepLoginSurfaceForHydration || loginStartedRef.current)

  // If already logged in, warm up the overview, then redirect.
  React.useEffect(() => {
    if (authLoading) return
    if (!auth?.authenticated) return
    if (loginStartedRef.current) return

    let cancelled = false
    let timer: number | undefined

    void (async () => {
      await prefetchOverviewData()

      if (cancelled) return
      if (!exitStartedRef.current) {
        exitStartedRef.current = true
        setIsExiting(true)
        timer = window.setTimeout(() => {
          replaceWithRouteProgress(router, nextRoute)
        }, 300)
      }
    })()

    return () => {
      cancelled = true
      if (timer) window.clearTimeout(timer)
    }
  }, [auth?.authenticated, authLoading, nextRoute, prefetchOverviewData, router])

  React.useEffect(() => {
    if (!loginReady) return
    if (exitStartedRef.current) return
    exitStartedRef.current = true
    setIsExiting(true)
    const timer = window.setTimeout(() => {
      replaceWithRouteProgress(router, nextRoute)
    }, 300)
    return () => window.clearTimeout(timer)
  }, [loginReady, nextRoute, router])

  React.useEffect(() => {
    if (!showLoginSurface) {
      setLoginVisualReady(false)
    }
  }, [showLoginSurface])

  const handleLoginVisualReady = React.useCallback(() => {
    setLoginVisualReady(true)
  }, [])

  const handleLogin = async (username: string, password: string) => {
    loginStartedRef.current = true
    setLoginReady(false)

    try {
      // Warm the destination route immediately and only hydrate overview data for its canonical entry.
      const [loginRes] = await Promise.all([
        login({ username, password }),
        router.prefetch(nextRoute),
      ])

      if (nextRoute === defaultReturnTo) {
        await prefetchOverviewData()
      }

      // Prime auth cache so AuthLayout doesn't flash a full-screen loading state.
      queryClient.setQueryData(["auth", "me"], {
        authenticated: true,
        user: loginRes.user,
      })

      setLoginReady(true)
    } catch (error) {
      loginStartedRef.current = false
      throw error
    }
  }

  return (
    <div
      data-auth-theme="public"
      data-boot-handoff-pending={showLoginSurface && !loginVisualReady ? "true" : undefined}
      className="bg-background relative flex min-h-svh flex-col text-foreground"
    >
      {showExitOverlay ? (
        <div className="bg-background fixed inset-0 z-50" />
      ) : null}

      {/* Fingerprint identifier - for FOFA/Shodan and other search engines to identify */}
      <meta name="generator" content="LunaFox ASM Platform" />

      <div className="relative z-10 flex flex-1">
        {showLoginSurface ? (
          <ContentReveal
            owner="login-page-content"
            className="flex w-full justify-center"
          >
            <VisualSplitLogin
              onLogin={handleLogin}
              authDone={loginReady}
              isPending={isPending}
              onVisualReady={handleLoginVisualReady}
              translations={{
                ariaLabel: tVisualLogin("ariaLabel"),
                consoleLabel: tVisualLogin("consoleLabel"),
                usernameLabel: tVisualLogin("usernameLabel"),
                usernamePlaceholder: tVisualLogin("usernamePlaceholder"),
                passwordLabel: tVisualLogin("passwordLabel"),
                passwordPlaceholder: tVisualLogin("passwordPlaceholder"),
                submit: tVisualLogin("submit"),
                accessHelp: tVisualLogin("accessHelp"),
              }}
              className={isExiting ? "opacity-0" : "opacity-100"}
            />
          </ContentReveal>
        ) : null}
      </div>
    </div>
  )
}
