# Prova Frontend QA Report

**App:** http://localhost:3000  
**Project:** /Users/eminkutlu/dev/Prova/frontend/web  
**Date:** 2026-09-05 (Europe/Istanbul)  
**Viewport:** desktop 1440×900, mobile 390×844  
**Stack:** Next.js 16.3.4, React 19, Turkish UI (`lang="tr"`)

## Routes tested

- `/`
- `/personas`
- `/assignments`
- `/sessions`
- `/sessions/o-5512`
- `/certificates`
- `/certificates/PRV-2026-0148`
- `/users`
- `/devices`
- `/audit-log`
- `/llm-routing`
- `/login`
- `/register`
- `/verify`
- `/privacy`

## App map (from source)

### Account group `(account)`
- `/login`, `/register`, `/verify`, `/verify-expired`, `/verify-result`, `/privacy`, `/delete-account`
- Key UI: `FormShell`, email/name inputs, Link CTAs to `/verify`, `PrivacyNotice`, OTP verify flow

### Admin group `(admin)` + sidebar
- `/` Panel, `/personas`, `/assignments`, `/sessions`, `/sessions/[sessionId]`, `/certificates`, `/certificates/[certificateId]`
- `/users`, `/devices`, `/llm-routing` (platform only), `/audit-log`
- Interactive: `AppSidebar` + role `Select`, `RoleGate`, `SessionTable`, invite/assignment dialogs, persona/rubric editors, recertification CTA
- Default simulated role: `org-admin` (`RoleProvider`)

## Console errors

_None collected._

## Network failures (4xx/5xx or requestfailed)

_None collected._

## Findings (severity-ranked)

### Critical

#### 1. Dashboard metric cards unreadable on mobile
- **Where:** `/` (Panel)
- **What:** Summary cards use a fixed `grid-cols-4` layout. At 390px with the persistent 220px sidebar, each card collapses to a few pixels wide; titles/values wrap to single characters and become unreadable.
- **Evidence:** `home_mobile.png` (compare `home_desktop.png`)
- **Suggested fix:** Responsive grid, e.g. `grid-cols-1 sm:grid-cols-2 xl:grid-cols-4`, and/or collapse the sidebar on narrow viewports so content has usable width.

### High

#### 1. Admin shell not mobile-adapted (sidebar always open)
- **Where:** All `(admin)` routes (`/`, `/sessions`, `/users`, …)
- **What:** `Sidebar` uses `collapsible="none"` with `--sidebar-width: 220px`. On 390px viewports the nav never collapses; main content is ~170px wide. Tables show only truncated columns; employee names cut off with ellipsis.
- **Evidence:** `home_mobile.png`, `sessions_mobile.png`, `users_mobile.png`
- **Suggested fix:** Use a collapsible/off-canvas sidebar (hamburger) below `md`/`lg`, or hide sidebar and show a top nav on small screens.

#### 2. Auth CTA bypasses form validation
- **Where:** `/login`
- **What:** Primary action is a `Link` to `/verify` (`Button` with `render={<Link href="/verify" />}`), not a validated form submit. Empty/invalid email still navigates.
- **Evidence:** `login_desktop.png`; source `src/app/(account)/login/page.tsx`
- **Suggested fix:** Use a real `<form onSubmit>` with `required` / `type="email"` (or zod) and only then `router.push('/verify')`.

#### 3. Auth CTA bypasses form validation
- **Where:** `/register`
- **What:** Same pattern as login — “Doğrulama kodu gönder” is a Link to `/verify` with no client-side required/email checks.
- **Evidence:** `register_desktop.png`; source `src/app/(account)/register/page.tsx`
- **Suggested fix:** Same as login — submit handler + validation before navigation.

### Medium

#### 1. Email field not marked required
- **Where:** `/login`
- **What:** Email input lacks `required` / `aria-required`; combined with Link CTA, empty submit is trivial.
- **Evidence:** `login_desktop.png`
- **Suggested fix:** Add `required` and associate error messaging with the Label.

#### 2. Email / name fields not marked required
- **Where:** `/register`
- **What:** Name and email inputs lack `required`; CTA still navigates.
- **Evidence:** `register_desktop.png`
- **Suggested fix:** Mark required fields; block submit until valid.

#### 3. Session / admin tables clip on narrow widths
- **Where:** `/`, `/sessions` (and similar tables on `/users`, `/devices`, `/certificates`)
- **What:** Multi-column tables remain wide; on mobile only ~2 columns peek through with heavy truncation. Parent uses `overflow-x-hidden`, so there is no horizontal scroll affordance either — content is simply clipped.
- **Evidence:** `sessions_mobile.png`, `home_mobile.png`
- **Suggested fix:** Card/stacked row layout under `md`, or allow intentional `overflow-x-auto` on the table wrapper with sticky first column.

