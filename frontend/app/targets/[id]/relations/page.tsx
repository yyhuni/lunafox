import { redirect } from "next/navigation"

export default async function TargetRelationsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  redirect(`/targets/${id}/websites/`)
}
