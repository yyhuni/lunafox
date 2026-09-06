# Auth UI Rules

`frontend/components/auth` owns the local composition rules for public auth entry surfaces and protected auth shell handoff. This README is the near-code contract for login visual readiness, boot loading handoff, and auth-specific fallback behavior.

## Protected Route Redirects

- Protected-route unauthenticated access must redirect directly to `/login/?returnTo=...`.
- `returnTo` must preserve the originally requested in-app destination so auth recovery can resume the workflow after login.
- The root app entry `/` is a redirect-only entry and must normalize to `/overview/` before it is stored or consumed as `returnTo`.
- `returnTo` must stay canonical and in-app only. Non-canonical or external values must be rejected and fall back to `/overview/`; auth recovery must not retain a legacy locale-prefix compatibility branch.
- Protected-route `401` is an auth transition, not a page-level error state. `AuthLayout`, `AuthGuard`, and transport ownership must redirect instead of rendering shell-scoped business error UI.
- **Auth pending must never mount the protected app shell.** When `authenticated` is false or auth state is unresolved on a protected route, `AuthLayout` must hold `data-boot-handoff-pending="true"` with a non-visual blocker so the global boot layer remains the only visible first-screen owner. It must not render `AppShellWarmup`, sidebar, header, protected shell chrome, or a second visible auth warmup card. The `ProtectedAuthLayout` component only renders after authentication is confirmed.
- **Protected-shell and route-child suspense use the same non-visual handoff.** While their lazy chunks are pending, they must keep `data-boot-handoff-pending="true"` so the root boot layer remains visible. Do not render `AppShellWarmup`, a route fallback, or a detail-shell skeleton before the route's own approved workspace owner commits.
- **Protected sidebar soft navigation stays framework-owned.** Every navigable sidebar `Link` reports its pending state through `useLinkStatus`; pathname remains the committed route source of truth. The shared sidebar scope suppresses the committed selection while the clicked Link is pending, so success commits the new item and cancellation or failure restores the previous item without a timer-backed route store.
- The protected shell uses the one `protectedAppShellContentFrameClassName` from `protected-app-shell.ts` in both the resolved layout and `AppShellWarmup`. Every authenticated business route uses the available main-workspace width; local route layouts retain readable-text, form, dialog, table, editor, and workflow-canvas constraints. The full-width scroll area and symmetric scrollbar gutter remain shell-owned.
- `ProtectedAuthLayout` MUST NOT observe sidebar pending state to hide, replace, or overlay the committed content region. Any first-screen skeleton belongs to the destination page or workspace through its own query/loading state or `ContentHandoff`; a destination that is already ready may commit content directly.
- If transport-level `401` handling runs while the browser is already on a public auth route such as `/login/`, it must redirect to the canonical `/login/` path without wrapping the current login URL into a new `returnTo`. Recursive `/login/?returnTo=/login?...` redirects are invalid.

## Login Boot Handoff

- `/login/` must keep the server-rendered `lunafox-boot-layer` as the only visible first-screen loading owner until the login surface can show a stable first visual frame.
- Protected unauthenticated entry should flow as: global boot layer remains visible, auth redirects to `/login/?returnTo=...`, login content mounts behind the boot layer, then the boot layer exits only after the login visual is ready. Do not insert the `AUTH / 加载中` warmup card between boot and login.
- `app/login/content.tsx` owns the route-level readiness gate through `loginVisualReady` and `data-boot-handoff-pending`.
- `/login/` must keep the login surface mounted while auth state is unresolved. Client-only token reads can make the browser know more than the server on the first render; that must not remove `ContentReveal` or `data-boot-handoff-pending` before hydration completes.
- `BootLayerController` must treat `data-boot-handoff-pending="true"` as a blocker before starting the boot layer exit.
- `VisualSplitLogin` must call `onVisualReady` only after the left-side visual is ready to hand off:
  - WebGL path: after `FaultyTerminal` reports `onFirstFrame` for a visible page-load frame.
  - Non-WebGL or low-budget path: after the shader decision resolves and the static fallback background is available.
- Do not use `document.readyState`, a fixed timeout, dynamic import mount, or canvas element existence as the route handoff signal. Those can fire before the left-side visual has a visible first frame.

