import { redirect } from "next/navigation"

export default async function WebsiteRelationDetailPage({
  params,
}: {
  params: Promise<{ id: string; websiteId: string; section?: string[] }>
}) {
  const { id, websiteId, section } = await params
  const sectionPath = section?.length ? `${section.join("/")}/` : ""
  redirect(`/targets/${id}/websites/${websiteId}/${sectionPath}`)
}
