// Internationalization navigation helper functions
import { createNavigation } from 'next-intl/navigation';
import { locales, defaultLocale } from './config';

// Create internationalization navigation tools
export const { Link, redirect, usePathname, useRouter, getPathname } = createNavigation({
  locales,
  defaultLocale,
});
