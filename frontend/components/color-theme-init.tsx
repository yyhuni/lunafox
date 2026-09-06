import {
  COLOR_THEMES,
  COLOR_THEME_COOKIE_KEY,
  THEME_MODE_COOKIE_KEY,
  DEFAULT_COLOR_THEME_ID,
  DEFAULT_DARK_COLOR_THEME_ID,
  DEFAULT_LIGHT_COLOR_THEME_ID,
  DEFAULT_THEME_MODE_ID,
  type ColorThemeId,
  type ThemeModeId,
} from "@/lib/color-themes"

type ColorThemeInitProps = {
  initialTheme?: ColorThemeId
  initialMode?: ThemeModeId
}

function buildThemeInitScript(
  initialTheme: ColorThemeId,
  initialMode: ThemeModeId
) {
  const darkThemes = COLOR_THEMES.filter((theme) => theme.isDark).map((theme) => theme.id)
  const legacyDarkThemes = [
    "cherry-cocoa-dark",
    "rose-cocoa-dark",
    "graphite-contrast-dark",
    "mist-teal-dark",
    "ink-navy-dark",
    "paper-ink-dark",
    "blue-dark",
    "smoked-amethyst-dark",
  ]
  const legacyLightThemes = [
    "cherry-cocoa-light",
    "rose-cocoa-light",
    "graphite-contrast-light",
    "mist-teal-light",
    "ink-navy-light",
    "paper-ink-light",
    "blue-light",
    "smoked-amethyst-light",
  ]

  return [
    `(function(){try{var r=document.documentElement,c=document.cookie,`,
    `D=${JSON.stringify([...darkThemes, ...legacyDarkThemes])},L=${JSON.stringify(legacyLightThemes)},`,
    `dt=${JSON.stringify(initialTheme)},dm=${JSON.stringify(initialMode)},dl=${JSON.stringify(DEFAULT_LIGHT_COLOR_THEME_ID)},dd=${JSON.stringify(DEFAULT_DARK_COLOR_THEME_ID)};`,
    `function g(k){var m=("; "+c).split("; "+k+"=");return m.length>1?decodeURIComponent(m.pop().split(";").shift()):null}`,
    `var t=g(${JSON.stringify(COLOR_THEME_COOKIE_KEY)}),m=g(${JSON.stringify(THEME_MODE_COOKIE_KEY)});`,
    `if(t==="bauhaus"||t===("shad"+"cn-default"))t=dt;`,
    `if(m!=="light"&&m!=="dark"&&m!=="system"){if(D.indexOf(t)>=0)m="dark";else if(L.indexOf(t)>=0)m="light";else m=dm;}`,
    `var e=m==="system"&&window.matchMedia&&window.matchMedia("(prefers-color-scheme: dark)").matches?1:m==="dark"?1:0,`,
    `n=e?dd:dl;`,
    `r.setAttribute("data-theme",n);(D.indexOf(n)>=0?r.classList.add:r.classList.remove).call(r.classList,"dark")`,
    `}catch(e){}})();`,
  ].join("")
}

export function ColorThemeInit({
  initialTheme = DEFAULT_COLOR_THEME_ID,
  initialMode = DEFAULT_THEME_MODE_ID,
}: ColorThemeInitProps) {
  return (
    <script
      id="color-theme-init"
      dangerouslySetInnerHTML={{ __html: buildThemeInitScript(initialTheme, initialMode) }}
    />
  )
}
