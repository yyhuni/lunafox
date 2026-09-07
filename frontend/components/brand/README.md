# Brand Components

`frontend/components/brand` owns the shared LunaFox brand mark and the source assets used by app-level favicon metadata files.

## Owners

- `LunaFoxMark` in `lunafox-mark.tsx` is the only shared product mark for production UI.

## Usage Rules

- Production brand surfaces SHOULD reuse `LunaFoxMark` instead of page-local `img`, avatar PNG, or duplicated logo markup.
- Brand color MUST come from theme tokens, currently `--brand-mark-foreground`. Do not hard-code logo colors in route components.
- When a UI surface needs the product mark inside an avatar-like shell, prefer `LunaFoxMark` as the fallback content instead of a fixed PNG avatar.
- Do not apply unconditional `grayscale` or other image-only filters to the shared brand mark. Filters are allowed only for real user avatars or a reviewed exception.

## Favicon Rules

- Browser favicon MUST use Next.js app-level metadata files in `frontend/app`.
- App-level favicon files SHOULD be derived from the shared brand mark source assets instead of route-local image copies.
- Route modules and layouts must not create page-local favicon generation logic.

## Known Pitfalls

- A static PNG avatar such as `/images/brand/fox-mark-64.png` will not track theme changes and should not be used as the default sidebar account mark.

## Verification

- Run the brand contract tests when changing this directory:
  - `cd frontend && pnpm vitest run components/brand/__tests__/lunafox-mark.contract.test.ts components/brand/__tests__/favicon-static.contract.test.ts`
- When the change affects sidebar branding, also verify:
  - `cd frontend && pnpm vitest run components/__tests__/nav-user.contract.test.ts components/__tests__/app-sidebar.contract.test.ts`
- For visual debugging, reproduce with:
  - `cd frontend && pnpm run dev:mock:noauth`
