import fs from "fs"
import path from "path"

export type ComponentGroup = {
  key: string
  title: string
  description?: string
  items: string[]
}

const GROUP_META: Record<string, { title: string; description?: string }> = {
  root: { title: "全局与壳层", description: "导航、布局、入口与全局组件" },
  common: { title: "通用组件", description: "多模块共享的基础组件" },
  dashboard: { title: "仪表盘", description: "统计、趋势与监控视图" },
  scan: { title: "扫描", description: "扫描历史、工作流与计划任务相关" },
  target: { title: "目标", description: "目标资产管理与详情视图" },
  organization: { title: "组织", description: "组织与关联资产管理" },
  settings: { title: "系统设置", description: "工作节点、日志与系统配置" },
  tools: { title: "工具", description: "工具配置、命令与词表管理" },
  vulnerabilities: { title: "漏洞", description: "漏洞列表与详情" },
  fingerprints: { title: "指纹", description: "指纹库与导入相关" },
  endpoints: { title: "端点", description: "资产端点与详情" },
  directories: { title: "目录", description: "目录扫描与列表视图" },
  websites: { title: "网站", description: "网站资产列表与详情" },
  subdomains: { title: "子域名", description: "子域名管理与详情" },
  "ip-addresses": { title: "IP 地址", description: "IP 资产视图" },
  search: { title: "搜索", description: "全局搜索与结果视图" },
  notifications: { title: "通知", description: "通知与提醒" },
  auth: { title: "认证", description: "登录与鉴权相关" },
  providers: { title: "Providers", description: "上下文与全局 Provider" },
  "animate-ui": { title: "动画与特效", description: "动效与背景组件" },
  prototypes: { title: "原型组件", description: "原型演示用组件" },
  disk: { title: "磁盘", description: "磁盘统计组件" },
  screenshots: { title: "截图", description: "截图视图组件" },
}

const GROUP_ORDER = [
  "root",
  "common",
  "dashboard",
  "scan",
  "target",
  "organization",
  "vulnerabilities",
  "search",
  "tools",
  "settings",
  "fingerprints",
  "endpoints",
  "directories",
  "websites",
  "subdomains",
  "ip-addresses",
  "notifications",
  "auth",
  "providers",
  "animate-ui",
  "prototypes",
  "disk",
  "screenshots",
]

const IGNORE_DIRS = new Set(["ui", "demo"])
const IGNORE_FILES = new Set(["index.ts", "index.tsx"])

function walk(dir: string, root: string, files: string[]) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (IGNORE_DIRS.has(entry.name)) continue
      walk(fullPath, root, files)
      continue
    }

    if (!entry.name.endsWith(".tsx")) continue
    if (IGNORE_FILES.has(entry.name)) continue
    if (entry.name.endsWith(".d.ts")) continue

    const rel = path.relative(root, fullPath)
    files.push(rel)
  }
}

export function getComponentIndex(): ComponentGroup[] {
  const root = path.join(process.cwd(), "components")
  const files: string[] = []
  walk(root, root, files)

  const groups = new Map<string, string[]>()

  for (const rel of files) {
    const normalized = rel.split(path.sep).join("/")
    const segments = normalized.split("/")
    const groupKey = segments.length > 1 ? segments[0] : "root"
    const name =
      segments.length > 1
        ? segments.slice(1).join("/").replace(/\\.tsx$/, "")
        : segments[0].replace(/\\.tsx$/, "")

    if (!groups.has(groupKey)) groups.set(groupKey, [])
    groups.get(groupKey)?.push(name)
  }

  const result: ComponentGroup[] = Array.from(groups.entries()).map(
    ([key, items]) => {
      const meta = GROUP_META[key] || { title: key }
      return {
        key,
        title: meta.title,
        description: meta.description,
        items: items.sort((a, b) => a.localeCompare(b)),
      }
    }
  )

  return result.sort((a, b) => {
    const ia = GROUP_ORDER.indexOf(a.key)
    const ib = GROUP_ORDER.indexOf(b.key)
    if (ia === -1 && ib === -1) return a.key.localeCompare(b.key)
    if (ia === -1) return 1
    if (ib === -1) return -1
    return ia - ib
  })
}
