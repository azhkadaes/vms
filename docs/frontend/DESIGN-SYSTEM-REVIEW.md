# Current design-system assessment

Updated 2 November 2026, WITA. Scope: frontendx only. The prior chronological review is preserved under anti-slop/history/2026-11-02.

The project keeps React/Vite, plain scoped CSS, shared institutional tokens, local fonts, Radix dialogs/popovers and the existing adapted calendar/time controls. Native accessible fields/buttons handle ordinary work; form schemas and fetched data are separate from local UI state. A framework migration is unnecessary for the requested flows.

Management pages now share conflict/discard behavior; booking detail is a lazy feature; data code separates session, HTTP, adapters and schemas. Public/staff/restricted responses remain separate. URL-owned filters/pages and query-owned server data avoid duplicated state. Large multistep form modules remain explicit feature owners; extracting generic forms merely to reduce line count would obscure their different privacy/authorization rules.

Production lists use bounded server pages and dashboard metrics. Small pages do not require virtualization or workers. Phone registration is deferred to first intent. Native photo motion replaces the GSAP runtime, and the existing hero is preloaded. The original photo remains about 517 KB; real device/network performance is a deployment measurement gate.

The design preserves OIKN identity, existing colors, responsive task navigation, opaque operational surfaces and restrained transitions. Shared names/headings/scroll regions/focus, employee cards and full list states were checked in the scoped audits. Automated accessibility results and screenshot widths are in VERIFICATION.md; manual device/assistive-technology acceptance remains explicit.

BUSINESS-REQUIREMENTS.md and DESIGN.md are current acceptance references. G14 remains deferred. G20 staff cancellation uses the existing decision feature and shared permission/validation/mutation boundaries. No claim of live production authorization, physical scanning or complete WCAG compliance is made by this review.
