import { redirect } from "next/navigation"

type Props = {
  params: Promise<{ locale: string; id: string }>
}

/**
 * Target detail page (compatible with old routes)
 * Automatically redirects to overview page
 */
export default async function TargetDetailsPage({
  params,
}: Props) {
  const { locale, id } = await params
  redirect(`/${locale}/target/${id}/overview/`)
}
