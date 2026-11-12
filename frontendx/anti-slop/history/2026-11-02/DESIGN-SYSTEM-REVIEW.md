# VMS design system — review sheet

**Status:** Updated implementation for review · 26 October 2026  
**Scope:** Current `frontendx` visitor and staff interfaces. Shared badges and filled action buttons now reflect the changes below.

Open [the visual review sheet](./design-system-review.html) to inspect the colors, components, states, spacing, and corner shapes together.

## Design direction

The VMS interface pairs neutral white and cool gray surfaces with OIKN navy, blue, and a limited gold accent. Operational actions use blue; gold marks VIP and selective brand emphasis. The visitor side is roomier and uses Plus Jakarta Sans for headings. The staff side uses Geist throughout, with compact data density.

## Color foundations

| Role | Visitor | Staff | Use |
| --- | --- | --- | --- |
| Canvas | `#f5f5f7` | `#f8fafc` | App background |
| Surface | `#ffffff` | `#ffffff` | Cards, fields, dialogs |
| Text | `#1d1d1f` | `#11131b` | Primary content |
| Muted text | `#63636c` | `#596273` | Secondary content |
| Divider | `#e5e5ea` | `#e4e4e7` | Rules, table rows |
| Subtle surface | `#f0f0f3` | `#f3f5f8` | Secondary groups |
| Brand navy | `#0a2342` | Shared | Institutional accent |
| Brand blue | `#185088` | Shared | Navigation, links, details |
| Brand gold | `#dbaf6c` | Shared | VIP and selected brand moments |
| Action blue | `#1769c2` | Shared | Primary actions and focus |
| Action danger | `#c43b32` | Shared | Destructive actions |

Primary action uses solid `#1769c2`, a quiet inner highlight, and a short soft shadow; hover is `#145dae` and pressed is `#105198`. Final destructive action uses solid `#c43b32`, hover `#b0332b`, pressed `#9d2d26`. An initial Reject action uses a warm white `#fff8f7` fill, dark red `#a32e29` text and a `#d99c97` inset outline. Secondary action: surface `#edf3fa`, label `#1769c2`, edge `#b5c6da`, hover `#e0ebf7`, pressed `#d4e3f3`. Disabled: surface `#e5e5ea`, label `#63636c`.

## Typography

| Style | Visitor | Staff |
| --- | --- | --- |
| Font | System sans body; Plus Jakarta Sans Variable headings | Geist Variable throughout |
| Body | `15px / 1.5`; visitor form text uses `16px` | `15px / 1.5` base, compact tables around `13px` |
| Page heading | `clamp(32px, 3vw, 44.8px)`, weight `650`, line `1.16`, tracking `-0.04em` | `clamp(28.8px, 2.5vw, 35.2px)`, weight `690`, tracking `-0.025em` |
| Public hero | `clamp(43.2px, 3.8vw, 66.4px)`, weight `650`, line `1.14` | — |
| Section heading | `22.4px` shared `h2` base | Contextual compact overrides |
| Field label | `13.28px`, weight `600` | Shared |
| Button | `14.4px`, weight `600` | Shared |
| Status badge | `11.68px`, weight `600` | Shared |

The public hero uses a responsive phone style near `28–40px`; the staff page title stays compact for operational scanning. Heading fonts are self hosted with system fallbacks. Numbers such as booking codes and metrics use tabular figures where applied.

## Spacing and size

| Purpose | Current value |
| --- | --- |
| Label to field | `7px` |
| Badge content gap | `5px` |
| Button content gap | `9px` |
| Toolbar gap | `12px` |
| Form group gap | `20px` |
| Content inset | `24px` common baseline; panel heading `23px 26px` |
| Section separation | `28px` |
| Main button | Minimum `44px` high, `11px 19px` padding |
| Input | Minimum `48px` high, `11px 14px` padding |
| Staff table cell | `9px 13px` in the current compact table |

These are recurring values in the implementation, not a strict universal spacing scale. Some context-specific values remain in the pages.

## Corners, lines, and depth

| Element | Radius | Edge/depth |
| --- | --- | --- |
| Status badge | `7px` | No shared border; staff badge adds a fine white edge and short shadow |
| VIP mark | `7px` | Same glass finish as status, with gold tint |
| Field | `10px` | `1px #cbd0d4` visitor; staff field uses `#cbd2dc` |
| Button | `16px` | Filled actions have a quiet highlight; outlined Reject has a warm edge |
| Staff panel | `16px` | `1px #e4e4e7`, `0 1px 2px #11131b0a` |
| Visitor panel/dialog | `20px` | Soft layered shadow and inset hairline |
| Public hero / mobile choice card | `28px` | Contextual image/card edge |

Dividers are generally `1px` and neutral. Focus uses a `3px #1769c2` outline with `4px` offset in the shared styles. Staff sidebar toggle uses its own `2px` focus treatment.

## Buttons and fields

The button family is **Primary**, **Secondary**, **Ghost**, **Danger outline**, **Danger filled**, and **Disabled**. Buttons use the same 44px minimum target and 16px corner. Continue uses a solid blue fill. Reject opens the rejection flow with a red outline; submitting the rejection uses the solid red button. The staff secondary variant adds a restrained light material treatment with an opaque fallback. Text links use brand blue and a 44px minimum hit area.

Inputs use white fill, dark text, a 1px neutral border, 10px corner, and 48px minimum height. Labels sit above fields. Invalid controls receive a red border and an adjacent error message. Visible focus remains on keyboard interaction.

Query timeouts and retry now share one pale red panel. The retry control uses a matching warm outline inside that panel rather than a separate blue secondary button. The raw timeout message is shown in the current UI language; the action stays labeled **Coba lagi**.

## Status badges

| Status | Shared text | Solid fallback |
| --- | --- | --- |
| Pending | `#754b00` | `#fff0bf` |
| Approved | `#146537` | `#d6f8df` |
| Rejected | `#a32135` | `#ffe0e2` |
| Cancelled | `#4b5563` | `#e7eaf0` |
| Checked in | `#075985` | `#d8efff` |
| Checked out | `#5636a5` | `#e5ddff` |

The brighter staff treatment is now shared by visitor and staff: translucent tinted fills, a subtle sheen, and a short shadow. **VIP** uses the same 7px corner, padding, sheen, and blur with its gold tint (`#f1e4c9`, text `#775020`). Reduced-transparency, increased-contrast, and unsupported-blur modes use solid fills. Status text must stay visible: color alone does not convey state. **Rescheduled** is an annotation on Approved.

## Review decisions

1. Confirm the visitor/staff typography pairing: Plus Jakarta Sans for public headings and Geist for staff.
2. Review the shared brighter status and gold VIP badges in both contexts.
3. Review the calmer Continue and outlined Reject pair, including their hover, pressed, and disabled states.
4. Confirm whether the `28px` public card and `16px` staff panel corners feel right together.

**Source of truth:** `packages/ui/src/tokens.css`, `packages/ui/src/styles.css`, `apps/admin/src/styles.css`, `apps/visitor/src/styles.css`, and `DESIGN.md` as read on 26 October 2026. The review sheet is a visual reference, not a production component library.
