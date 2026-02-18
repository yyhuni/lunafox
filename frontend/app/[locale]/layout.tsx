import type React from "react"
import type { Metadata } from "next"
import { NextIntlClientProvider } from 'next-intl'
import { getMessages, setRequestLocale, getTranslations } from 'next-intl/server'
import { notFound } from 'next/navigation'
import { locales, localeHtmlLang, type Locale } from '@/i18n/config'
import { DEFAULT_COLOR_THEME_ID } from "@/lib/color-themes"
import localFont from 'next/font/local'

// Import global style files
import "../globals.css"
// Import color themes
import "@/styles/themes/bauhaus.css"
import { QueryProvider } from "@/components/providers/query-provider"
import { ThemeProvider } from "@/components/providers/theme-provider"
import { UiI18nProvider } from "@/components/providers/ui-i18n-provider"
import { ColorThemeInit } from "@/components/color-theme-init"
import { ErrorBoundary } from "@/components/error-boundary"
import { LayoutClientEnhancements } from "@/components/layout-client-enhancements"

// Import common layout components
import { AuthLayout } from "@/components/auth/auth-layout"

// Dynamically generate metadata
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: 'metadata' })
  
  return {
    title: t('title'),
    description: t('description'),
    keywords: t('keywords').split(',').map(k => k.trim()),
    generator: "LunaFox ASM Platform",
    authors: [{ name: "yyhuni" }],
    icons: {
      icon: [
        { url: "/images/icon-64.png", sizes: "64x64", type: "image/png" },
        { url: "/images/icon-256.png", sizes: "256x256", type: "image/png" },
      ],
      apple: [{ url: "/images/icon-256.png", sizes: "256x256", type: "image/png" }],
    },
    openGraph: {
      title: t('ogTitle'),
      description: t('ogDescription'),
      type: "website",
      locale: locale === 'zh' ? 'zh_CN' : 'en_US',
    },
    robots: {
      index: true,
      follow: true,
    },
  }
}

// Optimize font loading with next/font/local
const harmonyOS = localFont({
  src: [
    {
      path: '../../public/fonts/harmonyos-sans/woff2/HarmonyOS_Sans_SC_Regular.woff2',
      weight: '400',
      style: 'normal',
    },
    {
      path: '../../public/fonts/harmonyos-sans/woff2/HarmonyOS_Sans_SC_Medium.woff2',
      weight: '500',
      style: 'normal',
    },
    {
      path: '../../public/fonts/harmonyos-sans/woff2/HarmonyOS_Sans_SC_Bold.woff2',
      weight: '700',
      style: 'normal',
    },
  ],
  variable: '--font-harmony',
  display: 'swap',
  preload: true,
  fallback: ['system-ui', '-apple-system', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'sans-serif'],
})

interface Props {
  children: React.ReactNode
  params: Promise<{ locale: string }>
}

/**
 * Language layout component
 * Wraps all pages, provides internationalization context
 */
export default async function LocaleLayout({
  children,
  params,
}: Props) {
  const { locale } = await params

  // Validate locale validity
  if (!locales.includes(locale as Locale)) {
    notFound()
  }

  // Enable static rendering
  setRequestLocale(locale)

  // Load translation messages
  const messages = await getMessages()
  const commonUiT = await getTranslations({ locale, namespace: 'common.ui' })

  const themeId = DEFAULT_COLOR_THEME_ID

  return (
    <html
      lang={localeHtmlLang[locale as Locale]}
      data-theme={themeId}
      suppressHydrationWarning
    >
      <body className={harmonyOS.className}>
        <a
          href="#main-content"
          className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-background focus:px-3 focus:py-2 focus:text-sm focus:font-medium focus:text-foreground focus:shadow-md"
        >
          {commonUiT('skipToMainContent')}
        </a>
        <ColorThemeInit />
        {/* ThemeProvider provides theme switching functionality */}
        <ThemeProvider>
          {/* NextIntlClientProvider provides internationalization context */}
          <NextIntlClientProvider messages={messages}>
            <LayoutClientEnhancements />
            {/* QueryProvider provides React Query functionality */}
            <QueryProvider>
              {/* ErrorBoundary catches component errors and prevents app crashes */}
              <ErrorBoundary>
                {/* UiI18nProvider provides UI component translations */}
                <UiI18nProvider>
                  {/* AuthLayout handles authentication and sidebar display */}
                  <AuthLayout>
                    {children}
                  </AuthLayout>
                </UiI18nProvider>
              </ErrorBoundary>
            </QueryProvider>
          </NextIntlClientProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
