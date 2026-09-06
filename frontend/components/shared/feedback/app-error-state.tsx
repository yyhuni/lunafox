"use client"

import Link from "next/link"
import { useRouter } from "next/navigation"
import { useLocale } from "next-intl"

import { semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"
import type { AppError, AppErrorKind } from "@/lib/errors/app-error"
import {
  getStatusToneSurfaceClass,
  getStatusToneTextClass,
  type StatusTone,
} from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export type AppErrorStateVariant = "page" | "section"

type AppErrorStateProps = {
  error: AppError
  title?: string
  description?: string
  resourceLabel?: string
  onRetry?: (() => void) | (() => Promise<unknown>)
  actionHref?: string
  actionLabel?: string
  variant?: AppErrorStateVariant
  className?: string
}

const ERROR_COPY: Record<
  "en" | "zh",
  Record<
    AppErrorKind,
    {
      title: (resourceLabel: string) => string
      description: (resourceLabel: string) => string
    }
  >
> = {
  en: {
    "resource-not-found": {
      title: (resourceLabel) => `${resourceLabel} not found`,
      description: (resourceLabel) =>
        `The requested ${resourceLabel.toLowerCase()} no longer exists or is no longer available in this workspace.`,
    },
    "permission-denied": {
      title: () => "Access denied",
      description: () => "Your current account does not have permission to view this content.",
    },
    "rate-limited": {
      title: () => "Too many requests",
      description: () => "The service is throttling requests right now. Please wait a moment and try again.",
    },
    "service-unavailable": {
      title: () => "Service unavailable",
      description: () => "The service is temporarily unavailable. Please retry in a moment.",
    },
    "network-error": {
      title: () => "Network connection lost",
      description: () => "We could not reach the service. Check your connection and try again.",
    },
    "unexpected-error": {
      title: () => "Could not load this content",
      description: () => "An unexpected problem interrupted this view. Please try again.",
    },
  },
  zh: {
    "resource-not-found": {
      title: (resourceLabel) => `未找到${resourceLabel}`,
      description: (resourceLabel) => `请求的${resourceLabel}不存在，或当前工作区已无法访问该资源。`,
    },
    "permission-denied": {
      title: () => "没有访问权限",
      description: () => "当前账号无权查看此内容。",
    },
    "rate-limited": {
      title: () => "请求过于频繁",
      description: () => "服务当前正在限流，请稍后再试。",
    },
    "service-unavailable": {
      title: () => "服务暂时不可用",
      description: () => "服务暂时不可用，请稍后重试。",
    },
    "network-error": {
      title: () => "网络连接异常",
      description: () => "暂时无法连接到服务，请检查网络后重试。",
    },
    "unexpected-error": {
      title: () => "内容加载失败",
      description: () => "当前视图遇到了未预期的问题，请重试。",
    },
  },
}

const ERROR_PRESENTATION: Record<
  AppErrorKind,
  {
    icon: (typeof semanticIcons.status)[keyof typeof semanticIcons.status]
    tone: StatusTone
  }
> = {
  "resource-not-found": { icon: semanticIcons.status.warning, tone: "warning" },
  "permission-denied": { icon: semanticIcons.status.disabled, tone: "muted" },
  "rate-limited": { icon: semanticIcons.status.warning, tone: "warning" },
  "service-unavailable": { icon: semanticIcons.status.error, tone: "error" },
  "network-error": { icon: semanticIcons.status.warning, tone: "warning" },
  "unexpected-error": { icon: semanticIcons.status.error, tone: "error" },
}

const ERROR_VARIANT_CLASSNAMES = {
  page: {
    root: "py-12",
    iconShell: "mb-4 p-3",
    icon: "size-10",
    title: cn("mb-2", textRole.panelTitle),
    description: "max-w-xl",
    actions: "mt-5",
    buttonSize: "default" as const,
  },
  section: {
    root: "py-8",
    iconShell: "mb-3 p-2",
    icon: "size-6",
    title: cn("mb-1.5", textRole.sectionTitle),
    description: "max-w-lg",
    actions: "mt-4",
    buttonSize: "sm" as const,
  },
} satisfies Record<
  AppErrorStateVariant,
  {
    root: string
    iconShell: string
    icon: string
    title: string
    description: string
    actions: string
    buttonSize: "default" | "sm"
  }
>

export function AppErrorState({
  error,
  title,
  description,
  resourceLabel,
  onRetry,
  actionHref,
  actionLabel,
  variant = "page",
  className,
}: AppErrorStateProps) {
  const router = useRouter()
  const locale = useLocale()
  const copyLocale = locale === "zh" ? "zh" : "en"
  const resolvedResourceLabel = resourceLabel ?? (copyLocale === "zh" ? "资源" : "Resource")
  const copy = ERROR_COPY[copyLocale][error.kind]
  const presentation = ERROR_PRESENTATION[error.kind]
  const variantClassNames = ERROR_VARIANT_CLASSNAMES[variant]
  const Icon = presentation.icon
  const showRetry = error.retryable && typeof onRetry === "function"
  const resolvedActionLabel = actionLabel ?? (copyLocale === "zh" ? "返回" : "Back")

  return (
    <div
      role="alert"
      aria-atomic="true"
      className={cn(
        "flex flex-col items-center justify-center text-center",
        variantClassNames.root,
        className
      )}
      data-app-error-kind={error.kind}
      data-app-error-variant={variant}
    >
      <div
        className={cn(
          "radius-round",
          variantClassNames.iconShell,
          getStatusToneSurfaceClass(presentation.tone)
        )}
      >
        <Icon
          aria-hidden="true"
          className={cn(
            variantClassNames.icon,
            getStatusToneTextClass(presentation.tone)
          )}
        />
      </div>

      <h2 className={variantClassNames.title}>
        {title ?? copy.title(resolvedResourceLabel)}
      </h2>
      <p className={cn(variantClassNames.description, textRole.bodySubtle)}>
        {description ?? copy.description(resolvedResourceLabel)}
      </p>

      <div className={cn("flex flex-wrap items-center justify-center gap-3", variantClassNames.actions)}>
        {showRetry ? (
          <Button
            type="button"
            size={variantClassNames.buttonSize}
            onClick={() => {
              void onRetry?.()
            }}
          >
            {copyLocale === "zh" ? "重试" : "Retry"}
          </Button>
        ) : null}

        {actionHref ? (
          <Button
            type="button"
            size={variantClassNames.buttonSize}
            variant={showRetry ? "outline" : "default"}
            render={<Link href={actionHref} />}
          >
            {resolvedActionLabel}
          </Button>
        ) : null}

        {!showRetry && !actionHref ? (
          <Button
            type="button"
            size={variantClassNames.buttonSize}
            variant="outline"
            onClick={() => router.back()}
          >
            {resolvedActionLabel}
          </Button>
        ) : null}
      </div>
    </div>
  )
}
