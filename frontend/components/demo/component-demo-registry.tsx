"use client"

import { uiDemoMeta } from "@/components/demo/ui-demo-manifest"
import { businessDemoItems } from "@/components/demo/business-demo-registry.generated"

export type DemoEntry = {
  slug: string
  title: string
  description?: string
  group: string
  kind: "ui" | "business"
  loader: () => Promise<Record<string, unknown>>
  props?: Record<string, unknown>
  fallbackRoute?: string
}

const BUSINESS_GROUP_TITLES: Record<string, string> = {
  root: "根组件",
  common: "通用",
  dashboard: "仪表盘",
  scan: "扫描",
  target: "目标",
  organization: "组织",
  settings: "系统设置",
  tools: "工具",
  vulnerabilities: "漏洞",
  search: "搜索",
  fingerprints: "指纹",
  endpoints: "端点",
  directories: "目录",
  websites: "网站",
  subdomains: "子域名",
  "ip-addresses": "IP 地址",
  notifications: "通知",
  auth: "认证",
  disk: "磁盘",
  screenshots: "截图",
  animate: "动画",
  "animate-ui": "动画",
}

const BUSINESS_ROUTES: Record<string, string> = {
  dashboard: "/dashboard/",
  scan: "/scan/",
  target: "/target/",
  organization: "/organization/",
  settings: "/settings/workers/",
  tools: "/tools/",
  vulnerabilities: "/vulnerabilities/",
  search: "/search/",
  fingerprints: "/tools/fingerprints/",
  endpoints: "/target/",
  directories: "/target/",
  websites: "/target/",
  subdomains: "/target/",
  "ip-addresses": "/target/",
  notifications: "/settings/notifications/",
  auth: "/login/",
  disk: "/settings/database-health/",
  screenshots: "/target/",
}

const resolveBusinessDemoProps = (slug: string, group: string) => {
  const props: Record<string, unknown> = {}

  if (slug.includes("detail-view") || slug.includes("view")) {
    if (["websites", "subdomains", "directories", "endpoints", "ip-addresses", "screenshots"].includes(group)) {
      props.targetId = 1
    }
  }

  if (slug.includes("dialog") || slug.includes("drawer") || slug.includes("sheet")) {
    props.open = true
    props.onOpenChange = () => {}
  }

  if (slug.includes("data-table")) {
    props.data = []
    props.columns = []
  }

  return props
}

const createUiLoader = (slug: string) => async () => {
  const mod = await import("@/components/demo/ui-demo-registry")
  const Component = mod.getUiDemoComponent?.(slug)

  if (!Component) return {}

  return { default: Component }
}

const businessEntries: DemoEntry[] = businessDemoItems.map((item) => {
  const groupTitle = BUSINESS_GROUP_TITLES[item.group] || item.group
  const fallbackRoute = BUSINESS_ROUTES[item.group]
  const props = resolveBusinessDemoProps(item.slug, item.group)

  return {
    slug: `biz-${item.slug}`,
    title: item.title,
    description: `模块：${groupTitle}`,
    group: groupTitle,
    kind: "business",
    loader: item.loader,
    props,
    fallbackRoute,
  }
})

const uiEntries: DemoEntry[] = uiDemoMeta.map((item) => ({
  slug: `ui-${item.slug}`,
  title: item.title,
  description: item.description,
  group: item.group,
  kind: "ui",
  loader: createUiLoader(item.slug),
}))

export const demoEntries: DemoEntry[] = [...uiEntries, ...businessEntries]

export const demoMap = demoEntries.reduce<Record<string, DemoEntry>>((acc, item) => {
  acc[item.slug] = item
  return acc
}, {})

export const demoGroups = demoEntries.reduce<Record<string, DemoEntry[]>>((acc, item) => {
  const key = `${item.kind}:${item.group}`
  if (!acc[key]) acc[key] = []
  acc[key].push(item)
  return acc
}, {})
