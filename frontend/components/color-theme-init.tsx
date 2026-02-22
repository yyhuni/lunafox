/* eslint-disable @next/next/no-before-interactive-script-outside-document */
import Script from "next/script"

const themeInitScript = `(function(){try{
  var root=document.documentElement;
  root.setAttribute('data-theme','bauhaus');
  root.classList.remove('dark');
}catch(e){}})();`

export function ColorThemeInit() {
  return (
    <Script id="color-theme-init" strategy="beforeInteractive">
      {themeInitScript}
    </Script>
  )
}
