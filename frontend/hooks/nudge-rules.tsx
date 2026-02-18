"use client"

import * as React from "react"
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
  IconBug,
} from "@/components/icons"
import type { NudgeToastVariant } from "@/hooks/use-nudge-toast"

// Care nudge variants
const LATE_NIGHT_VARIANTS: NudgeToastVariant[] = [
  {
    icon: <IconMoon className="size-6 text-indigo-500" />,
    title: "凌晨修仙警告",
    description: "指挥官，现在的流量最干净，但你的发际线正在报警。守护 Root 权限的同时，别忘了守护发量。",
    primaryAction: { label: "睡了睡了" },
    secondaryAction: { label: "再挖一个洞" },
  },
  {
    icon: <IconTerminal className="size-6 text-indigo-400" />,
    title: "It's Late...",
    description: "午夜钟声敲响，灰姑娘丢了水晶鞋，而你刚刚打开了 Burp Suite。",
    primaryAction: { label: "继续抓包" },
    secondaryAction: { label: "休息一下" },
  },
]

const MORNING_VARIANTS: NudgeToastVariant[] = [
  {
    icon: <IconSun className="size-6 text-orange-500" />,
    title: "早起的黑客有洞挖",
    description: "当别人还在睡梦中，你已经完成了第一波资产测绘。今天的目标：拿下一个 Shell。",
    primaryAction: { label: "Get Shell!" },
  },
]

const COFFEE_VARIANTS: NudgeToastVariant[] = [
  {
    icon: <IconCoffee className="size-6 text-amber-700" />,
    title: "咖啡因浓度过低",
    description: "SQL 注入失败？也许不是 Payload 的问题，是你血液里的咖啡因不足了。来杯 Java 提提神。",
    primaryAction: { label: "去倒咖啡" },
    secondaryAction: { label: "红牛万岁" },
  },
]

const FOOD_VARIANTS: NudgeToastVariant[] = [
  {
    icon: <IconCpu className="size-6 text-rose-500" />,
    title: "缓冲区下溢警告",
    description: "检测到你的胃部缓冲区即将 Underflow。请立即执行 eat() 函数，防止身体 Crash。",
    primaryAction: { label: "去干饭" },
    secondaryAction: { label: "还能再扛" },
  },
]

const FRIDAY_VARIANTS: NudgeToastVariant[] = [
  {
    icon: <IconCalendar className="size-6 text-purple-500" />,
    title: "Read-Only Friday",
    description: "周五不改生产库，这是江湖规矩。除非你想在这个周末收到报警短信，否则请放下手中的 DROP TABLE。",
    primaryAction: { label: "遵命" },
    secondaryAction: { label: "头铁" },
  },
  {
    icon: <IconHeart className="size-6 text-pink-500" />,
    title: "节假日加班警告",
    description: "大过节的还在挖洞？你的敬业程度让我 CPU 温度都升高了。记得给自己发三倍工资（或者三倍快乐水）。",
    primaryAction: { label: "搞完就撤" },
    secondaryAction: { label: "我爱加班" },
  },
]

const OVERTIME_VARIANTS: NudgeToastVariant[] = [
  {
    icon: <IconMoon className="size-6 text-indigo-500" />,
    title: "还在肝？",
    description: "听说深夜的代码 Bug 更少？也许吧，但发际线后移的速度肯定更快。保重啊，指挥官。",
    primaryAction: { label: "最后一行" },
    secondaryAction: { label: "再战三百回" },
  },
  {
    icon: <IconCoffee className="size-6 text-amber-700" />,
    title: "深夜食堂",
    description: "这个点还在屏幕前，是不是饿了？泡面虽好，可不要贪杯。或者去阳台看看星星（如果有的话）。",
    primaryAction: { label: "去觅食" },
    secondaryAction: { label: "我不饿" },
  },
]

const MEME_VARIANTS: NudgeToastVariant[] = [
  {
    icon: <IconAlertTriangle className="size-6 text-red-600" />,
    title: "rm -rf /*",
    description: "这是一个危险的命令... 哪怕你是在虚拟机里。时刻保持敬畏，记得备份，记得看清你在哪个窗口。",
    primaryAction: { label: "已经在跑路了" },
  },
  {
    icon: <IconLock className="size-6 text-blue-500" />,
    title: "密码是 123456？",
    description: "希望不是。否则我的字典生成器第一秒就能跑出来。去改个强密码吧，比如 P@$$w0rd (开玩笑的)。",
    primaryAction: { label: "我很强" },
    secondaryAction: { label: "这就改" },
  },
  {
    icon: <IconUser className="size-6 text-pink-500" />,
    title: "面向对象编程",
    description: "找不到对象？没关系，new 一个就行了。但在现实里，可能需要你合上电脑出去走走。",
    primaryAction: { label: "我去 new 一个" },
    secondaryAction: { label: "代码就是恋人" },
  },
  {
    icon: <IconActivity className="size-6 text-emerald-500" />,
    title: "姿势不对，Shell 不会",
    description: "长期低头会压迫颈椎，导致 Payload 构造思路受阻。建议立即执行：stand_up(); stretch();",
    primaryAction: { label: "活动一下" },
    secondaryAction: { label: "再坐五百年" },
  },
  {
    icon: <IconHeart className="size-6 text-red-500" />,
    title: "别扫了，扫我吧",
    description: "我是你的小狐狸助手，一直在默默守护你的控制台。要不要给我点个 Star？",
    primaryAction: {
      label: "去点 Star",
      onClick: () => window.open("https://github.com/yyhuni/xingrin", "_blank"),
    },
    secondaryAction: { label: "下次一定" },
  },
]

