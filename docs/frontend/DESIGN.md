---
name: Kunjungan Nusantara
description: Neutral materials, restrained IKN accents, and a compact Geist backoffice adaptation.
colors:
  navy: "oklch(0.25592260 0.06610870 255.485659)"
  blue: "oklch(0.42600465 0.10908810 252.076080)"
  blue-pale: "oklch(0.93922655 0.01662050 250.845310)"
  gold: "oklch(0.77936785 0.09923980 76.995836)"
  page: "oklch(0.97071377 0.00265048 286.350361)"
  white: "oklch(0.99999999 0 0.000000)"
  ink: "oklch(0.23157626 0.00380518 286.098854)"
  muted: "oklch(0.50282099 0.01413969 285.880285)"
  line: "oklch(0.92334308 0.00671753 286.267430)"
  focus: "oklch(0.52411830 0.15787954 254.785480)"
  action-primary: "oklch(0.52411830 0.15787954 254.785480)"
  action-primary-hover: "oklch(0.48223341 0.14616866 255.093956)"
  action-primary-pressed: "oklch(0.43776090 0.13168964 254.840155)"
  action-secondary: "oklch(0.96163647 0.01137803 252.088071)"
  action-secondary-hover: "oklch(0.93534239 0.02009266 250.388412)"
  action-secondary-pressed: "oklch(0.90942103 0.02719030 249.860971)"
  action-secondary-border: "oklch(0.81995299 0.03378766 252.499248)"
  action-disabled: "oklch(0.92334308 0.00671753 286.267430)"
  action-disabled-text: "oklch(0.50282099 0.01413969 285.880285)"
  control-white: "oklch(0.99999999 0 0.000000)"
  danger: "oklch(0.55311213 0.17520923 28.023646)"
  danger-hover: "oklch(0.50933367 0.16283266 28.026741)"
  danger-pressed: "oklch(0.46879990 0.14909035 27.916608)"
  field-line: "oklch(0.85489312 0.00780513 241.680674)"
  pending-text: "oklch(0.44956814 0.09594386 72.623907)"
  pending-bg: "oklch(0.95511725 0.06504919 93.055628)"
  approved-text: "oklch(0.44806736 0.10590388 152.990236)"
  approved-bg: "oklch(0.94802096 0.04859548 153.513623)"
  rejected-text: "oklch(0.47088536 0.16447328 18.532170)"
  rejected-bg: "oklch(0.93273038 0.03435065 13.349119)"
  cancelled-text: "oklch(0.44611200 0.02631248 256.801751)"
  cancelled-bg: "oklch(0.93648325 0.00870663 264.520717)"
  checked-in-text: "oklch(0.44337598 0.10003252 240.789729)"
  checked-in-bg: "oklch(0.94012610 0.03259791 238.135990)"
  checked-out-text: "oklch(0.43641189 0.16890799 290.987750)"
  checked-out-bg: "oklch(0.91432811 0.04664360 296.243246)"
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
    fontSize: "0.875rem"
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
  field-gap: "8px"
  action-gap: "12px"
  toolbar-gap: "12px"
  form-gap: "20px"
  content: "24px"
  section-gap: "24px"
components:
  button-primary:
    backgroundColor: "{colors.action-primary}"
    textColor: "{colors.control-white}"
    typography: "{typography.button}"
    rounded: "{rounded.control}"
    padding: "8px 20px"
  button-primary-hover:
    backgroundColor: "{colors.action-primary-hover}"
  button-primary-pressed:
    backgroundColor: "{colors.action-primary-pressed}"
  button-secondary:
    backgroundColor: "{colors.action-secondary}"
    textColor: "{colors.action-primary}"
    rounded: "{rounded.control}"
    padding: "8px 20px"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.action-primary}"
    rounded: "{rounded.control}"
    padding: "8px 20px"
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.control-white}"
    rounded: "{rounded.control}"
    padding: "8px 20px"
  button-disabled:
    backgroundColor: "{colors.action-disabled}"
    textColor: "{colors.action-disabled-text}"
    rounded: "{rounded.control}"
    padding: "8px 20px"
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

# Current interface and component rules

## Spacing and sizes

Use the shared tokens in `frontendx/packages/ui/src/tokens.css`. The spacing scale uses 4, 8, 12, 16, 20, 24, and 32 px. Size tokens describe the purpose of a control so changes to forms can be made in one place.

The staff theme sets Geist typography and compact neutral surfaces at the document root. Dialogs and popovers inherit the same theme as the staff page. Button colors, interaction states, and field borders come from the shared component styles.

