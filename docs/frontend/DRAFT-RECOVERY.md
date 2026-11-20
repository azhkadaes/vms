# Visitor booking draft recovery plan

31 October G14: the user explicitly chose to keep this feature planned for a later task. No draft/contact/roster/document storage is enabled by the current audit fixes. The implemented uncertain-submission reference recovery remains a separate concern.

30 October distinction: G07 now retains only a pending random submission reference in sessionStorage for safe outcome resolution, with the original email supplied again after reload. This does **not** implement the optional draft storage/30-minute recovery plan below; no draft, contact, roster or document is saved. If this draft feature is later selected, coordinate its submission reference with the existing controller rather than creating a second attempt owner.

21 October 2026. User selected optional same-tab recovery with a 30-minute inactivity expiry. This document is the implementation plan; this staff-first pass does not enable visitor-data storage.

## Recommended behavior

The public VMS has named groups, contact details and free-text visit purposes, and may be used on shared phones. Keep recovery optional, tab-scoped and short-lived. Do not require an account or change the booking/status verification flow.

1. The form works without recovery. Offer an unchecked control: **Simpan draft di tab ini selama 30 menit**. Explain: **Opsional. Hindari menyimpan di perangkat bersama. Dokumen tidak ikut disimpan.**
2. On opt-in, save the permitted fields after edits settle (500 ms debounce). Display **Draft tersimpan di tab ini** only after storage succeeds. Keep unsaved input usable when storage is blocked or full.
3. On reload/re-entry, offer **Lanjutkan draft** and **Hapus draft** before populating the form. Show only its last-saved time, not names or contact details in the invitation. Never silently restore a draft.
4. Restore the first incomplete valid step. Validate its current destination, WITA date/time and roster again. A past schedule returns to step one with an explanation. Privacy acknowledgment always resets to unchecked.
5. Require ND reselection and say so on recovery. Do not imply any document is retained or submitted. Review still requires an explicit final submission.
6. Clear on successful confirmed submission, **Hapus draft**, opt-out or expiry. Keep the draft on validation/network failure. Never clear merely because the submit request was sent.
7. If recovery is off or storage fails, protect a populated form with an in-app departure warning and best-effort browser unload warning. Browser/OS shutdown warnings are not guaranteed on mobile.

## Data and lifecycle

- One versioned sessionStorage entry: `vms.visitor.bookingDraft.v1`. No localStorage, cookies, IndexedDB or server draft endpoint in phase one. Same-tab scope means no promised cross-device or cross-tab recovery. Some browsers can copy/restore tab sessions; do not promise that closing a tab reliably erases data.
- Allowlist: officerId, date, time, purpose, representative name/organization/email/phone, identity type and named roster. Do not serialize the full form object. The representative's sensitive free-text/contact/roster data is saved only after opt-in.
- Exclude identity numbers/images, ND file/blob/name/path, booking/status verification codes, OTPs, authentication/session credentials, privacy acknowledgment, API responses and form error state.
- Envelope: schemaVersion, savedAt, expiresAt, permitted values, last valid step and create-request idempotency key. The idempotency key prevents a recovered retry from creating another booking after an ambiguous network response; it grants no authority. Its reuse needs a backend-supported idempotency contract before production.
- Thirty minutes is measured from the last meaningful form edit. Route navigation, storage reads, restore prompts and background polling do not extend it. Timestamp expiry is checked on load, visibility return, before restore and before save. An open-page timer can clear an expired entry. Expiry must not erase the visitor's currently visible unsent input.
- A suspended/closed browser cannot run deletion code: expired entries are rejected and removed at the next opportunity. This is a recovery limit, not a guarantee of physical deletion at exactly 30 minutes.
- Keep data out of logs, URLs, analytics and crash payloads. Session storage is readable by scripts on the same origin; it is not encrypted storage or an authorization boundary.

## Implementation boundaries

Use one small draft-storage adapter with parsing/allowlist/expiry checks, separate from RHF. A visitor-local hook owns the opt-in, save status, restore offer and expiry lifecycle. RHF remains the only owner of active form values. Keep route protection in the visitor shell so home/status links and browser navigation cannot silently drop populated work. File objects remain in memory only.

No new dependencies or worker are needed for a roster capped at 20 in this preview. Save only after edits, never on every render. Do not upload drafts or alter Go contracts in this frontend phase.

## Acceptance criteria

- Opt-out creates no saved personal-data entry; normal booking still works.
- Opt-in, edit, reload, explicit resume restores permitted values in the same tab.
- Expired, corrupt, wrong-version or invalid entries are removed safely; no input or verification state is silently populated.
- ND must be reselected, acknowledgment is unchecked, and stale schedules require correction.
- Failed submission preserves recovery; a successful response clears it. Ambiguous retries use the same idempotency key.
- Storage denial/quota failure produces an accurate unsaved warning and leaves the form usable.
- Keyboard users can reach and dismiss the restore choice; focus goes to the restored step or start action. Mobile supports at least 320px width and 16px inputs.
- Use fake data for browser checks; test allowlisting/expiry/corrupt input/idempotency handling in focused logic tests. Do not claim GDPR/PDP compliance from this mechanism.
