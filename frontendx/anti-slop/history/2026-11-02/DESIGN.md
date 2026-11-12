---
name: Kunjungan Nusantara
description: Neutral materials, restrained IKN accents, and a compact Geist backoffice adaptation.
colors:
  navy: "#0a2342"
  blue: "#185088"
  blue-pale: "#e3ecf6"
  gold: "#dbaf6c"
  page: "#f5f5f7"
  white: "#ffffff"
  ink: "#1d1d1f"
  muted: "#63636c"
  line: "#e5e5ea"
  focus: "#1769c2"
  action-primary: "#1769c2"
  action-primary-hover: "#145dae"
  action-primary-pressed: "#105198"
  action-secondary: "#edf3fa"
  action-secondary-hover: "#e0ebf7"
  action-secondary-pressed: "#d4e3f3"
  action-secondary-border: "#b5c6da"
  action-disabled: "#e5e5ea"
  action-disabled-text: "#63636c"
  control-white: "#fff"
  danger: "#c43b32"
  danger-hover: "#b0332b"
  danger-pressed: "#9d2d26"
  field-line: "#cbd0d4"
  pending-text: "#754b00"
  pending-bg: "#fff0bf"
  approved-text: "#146537"
  approved-bg: "#d6f8df"
  rejected-text: "#a32135"
  rejected-bg: "#ffe0e2"
  cancelled-text: "#4b5563"
  cancelled-bg: "#e7eaf0"
  checked-in-text: "#075985"
  checked-in-bg: "#d8efff"
  checked-out-text: "#5636a5"
  checked-out-bg: "#e5ddff"
typography:
  display:
    fontFamily: '"Plus Jakarta Sans Variable", -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, sans-serif'
    fontSize: "clamp(2.7rem, 3.8vw, 4.15rem)"
    fontWeight: 650
    lineHeight: 1.14
    letterSpacing: "-0.055em"
  headline:
    fontFamily: '"Plus Jakarta Sans Variable", -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, sans-serif'
    fontSize: "clamp(2rem, 3vw, 2.8rem)"
    fontWeight: 650
    lineHeight: 1.16
    letterSpacing: "-0.04em"
  body:
    fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, sans-serif'
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, sans-serif'
    fontSize: "0.83rem"
    fontWeight: 600
  button:
    fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, sans-serif'
    fontSize: "0.9rem"
    fontWeight: 600
rounded:
  badge: "7px"
  field: "10px"
  control: "16px"
  panel: "20px"
  dialog: "20px"
  hero: "28px"
spacing:
  field-gap: "7px"
  action-gap: "10px"
  toolbar-gap: "12px"
  form-gap: "20px"
  content: "24px"
  section-gap: "28px"
components:
  button-primary:
    backgroundColor: "{colors.action-primary}"
    textColor: "{colors.control-white}"
    typography: "{typography.button}"
    rounded: "{rounded.control}"
    padding: "11px 19px"
  button-primary-hover:
    backgroundColor: "{colors.action-primary-hover}"
  button-primary-pressed:
    backgroundColor: "{colors.action-primary-pressed}"
  button-secondary:
    backgroundColor: "{colors.action-secondary}"
    textColor: "{colors.action-primary}"
    rounded: "{rounded.control}"
    padding: "11px 19px"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.action-primary}"
    rounded: "{rounded.control}"
    padding: "10px"
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.control-white}"
    rounded: "{rounded.control}"
    padding: "11px 19px"
  button-disabled:
    backgroundColor: "{colors.action-disabled}"
    textColor: "{colors.action-disabled-text}"
    rounded: "{rounded.control}"
    padding: "11px 19px"
  input:
    backgroundColor: "{colors.control-white}"
    textColor: "{colors.ink}"
    rounded: "{rounded.field}"
    padding: "11px 14px"
  panel:
    backgroundColor: "{colors.white}"
    rounded: "{rounded.panel}"
  status-pending:
    backgroundColor: "{colors.pending-bg}"
    textColor: "{colors.pending-text}"
    rounded: "{rounded.badge}"
    padding: "5px 9px"
