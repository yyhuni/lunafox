import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/common/page-header"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ArrowRight, PackageOpen, Settings } from "@/components/icons"
import { getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import Link from "next/link"
import { getTranslations } from "next-intl/server"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"

/**
 * Tools overview page
 * Displays entry points for open source tools and custom tools
 */
export default async function ToolsPage() {
  const t = await getTranslations("pages.tools")

  // Feature modules
  const modules = [
    {
      title: t("wordlists.title"),
      description: t("wordlists.description"),
      href: "/tools/wordlists/",
      icon: PackageOpen,
      status: "available",
      stats: {
        total: "-",
        active: "-",
      },
    },
    {
      title: t("nuclei.title"),
      description: t("nuclei.description"),
      href: "/tools/nuclei/",
      icon: Settings,
      status: "available",
      stats: {
        total: "-",
        active: "-",
      },
    },
  ]

  return (
    <div className={COMPACT_PAGE_SHELL_CLASS}>
      <PageHeader
        code="TLS-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Statistics cards */}
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <div className="grid gap-3 md:grid-cols-2">
          {modules.map((module) => (
            <Card
              key={module.title}
              variant="compact"
              className="relative hover:shadow-lg transition-shadow"
            >
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <module.icon className="h-5 w-5" />
                    <CardTitle className={textRole.panelTitle}>{module.title}</CardTitle>
                  </div>
                  {module.status === "coming-soon" && (
                    <span className={cn("rounded-full bg-warning/10 px-2 py-1", textRole.badge, getStatusToneTextClass("warning"))}>
                      {t("comingSoon")}
                    </span>
                  )}
                </div>
                <CardDescription>{module.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {/* Statistics information */}
                  <div className="flex items-center gap-3">
                    <div className="flex items-baseline gap-1">
                      <span className={textRole.metadataLabel}>{t("stats.total")}</span>
                      <span className={textRole.metadataValueStrong}>{module.stats.total}</span>
                    </div>
                    <div className="flex items-baseline gap-1">
                      <span className={textRole.metadataLabel}>{t("stats.active")}</span>
                      <span className={cn(textRole.metadataValueStrong, getStatusToneTextClass("success"))}>{module.stats.active}</span>
                    </div>
                  </div>

                  {/* Action buttons */}
                  {module.status === "available" ? (
                    <Link href={module.href} className="block">
                      <Button>
                        {t("enterManagement")}
                        <ArrowRight className="h-4 w-4" />
                      </Button>
                    </Link>
                  ) : (
                    <div className="block">
                      <Button disabled>
                      {t("comingSoon")}
                      </Button>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      {/* Quick actions */}
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <Card variant="compact">
          <CardHeader>
            <CardTitle>{t("quickActions.title")}</CardTitle>
            <CardDescription>
              {t("quickActions.description")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              <Link href="/tools/wordlists/">
                <Button variant="outline" size="sm">
                  <PackageOpen className="h-4 w-4" />
                  {t("wordlists.title")}
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
