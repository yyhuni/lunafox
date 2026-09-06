import type React from "react"
import { Suspense } from "react"
import { NextIntlClientProvider } from "next-intl"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { DEFAULT_COLOR_THEME_ID } from "@/lib/color-themes"
import { DEFAULT_THEME_MODE_ID } from "@/lib/color-themes"
import { localeHtmlLang } from "@/i18n/config"
import { loadLocaleMessages, resolveRequestLocale } from "@/i18n/locale"
import { QueryProvider } from "@/components/providers/query-provider"
import { ThemeProvider } from "@/components/providers/theme-provider"
import { MockProvider } from "@/components/providers/mock-provider"
import { UiI18nProvider } from "@/components/providers/ui-i18n-provider"
import { USE_MOCK } from "@/mock/config"
import { ColorThemeInit } from "@/components/color-theme-init"
import { LayoutClientEnhancements } from "@/components/layout-client-enhancements"
import { BootLayerController } from "@/components/boot-layer-controller"
import { Toaster } from "@/components/ui/sonner"
import { AuthLayout } from "@/components/auth/auth-layout"

import "./globals.css"

const BOOT_ROSE_PARTICLE_COUNT = 80

function generateBootRoseSvgMarkup(particleCount: number): string {
  const circles = Array.from(
    { length: particleCount },
    (_, i) => `<circle fill="currentColor" cx="50" cy="50" r="1" opacity="0" data-index="${i}" />`,
  ).join("")
  return `<svg viewBox="0 0 100 100" fill="none" aria-hidden="true"><g id="lunafox-boot-rose-group"><path id="lunafox-boot-rose-path" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="5.5" opacity="0.1" />${circles}</g></svg>`
}

const bootRoseSvgHtml = generateBootRoseSvgMarkup(BOOT_ROSE_PARTICLE_COUNT)

const bootCriticalCss = `
.lunafox-boot-layer {
  position: fixed;
  inset: 0;
  z-index: 2147483000;
  display: grid;
  place-items: center;
  align-content: center;
  background: Canvas;
  background: var(--background);
  color: CanvasText;
  color: var(--foreground);
  color: var(--brand-mark-foreground);
  opacity: 1;
  pointer-events: none;
  transition:
    opacity 220ms cubic-bezier(0.22, 1, 0.36, 1),
    visibility 220ms cubic-bezier(0.22, 1, 0.36, 1);
}

.lunafox-boot-layer--leaving {
  opacity: 0;
  visibility: hidden;
}

html[data-boot-layer-complete="true"] .lunafox-boot-layer {
  display: none;
  opacity: 0;
  visibility: hidden;
}

.lunafox-boot-loader {
  width: clamp(4.5rem, 12vmin, 7rem);
  aspect-ratio: 1;
  display: grid;
  place-items: center;
}

.lunafox-boot-loader__container {
  width: 100%;
  height: 100%;
  display: grid;
  place-items: center;
}

.lunafox-boot-loader__container svg {
  width: 100%;
  height: 100%;
  display: block;
  overflow: visible;
}

@media (prefers-reduced-motion: reduce) {
  .lunafox-boot-layer,
  .lunafox-boot-loader {
    animation: none;
    transition: none;
  }
}
`

