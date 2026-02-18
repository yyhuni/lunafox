import createMiddleware from 'next-intl/middleware';
import { locales, defaultLocale } from '@/i18n/config';

export default createMiddleware({
  // Supported language list
  locales,
  // Default language
  defaultLocale,
  // Always show language prefix in URL
  localePrefix: 'always',
  // Auto-detect browser language preference
  localeDetection: true,
});

export const config = {
  // Match all paths, excluding API, static files, Next.js internal paths, etc.
  matcher: [
    // Match all paths
    '/((?!api|_next|_vercel|.*\\..*).*)',
  ],
};