export function getCareNudgeVariants(now: Date = new Date()): NudgeToastVariant[] {
  const hour = now.getHours()
  const day = now.getDay() // 0 = Sunday, 5 = Friday

  let pool: NudgeToastVariant[] = [...MEME_VARIANTS]

  if (hour >= 23 || hour <= 4) {
    return [...LATE_NIGHT_VARIANTS, ...OVERTIME_VARIANTS]
  }

  if (hour >= 5 && hour <= 9) {
    pool = [...pool, ...MORNING_VARIANTS]
  } else if ((hour >= 11 && hour <= 13) || (hour >= 17 && hour <= 19)) {
    pool = [...pool, ...FOOD_VARIANTS]
  } else if (hour >= 14 && hour <= 16) {
    pool = [...pool, ...COFFEE_VARIANTS]
  }

  if (day === 5 || day === 0 || day === 6) {
    pool = [...pool, ...FRIDAY_VARIANTS]
  }

  return pool
}

// Guardian rules
export type GuardianRuleId = "late_night" | "long_session" | "cyber_fox"

export interface GuardianVariantTemplate {
  title: string
  desc: string
  icon: React.ComponentType<{ className?: string }>
  color: string
  primary?: string
  secondary?: string
}

export interface GuardianRuleContext {
  hour: number
  sessionDurationMinutes: number
}

export interface GuardianRule {
  id: GuardianRuleId
  cooldownHours: number
  shouldTrigger: (context: GuardianRuleContext) => boolean
  variants: GuardianVariantTemplate[]
}

const GUARDIAN_VARIANTS: Record<GuardianRuleId, GuardianVariantTemplate[]> = {
  cyber_fox: [
    {
      title: "系统监测到毛茸茸生物",
      desc: "🦊 嘤？LunaFox 的吉祥物跑出来了！它似乎对你的鼠标指针很好奇。工作累了就陪它玩会儿吧。",
      icon: IconBug,
      color: "text-orange-500",
      primary: "陪它玩",
      secondary: "摸摸头",
    },
  ],
  late_night: [
    {
      title: "深夜修仙警告",
      desc: "指挥官，流量最干净的时候确实适合挖洞，但发际线也在报警。守护 Root 权限的同时，别忘了守护发量。",
      icon: IconMoon,
      color: "text-indigo-500",
      primary: "再熬半小时",
      secondary: "这就睡",
    },
    {
      title: "夜深了",
      desc: "这个点还在屏幕前，是不是饿了？泡面虽好，可不要贪杯。或者去阳台看看星星（如果有的话），然后关机。",
      icon: IconCoffee,
      color: "text-amber-600",
      primary: "去看星星",
      secondary: "关机",
    },
    {
      title: "凌晨 1 点的月狐",
      desc: "你和服务器是现在唯一醒着的伙伴。别太累了，服务器有 UPS 撑着，你可没有。",
      icon: IconMoon,
      color: "text-purple-500",
      primary: "陪它聊会儿",
      secondary: "去休息",
    },
  ],
  long_session: [
    {
      title: "姿势不对，Shell 不会",
      desc: "长期低头会压迫颈椎，导致 Payload 构造思路受阻。建议立即执行：stand_up(); stretch();",
      icon: IconActivity,
      color: "text-emerald-500",
      primary: "起来活动",
      secondary: "再坐会儿",
    },
    {
      title: "视觉模块过热",
      desc: "你盯着屏幕太久了，视网膜可能出现了残影。去窗边看看有没有真的狐狸，或者哪怕只是看看那棵树。",
      icon: IconSun,
      color: "text-amber-500",
      primary: "去看狐狸",
      secondary: "看看树",
    },
    {
      title: "久坐提醒",
      desc: "已经连续战斗 2 小时了。起来接杯水，活动一下腰背，为了更长远的黑客生涯。",
      icon: IconCpu,
      color: "text-indigo-500",
      primary: "接水去",
      secondary: "活动腰背",
    },
    {
      title: "Drink Water",
      desc: "多喝水，多排毒。身体是革命的本钱，也是挖洞的本钱。",
      icon: IconCoffee,
      color: "text-cyan-500",
      primary: "去接水",
      secondary: "一会儿喝",
    },
  ],
}

export const GUARDIAN_RULES: GuardianRule[] = [
  {
    id: "late_night",
    cooldownHours: 16,
    shouldTrigger: ({ hour }) => hour >= 23 || hour < 4,
    variants: GUARDIAN_VARIANTS.late_night,
  },
  {
    id: "long_session",
    cooldownHours: 3,
    shouldTrigger: ({ sessionDurationMinutes }) => sessionDurationMinutes > 120,
    variants: GUARDIAN_VARIANTS.long_session,
  },
  {
    id: "cyber_fox",
    cooldownHours: 20,
    shouldTrigger: ({ hour }) => hour === 15 && Math.random() < 0.01,
    variants: GUARDIAN_VARIANTS.cyber_fox,
  },
]