---

# Design System: Kunjungan Nusantara

## Overview

30 October reservation/access decisions: preserve the existing creation and decision surfaces. Requested time overlaps do not block a Pending submission. Approval/reschedule conflicts use the existing inline error and retain selected room/duration or schedule inputs so staff can correct the decision. Email/code lookup remains the visitor entry; no separate proof-link, resend-link or OTP surface is planned under the revised G02 requirement. Production QR remains server-issued; no new visual asset or navigation is introduced by these decisions.

30 October staff freshness control: a compact wrapping row inside the operational page provides one Perbarui action, the last successful WITA timestamp and plain-language failure/offline feedback. Use the existing Geist staff typography and muted colors; warnings use text plus color. The button remains at least 44px tall and works with the keyboard. A polite live region announces manual success and connection/refresh failures without announcing every periodic success. One shared poll owner serves all widgets; background refresh preserves content, filters, scroll and forms. There is no added animation or visual asset. Scope is frontendx with a future unbuilt backend at `D:/oikn/VMS/backend`.

**Creative North Star: Neutral Apple materials, IKN accents.**

Updated on 22 October 2026. White panels, cool gray canvas and graphite text lead both apps. Plus Jakarta Sans remains the visitor heading voice. The backoffice uses a compact self-hosted Geist face for its interface and data. Aksara navy, blue and gold remain accents, with the original institutional emblem and photograph retained.

26 October 2026 review update: all six status badges now use the brighter staff palette and glass finish through the shared UI styles. VIP follows the same badge surface and size with its gold tint. Shared filled actions use solid blue or red, a quiet inner highlight, and a short shadow; the shared button corner is 16px. The initial rejection action uses a red outline while final rejection remains filled. Visitor and staff typography remain distinct because the public flow benefits from expressive headings while staff screens need compact scanning. See `DESIGN-SYSTEM-REVIEW.md` for the current inventory.

Query timeout errors now group the message and retry control inside one pale red panel. The retry action uses a warm outlined style within that state, and raw timeout text is localized before display.

This is a code-led preview, not an approved raster comp. Shared tokens and styles are authoritative. Application state, validation, routing, permission scopes and backend boundaries are unchanged.

## 29 October approved audit refinements (planned)

G10 implemented and verified in preview on 31 October: employee cards on phones/tablets with name, email and active role; a desktop table above 1000px; stacked narrow-screen heading/actions and labeled search below the directory heading. Long fields wrap, no-match feedback includes a reset action and result changes are announced. Reuse existing opaque panels, typography, button and focus tokens. Rendered 320/390/768/1280px checks pass; actual browser zoom remains unverified.

Use consistent loading, empty, no-match/reset, error/retry and access-denied feedback where list pages lack it. Background refresh keeps filters, scroll and drafts stable, shows last successful update and labels older results after failure. Do not present failed or forbidden report data as zeros. Reporting starts with scoped date/status summaries and summary exports.

G11 requirement revision, 31 October: retain the existing visitor choice-card selection, Kembali/browser Back and responsive language-control placement. Direct links between registration and status are no longer required. Expanded mobile tasks keep their sticky Back/logo/language toolbar, background isolation and input continuity. No visitor UI change is made for this decision; accessible control names and heading structure are separate audit items.

G12/G13 implemented and verified on 31 October: public history renders localized system actions and WITA timestamps without raw staff messages/attribution. Explicit public rescheduling text and the public rejection reason keep their wording; staff audit stays separate. Expanded/standalone tasks use h1, inline desktop tasks use h2 below the landing h1, and sections follow their parent. Preserve visual sizes with selectors/classes. Initial lookup focuses the result heading, refresh restores its action focus, and check-another focuses the code input. G15 disables cancellation/confirmation at the service-time cutoff with an explanation; a stale conflict refreshes the displayed record. G09 retains one server-assigned active role per account, with administrative reassignment requiring session/cache refresh.

