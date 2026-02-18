import { redirect } from "next/navigation"

type Props = {
  params: Promise<{ locale: string; id: string }>
}

/**
 * Target detail default page
 * Automatically redirects to overview page
 */
export default async function TargetDetailPage({
  params,
}: Props) {
  const { locale, id } = await params
  redirect(`/${locale}/target/${id}/overview/`)
}
