"use client";
import * as React from "react";
import Link from "next/link";
import { useLocale } from "next-intl";
import { semanticIcons } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
const LocaleErrorIcon = semanticIcons.status.error;
const LOCALE_ERROR_COPY = {
    en: {
        title: "This page hit an unexpected problem",
        description: "The route is available, but this part of the page could not finish rendering.",
        retry: "Try again",
        action: "Go to overview",
    },
    zh: {
        title: "当前页面遇到了异常",
        description: "路由仍然可用，但这部分内容暂时无法完成渲染。",
        retry: "重试",
        action: "返回概览",
    },
} as const;
export default function LocaleError({ error, reset, }: {
    error: Error & {
        digest?: string;
    };
    reset: () => void;
}) {
    const locale = useLocale();
    const copy = locale === "zh" ? LOCALE_ERROR_COPY.zh : LOCALE_ERROR_COPY.en;
    React.useEffect(() => {
        console.error(error);
    }, [error]);
    return (<main className="flex min-h-screen items-center justify-center px-6 py-16">
      <div className="mx-auto flex w-full max-w-2xl flex-col items-center text-center">
        <div className="mb-4 rounded-full bg-destructive/10 p-3">
          <LocaleErrorIcon className="h-10 w-10 text-destructive"/>
        </div>
        <h1 className={cn("mb-2", textRole.pageTitle)}>{copy.title}</h1>
        <p className={cn("max-w-xl", textRole.bodySubtle)}>{copy.description}</p>
        <div className="mt-5 flex flex-wrap items-center justify-center gap-3">
          <Button type="button" onClick={() => reset()}>
            {copy.retry}
          </Button>
          <Button variant="outline" render={<Link href="/overview/"/>}>{copy.action}</Button>
        </div>
      </div>
    </main>);
}
