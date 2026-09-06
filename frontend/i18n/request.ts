import { getRequestConfig } from "next-intl/server"

import { loadLocaleMessages, resolveRequestLocale } from "@/i18n/locale"

export default getRequestConfig(async () => {
  const locale = await resolveRequestLocale()

  return {
    locale,
    messages: await loadLocaleMessages(locale),
  }
})
