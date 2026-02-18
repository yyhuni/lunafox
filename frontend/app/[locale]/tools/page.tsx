"use client"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/common/page-header"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { PackageOpen, Settings, ArrowRight } from "@/components/icons"
import Link from "next/link"
import { useTranslations } from "next-intl"

/**
 * Tools overview page
 * Displays entry points for open source tools and custom tools
 */
export default function ToolsPage() {
  const t = useTranslations("pages.tools")

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
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="TLS-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Statistics cards */}
      <div className="px-4 lg:px-6">
        <div className="grid gap-4 md:grid-cols-2">
          {modules.map((module) => (
            <Card key={module.title} className="relative hover:shadow-lg transition-shadow">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <module.icon className="h-5 w-5" />
                    <CardTitle className="text-lg">{module.title}</CardTitle>
                  </div>
                  {module.status === "coming-soon" && (
                    <span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-1 rounded-full">
                      {t("comingSoon")}
                    </span>
                  )}
                </div>
                <CardDescription>{module.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {/* Statistics information */}
                  <div className="flex items-center gap-6 text-sm">
                    <div>
                      <span className="text-muted-foreground">{t("stats.total")}</span>
                      <span className="font-semibold ml-1">{module.stats.total}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">{t("stats.active")}</span>
                      <span className="font-semibold ml-1 text-green-600">{module.stats.active}</span>
                    </div>
                  </div>

                  {/* Action buttons */}
                  {module.status === "available" ? (
                    <Link href={module.href}>
                      <Button className="w-full">
                        {t("enterManagement")}
                        <ArrowRight className="h-4 w-4" />
                      </Button>
                    </Link>
                  ) : (
                    <Button disabled className="w-full">
                      {t("comingSoon")}
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      {/* Quick actions */}
      <div className="px-4 lg:px-6">
        <Card>
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
