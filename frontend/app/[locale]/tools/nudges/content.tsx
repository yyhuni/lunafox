"use client"

import * as React from "react"
import { toast } from "sonner"
import {
  IconActivity,
  IconAlertTriangle,
  IconCalendar,
  IconCoffee,
  IconCpu,
  IconHeart,
  IconLock,
  IconMoon,
  IconSun,
  IconTerminal,
  IconUser,
  IconTrophy,
  IconBug,
} from "@/components/icons"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { NudgeToastCard } from "@/components/nudges/nudge-toast-card"
import { PageHeader } from "@/components/common/page-header"

interface DemoVariant {
  id: string
  icon: React.ReactNode
  title: string
  desc: string
  primaryAction: { label: string }
  secondaryAction?: { label: string }
}

interface AiNudgeResult {
  icon?: string
  title: string
  description: string
  primaryAction?: { label: string }
  secondaryAction?: { label: string }
}

// --- Copy Variants ---

const HEALTH_VARIANTS = [
  {
    id: "late-night",
    icon: <IconMoon className="size-6 text-indigo-500" />,
    title: "凌晨修仙警告",
    desc: "指挥官，现在的流量最干净，但你的发际线正在报警。守护 Root 权限的同时，别忘了守护发量。",
    primaryAction: { label: "睡了睡了" },
    secondaryAction: { label: "再挖一个洞" },
  },
  {
    id: "morning",
    icon: <IconSun className="size-6 text-orange-500" />,
    title: "早起的黑客有洞挖",
    desc: "当别人还在睡梦中，你已经完成了第一波资产测绘。今天的目标：拿下一个 Shell。",
    primaryAction: { label: "Get Shell!" },
  },
  {
    id: "posture",
    icon: <IconActivity className="size-6 text-emerald-500" />,
    title: "姿势不对，Shell 不会",
    desc: "长期低头会压迫颈椎，导致 Payload 构造思路受阻。建议立即执行：stand_up(); stretch();",
    primaryAction: { label: "活动一下" },
    secondaryAction: { label: "再坐五百年" },
  },
  {
    id: "food",
    icon: <IconCpu className="size-6 text-rose-500" />,
    title: "缓冲区下溢警告",
    desc: "检测到你的胃部缓冲区即将 Underflow。请立即执行 eat() 函数，防止身体 Crash。",
    primaryAction: { label: "去干饭" },
    secondaryAction: { label: "还能再扛" },
  },
  {
    id: "eyes",
    icon: <IconAlertTriangle className="size-6 text-amber-500" />,
    title: "视觉模块过热",
    desc: "你盯着屏幕太久了，视网膜可能出现了残影。去窗边看看有没有真的狐狸（虽然大概率没有）。",
    primaryAction: { label: "远眺一下" },
  },
  {
    id: "overtime",
    icon: <IconMoon className="size-6 text-indigo-500" />,
    title: "还在肝？",
    desc: "听说深夜的代码 Bug 更少？也许吧，但发际线后移的速度肯定更快。保重啊，指挥官。",
    primaryAction: { label: "最后一行" },
    secondaryAction: { label: "再战三百回" },
  },
  {
    id: "late-food",
    icon: <IconCoffee className="size-6 text-amber-700" />,
    title: "深夜食堂",
    desc: "这个点还在屏幕前，是不是饿了？泡面虽好，可不要贪杯。或者去阳台看看星星（如果有的话）。",
    primaryAction: { label: "去觅食" },
    secondaryAction: { label: "我不饿" },
  },
]

const SOCIAL_VARIANTS = [
  {
    id: "single",
    icon: <IconUser className="size-6 text-pink-500" />,
    title: "面向对象编程",
    desc: "找不到对象？没关系，new 一个就行了。但在现实里，可能需要你合上电脑出去走走。",
    primaryAction: { label: "我去 new 一个" },
    secondaryAction: { label: "代码就是恋人" },
  },
  {
    id: "hair",
    icon: <IconActivity className="size-6 text-stone-500" />,
    title: "秃头风险评估",
    desc: "根据你的 commit 频率分析，你的发际线后移概率增加了 15%。建议配合生发液使用本系统。",
    primaryAction: { label: "扎心了" },
    secondaryAction: { label: "我变强了" },
  },
  {
    id: "coffee",
    icon: <IconCoffee className="size-6 text-amber-700" />,
    title: "咖啡因浓度过低",
    desc: "SQL 注入失败？也许不是 Payload 的问题，是你血液里的咖啡因不足了。来杯 Java 提提神。",
    primaryAction: { label: "去倒咖啡" },
    secondaryAction: { label: "红牛万岁" },
  },
]

