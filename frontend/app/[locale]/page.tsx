import { redirect } from 'next/navigation';
import { defaultLocale } from '@/i18n/config';

export default function Home() {
  // Redirect directly to dashboard page (with language prefix)
  redirect(`/${defaultLocale}/dashboard/`);
}