These approved refinements are not implemented by this documentation update. See [the implementation plan](anti-slop/IMPLEMENTATION-PLAN.md). Existing visual direction remains the reference for their implementation.

## Colors

Canvas #f5f5f7, panel #ffffff, ink #1d1d1f, muted #63636c and divider #e5e5ea form the neutral foundation. Subtle surfaces use #f0f0f3; selected navigation uses #edf2f8. Following the user's approved 22 October button proposal, primary controls use balanced blue #1769c2 with white labels, #145dae hover and #105198 pressed states. Secondary controls use #edf3fa with #1769c2 labels and a #b5c6da edge; hover is #e0ebf7, and pressed is #d4e3f3 with #105198 labels. Destructive controls use #c43b32, #b0332b hover and #9d2d26 pressed. Disabled buttons use #e5e5ea with #63636c labels. These are custom web colors inspired by Apple's selective emphasis, not fixed native system color values. The existing IKN blue #185088 continues on navigation, links, icons and semantic status details. Gold #dbaf6c marks the hero underline, staff selection edge and VIP details. Navy #0a2342 remains available for small accents.

The backoffice adapts the supplied Lightswind reference within these shared action and status colors. Its canvas is #f8fafc, ink #11131b, muted text #596273, divider #e4e4e7 and subtle surface #f3f5f8. Primary actions keep OIKN's balanced #1769c2; the reference's electric blue and teal are not copied across staff workflows.

The six semantic status pairs retain labels and icons. Rescheduled remains a secondary annotation on Approved; VIP is independent. Never rely on color alone.

## Typography

Visitor headings, brand and metric numbers use self-hosted **Plus Jakarta Sans Variable**, weights 200–800 supported by one 27.35 KB Latin WOFF2 from @fontsource-variable/plus-jakarta-sans 5.3.0 (OFL-1.1). The backoffice uses self-hosted **Geist Variable** from @fontsource-variable/geist 5.3.0 for headings, body, controls and tabular data. Both use font-display: swap and system fallbacks. No font service or font JavaScript is loaded.

The public heading is clamp(2.7rem, 3.8vw, 4.15rem), weight 650, line-height 1.14, tracking -0.055em. Phones use clamp(1.75rem, 7.4vw, 2.5rem), line-height 1.2, tracking -0.04em and balanced wrapping. Shared page headings use clamp(2rem, 3vw, 2.8rem), weight 650 and tracking -0.04em. Body is 15px/1.5 with normal tracking; visitor body and fields use 16px. Metrics and booking codes retain tabular numerals.

## Layout

29 October navigation refinement: the visitor navbar uses one row, 72px on desktop and 64px on phones. The logo is the Home link. Keep only status and visit registration as navigation choices, omit the action for the current dedicated task page, and use a single native ID/EN selector. At phone widths, shorten the visible registration label while retaining its full accessible name. The existing mobile reveal and full-screen task behavior remain. The backoffice replaces its desktop collapse button with a visible grip on the sidebar's right edge: dragging follows the pointer, then snaps to expanded (250px, or 220px at compact widths) or icon-only (76px). The edge also supports clicking, arrow keys, Home/End, Enter/Space, and Escape to cancel a drag. Preserve the saved expanded preference, reduced-motion behavior, and the mobile drawer's separate open/close controls.

Visitor desktop hero separates left-aligned text on white from an undimmed architectural image on the right. Following the user's middle phone reference and later 22 October refinements, phones lead with an edge-to-edge IKN photo at 82svh; a white choice card overlaps its lower 84px. The image stays in view as the card rises. A 48px-tall link combines the decorative handle and Geser ke atas untuk mulai and scrolls to the choices. Initial mobile document content ends after Sudah punya booking?: form, guidance and footer are unavailable until an action is selected. The mobile landing navbar starts offscreen and appears after 60px of scroll or focus within it, using a compact 64px content height. Initial photo controls provide the brand/home link and Cek status. The visitor shell caps at 1440px and booking at 1140px. Staff retains a 250px neutral sidebar (220px at compact widths), a 1600px content cap and its existing task hierarchy. Dashboard welcome and metrics are white rather than brand-colored blocks.

