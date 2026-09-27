"use client"

import * as React from "react"
import { useLocale, useTranslations } from "next-intl"

import { Clock, InfoIcon, semanticIcons } from "@/components/icons"
import { PageHeader } from "@/components/common/page-header"
import { EngineDurationInput } from "@/components/scan/engine-duration-input"
import {
  InitiateScanConfigStep,
  InitiateScanStepHeader,
} from "@/components/scan/initiate-scan-dialog-sections"
import { QuickScanFooter, QuickScanHeader } from "@/components/scan/quick-scan-dialog-sections"
import type { EngineParamControlRenderer } from "@/components/scan/engine-config-form"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Drawer, DrawerContent } from "@/components/ui/drawer"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_FORM_OVERLAY_HORIZONTAL_INSET_CLASS,
  COMPACT_FORM_OVERLAY_INSET_CLASS,
  COMPACT_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"
import { scanWorkbenchDrawerContentClassName } from "@/lib/ui/overlay-styles"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Locale } from "@/i18n/config"
import type { EngineCatalogDetail } from "@/types/engine-catalog.types"
import type { ScanWorkflow, ScanWorkflowStepView } from "@/types/scan-workflow.types"

const PROTOTYPE_TIMEOUT_MINIMUM_SECONDS = 60
const PROTOTYPE_TIMEOUT_DEFAULT_SECONDS = 14400

const COPY = {
  zh: {
    prototype: "仅 UI 原型",
    prototypeHint: "复用真实扫描抽屉和引擎配置表单，不会提交扫描",
    workspaceTitle: "扫描配置 UI 原型",
    workspaceDescription: "在现有扫描配置界面中验证秒数时长滚轮选择器。",
    openDrawer: "打开扫描配置",
    prototypeOnly: "原型状态：修改只保存在当前页面内",
  },
  en: {
    prototype: "UI prototype only",
    prototypeHint: "Reuses the real scan drawer and engine form; no scan is submitted",
    workspaceTitle: "Scan configuration UI prototype",
    workspaceDescription: "Verify the seconds duration wheel picker inside the existing scan configuration surface.",
    openDrawer: "Open scan configuration",
    prototypeOnly: "Prototype state: changes stay in this page",
  },
} as const

type Copy = (typeof COPY)[keyof typeof COPY]

const PROTOTYPE_STEP: ScanWorkflowStepView = {
  stageId: "discovery",
  stepId: "httpx",
  engineId: "engine.lunafox.httpx",
  profileDefaultEnabled: true,
}

const PROTOTYPE_WORKFLOW: ScanWorkflow = {
  name: "Website Discovery",
  displayName: "Website Discovery",
  description: "A local fixture for the seconds duration picker.",
  stages: [{ stageId: "discovery", steps: [PROTOTYPE_STEP] }],
  steps: [PROTOTYPE_STEP],
  isBuiltin: true,
  isExecutable: true,
  etag: "prototype",
  createTime: "2026-01-01T00:00:00Z",
  updateTime: "2026-01-01T00:00:00Z",
}

const PROTOTYPE_CONFIG = `steps:
  httpx:
    enabled: true
    engineConfig:
      httpx:
        enabled: true
        timeout: ${PROTOTYPE_TIMEOUT_DEFAULT_SECONDS}
        request-timeout: 30
        rate: 20
        threads: 10`