const HOLIDAY_VARIANTS = [
  {
    id: "friday",
    icon: <IconCalendar className="size-6 text-purple-500" />,
    title: "Read-Only Friday",
    desc: "周五不改生产库，这是江湖规矩。除非你想在这个周末收到报警短信，否则请放下手中的 DROP TABLE。",
    primaryAction: { label: "遵命" },
    secondaryAction: { label: "头铁" },
  },
  {
    id: "holiday-work",
    icon: <IconHeart className="size-6 text-pink-500" />,
    title: "节假日加班警告",
    desc: "大过节的还在挖洞？你的敬业程度让我 CPU 温度都升高了。记得给自己发三倍工资（或者三倍快乐水）。",
    primaryAction: { label: "搞完就撤" },
    secondaryAction: { label: "我爱加班" },
  },
  {
    id: "1024",
    icon: <IconTerminal className="size-6 text-green-500" />,
    title: "1024 Happy Hacking!",
    desc: "愿你手里全是 0day，扫出的资产全是弱口令，WAF 对你永远放行。节日快乐！",
    primaryAction: { label: "收下祝福" },
  },
  {
    id: "cny",
    icon: <IconHeart className="size-6 text-red-500" />,
    title: "新春快乐",
    desc: "代码 freeze，烦恼 delete，好运 commit，红包 push！给您拜年啦！🧧",
    primaryAction: { label: "红包拿来" },
  },
]

const MEME_VARIANTS = [
  {
    id: "rmrf",
    icon: <IconAlertTriangle className="size-6 text-red-600" />,
    title: "rm -rf /*",
    desc: "这是一个危险的命令… 哪怕你是在虚拟机里。时刻保持敬畏，记得备份，记得看清你在哪个窗口。",
    primaryAction: { label: "已经在跑路了" },
  },
  {
    id: "password",
    icon: <IconLock className="size-6 text-blue-500" />,
    title: "密码是 123456？",
    desc: "希望不是。否则我的字典生成器第一秒就能跑出来。去改个强密码吧，比如 P@$$w0rd (开玩笑的)。",
    primaryAction: { label: "我很强" },
    secondaryAction: { label: "这就改" },
  },
]

const MILESTONE_VARIANTS = [
  {
    id: "day-1",
    icon: <IconHeart className="size-6 text-pink-500" />,
    title: "Hello World! 👋",
    desc: "这是我们在月狐控制台共度的第一天。很高兴认识你，指挥官。",
    primaryAction: { label: "你好呀" },
  },
  {
    id: "day-7",
    icon: <IconTrophy className="size-6 text-yellow-500" />,
    title: "第一周达成！🏅",
    desc: "已经过去一周了。你的资产库是不是也跟着胖了一圈？保持这个节奏！",
    primaryAction: { label: "继续冲" },
  },
  {
    id: "day-30",
    icon: <IconMoon className="size-6 text-indigo-500" />,
    title: "满月纪念 🌕",
    desc: "30 天的陪伴。这一个月里，感谢你为了互联网安全所做的每一次扫描。",
    primaryAction: { label: "干杯" },
  },
  {
    id: "day-100",
    icon: <IconActivity className="size-6 text-emerald-500" />,
    title: "百日修仙达成 💯",
    desc: "100 天的坚持。今天的你，一定比 100 天前更强了。",
    primaryAction: { label: "确实" },
    secondaryAction: { label: "还得练" },
  },
]