Selecting Daftar kunjungan or Sudah punya booking? sets the validated registration/status URL choice and replaces the choice content inside the same persistent mobile card. That existing element rises from its measured current position to the space below the header over 480ms with cubic-bezier(0.16, 1, 0.3, 1). The task wrappers remain transparent/static inside it; the mobile status panel is flattened, with no separate fixed task surface or independent entrance. The card alone contains native overflow scrolling, while body position stays locked/preserved and photo/footer content is inert. Reserve the photo scene's flow height using its viewport size plus the saved background scroll, preventing a sticky-image jump when the card becomes fixed. Upward scrolling cannot return to the landing. The OIKN brand stays in the bar as a non-link while locked. Kembali retains its 44px target 16px below the bar and sticky access. The registration instance stays mounted while hidden to retain input, including across booking steps; privacy closes when inactive. Reduced motion removes the expansion motion. Direct booking/status routes and desktop retain regular page scrolling and the desktop split hero. This supersedes separate overlapping task cards as well as the earlier swipe-straight-to-form approach; no touch/wheel interception or scroll-position clamping is used.

Latest full-screen refinement supersedes the 65px header reservation above: selected mobile cards use inset:0, zero corner radius and z-index 40 to cover the entire viewport. Landing navigation is hidden, inert and aria-hidden while selected. Back and the static OIKN logo move into one compact sticky toolbar within the form, with safe-area-aware internal padding and 44px targets. The card keeps its glass surface and opaque fields. Expansion morphs its corners from 28px to 0; Back reverses them while lowering that same card to its former position. The toolbar scrolls with the card's transition and remains at its top during task scrolling. Desktop retains its ordinary navigation/back link and has no extra logo toolbar.

The bounded photo sequence uses GSAP 3.15 ScrollTrigger: scale 1.08 to 1.02 with at most -0.5% vertical translation, scrub 0.3, over 55% of viewport height. Photo toolbar/caption fade as navigation returns. The card hint remains outside that fade so its tap/focus target stays stable. Native sticky positioning provides the visual overlap without a pin spacer; no wheel/touch interception or animation delay is added to form controls. The motion chunk loads only at widths up to 760px, heights above 500px and with no reduced-motion preference. MatchMedia contexts revert on breakpoint/preference changes and route unmount; the header observer updates only at its threshold. Reduced motion disables photo motion and sticky positioning, retaining all content/actions. Short landscape screens use a 92svh static photo. The initial 42svh photo and always-visible mobile landing header from the first pass are superseded.

Latest Back refinement: ordinary mobile activation restores the choices inside the same card, then lowers that card to its remembered earlier position over 420ms using cubic-bezier(0.4, 0, 0.2, 1). It starts at the current computed transform, allowing an early Back press during expansion. Background interaction/scroll remain locked until the return completes. Then the selected URL view closes and native smooth scrolling returns the landing to the IKN photo at page top; main receives focus without a jump. Repeated activation is ignored during return. Route/breakpoint changes and unmount cancel pending motion/navigation; a new reduced-motion preference finishes immediately. Reduced motion skips both card motion and smooth scrolling. Modified links, desktop anchors and browser history remain available. The earlier animation of a separate surface down offscreen is superseded.

Breakpoints remain shared 640px, visitor 760px/1000px/1100px, and staff 760px/1150px/1500px. Phone fields stack; tablet booking uses one full-width column, and the contextual summary appears only above 1000px. Staff navigation becomes a modal drawer. At phone widths booking rows become compact cards with name, VIP, status and time together and a native status selector. Desktop retains the sortable table; dashboard/reception summaries fit with wrapping columns. Table headers establish a positioning context so their visually hidden labels cannot create page overflow.

## Elevation & Depth

