"use client";
import { useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { ExternalLink, Eye, GitBranch, IconBrandGithub, IconInfoCircle, IconStar, } from "@/components/icons";
import { LunaFoxMark } from "@/components/brand/lunafox-mark";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { readSessionValue, removeSessionValue, writeSessionValue } from "@/lib/browser-storage";
import { unlockLoginVisual } from "@/lib/login-visual-unlock";
import { useUnlockLoginVisualDiscoverability } from "@/hooks/use-login-visual";
import { textRole } from "@/lib/typography";
import { shellOverlaySideOffsets } from "@/lib/ui/overlay-styles";
import { cn } from "@/lib/utils";
type GithubRepoSnapshot = {
    name: string;
    fullName: string;
    htmlUrl: string;
    description: string | null;
    stars: number;
    forks: number;
    watchers: number;
    issues: number;
};
const GITHUB_REPO = "yyhuni/xingrin";
const GITHUB_REPO_URL = `https://github.com/${GITHUB_REPO}`;
const GITHUB_REPO_FALLBACK_API = "/api/github/repo";
const GITHUB_REPO_CACHE_KEY = "github-repo-snapshot";
const GITHUB_REPO_CACHE_TIME_KEY = "github-repo-snapshot-time";
const GITHUB_REPO_CACHE_TTL = 5 * 60 * 1000;
const GITHUB_STAR_SPRINT_SIZE = 50;
const appVersion = process.env.NEXT_PUBLIC_IMAGE_TAG || "dev";
export function GithubStarButton() {
    const t = useTranslations("githubStar");
    const [repo, setRepo] = useState<GithubRepoSnapshot | null>(null);
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [unlockDialogOpen, setUnlockDialogOpen] = useState(false);
    const unlockDiscoverability = useUnlockLoginVisualDiscoverability();
    const latestRequestRef = useRef(0);
    useEffect(() => {
        const cached = readSessionValue(GITHUB_REPO_CACHE_KEY);
        const cachedTime = readSessionValue(GITHUB_REPO_CACHE_TIME_KEY);
        const now = Date.now();
        let cachedFallback: GithubRepoSnapshot | null = null;
        if (cached) {
            try {
                cachedFallback = parseGithubRepoSnapshot(JSON.parse(cached));
                setRepo(cachedFallback);
                if (cachedTime && now - parseInt(cachedTime) < GITHUB_REPO_CACHE_TTL) {
                    return;
                }
            }
            catch {
                removeSessionValue(GITHUB_REPO_CACHE_KEY);
                removeSessionValue(GITHUB_REPO_CACHE_TIME_KEY);
            }
        }
        const requestId = ++latestRequestRef.current;
        fetchGithubRepoSnapshot()
            .then((snapshot) => {
            if (requestId !== latestRequestRef.current) {
                return;
            }
            setRepo(snapshot);
            writeGithubRepoCache(snapshot);
        })
            .catch(() => {
            if (requestId !== latestRequestRef.current) {
                return;
            }
            setRepo(cachedFallback);
        });
    }, []);
    const handlePopoverOpenChange = (open: boolean) => {
        if (!open) {
            return;
        }

        setIsRefreshing(true);
        const requestId = ++latestRequestRef.current;
        fetchGithubRepoSnapshot()
            .then((snapshot) => {
            if (requestId !== latestRequestRef.current) {
                return;
            }
            setRepo(snapshot);
            writeGithubRepoCache(snapshot);
        })
            .catch(() => {
            // Keep the cached or previously rendered snapshot when a refresh fails.
        })
            .finally(() => {
            if (requestId === latestRequestRef.current) {
                setIsRefreshing(false);
            }
        });
    };
    const repoName = repo?.name ?? GITHUB_REPO.split("/").at(-1) ?? GITHUB_REPO;
    const repoUrl = repo?.htmlUrl ?? GITHUB_REPO_URL;
    const starMilestone = repo ? resolveGithubStarMilestone(repo.stars) : null;
    const starsRemaining = repo && starMilestone ? Math.max(0, starMilestone - repo.stars) : 0;
    const starProgress = repo && starMilestone ? Math.min(100, Math.round((repo.stars / starMilestone) * 100)) : 0;
    const handleRepositoryOpen = () => {
        if (unlockLoginVisual()) {
            setUnlockDialogOpen(true);
        }
        void unlockDiscoverability.mutateAsync().catch(() => {
            // Keep the optimistic local unlock; the next authenticated check retries the account sync.
        });
    };
    return (<>
      <Popover onOpenChange={handlePopoverOpenChange}>
      <PopoverTrigger render={<Button size="sm" variant="ghost" aria-label={t("openPopover")} aria-busy={isRefreshing || undefined}/>}>
          <IconBrandGithub className="size-4"/>
          {repo ? (<span className={cn(textRole.monoLabel, "hidden tabular-nums text-muted-foreground sm:inline")}>{repo.stars.toLocaleString()}</span>) : null}
        </PopoverTrigger>
      <PopoverContent align="end" sideOffset={shellOverlaySideOffsets.header} className="w-80 p-0 sm:w-96">
        <div className="space-y-4 p-4">
          <div className="flex items-start gap-3">
            <span className="radius-round flex size-12 shrink-0 items-center justify-center border bg-background">
              <LunaFoxMark className="size-7" decorative/>
            </span>
            <div className="min-w-0 flex-1 space-y-1">
              <p className={cn("truncate", textRole.sectionTitle)}>{repoName}</p>
              <p className={cn("truncate", textRole.monoLabel)}>{appVersion}</p>
            </div>
          </div>

          {repo?.description ? <p className={cn("leading-6", textRole.bodySubtle)}>{repo.description}</p> : null}

          <div className="h-px bg-border"/>

          {repo ? (<>
              {starMilestone ? (<div className="space-y-2">
                  <div className="flex items-center justify-between gap-3">
                    <div className="flex items-center gap-2">
                      <IconStar className="size-4 text-muted-foreground"/>
                      <span className={textRole.navLabel}>{t("stars", { count: repo.stars })}</span>
                    </div>
                    <span className={cn("tabular-nums", textRole.helperText)}>{t("remaining", { count: starsRemaining })}</span>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="h-2 flex-1 overflow-hidden radius-pill bg-muted">
                      <div className="h-full bg-primary transition-[width]" style={{ width: `${starProgress}%` }}/>
                    </div>
                    <span className={cn("w-9 text-right tabular-nums", textRole.navLabel)}>{starProgress}%</span>
                  </div>
                </div>) : null}

              <div className="grid grid-cols-3 divide-x divide-border text-center">
                <GithubMetric icon={<GitBranch className="size-4"/>} label={t("forks")} value={repo.forks}/>
                <GithubMetric icon={<Eye className="size-4"/>} label={t("watchers")} value={repo.watchers}/>
                <GithubMetric icon={<IconInfoCircle className="size-4"/>} label={t("issues")} value={repo.issues}/>
              </div>
            </>) : null}

          <Button size="lg" layout="fullWidth" aria-label={t("openRepository")} render={<a href={repoUrl} target="_blank" rel="noopener noreferrer" onClick={handleRepositoryOpen}/> }>
              <IconStar className="size-4"/>
              <span>{starMilestone ? t("sprintCta", { count: starMilestone }) : t("cta")}</span>
              <ExternalLink className="size-4"/>
            </Button>
        </div>
      </PopoverContent>
      </Popover>
      <Dialog open={unlockDialogOpen} onOpenChange={setUnlockDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("unlock.title")}</DialogTitle>
            <DialogDescription>{t("unlock.description")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" onClick={() => setUnlockDialogOpen(false)}>{t("unlock.confirm")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>);
}
function resolveGithubStarMilestone(stars: number) {
    return Math.ceil((stars + 1) / GITHUB_STAR_SPRINT_SIZE) * GITHUB_STAR_SPRINT_SIZE;
}
async function fetchGithubRepoSnapshot(): Promise<GithubRepoSnapshot> {
    try {
        return await fetchGithubRepoSnapshotFrom(`https://api.github.com/repos/${GITHUB_REPO}`);
    }
    catch {
        return fetchGithubRepoSnapshotFrom(GITHUB_REPO_FALLBACK_API);
    }
}
async function fetchGithubRepoSnapshotFrom(url: string): Promise<GithubRepoSnapshot> {
    const response = await fetch(url);
    if (!response.ok) {
        throw new Error("Failed to fetch GitHub repository");
    }
    return parseGithubRepoSnapshot(await response.json());
}
function writeGithubRepoCache(snapshot: GithubRepoSnapshot) {
    writeSessionValue(GITHUB_REPO_CACHE_KEY, JSON.stringify(snapshot));
    writeSessionValue(GITHUB_REPO_CACHE_TIME_KEY, Date.now().toString());
}
function parseGithubRepoSnapshot(data: unknown): GithubRepoSnapshot {
    if (!data || typeof data !== "object") {
        throw new Error("Invalid GitHub repository response");
    }
    const repo = data as Record<string, unknown>;
    if (typeof repo.name === "string" &&
        typeof repo.fullName === "string" &&
        typeof repo.htmlUrl === "string" &&
        (typeof repo.description === "string" || repo.description === null) &&
        typeof repo.stars === "number" &&
        typeof repo.forks === "number" &&
        typeof repo.watchers === "number" &&
        typeof repo.issues === "number") {
        return repo as GithubRepoSnapshot;
    }
    if (typeof repo.name !== "string" ||
        typeof repo.full_name !== "string" ||
        typeof repo.html_url !== "string" ||
        typeof repo.stargazers_count !== "number" ||
        typeof repo.forks_count !== "number" ||
        typeof repo.subscribers_count !== "number" ||
        typeof repo.open_issues_count !== "number") {
        throw new Error("Invalid GitHub repository response");
    }
    return {
        name: repo.name,
        fullName: repo.full_name,
        htmlUrl: repo.html_url,
        description: typeof repo.description === "string" && repo.description.trim().length > 0 ? repo.description : null,
        stars: repo.stargazers_count,
        forks: repo.forks_count,
        watchers: repo.subscribers_count,
        issues: repo.open_issues_count,
    };
}
function GithubMetric({ icon, label, value, }: {
    icon: React.ReactNode;
    label: string;
    value: number;
}) {
    return (<div className="space-y-1 px-2">
      <div className="flex items-center justify-center gap-1.5 text-muted-foreground">
        {icon}
        <span className={textRole.helperText}>{label}</span>
      </div>
      <p className={cn("tabular-nums", textRole.navLabel)}>{value.toLocaleString()}</p>
    </div>);
}