const bootRoseScript = `
(() => {
  const bootLayer = document.getElementById("lunafox-boot-layer");
  const container = document.getElementById("lunafox-boot-rose-container");
  if (!bootLayer || !container) return;

  const svgRoot = container.querySelector("svg");
  const svgGroup = container.querySelector("#lunafox-boot-rose-group");
  const svgPath = container.querySelector("#lunafox-boot-rose-path");
  const circles = Array.from(container.querySelectorAll("circle"));
  if (!svgRoot || !svgGroup || !svgPath || circles.length === 0) return;

  const particleCount = ${BOOT_ROSE_PARTICLE_COUNT};
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const config = {
    rotate: true,
    particleCount: particleCount,
    trailSpan: 0.38,
    durationMs: 4600,
    rotationDurationMs: 28000,
    pulseDurationMs: 4200,
    pathSteps: 200,
    strokeWidth: 5.5,
    baseRadius: 7,
    detailAmplitude: 3,
    petalCount: 7,
    curveScale: 3.9,
    targetFrameMs: 1000 / 45,
  };

  let lastDrawAt = -Infinity;

  function normalizeProgress(progress) {
    return ((progress % 1) + 1) % 1;
  }

  function getDetailScale(time) {
    const pulseProgress = (time % config.pulseDurationMs) / config.pulseDurationMs;
    const pulseAngle = pulseProgress * Math.PI * 2;
    return 0.52 + ((Math.sin(pulseAngle + 0.55) + 1) / 2) * 0.48;
  }

  function getRotation(time) {
    if (!config.rotate) return 0;
    return -((time % config.rotationDurationMs) / config.rotationDurationMs) * 360;
  }

  function getPoint(progress, detailScale) {
    const t = progress * Math.PI * 2;
    const petals = Math.round(config.petalCount);
    const x = config.baseRadius * Math.cos(t) - config.detailAmplitude * detailScale * Math.cos(petals * t);
    const y = config.baseRadius * Math.sin(t) - config.detailAmplitude * detailScale * Math.sin(petals * t);
    return {
      x: 50 + x * config.curveScale,
      y: 50 + y * config.curveScale,
    };
  }

  function buildPath(detailScale) {
    const parts = new Array(config.pathSteps + 1);
    for (let i = 0; i <= config.pathSteps; i += 1) {
      const point = getPoint(i / config.pathSteps, detailScale);
      parts[i] = (i === 0 ? "M" : "L") + " " + point.x.toFixed(2) + " " + point.y.toFixed(2);
    }
    return parts.join(" ");
  }

  function getParticle(index, progress, detailScale) {
    const tailOffset = index / Math.max(config.particleCount - 1, 1);
    const point = getPoint(normalizeProgress(progress - tailOffset * config.trailSpan), detailScale);
    const fade = Math.pow(1 - tailOffset, 0.56);
    return {
      x: point.x,
      y: point.y,
      radius: 0.9 + fade * 2.7,
      opacity: 0.04 + fade * 0.96,
    };
  }

  const bootTimelineOrigin = Date.now() - performance.now();
  let cachedDetailScale = -1;
  let cachedPathD = "";

  function getBootTimelineTime(now = performance.now()) {
    return bootTimelineOrigin + now;
  }

  function draw(time) {
    const progress = (time % config.durationMs) / config.durationMs;
    const detailScale = getDetailScale(time);
    svgGroup.setAttribute("transform", "rotate(" + getRotation(time) + " 50 50)");
    if (Math.abs(detailScale - cachedDetailScale) > 0.005) {
      cachedDetailScale = detailScale;
      cachedPathD = buildPath(detailScale);
    }
    svgPath.setAttribute("d", cachedPathD);
    for (let i = 0; i < circles.length; i++) {
      const particle = getParticle(i, progress, detailScale);
      const c = circles[i];
      c.setAttribute("cx", particle.x.toFixed(2));
      c.setAttribute("cy", particle.y.toFixed(2));
      c.setAttribute("r", particle.radius.toFixed(2));
      c.setAttribute("opacity", particle.opacity.toFixed(3));
    }
  }

  draw(getBootTimelineTime());

  if (reduceMotion) return;

  function render(now) {
    if (bootLayer.hidden) return;
    if (now - lastDrawAt < config.targetFrameMs) {
      requestAnimationFrame(render);
      return;
    }
    lastDrawAt = now;
    draw(getBootTimelineTime(now));
    requestAnimationFrame(render);
  }

  requestAnimationFrame(render);
})();
`

const bootHandoffScript = `
(() => {
  const BOOT_LAYER_EXIT_CLASS = "lunafox-boot-layer--leaving";
  const BOOT_LAYER_EXIT_MS = 220;
  const APP_SHELL_WARMUP_SELECTOR = "[data-slot='app-shell-warmup']";
  const bootLayer = document.getElementById("lunafox-boot-layer");
  if (!bootLayer) return;

  const hasPendingBootHandoff = () => Boolean(document.querySelector('[data-boot-handoff-pending="true"]'));

  const isInsideAppShellWarmup = (element) => Boolean(element.closest(APP_SHELL_WARMUP_SELECTOR));

  const isAppShellWarmupOwner = (element) => element.matches(APP_SHELL_WARMUP_SELECTOR) && element.hasAttribute("data-loading-owner");

  const hasRealLoadingOwner = () => Array.from(document.querySelectorAll("[data-loading-owner]")).some((element) => {
    if (element === bootLayer || bootLayer.contains(element)) return false;
    if (isInsideAppShellWarmup(element) && !isAppShellWarmupOwner(element)) return false;
    return element.getAttribute("data-loading-owner") !== "initial-boot";
  });

  // Shell chrome presence indicates a visible next owner once it is outside the temporary warmup shell.
  // Do NOT rely on mainContent.textContent alone — text in the DOM does not prove a
  // visually stable owner occupies the first-screen region.
  const hasVisibleAppLayer = () => {
    return Array.from(document.querySelectorAll("[data-slot='sidebar-wrapper'], [data-slot='sidebar-inset']"))
      .some((element) => !isInsideAppShellWarmup(element));
  };

  const hideBootLayer = () => {
    document.documentElement.setAttribute("data-boot-layer-complete", "true");
    bootLayer.hidden = true;
    bootLayer.setAttribute("aria-hidden", "true");
  };

  const startExit = ({ immediate = false } = {}) => {
    if (bootLayer.getAttribute("data-boot-exit-started") === "true") return;
    bootLayer.setAttribute("data-boot-exit-started", "true");
    bootLayer.classList.add(BOOT_LAYER_EXIT_CLASS);
    bootLayer.setAttribute("aria-hidden", "true");
    if (immediate) {
      hideBootLayer();
      return;
    }
    window.setTimeout(hideBootLayer, BOOT_LAYER_EXIT_MS);
  };

  const getHandoffReadiness = () => {
    if (hasPendingBootHandoff()) return { ready: false, immediate: false };

    const hasLoadingOwner = hasRealLoadingOwner();
    return {
      ready: hasLoadingOwner || hasVisibleAppLayer(),
      // Once a route/workspace skeleton owns the next frame, do not fade the
      // full-screen boot mark over it; that reads as a second loader.
      immediate: hasLoadingOwner,
    };
  };

  const initialReadiness = getHandoffReadiness();
  if (initialReadiness.ready) {
    startExit({ immediate: initialReadiness.immediate });
    return;
  }

  const observer = new MutationObserver(() => {
    const readiness = getHandoffReadiness();
    if (!readiness.ready) return;
    observer.disconnect();
    startExit({ immediate: readiness.immediate });
  });

  observer.observe(document.body, {
    attributes: true,
    attributeFilter: ["data-loading-owner", "data-slot", "data-boot-handoff-pending", "id"],
    childList: true,
    subtree: true,
  });
})();
`

