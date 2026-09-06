"use client"

import React from "react"
import { useLocale, useTranslations } from "next-intl"
import { useRouter } from "next/navigation"
import { LOCALE_COOKIE_NAME, locales, localeNames, type Locale } from "@/i18n/config"
import { useUpdateNotificationLocale } from "@/hooks/use-notification-settings"
import {
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from "@/components/ui/dropdown-menu"
import { IconLanguage } from "@/components/icons"
import { HeaderIconActionMenu } from "@/components/shared/dropdown-menu-owners"

/**
 * Language switcher component
 * Displays current language, click to switch to other supported languages
 */
export function LanguageSwitcher() {
  const locale = useLocale() as Locale
  const router = useRouter()
  const tCommon = useTranslations("common")
  const [isPending, startTransition] = React.useTransition()
  const updateNotificationLocale = useUpdateNotificationLocale()

  const handleLocaleChange = async (newLocale: Locale) => {
    if (newLocale === locale || isPending || updateNotificationLocale.isPending) return

    // Persist the worker's locale before changing the page cookie so a failed
    // request cannot leave the visible page and future notifications split.
    try {
      await updateNotificationLocale.mutateAsync(newLocale)
    } catch {
      return
    }
    document.cookie = `${LOCALE_COOKIE_NAME}=${newLocale}; Path=/; Max-Age=31536000; SameSite=Lax`
    startTransition(() => {
      router.refresh()
    })
  }

  return (
    <HeaderIconActionMenu
      ariaLabel={tCommon("language.switchLanguage")}
      icon={<IconLanguage className="h-4 w-4" />}
    >
      <DropdownMenuRadioGroup value={locale}>
        {locales.map((l) => (
          <DropdownMenuRadioItem
            key={l}
            value={l}
            onClick={() => handleLocaleChange(l)}
            disabled={isPending || updateNotificationLocale.isPending}
          >
            <span>{localeNames[l]}</span>
          </DropdownMenuRadioItem>
        ))}
      </DropdownMenuRadioGroup>
    </HeaderIconActionMenu>
  )
}