## Protected App Shell Scroll Geometry

- `protectedAppShellStyle` must not override `--sidebar-width`; `SidebarProvider` in `frontend/components/ui/sidebar.tsx` is the shared owner of expanded, icon-only, and mobile sidebar widths.
- The protected app shell main content scroller must use `protectedAppShellScrollAreaClassName` from `frontend/components/auth/protected-app-shell.ts`.
- That shared class owns `overflow-x-hidden`, `overflow-y-auto`, and symmetric `scrollbar-gutter: stable both-edges` reservation so the centered route frame does not shift left when the vertical scrollbar appears or disappears.
- `ProtectedAuthLayout` and `AppShellWarmup` must use the same scroll-area class. Do not hand-roll a separate page-level main scroller in route code to compensate for scrollbar jitter; fix or extend the shared shell owner instead.
- The inner protected route frame must use `protectedAppShellContentFrameClassName` from the same module. It owns the shared uncapped `w-full`, `max-w-none`, centering, and `min-w-0` boundary for all authenticated routes.
- Keep the scroll area full width and apply route-owned readable-width or overflow constraints inside the inner frame. Dense tables, logs, editors, and workflow canvases retain their existing internal overflow behavior.
- `ProtectedAuthLayout` must not add a whole-shell entry animation. Route `ContentReveal` and workspace `ContentHandoff` already own the visible handoff motion; adding shell-level fade creates a double reveal on hard reload.

## Protected Session Renewal

- `ProtectedAuthLayout` owns exactly one `useSessionRenewal` lifecycle while authentication is confirmed. Do not move this ownership into dynamically loaded shell consumers such as `NotificationDrawer`.
- The coordinator schedules from the current JWT `exp`, wakes when the document becomes visible or the browser returns online, and cleans up its timer/listeners with the protected shell.
- Axios and fetch-based notification SSE must still run the shared request-time freshness check. Browser timers can be throttled, so the existing one-retry `401` and SSE `TOKEN_EXPIRED` recovery paths remain required.
- Renewal failure is terminal for the current local session and remains owned by the shared auth redirect boundary; shell or stream code must not add a parallel retry loop.

## Left Visual Fallback

- The left login visual may use a static, theme-owned fallback background while the WebGL shader is unavailable or still preparing.
- The left fallback must not render its own animated loading indicator such as dots, spinners, pulses, shimmer, or progress bars. A second local loading scene creates the broken sequence: global boot loader exits, then a local fallback loader appears, then WebGL appears.
- The intended sequence is:
  1. global boot loader is visible;
  2. login page mounts the same login surface on the server and first client render while `data-boot-handoff-pending="true"`;
  3. WebGL first visible frame or static fallback readiness clears the pending flag;
  4. global boot loader exits over an already-ready login visual.
- If the WebGL visual changes its startup animation, keep `onFirstFrame` tied to the first visible frame rather than the first draw call. A drawn background-only frame is not enough for handoff.

## FaultyTerminal Budget

- Login usage of `FaultyTerminal` should stay capped at `maxFps={30}`.
- Login usage should keep `mouseReact={false}` unless a reviewed interaction need is added. The login background is decorative and should not spend per-pointer frame budget.
- `FaultyTerminal` may continue using canvas/WebGL internals as a `canvas-webgl` foundation exception, but surrounding readable auth UI must keep using theme tokens, typography roles, and shared form controls.

## Verification

- Code guards:
  - `components/__tests__/boot-layer-controller.contract.test.ts`
  - `app/login/__tests__/content.contract.test.ts`
  - `app/login/__tests__/content.test.tsx`
  - `components/auth/__tests__/visual-split-login.contract.test.ts`
  - `components/auth/__tests__/auth-layout.contract.test.ts`
  - `components/__tests__/faulty-terminal.contract.test.ts`
- Browser smoke for this handoff should sample `/login/` in an unauthenticated context and confirm:
  - `data-boot-handoff-pending` stays true while the left visual is not ready;
  - `.auth-shader-fallback__dots` and `.auth-shader-fallback__dot` do not exist;
  - the boot layer exits only after `data-shader-ready="true"` on the WebGL path, or after static fallback readiness on the non-WebGL path.
