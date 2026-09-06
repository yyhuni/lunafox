"use client";
import Link from "next/link";
import { useLocale } from "next-intl";
import { semanticIcons } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
const NotFoundIcon = semanticIcons.status.warning;
const NOT_FOUND_COPY = {
    en: {
        title: "Page not found",
        description: "The address is invalid or this route no longer exists in LunaFox.",
        action: "Go to overview",
    },
    zh: {
        title: "页面不存在",
        description: "当前地址无效，或该路由已不再存在于 LunaFox 中。",
        action: "返回概览",
    },
} as const;
export default function NotFound() {
    const locale = useLocale();
    const copyLocale = locale === "zh" ? "zh" : "en";
    const notFoundCopy = NOT_FOUND_COPY[copyLocale];
    return (<main className="flex min-h-screen items-center justify-center px-6 py-16">
      <div className="mx-auto flex w-full max-w-2xl flex-col items-center text-center">
        <div className="mb-4 rounded-full bg-destructive/10 p-3">
          <NotFoundIcon className="h-10 w-10 text-destructive"/>
        </div>
        <p className={cn("mb-2 text-sm uppercase text-muted-foreground", textRole.metadataLabel)}>404</p>
        <h1 className={cn("mb-2", textRole.pageTitle)}>{notFoundCopy.title}</h1>
        <p className={cn("max-w-xl", textRole.bodySubtle)}>{notFoundCopy.description}</p>
        <div className="mt-5">
          <Button render={<Link href="/overview/"/>}>{notFoundCopy.action}</Button>
        </div>
      </div>
    </main>);
}
