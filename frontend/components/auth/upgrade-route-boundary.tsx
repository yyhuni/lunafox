"use client"

import * as React from "react"
import { usePathname, useRouter } from "next/navigation"
import { useTranslations } from "next-intl"

import { replaceWithRouteProgress } from "@/components/route-progress"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { semanticIcons } from "@/components/icons"
import { useUpgradeOperation } from "@/hooks/use-version"
import { textRole } from "@/lib/typography"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"

const SYSTEM_UPGRADE_PATH = "/system-upgrade/"

function normalizePathname(pathname: string | null): string {
  if (!pathname || pathname === "/") return pathname ?? ""
  return pathname.endsWith("/") ? pathname : `${pathname}/`
}

export function isSystemUpgradePathname(pathname: string | null): boolean {
  return normalizePathname(pathname) === SYSTEM_UPGRADE_PATH
}

function UpgradeRouteOwner({ resolving = false }: { resolving?: boolean }) {
  const t = useTranslations("systemUpgrade")
  const StatusIcon = resolving ? semanticIcons.status.unknown : semanticIcons.status.warning

  return (
    <main
      {...getLoadingOwnerAttributes({ owner: "upgrade-route-boundary", layer: "auth-shell", intent: "route" })}
      className="flex min-h-svh w-full items-center justify-center bg-background px-4 py-8 sm:px-8"
      data-testid="upgrade-route-boundary"
    >
      <Card className="w-full max-w-lg" variant="compact">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <StatusIcon className="size-5 text-primary" aria-hidden="true" />
            {resolving ? t("boundary.resolvingTitle") : t("boundary.lockedTitle")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className={textRole.bodySubtle}>
            {resolving ? t("boundary.resolvingDescription") : t("boundary.lockedDescription")}
          </p>
        </CardContent>
      </Card>
    </main>
  )
}

interface UpgradeRouteBoundaryProps {
  children: React.ReactNode
  renderProtectedShell: (children: React.ReactNode) => React.ReactNode
}

/**
 * Keeps ordinary authenticated routes out of the tree while an accepted
 * upgrade is non-terminal. The server Operation, rather than a browser event,
 * decides when the boundary can release the application again.
 */
export function UpgradeRouteBoundary({ children, renderProtectedShell }: UpgradeRouteBoundaryProps) {
  const pathname = usePathname()
  const router = useRouter()
  const isUpgradeRoute = isSystemUpgradePathname(pathname)
  const [hydrated, setHydrated] = React.useState(false)
  const operation = useUpgradeOperation(undefined, { enabled: hydrated && !isUpgradeRoute })
  const redirectStartedRef = React.useRef(false)
  const shouldLock = !isUpgradeRoute && (
    !hydrated ||
    operation.isResolving ||
    operation.isReconnecting ||
    operation.isError ||
    operation.isActive
  )
  const shouldRedirect = hydrated && !isUpgradeRoute && (
    operation.isReconnecting ||
    operation.isError ||
    operation.isActive
  )

  React.useEffect(() => {
    setHydrated(true)
  }, [])

  React.useEffect(() => {
    if (!shouldRedirect) return
    if (redirectStartedRef.current) return
    redirectStartedRef.current = true
    replaceWithRouteProgress(router, SYSTEM_UPGRADE_PATH)
  }, [router, shouldRedirect])

  React.useEffect(() => {
    if (!shouldLock) redirectStartedRef.current = false
  }, [shouldLock])

  if (isUpgradeRoute) return <>{children}</>
  if (shouldLock) return <UpgradeRouteOwner resolving={!hydrated || operation.isResolving} />
  return <>{renderProtectedShell(children)}</>
}