// --- Local trigger rules (mirrors useNudgeGuardian) ---
const LOCAL_TRIGGER_VARIANTS = [
  {
    id: "late-night-1",
    icon: <IconMoon className="size-6 text-indigo-500" />,
    title: "深夜修仙警告",
    desc: "指挥官,流量最干净的时候确实适合挖洞,但发际线也在报警。守护 Root 权限的同时,别忘了守护发量。",
    primaryAction: { label: "再熬半小时" },
    secondaryAction: { label: "这就睡" },
    trigger: "23:00 - 04:00 自动触发",
  },
  {
    id: "late-night-3",
    icon: <IconCoffee className="size-6 text-amber-600" />,
    title: "夜深了",
    desc: "这个点还在屏幕前,是不是饿了？泡面虽好,可不要贪杯。或者去阳台看看星星（如果有的话）,然后关机。",
    primaryAction: { label: "去看星星" },
    secondaryAction: { label: "关机" },
    trigger: "23:00 - 04:00 自动触发",
  },
  {
    id: "late-night-4",
    icon: <IconMoon className="size-6 text-purple-500" />,
    title: "凌晨 1 点的月狐",
    desc: "你和服务器是现在唯一醒着的伙伴。别太累了,服务器有 UPS 撑着,你可没有。",
    primaryAction: { label: "陪它聊会儿" },
    secondaryAction: { label: "去休息" },
    trigger: "23:00 - 04:00 自动触发",
  },
  {
    id: "long-session-1",
    icon: <IconActivity className="size-6 text-emerald-500" />,
    title: "姿势不对,Shell 不会",
    desc: "长期低头会压迫颈椎,导致 Payload 构造思路受阻。建议立即执行：stand_up(); stretch();",
    primaryAction: { label: "起来活动" },
    secondaryAction: { label: "再坐会儿" },
    trigger: "连续使用 2 小时后自动触发",
  },
  {
    id: "long-session-2",
    icon: <IconSun className="size-6 text-amber-500" />,
    title: "视觉模块过热",
    desc: "你盯着屏幕太久了,视网膜可能出现了残影。去窗边看看有没有真的狐狸,或者哪怕只是看看那棵树。",
    primaryAction: { label: "去看狐狸" },
    secondaryAction: { label: "看看树" },
    trigger: "连续使用 2 小时后自动触发",
  },
  {
    id: "long-session-3",
    icon: <IconCpu className="size-6 text-indigo-500" />,
    title: "久坐提醒",
    desc: "已经连续战斗 2 小时了。起来接杯水,活动一下腰背,为了更长远的黑客生涯。",
    primaryAction: { label: "接水去" },
    secondaryAction: { label: "活动腰背" },
    trigger: "连续使用 2 小时后自动触发",
  },
  {
    id: "long-session-4",
    icon: <IconCoffee className="size-6 text-cyan-500" />,
    title: "Drink Water",
    desc: "多喝水,多排毒。身体是革命的本钱,也是挖洞的本钱。",
    primaryAction: { label: "去接水" },
    secondaryAction: { label: "一会儿喝" },
    trigger: "连续使用 2 小时后自动触发",
  },
  {
    id: "cyber-fox",
    icon: <IconBug className="size-6 text-orange-500" />,
    title: "系统监测到毛茸茸生物",
    desc: "🦊 嘤？LunaFox 的吉祥物跑出来了！它似乎对你的鼠标指针很好奇。工作累了就陪它玩会儿吧。",
    primaryAction: { label: "陪它玩" },
    secondaryAction: { label: "摸摸头" },
    trigger: "15:00-16:00 随机偶遇 (1% 概率)",
  },
]

const NUDGE_TOAST_ID = "nudge-singleton"