export async function generateMetadata(): Promise<Metadata> {
  const locale = await resolveRequestLocale()
  const t = await getTranslations({ locale, namespace: "metadata" })

  return {
    title: t("title"),
    description: t("description"),
    keywords: t("keywords").split(",").map((keyword) => keyword.trim()),
    generator: "LunaFox ASM Platform",
    authors: [{ name: "yyhuni" }],
    openGraph: {
      title: t("ogTitle"),
      description: t("ogDescription"),
      type: "website",
      locale: locale === "zh" ? "zh_CN" : "en_US",
    },
    robots: {
      index: true,
      follow: true,
    },
  }
}

/**
 * Root layout component
 * Owns the required html/body shell so root-level not-found and global-error
 * surfaces can render through official App Router boundaries.
 */
export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  const resolvedLocale = await resolveRequestLocale()
  const mockEnabled = USE_MOCK

  return (
    <html
      lang={localeHtmlLang[resolvedLocale]}
      data-theme={DEFAULT_COLOR_THEME_ID}
      suppressHydrationWarning
    >
      <body>
        <ColorThemeInit
          initialTheme={DEFAULT_COLOR_THEME_ID}
          initialMode={DEFAULT_THEME_MODE_ID}
        />
        <style dangerouslySetInnerHTML={{ __html: bootCriticalCss }} />
        {/* The inline handoff script may start the exit before React hydrates this node. */}
        <div
          id="lunafox-boot-layer"
          data-loading-owner="initial-boot"
          data-loading-layer="boot"
          data-loading-intent="boot"
          data-slot="boot-layer"
          role="status"
          aria-live="polite"
          aria-label={resolvedLocale === "zh" ? "正在加载" : "Loading"}
          className="lunafox-boot-layer"
          suppressHydrationWarning
        >
          <div className="lunafox-boot-loader" aria-hidden="true">
          <div id="lunafox-boot-rose-container" className="lunafox-boot-loader__container" dangerouslySetInnerHTML={{ __html: bootRoseSvgHtml }} suppressHydrationWarning />
          </div>
        </div>
        <script dangerouslySetInnerHTML={{ __html: bootRoseScript }} />
        <script dangerouslySetInnerHTML={{ __html: bootHandoffScript }} />
        <BootLayerController />
        <a href="#main-content" className="skip-link">
          {resolvedLocale === "zh" ? "跳转到主要内容" : "Skip to main content"}
        </a>
        <ThemeProvider>
          <Suspense fallback={null}>
            <MockProvider enabled={mockEnabled}>
              <RootAppProviders>{children}</RootAppProviders>
            </MockProvider>
          </Suspense>
        </ThemeProvider>
      </body>
    </html>
  )
}

async function RootAppProviders({
  children,
}: {
  children: React.ReactNode
}) {
  const locale = await resolveRequestLocale()
  const messages = await loadLocaleMessages(locale)

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <LayoutClientEnhancements />
      <QueryProvider>
        <UiI18nProvider>
          <AuthLayout>{children}</AuthLayout>
        </UiI18nProvider>
      </QueryProvider>
      <Toaster />
    </NextIntlClientProvider>
  )
}
