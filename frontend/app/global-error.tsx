"use client";
import * as React from "react";
import Link from "next/link";
import { semanticIcons } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
const GlobalErrorIcon = semanticIcons.status.error;
const GLOBAL_ERROR_COPY = {
    en: {
        title: "LunaFox could not recover from this error",
        description: "A root-level failure interrupted the application. You can retry once or return to the app entry.",
        retry: "Try again",
        action: "Return to app",
    },
    zh: {
        title: "LunaFox 当前无法从这个错误中恢复",
        description: "应用根层出现异常。你可以重试一次，或返回应用入口。",
        retry: "重试",
        action: "返回应用",
    },
} as const;
export default function GlobalError({ error, reset, }: {
    error: Error & {
        digest?: string;
    };
    reset: () => void;
}) {
    const [isZh, setIsZh] = React.useState(false);
    React.useEffect(() => {
        console.error(error);
        if (typeof document !== "undefined" && document.documentElement.lang.startsWith("zh")) {
            setIsZh(true);
            return;
        }
        if (typeof navigator !== "undefined" && navigator.language.toLowerCase().startsWith("zh")) {
            setIsZh(true);
        }
    }, [error]);
    const copy = isZh ? GLOBAL_ERROR_COPY.zh : GLOBAL_ERROR_COPY.en;
    return (<html lang={isZh ? "zh-CN" : "en"}>
      <body>
        <main className="flex min-h-screen items-center justify-center px-6 py-16">
          <div className="mx-auto flex w-full max-w-2xl flex-col items-center text-center">
            <div className="mb-4 rounded-full bg-destructive/10 p-3">
              <GlobalErrorIcon className="h-10 w-10 text-destructive"/>
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
        </main>
      </body>
    </html>);
}