export default function NudgeDemoPage() {
  const triggerToast = (variant: DemoVariant) => {
    toast.custom(() => (
      <NudgeToastCard
        title={variant.title}
        description={variant.desc}
        icon={variant.icon}
        primaryAction={{
          ...variant.primaryAction,
          onClick: () => toast.dismiss(NUDGE_TOAST_ID),
        }}
        secondaryAction={
          variant.secondaryAction
            ? { ...variant.secondaryAction, onClick: () => toast.dismiss(NUDGE_TOAST_ID), buttonVariant: "outline" }
            : undefined
        }
        onDismiss={() => toast.dismiss(NUDGE_TOAST_ID)}
      />
    ), { id: NUDGE_TOAST_ID, duration: 8000, position: "bottom-right" })
  }

  return (
    <div className="flex h-screen flex-col bg-background">
      <PageHeader
        title="Nudge & Care Demo"
        description="预览所有关怀、提醒与互动弹窗文案"
        breadcrumbItems={[
          { label: "Tools", href: "/tools" },
          { label: "Nudges", href: "/tools/nudges" },
        ]}
      />

      <div className="flex-1 overflow-auto p-6 md:p-8">
        <div className="mx-auto max-w-5xl space-y-8">
          <Tabs defaultValue="local">
            <TabsList className="grid w-full grid-cols-7 lg:w-[700px]">
              <TabsTrigger value="local">🦊 本地触发</TabsTrigger>
              <TabsTrigger value="ai">🤖 AI</TabsTrigger>
              <TabsTrigger value="health">健康</TabsTrigger>
              <TabsTrigger value="social">社交/梗</TabsTrigger>
              <TabsTrigger value="holiday">节日</TabsTrigger>
              <TabsTrigger value="memes">黑客文化</TabsTrigger>
              <TabsTrigger value="milestone">里程碑</TabsTrigger>
            </TabsList>

            <TabsContent value="local" className="mt-6 space-y-6">
              <div className="mb-4 rounded-lg border bg-muted/50 p-4 text-sm text-muted-foreground">
                <p>这些是 <code className="bg-muted px-1 rounded">useNudgeGuardian</code> Hook 中定义的本地触发规则。它们会根据时间和使用时长自动弹出，无需后端支持。</p>
              </div>
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {LOCAL_TRIGGER_VARIANTS.map((v) => (
                  <LocalTriggerCard key={v.id} variant={v} onTrigger={() => triggerToast(v)} />
                ))}
              </div>
            </TabsContent>

            <TabsContent value="ai" className="mt-6 space-y-6">
              <AiNudgeTestPanel />
            </TabsContent>

            <TabsContent value="health" className="mt-6 space-y-6">
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {HEALTH_VARIANTS.map((v) => (
                  <DemoCard key={v.id} variant={v} onTrigger={() => triggerToast(v)} />
                ))}
              </div>
            </TabsContent>

            <TabsContent value="social" className="mt-6 space-y-6">
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {SOCIAL_VARIANTS.map((v) => (
                  <DemoCard key={v.id} variant={v} onTrigger={() => triggerToast(v)} />
                ))}
              </div>
            </TabsContent>

            <TabsContent value="holiday" className="mt-6 space-y-6">
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {HOLIDAY_VARIANTS.map((v) => (
                  <DemoCard key={v.id} variant={v} onTrigger={() => triggerToast(v)} />
                ))}
              </div>
            </TabsContent>

            <TabsContent value="memes" className="mt-6 space-y-6">
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {MEME_VARIANTS.map((v) => (
                  <DemoCard key={v.id} variant={v} onTrigger={() => triggerToast(v)} />
                ))}
              </div>
            </TabsContent>

            <TabsContent value="milestone" className="mt-6 space-y-6">
              <div className="mb-4 rounded-lg border bg-muted/50 p-4 text-sm text-muted-foreground">
                <p>这里展示的是基于纯前端 localStorage 记录的“首次访问时间”计算出的里程碑。实际应用中会自动判断天数触发。</p>
              </div>
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {MILESTONE_VARIANTS.map((v) => (
                  <DemoCard key={v.id} variant={v} onTrigger={() => triggerToast(v)} />
                ))}
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </div>
  )
}

const AI_API_URL = "https://lunafox-ai-proxy-fzqn2tz4eb4f.yyhunisec.deno.net/"

const AI_CONTEXTS = [
  { label: "🌙 深夜 (23:00)", context: { hour: 23, day: 3, event: "daily_care" } },
  { label: "🌅 早晨 (07:00)", context: { hour: 7, day: 1, event: "daily_care" } },
  { label: "☕ 下午茶 (15:00)", context: { hour: 15, day: 2, event: "daily_care" } },
  { label: "🍜 饭点 (12:00)", context: { hour: 12, day: 4, event: "daily_care" } },
  { label: "🎉 周五 (18:00)", context: { hour: 18, day: 5, event: "daily_care" } },
  { label: "🌟 周末 (14:00)", context: { hour: 14, day: 6, event: "daily_care" } },
  { label: "💻 第一次扫描完成", context: { hour: 10, day: 1, event: "first_scan_complete" } },
  { label: "🎖️ 连续7天登录", context: { hour: 9, day: 3, event: "streak_7_days" } },
  { label: "🎯 扫描100个目标", context: { hour: 16, day: 2, event: "scanned_100_targets" } },
]

