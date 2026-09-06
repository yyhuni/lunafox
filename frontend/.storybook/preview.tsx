import * as React from "react"
import type { Preview } from "@storybook/nextjs-vite"
import { NextIntlClientProvider } from "next-intl"
import { initialize, mswLoader } from "msw-storybook-addon"

import { QueryProvider } from "@/components/providers/query-provider"
import { ThemeProvider } from "@/components/providers/theme-provider"
import { UiI18nProvider } from "@/components/providers/ui-i18n-provider"
import { setMockScenario } from "@/mock"

import enMessages from "../messages/en.json"
import zhMessages from "../messages/zh.json"
import "../app/globals.css"

initialize({
  onUnhandledRequest: "bypass",
})

const messageMap = {
  en: enMessages,
  zh: zhMessages,
} as const

function StorybookProviders({
  children,
  locale,
}: {
  children: React.ReactNode
  locale: "en" | "zh"
}) {
  return (
    <ThemeProvider>
      <NextIntlClientProvider locale={locale} messages={messageMap[locale]}>
        <QueryProvider>
          <UiI18nProvider>{children}</UiI18nProvider>
        </QueryProvider>
      </NextIntlClientProvider>
    </ThemeProvider>
  )
}

const preview: Preview = {
  loaders: [mswLoader],
  globalTypes: {
    locale: {
      name: "Locale",
      toolbar: {
        icon: "globe",
        items: [
          { value: "zh", title: "中文" },
          { value: "en", title: "English" },
        ],
      },
    },
    mockScenario: {
      name: "Mock Scenario",
      toolbar: {
        icon: "database",
        items: [
          { value: "happy", title: "Happy" },
          { value: "empty", title: "Empty" },
          { value: "stress", title: "Stress" },
          { value: "edge", title: "Edge" },
          { value: "error", title: "Error" },
        ],
      },
    },
  },
  initialGlobals: {
    locale: "zh",
    mockScenario: "happy",
  },
  parameters: {
    layout: "fullscreen",
    nextjs: {
      appDirectory: true,
    },
  },
  decorators: [
    (Story, context) => {
      const locale = context.globals.locale === "en" ? "en" : "zh"
      const scenario = String(context.parameters.mockScenario ?? context.globals.mockScenario ?? "happy")

      setMockScenario(scenario)

      return (
        <StorybookProviders key={`${locale}-${scenario}`} locale={locale}>
          <Story />
        </StorybookProviders>
      )
    },
  ],
}

export default preview
