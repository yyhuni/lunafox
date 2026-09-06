// Internationalization configuration file

export const locales = ['zh', 'en'] as const;
export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = 'en';
export const LOCALE_COOKIE_NAME = "NEXT_LOCALE";

export const localeNames: Record<Locale, string> = {
  zh: '中文',
  en: 'English',
};

// HTML lang attribute corresponding to languages
export const localeHtmlLang: Record<Locale, string> = {
  zh: 'zh-CN',
  en: 'en',
};

export function isLocale(value: string): value is Locale {
  return (locales as readonly string[]).includes(value)
}
