import { redirect } from "next/navigation"

type Props = {
  params: Promise<{ locale: string; id: string }>
}

export default async function ScanHistoryDetailPage({
  params,
}: Props) {
  const { locale, id } = await params
  redirect(`/${locale}/scan/history/${id}/overview/`)
}