Panels use 0 2px 6px #1d1d1f03, 0 12px 32px -20px #1d1d1f1c and an inset #1d1d1f06 hairline. Dialog shadow is 0 25px 100px #1d1d1f33. Shared glass remains rgba(255,255,255,0.84), static blur(18px) saturate(135%) for contextual summaries. The user's latest visitor refinement sets its top bar and mobile choice card to 78% white, static blur(24px) saturate(145%). The card has a subtle inset light edge and soft upper shadow over the existing photograph. Its secondary text uses #465160; header secondary text uses #43515f, preserving at least 4.5:1 even over the darkest backdrop. The selected mobile task uses rgba(245,245,247,0.94), blur(24px) saturate(135%), with transparent embedded form wrappers and opaque white fields. Hidden background choice copy cannot show through the task. Back uses 92% white with a static 18px blur. Other content panels remain opaque. This replaces the earlier solid-white mobile card. The photo location caption retains its navy backing, and the desktop photo note hides on phones. Reduced transparency, increased contrast and unsupported standard/prefixed blur use opaque white fallbacks. Blur itself is never animated; the requested photo/task motion remains unchanged.

Backoffice panels use a 1px #e4e4e7 border, 16px radius and 0 1px 2px #11131b0a shadow. Dashboard metrics are separate linked cards with the same surface and a subtle focus/hover edge. The backoffice top bar alone keeps translucent white; operational cards, tables, forms and dialogs stay opaque.

Light secondary buttons in the backoffice use a translucent white-to-blue surface, a fine blue-gray edge, inset highlight, soft offset shadow and static 12px backdrop blur. Hover and pressed states deepen the tint while retaining dark blue text. Reduced-transparency preferences and browsers without backdrop blur receive an opaque #edf3fa surface; disabled controls keep the shared disabled treatment.

At desktop widths above 1150px, the page introduction has a thin bottom divider, the welcome panel places its primary actions beside the copy, and metrics and table rows use tighter spacing. Reception actions appear as two standalone cards so the next operational table stays visible sooner. These rules are scoped to desktop; narrower layouts retain their responsive arrangement.

Backoffice data tables use compact 11px by 16px body cell padding, restrained cool-gray dividers and a faint blue hover; phone dashboard cells keep their narrower 10px horizontal inset. The shared status badges use the brighter semantic tints across visitor and staff: amber pending, green approved, rose rejected, slate cancelled, sky checked in and violet checked out. Each retains its label and icon with dark text for contrast.

The shared status tints render as translucent glass chips with a soft white sheen, inset highlight, short shadow and static 8px blur. VIP follows the same material with a gold tint. Reduced-transparency, increased-contrast and unsupported-blur modes use their opaque semantic tints. Sign out sits in the sidebar footer beneath the visitor portal link, including in the phone navigation drawer.

On phones the dashboard agenda uses a compact linked list showing time, visitor, organization and status; desktop keeps its table. This prevents narrow columns from breaking names at 320px while keeping the same scoped data and links.

The floating photo-logo tile and status shortcut use 78% white with static blur(18px) saturate(145%); the logo keeps its 48px tile and 34px emblem. Their opaque accessibility/unsupported-blur fallbacks match the bar/card. Floating photo controls hide while the card is expanded so they cannot peek from beneath the active bar. The glass card owns the material; its intro and task wrappers do not paint another background.

## Shapes

Badges 7px, fields 10px, controls 12px, visitor panels/dialogs 20px, backoffice panels 16px and desktop public hero 28px. The mobile hero photo runs edge to edge; its overlapping content/form surface uses 28px corners at its top and bottom. Controls retain 44px minimum targets and fields 48px. Dividers are subtle neutral strokes; avatars and progress indicators retain their established shapes.

## Components