interface AiContext {
  hour: number
  day: number
  event: string
}

function AiNudgeTestPanel() {
  const [loading, setLoading] = React.useState<string | null>(null)
  const [results, setResults] = React.useState<Record<string, AiNudgeResult>>({})
  const [error, setError] = React.useState<string | null>(null)

  const testAi = async (label: string, context: AiContext) => {
    setLoading(label)
    setError(null)
    try {
      const res = await fetch(AI_API_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ context }),
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setResults((prev) => ({ ...prev, [label]: data }))

      // Pop up Toast at the same time
      toast.custom(() => (
        <NudgeToastCard
          title={data.title}
          description={data.description}
          icon={<span className="text-2xl">{data.icon || "🤖"}</span>}
          primaryAction={{
            label: data.primaryAction?.label || "OK",
            onClick: () => toast.dismiss(NUDGE_TOAST_ID),
          }}
          secondaryAction={
            data.secondaryAction
              ? { label: data.secondaryAction.label, onClick: () => toast.dismiss(NUDGE_TOAST_ID), buttonVariant: "outline" }
              : undefined
          }
          onDismiss={() => toast.dismiss(NUDGE_TOAST_ID)}
        />
      ), { id: NUDGE_TOAST_ID, duration: 8000, position: "bottom-right" })
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="rounded-lg border bg-muted/50 p-4 text-sm text-muted-foreground">
        <p>测试 AI Proxy API 生成的 Nudge 文案。点击按钮会调用 AI 生成并弹出 Toast。</p>
        <p className="mt-1 font-mono text-xs">{AI_API_URL}</p>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive bg-destructive/10 p-4 text-sm text-destructive">
          错误: {error}
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {AI_CONTEXTS.map(({ label, context }) => (
          <Card key={label} className="flex flex-col">
            <CardHeader className="pb-2">
              <CardTitle className="text-base">{label}</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col flex-1 gap-3">
              <pre className="rounded bg-muted p-2 text-xs overflow-auto flex-1">
                {JSON.stringify(context, null, 2)}
              </pre>
              <Button
                onClick={() => testAi(label, context)}
                disabled={loading === label}
                className="w-full"
              >
                {loading === label ? "生成中…" : "🤖 调用 AI"}
              </Button>
              {results[label] && (
                <div className="rounded border bg-background p-3 text-sm space-y-1">
                  <p><span className="text-muted-foreground">Icon:</span> {results[label].icon}</p>
                  <p><span className="text-muted-foreground">Title:</span> {results[label].title}</p>
                  <p className="text-muted-foreground text-xs leading-relaxed">{results[label].description}</p>
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}

function DemoCard({ variant, onTrigger }: { variant: DemoVariant; onTrigger: () => void }) {
  return (
    <Card className="flex flex-col h-full">
      <CardHeader className="flex flex-row items-center gap-4 pb-2">
        <div className="flex size-10 items-center justify-center rounded-lg border bg-background shadow-sm">
          {variant.icon}
        </div>
        <div>
          <CardTitle className="text-base">{variant.title}</CardTitle>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col flex-1 gap-4">
        <p className="text-sm text-muted-foreground leading-relaxed flex-1">
          {variant.desc}
        </p>
        <Button onClick={onTrigger} variant="secondary" className="w-full">
          Preview Toast
        </Button>
      </CardContent>
    </Card>
  )
}

interface LocalTriggerVariant extends DemoVariant {
  trigger: string
}

function LocalTriggerCard({ variant, onTrigger }: { variant: LocalTriggerVariant; onTrigger: () => void }) {
  return (
    <Card className="flex flex-col h-full">
      <CardHeader className="flex flex-row items-center gap-4 pb-2">
        <div className="flex size-10 items-center justify-center rounded-lg border bg-background shadow-sm">
          {variant.icon}
        </div>
        <div>
          <CardTitle className="text-base">{variant.title}</CardTitle>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col flex-1 gap-4">
        <p className="text-sm text-muted-foreground leading-relaxed flex-1">
          {variant.desc}
        </p>
        <div className="rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground">
          触发条件: {variant.trigger}
        </div>
        <Button onClick={onTrigger} variant="secondary" className="w-full">
          Preview Toast
        </Button>
      </CardContent>
    </Card>
  )
}