const PROTOTYPE_ENGINE: EngineCatalogDetail = {
  name: "engines/httpx",
  engineId: "engine.lunafox.httpx",
  manifestVersion: "engine.v5",
  publisher: "lunafox",
  packageVersion: "prototype",
  artifactRef: "prototype://engine.lunafox.httpx",
  packageDigest: "prototype",
  execution: {
    engineApiMajor: 2,
    supportedTargetTypes: ["domain"],
    configSections: [{
      id: "httpx",
      defaultEnabled: true,
      requiredEnabled: true,
      params: [
        {
          key: "timeout",
          type: "integer",
          unit: "seconds",
          default: PROTOTYPE_TIMEOUT_DEFAULT_SECONDS,
          minimum: PROTOTYPE_TIMEOUT_MINIMUM_SECONDS,
        },
        {
          key: "request-timeout",
          type: "integer",
          unit: "seconds",
          default: 30,
          minimum: 1,
          maximum: 120,
        },
        {
          key: "rate",
          type: "integer",
          default: 20,
          minimum: 1,
          maximum: 1000,
        },
        {
          key: "threads",
          type: "integer",
          default: 10,
          minimum: 1,
          maximum: 100,
        },
      ],
    }],
  },
  localeResources: {
    zh: {
      engine: { displayName: "HTTPX", description: "网站发现引擎" },
      sections: {
        httpx: {
          name: "Website Discovery",
          description: "发现目标上的网站和 HTTP 服务。",
          params: {
            timeout: { description: "HTTPX 进程允许运行的最长时间。" },
            "request-timeout": { description: "单个 HTTP 请求的超时上限。" },
            rate: { description: "出站请求的速率限制。" },
            threads: { description: "并行执行的工作线程数。" },
          },
        },
      },
    },
    en: {
      engine: { displayName: "HTTPX", description: "Website discovery engine" },
      sections: {
        httpx: {
          name: "Website Discovery",
          description: "Discover websites and HTTP services on the target.",
          params: {
            timeout: { description: "Maximum time allowed for the HTTPX process." },
            "request-timeout": { description: "Timeout limit for an individual HTTP request." },
            rate: { description: "Outgoing request rate limit." },
            threads: { description: "Number of workers running in parallel." },
          },
        },
      },
    },
  },
}

function PrototypeWorkspace({
  copy,
  onOpen,
}: {
  copy: Copy
  onOpen: () => void
}) {
  return (
    <div className={cn(COMPACT_PAGE_SHELL_CLASS, "min-w-0")}>
      <PageHeader
        code="SCN-02"
        title={copy.workspaceTitle}
        description={copy.workspaceDescription}
        descriptionSupplement={<Badge variant="warning">{copy.prototype}</Badge>}
      />
      <div className={cn(COMPACT_CONTENT_GUTTER_CLASS, "min-w-0 pb-6")}>
        <div className="mx-auto flex w-full max-w-406 flex-col gap-3">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border/70 px-1 pb-3">
            <div className="flex min-w-0 items-start gap-2">
              <Clock aria-hidden className="mt-0.5 size-4 shrink-0 text-primary" />
              <div className="min-w-0">
                <p className={textRole.sectionTitle}>Issue #161 · timeout 控件评估</p>
                <p className={cn("mt-1", textRole.bodySubtle)}>{copy.prototypeHint}</p>
              </div>
            </div>
            <Button type="button" variant="outline" size="sm" onClick={onOpen}>
              <semanticIcons.action.edit aria-hidden />
              {copy.openDrawer}
            </Button>
          </div>
          <div className="flex items-start gap-2 px-1 text-muted-foreground">
            <InfoIcon aria-hidden className="mt-0.5 size-4 shrink-0" />
            <p className={textRole.helperText}>{copy.prototypeOnly}</p>
          </div>
        </div>
      </div>
    </div>
  )
}

