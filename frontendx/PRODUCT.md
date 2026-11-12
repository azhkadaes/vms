# Visitor Management System product contract

Consolidated 2 November 2026, WITA. This document and BUSINESS-REQUIREMENTS.md describe D:/oikn/VMS/frontendx. The future backend at D:/oikn/VMS/backend is unbuilt.

Visitors request and review a visit, then use the matching registration email and eight-digit visit code for status, eligible admission QR and cancellation. No extra OTP or email-proof link is required. A pending request is not an entry permit. Visitor wording uses visit/kunjungan; staff scheduling uses appointment/janji temu. Technical /booking and bookingCode contracts retain their established meaning.

Staff have one server-assigned active role. PA decisions follow assigned non-VIP officers; Protocol manages VIP and Terbatas within its authority; Reception handles eligible arrivals/departures; Super Admin manages accounts/access/integrations and authorized summaries, and can read Terbatas without gaining decision rights. The server authorizes each read/action and projects private data before search/count/export.

Ordinary registration collects destination, WITA start, purpose, representative, organization, contact, identity type and named roster including the representative. Initial capacity is 20 people. An optional Nota Dinas atau Permohonan Kunjungan Resmi accepts PDF/JPEG/PNG up to 5 MB. Identity originals are checked at reception; ordinary ID numbers/photos are not collected. Staff sets duration/end before approval. Existing scheduling validation uses 08.00–16.00 WITA start times.

Pending requests may overlap. Approval/rescheduling reserves the same officer or occupied room transactionally; adjacent non-overlapping times are allowed. Public rejection text is separate from internal notes. Authorized rescheduling fixes the new time/place/reason immediately and invalidates old QR credentials. Visitor cancellation is strictly before start. G20 permits staff with explicit cancel permission and existing decision authority to cancel Pending/Approved visits, including after start, only before check-in and with a mandatory audit reason. The public history exposes the status change, not that internal reason.

Check-in opens 30 minutes before start through scheduled end, using server time. Approved visits without check-in become derived Missed after end; Missed is not a seventh stored status. Checked-in visitors can check out later. Both manual code and QR require reception identity verification and the same attendance rules.

Terbatas requires only date/time and exposes four optional fields: Nama, Instansi, No ID Dinas, target officer. Its current privacy matrix is in KUNJUNGAN-TERBATAS.md. No public discovery or general export of Terbatas records/events is allowed.

Administration includes employee create/edit/deactivate/assign, one role per account, administrator-set initial password and backend-required password change; versioned role permissions; server-defined integration configuration with write-only secrets. No real account or integration changes occur in the local preview.

Reports use an independent aggregate contract and summary-only CSV. Date/status filters and server scope determine counts; missing/failed/denied data does not become zero. Read and export rights are separate. The UI defaults to the current WITA month with a bounded 366-day query; this is a query limit, not data-retention policy.

Phone visitors keep the existing photo, choice card, expanded task, Back and responsive language controls. Registration mounts on first intent, then keeps input on Back. Staff uses phone cards or readable tables as appropriate. Headings, labels, keyboard focus, reduced motion and loading/error/empty/denied feedback are part of acceptance.

Current evidence is in VERIFICATION.md; backend obligations are in API-CONTRACT.md. G14 draft persistence is deferred. Official privacy/retention policy, real service integration and operational/device acceptance remain release gates.