Date and time controls use source-owned shadcn/ui components and the ReUI time picker, adapted to the existing tokens without a global styling migration. Dates compose Calendar and Radix Popover; time uses ReUI's scrollable hour/minute columns in a 24-hour format. Calendar captions have Indonesian month/year dropdowns, blue selected days, neutral today states and muted unavailable dates. The calendar dropdown is 284px wide and the time dropdown 216px, with softer time selection and larger targets on touch screens. Time changes apply through Pilih; Escape discards the draft. Calendar code preloads on intent and renders only when opened. Keep required marks, errors, WITA limits and ISO date/HH:mm time contracts consistent across all four scheduling flows. Restricted-visit detail editing keeps both schedule controls read-only. A separate modal scrolling body prevents popup clipping while preserving nested Escape and trigger focus. See DATE-PICKERS.md for sources, behavior and verification.

Required form labels use the shared red (#b42318), bold asterisk and the hint “Kolom bertanda * wajib diisi.” The asterisk is decorative (aria-hidden); native required attributes carry the requirement for assistive technology, while the hint includes a visually hidden “bintang.” Apply this consistently to visitor/staff fields, email parts, named group members and required acknowledgments. Optional official documents/internal notes, search/filter controls and the preview-role selector stay unmarked. This convention was implemented and checked on 22 October 2026.

Visitor email separates the username from one native dropdown of complete suffixes, including @ikn.go.id. No provider/TLD combination steps are needed. Domain lain reveals a domain-only field; full pasted/autofilled addresses split without losing custom domains. Username uses type=text/inputMode=email because it is only part of an address; the composed value uses the shared email schema. Labels, error descriptions, 16px controls and 48px targets remain, with stacked parts below 360px. Phone uses a numeric keyboard with digit-only text input and preserved leading zeroes. Neither field uses a numerical data type for contact identifiers.

Rejection forms use two full-width textareas: Pesan penolakan untuk pengaju with the required asterisk and Catatan internal (opsional), with explicit audience hints, 1000/2000 character limits and associated inline validation. Focus starts at the public message and returns to the first invalid field after validation. Staff detail presents the messages in distinct labeled sections; visitor status includes only the public reason. Preserve line breaks and wrap long text. Existing decision roles and scope remain.

Staff appointment creation is available from Dashboard and the Booking heading for PA/Protocol with create permission. Load the form only on opening. Use the existing modal, shared email suffix control, digit-only phone, labeled fieldsets, roster controls and optional official-document field. Select destinations only within the current role's scope; preselect a sole valid destination. New appointments explicitly start Pending and open their detail for approval. Keep a filled form when submission fails and offer continue/discard when closing. Phone header actions stack, form fields use 16px text, and the existing single-column modal breakpoint applies. Internal creator attribution appears only in staff detail.

Receptionist status navigation contains All, Approved, Checked in and Checked out; All counts only those three states. Use visit time as the default sort. Dashboard remains arrival-first and includes completed visits in counts/history. Out-of-scope selected records use a generic unavailable notice; code/QR resolution reveals no visitor identity for Pending, Rejected or Cancelled. An old disallowed status filter resets to All. PA/Protocol retain the full six-status decision workflow.

Buttons, forms, filters, status badges, loading/error/empty states and identity dialogs retain their existing behavior. Shared filled primary and danger buttons use solid color, a quiet inner highlight and a short shadow. An outlined danger variant marks entry into a rejection flow; the final rejection action is filled. Ghost actions use blue labels and a pale hover fill. Controls retain 44px minimum targets, 16px corners and system typography. Disabled buttons retain their neutral colors when hovered or pressed; reduced motion removes press scaling as well as transitions. Focus uses a 3px #1769c2 ring with the existing 4px offset. Staff active navigation retains blue text, a pale selected fill and a slim gold edge. Reception method choices are neutral with blue icons and a light blue hover. The mobile drawer retains Radix focus containment, Escape and trigger restoration.

## Do's and Don'ts

- Keep IKN color selective; broad surfaces stay neutral.
- Use the display face for hierarchy and the system face for sustained operational reading.
- Preserve visible status labels, semantic colors, responsive table containment, focus states and minimum control sizes.
- Keep the 27.35 KB font self-hosted and retain its license in the dependency.
- Preserve the original IKN emblem (16,760 bytes) and existing photograph (516,917 bytes); provenance remains in .impeccable/direction.md.
- Do not infer real backend or production readiness from this synthetic preview. New workflow HTTP operations remain unavailable until contracts are reconciled.
- No Impeccable engine/detector ran. Earlier review applies to the previous implementation; this visual revision is manually checked by the parent.

Current visual evidence: .impeccable/review/accent-visitor-desktop.png, accent-bookings-desktop.png, accent-visitor-mobile.png, accent-form-mobile.png, accent-dashboard-mobile.png and accent-bookings-mobile.png. See VERIFICATION.md for the exact checks and limits.

21 October refinement: Receptionist dashboard prioritizes arrivals and departures. Shared dialogs accept an explicit opening element and restore it on close; when it no longer exists, focus goes to main. Visitor field errors use stable descriptions; active table sort direction is exposed on every sortable header. Staff detail displays optional ND metadata without a false download action. Visitor form is single-column by default, adds summary above 760px, uses 16px inputs and compact phone headings. Homepage note has no decorative action arrow. Status lookup has one email/code form without OTP. Draft storage remains a separate plan.


## Restricted identity and arrival display — 30 October

G03/G05 are implemented in the synthetic preview. Keep the existing form/dialog/typography tokens. Terbatas shows date/start time and only four optional fields: Nama, Instansi, No ID Dinas (text) and target officer. RestrictedIdentity renders the service-projected audience: full Protocol/Admin, last-four ID for assigned PA, reference-only reception general detail, and name/last-four ID/target after valid arrival code/QR. Hidden fields are omitted by the service, not merely styled away. No extra Protocol acknowledgment is introduced. General reports/exports exclude Terbatas.

Terlewat/Missed is a derived display for Approved visits after their scheduled end without check-in, using the shared neutral status treatment plus clock icon/text. It is not a seventh stored status. Entry QR is absent for Missed; check-in opens 30 minutes early through end, and Checked-in visitors retain late checkout. Timing metadata comes from the service and refreshes through the existing shared owner; every mutation rechecks service time. No per-row timer, new asset, dependency or animation was added.


## Submission validation and recovery — 30 October

G06/G07 keep the existing three booking steps, controls and review confirmation. Server field errors return to the owning step and relevant input without clearing other entries. No request is sent when moving into review. A small SubmissionRecovery panel explains uncertain outcomes, requests the original email and offers an explicit check. Localized Indonesian/English copy, labeled errors, busy/disabled controls and programmatic heading/field/receipt focus use existing tokens. The panel reflows at 320px through desktop. Only a pending random reference survives reload; the form draft does not. A confirmed missing result keeps the current draft and requires another explicit send. A recovered completed result shows the original receipt/current status, never automatically sends edited details. No extra polling, asset, dependency or animation.


## Staff role requirement — 30 October

G09 now specifies one active server-assigned role per account. The multiple-assigned-workspace switch is superseded. The development selector remains visibly a test fixture; no production self-switch or union of role permissions is required. Authorized administrative role changes must refresh/invalidate sessions and private scopes. Actual backend/session integration remains unbuilt and unverified.


## Summary reports — 30 October

G08 uses a lazy Reports page for analytics and visit reports. Native labeled WITA date/status filters live in the URL after Apply. Pending filter edits disable refresh/export; invalid filters focus the relevant control. A successful summary shows period, authorized scope, update time, total and a semantic status-count table with labeled meters. The count column stays inside its panel at phone widths. Six stored lifecycle states plus derived Terlewat are mutually exclusive display groups.

Loading/error/denied/offline states do not substitute zero. Temporary failures may retain clearly marked older successful counts with export disabled; denial hides cached totals. Successful zero has explicit no-match feedback. Read/export permissions are separate; scope/counters are recalculated before revision-checked CSV, with no visitor rows or restricted events. Changed summaries refresh and require another explicit export. Current calendar month and 366-day bound are preview defaults. No new chart library, animation or asset.
