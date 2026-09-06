import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { resolveRequestLocale } from "@/i18n/locale"

export async function generateMetadata(): Promise<Metadata> {
  const locale = await resolveRequestLocale()
  const t = await getTranslations({ locale, namespace: "auth" })

  return {
    title: t("pageTitle"),
    description: t("pageDescription"),
  }
}

/**
 * Login page layout
 * Does not include sidebar and header
 */
export default function LoginLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <>
      {children}
    </>
  )
}