export function EngineDurationInputPrototype() {
  const localeValue = useLocale()
  const locale: "zh" | "en" = localeValue === "zh" ? "zh" : "en"
  const appLocale = locale as Locale
  const copy = COPY[locale]
  const tQuickScan = useTranslations("quickScan")
  const tActions = useTranslations("common.actions")
  const tInitiate = useTranslations("scan.initiate")
  const [drawerOpen, setDrawerOpen] = React.useState(true)
  const [configuration, setConfiguration] = React.useState(PROTOTYPE_CONFIG)
  const [isConfigEdited, setIsConfigEdited] = React.useState(false)
  const [isYamlValid, setIsYamlValid] = React.useState(true)
  const [prototypeNotice, setPrototypeNotice] = React.useState(false)
  const defaultExpandedStepIds = React.useMemo(() => new Set([PROTOTYPE_STEP.stepId]), [])

  const renderParamControl = React.useCallback<EngineParamControlRenderer>((context) => {
    if (context.param.type !== "integer" || context.param.unit !== "seconds") return null
    const currentValue = typeof context.value === "number" ? context.value : 0
    return (
      <EngineDurationInput
        id={context.fieldId}
        value={currentValue}
        defaultValue={typeof context.param.default === "number" ? context.param.default : undefined}
        minimum={context.param.minimum}
        maximum={context.param.maximum}
        disabled={context.disabled}
        onChange={context.onChange}
      />
    )
  }, [])

  const handleConfigChange = React.useCallback((nextConfiguration: string) => {
    setConfiguration(nextConfiguration)
    setIsConfigEdited(true)
    setPrototypeNotice(false)
  }, [])

  const handleConfigSync = React.useCallback((nextConfiguration: string) => {
    setConfiguration(nextConfiguration)
  }, [])

  const resetConfiguration = React.useCallback(() => {
    setConfiguration(PROTOTYPE_CONFIG)
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setPrototypeNotice(false)
  }, [])

  const steps = [
    { id: 1, title: tQuickScan("scanTargets") },
    { id: 2, title: tInitiate("steps.selectWorkflow") },
    { id: 3, title: tInitiate("steps.engineConfig") },
  ]

  return (
    <>
      <PrototypeWorkspace copy={copy} onOpen={() => setDrawerOpen(true)} />

      <Drawer open={drawerOpen} onOpenChange={setDrawerOpen} swipeDirection="right">
        <DrawerContent
          data-prototype="engine-duration"
          className={cn(
            scanWorkbenchDrawerContentClassName,
            "data-[swipe-direction=right]:sm:max-w-[640px]",
          )}
        >
          <form className="flex h-full min-h-0 w-full flex-col">
            <QuickScanHeader
              t={tQuickScan}
              step={3}
              totalSteps={3}
              closeLabel={tActions("close")}
              isSubmitting={false}
              stepHeader={<InitiateScanStepHeader steps={steps} currentStep={3} progress={100} />}
            />

            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <section
                className={cn(
                  "flex min-h-0 flex-1 flex-col overflow-hidden py-3",
                  COMPACT_FORM_OVERLAY_HORIZONTAL_INSET_CLASS,
                )}
              >
                <InitiateScanConfigStep
                  t={tInitiate}
                  configuration={configuration}
                  selectedWorkflows={[PROTOTYPE_WORKFLOW]}
                  isConfigEdited={isConfigEdited}
                  isYamlValid={isYamlValid}
                  hasConfig={configuration.trim().length > 0}
                  selectMode="custom"
                  selectedWorkflowNames={[PROTOTYPE_WORKFLOW.name]}
                  locale={appLocale}
                  engineCatalogDetails={[PROTOTYPE_ENGINE]}
                  isEngineCatalogLoading={false}
                  isEngineCatalogError={false}
                  isSubmitting={false}
                  onConfigSync={handleConfigSync}
                  onConfigChange={handleConfigChange}
                  onResetConfig={resetConfiguration}
                  onYamlValidationChange={setIsYamlValid}
                  renderParamControl={renderParamControl}
                  defaultExpandedStepIds={defaultExpandedStepIds}
                />
              </section>
            </div>

            <div className={cn("shrink-0 border-t bg-card", COMPACT_FORM_OVERLAY_INSET_CLASS)}>
              {prototypeNotice ? (
                <div className="pb-2">
                  <Badge variant="info">{copy.prototypeOnly}</Badge>
                </div>
              ) : null}
              <QuickScanFooter
                t={tQuickScan}
                step={3}
                validCount={0}
                invalidCount={0}
                selectedScanWorkflowCount={1}
                isSubmitting={false}
                canProceedToStep2={false}
                canProceedToStep3={false}
                canSubmit={isYamlValid && configuration.trim().length > 0}
                onBack={() => setDrawerOpen(false)}
                onNext={() => undefined}
                onSubmit={() => setPrototypeNotice(true)}
              />
            </div>
          </form>
        </DrawerContent>
      </Drawer>
    </>
  )
}