#### 4. Primary control may lack visible focus indicator
- **Where:** `/verify`
- **What:** Keyboard focus probe on first focusable control reported `outline=solid 0px` with no clear ring class detection.
- **Evidence:** `verify_desktop.png`
- **Suggested fix:** Ensure `:focus-visible` ring tokens on OTP inputs / Button variants (invite dialog focus ring on `/users` looked clearer — reuse that pattern).

### Low

#### 1. Next.js / tooling badge overlays content
- **Where:** Multiple admin + account pages
- **What:** Circular “N” badge (Next.js / tooling indicator) sits over table/footer area in screenshots. `next.config.ts` already moves `devIndicators` to `bottom-right`, but it still overlaps UI chrome.
- **Evidence:** `home_mobile.png`, `sessions_mobile.png`, `login_desktop.png`
- **Suggested fix:** Acceptable in local `next dev`; confirm absent in production `next start` builds. Optionally add bottom padding so footer isn’t covered during demos.

### Pass

#### 1. Dialog opens from CTA
- **Where:** `/users`
- **What:** Invite (“Davet” / “Üye ekle”) opens “Kullanıcı davet et” modal with name, email, role fields and Vazgeç / Daveti gönder actions. Focus ring visible on name field.
- **Evidence:** `users_dialog.png`

#### 2. Role gate shown as designed
- **Where:** `/llm-routing`
- **What:** With default `org-admin` role, `RoleGate` correctly shows “Bu sayfaya erişiminiz yok” instead of LLM routing content (platform-only route).
- **Evidence:** `llm-routing_desktop.png`

#### 3. Root layout lang=tr + Turkish copy
- **Where:** `/` and account pages
- **What:** `html lang="tr"`; sidebar, Panel, login/register copy all Turkish. Desktop Panel layout and typography look polished.
- **Evidence:** `home_desktop.png`, `login_desktop.png`

#### 4. No console / network failures on tested routes
- **Where:** All 15 routes
- **What:** Playwright collected zero console errors and zero failed/4xx–5xx same-origin network requests during the crawl.
- **Evidence:** `summary.json` (`consoleErrors: []`, `networkFailures: []`)

#### 5. Certificate detail and session detail resolve
- **Where:** `/certificates/PRV-2026-0148`, `/sessions/o-5512`
- **What:** Dynamic routes render without HTTP errors; screenshots captured successfully.
- **Evidence:** `certificates_PRV-2026-0148_desktop.png`, `sessions_o-5512_desktop.png`

## Screenshots

- `/tmp/prova-qa/shots/assignments_desktop.png`
- `/tmp/prova-qa/shots/assignments_mobile.png`
- `/tmp/prova-qa/shots/audit-log_desktop.png`
- `/tmp/prova-qa/shots/audit-log_mobile.png`
- `/tmp/prova-qa/shots/certificates_PRV-2026-0148_desktop.png`
- `/tmp/prova-qa/shots/certificates_PRV-2026-0148_mobile.png`
- `/tmp/prova-qa/shots/certificates_desktop.png`
- `/tmp/prova-qa/shots/certificates_mobile.png`
- `/tmp/prova-qa/shots/devices_desktop.png`
- `/tmp/prova-qa/shots/devices_mobile.png`
- `/tmp/prova-qa/shots/home_desktop.png`
- `/tmp/prova-qa/shots/home_mobile.png`
- `/tmp/prova-qa/shots/llm-routing_desktop.png`
- `/tmp/prova-qa/shots/llm-routing_mobile.png`
- `/tmp/prova-qa/shots/login_desktop.png`
- `/tmp/prova-qa/shots/login_mobile.png`
- `/tmp/prova-qa/shots/personas_desktop.png`
- `/tmp/prova-qa/shots/personas_mobile.png`
- `/tmp/prova-qa/shots/privacy_desktop.png`
- `/tmp/prova-qa/shots/privacy_mobile.png`
- `/tmp/prova-qa/shots/register_desktop.png`
- `/tmp/prova-qa/shots/register_mobile.png`
- `/tmp/prova-qa/shots/sessions_desktop.png`
- `/tmp/prova-qa/shots/sessions_mobile.png`
- `/tmp/prova-qa/shots/sessions_o-5512_desktop.png`
- `/tmp/prova-qa/shots/sessions_o-5512_mobile.png`
- `/tmp/prova-qa/shots/users_desktop.png`
- `/tmp/prova-qa/shots/users_dialog.png`
- `/tmp/prova-qa/shots/users_mobile.png`
- `/tmp/prova-qa/shots/verify_desktop.png`
- `/tmp/prova-qa/shots/verify_mobile.png`

*(Copies also under `/Users/eminkutlu/dev/Prova/frontend/web/.qa-shots/` for local viewing.)*

## Notes

- Auth is mock/dev: role simulated via sidebar Select (`RoleProvider` default `org-admin`); no real login gate on admin shell.
- Skipped destructive path `/delete-account` (no submit of account deletion).
- Did not submit real payments (none found).
- Automated `scrollWidth` overflow probe was muted by `overflow-x-hidden` on the admin content wrapper; mobile issues were confirmed via full-page screenshots instead.