| Pattern | Spacing | Size |
| --- | --- | --- |
| Field | 8 px between label, control, helper, and error | Label 14/20 px, control text and placeholder 16/24 px, helper and error 13/20 px |
| Field focus | Show the same blue focus ring for pointer and keyboard focus. Keep it visible while a picker is open. | Use a 3 px outline with a 4 px offset. Grouped controls show the ring around the whole control. Invalid fields retain their error border. |
| Standard input, select, date, time, and grouped email | 14 px content inset. Reserve room for a select chevron at the trailing edge. | 48 px minimum height, 10 px corner radius |
| Compact staff fields | Apply `form-density-compact` at the form boundary. Uses 12 px content insets and 12 px desktop grid gaps. | 40 px minimum height and 14/20 px text on desktop. Phone and touch layouts use at least 44 px height and 16/24 px text. |
| Form grid and sections | 20 px between fields on desktop, 16 px on narrow screens, 24 px between groups | Section titles 16/24 px. Use `FormSection` for a fieldset and its legend. |
| Upload and visitor group | 8 px within each group, 24 px between groups. Align the remove action with its input. | Use `UploadField`, `form-stack`, and `form-roster-row`. |
| Buttons and action groups | 8 px icon gap, 12 px between actions. Default button inline inset is 20 px and falls to 16 px on narrow screens. | Shared actions have a 44 px minimum height. |
| Icon and contextual actions | Use Button with `size="icon"` and an accessible name for icon-only actions. Use the action-card, menu-option, sort-label, row-link, and inline-link variants for the corresponding patterns. | Icon buttons are 44 × 44 px. Inline actions inside a sentence follow the surrounding text. |
| Segmented control | Use `SegmentedControl` for visible, mutually exclusive choices such as language and quick date ranges. | Reuses the secondary Button for the selected choice and the ghost Button for other choices. Each choice has a 44 px minimum hit area. |
| Single-choice menus | 4 px menu inset, 4 px vertical and 8 px horizontal option insets | 32 px minimum option height. Use `Select` for compatible choices. |
| Search and multi-select | Use `SearchField` and `SearchableMultiSelect` for their respective controls. | Both share the field height, border, and radius tokens. |
| Standard tables | 12 px block and 16 px inline body-cell padding. Narrow tables use a 12 px inline inset. | Use `TableRegion` for scrolling tables to provide an accessible name and keyboard scrolling. Preserve numeric and icon alignment. |
| Panels and dialogs | Use 24 px for content groups. Shared panel headings use 24 px block padding and 20 px insets on narrow screens. | Keep compact surfaces in the existing component variants. |
| Status and visit type badges | Use `StatusBadge`, `VisitKindBadge`, or `Badge` for their respective meanings. | Badge typography, spacing, and color treatments live in the shared stylesheet. |

Use `FieldMessage` for helper text and validation errors. Put it inside its field after the control and connect its ID with `aria-describedby`. Keep a visible label even when a control has a placeholder. When a density variant is needed, set its gap or size at the component boundary using the shared scale.

Consolidated 2 November 2026, WITA. Existing OIKN identity, approved photo/logo, color tokens and typography above remain. Runtime tokens live in packages/ui/src/tokens.css. Superseded chronological design notes are preserved under anti-slop/history/2026-11-02.

Visitor UI is photo-first on phones with the existing choice card. Registration/status expand in place with task Back/logo/language controls, background isolation and preserved same-tab inputs after Back. Registration is mounted on first intent, then retained. One active main heading and logical subheadings follow the current inline/expanded/standalone context. Code entry stays prominent; email stays usable and secondary.

Staff navigation and actions follow one server role and permissions. Phone staff pages keep readable operational content; employee directory uses cards below the desktop breakpoint and a table above it. Text wraps without document overflow. Search, sort, page and selected detail live in the URL; form drafts stay local. Loading, error/retry, denied, empty, no-match/reset and stale data have explicit feedback. Unknown counts are not zero.

Use existing Button, Modal, field errors, RequiredMark/RequiredFieldsHint, QueryState/Empty, DatePicker/TimePicker and tokens. Shared fields retain native required semantics. Operational forms, identity data and tables remain opaque. Gold is reserved for institutional/VIP meaning; ordinary actions use the shared blue palette. Shared table overflow regions are named and keyboard reachable.

Employee, role and integration editors share ManagementDialog. Unsaved edits require explicit discard; 409 offers explicit version reload. Busy forms disable their controls. Account role/status self-edits are guarded; server enforcement remains required. Initial passwords are masked and write-only, with service-supplied minimum and required-change handling. Secret integration fields start blank; configured markers never expose values, blank means keep and removal is explicit. Existing unavailable assignments are retained until deliberately removed.

Public success is a receipt, not an admission permit. Only Approved eligibility mounts the admission feature. Production QR encodes the opaque server payload; expiry removes the image and replacement requests a new credential. Preview wording always identifies demo QR. Public history uses localized actions and WITA time without raw actor/internal text. Staff-written public reasons stay verbatim.

Supporting-document inputs validate the approved 5 MB PDF/JPEG/PNG limit. Production sends the file before the idempotent booking payload; receipt metadata does not claim a fake upload. Download appears only from server-provided availability, with permission failures explained. Ordinary originals are checked at reception; no ID number/photo collection is introduced. Terbatas keeps only its four optional identity fields and privacy projections.

Motion remains outside operational decisions. Native scrolling raises the landing card; the photo uses a small passive-scroll/frame loop, scoped to eligible phones and disabled on reduced motion. Keyboard focus must keep navigation discoverable. Calendar, QR/scanner and feature modules load at their boundaries. No GSAP, new visual framework, worker or visualization library is required.

Retain labeled controls, visible focus, suitable touch areas, logical dialogs, ID/EN text and WITA scheduling. Automated axe results are evidence for tested states, not full WCAG certification. Actual zoom, screen readers, Safari and physical scanners require acceptance in the deployment environment.

Current source/evidence is mapped in REQUIREMENTS-MATRIX.md and VERIFICATION.md. G14 draft recovery is deferred. G20 uses an explicit destructive cancellation form with a required internal audit reason; permission/status gating, retained conflict input and deliberate retry/review are part of acceptance. Live services and official privacy policy remain release gates.
