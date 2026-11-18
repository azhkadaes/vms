# Frontend data traceability

Every application feature/page and shared data adapter was inspected. `confirmed requirement` refers to source or retained decisions; `current` describes actual consumer code, never a claim that the backend exists. All source-registered versus missing backend status comes from [manifest](manifest.json). Proposed interfaces are explicitly marked in [INTERFACES](INTERFACES.md). Preview fixtures and local UI preferences are not server persistence.

| ID / source / component | Route | UI element / action / required data | Existing integration | Endpoint operation IDs | Evidence / gaps |
| --- | --- | --- | --- | --- | --- |
| [F01](#f01); [frontendx/apps/visitor/src/pages/Booking.tsx](../../frontendx/apps/visitor/src/pages/Booking.tsx); `BookingForm` | /; /?view=registration; /booking | Registration destination/official dropdown in the first booking step; Open destination step; Active officials, VIP classification, privacy notice | useCatalogs -> repository.catalogs; preview uses domain fixtures | publicCatalogs | confirmed requirement; inspected source; FE-CATALOG |
| [F02](#f02); [frontendx/apps/visitor/src/pages/Booking.tsx](../../frontendx/apps/visitor/src/pages/Booking.tsx); `BookingForm` | /; /booking | Three-step booking form, companion-name rows, optional supporting-document picker and submit/receipt panel; Submit three-step ordinary request; Schedule, representative/contact, purpose, identity type, 0-19 companions converted to 1-20 names, ack, optional file token | BookingSubmission; actual create adapter exists; no backend booking route | publicCreateBooking, publicUpload | confirmed requirement; inspected source; BE-VISITS, FE-BOUNDS, FE-UPLOAD |
| [F03](#f03); [frontendx/apps/visitor/src/features/PrivacyNotice.tsx](../../frontendx/apps/visitor/src/features/PrivacyNotice.tsx); `PrivacyNotice` | /; /booking; /status | Bilingual privacy notice dialog, consent text and retry button; Read bilingual notice / retry; version and Indonesian/English notice text | Shares cached useCatalogs; real form blocks without published notice | publicCatalogs | confirmed requirement; inspected source; FE-CATALOG |
| [F04](#f04); [frontendx/apps/visitor/src/features/SubmissionRecovery.tsx](../../frontendx/apps/visitor/src/features/SubmissionRecovery.tsx); `SubmissionRecovery` | /; /booking | Uncertain-submission recovery panel and email recovery action; Resolve uncertain create by email; Retained UUID reference; committed booking or closed not_committed | BookingForm -> BookingSubmission.recover; only reference in sessionStorage | resolveSubmission | confirmed requirement; inspected source; BE-VISITS |
| [F05](#f05); [frontendx/apps/visitor/src/pages/Status.tsx](../../frontendx/apps/visitor/src/pages/Status.tsx); `Status` | /status; /?view=status | Status lookup form: eight-digit code input, email input, View visit and Update buttons; Lookup / refresh; Eight-digit code plus email -> complete public booking/history/timing | repository.lookup; invalid pair errors preserve staff auth | publicLookup | confirmed requirement; inspected source; BE-VISITS |
| [F06](#f06); [frontendx/apps/visitor/src/pages/Status.tsx](../../frontendx/apps/visitor/src/pages/Status.tsx); `Status` | /status; /?view=status | Cancel visit button and confirmation dialog on the status card; Confirm cancellation; code, email, expected version; before-start eligibility | repository.cancel; conflict attempts a fresh lookup | publicCancel, publicLookup | confirmed requirement; inspected source; BE-VISITS |
| [F07](#f07); [frontendx/apps/visitor/src/features/AdmissionQR.tsx](../../frontendx/apps/visitor/src/features/AdmissionQR.tsx); `AdmissionQR` | /status; booking receipt | Admission QR image, expiry/countdown and Renew QR code/Try again buttons; Issue / replace / renew QR; Opaque payload, expiry, serverTime; visit credentials/version | repository.admission; preview constructs VMS-DEMO credential locally | publicAdmission | confirmed requirement; inspected source; BE-ATTENDANCE |
| [F08](#f08); [frontendx/apps/visitor/src/features/VisitTicket.tsx](../../frontendx/apps/visitor/src/features/VisitTicket.tsx); `VisitTicket` | /status; booking receipt | Booking receipt / printable VisitTicket card; View / print ticket; Returned public schedule, code, roster and QR | Local rendering/print; shares existing booking; no ticket endpoint | publicLookup, publicCreateBooking, publicAdmission | confirmed requirement; inspected source; BE-VISITS |
| [F09](#f09); [frontendx/apps/visitor/src/features/PublicVisitHistory.tsx](../../frontendx/apps/visitor/src/features/PublicVisitHistory.tsx); `PublicVisitHistory` | /status | Public visit-history timeline within the status view; Read public event history; Allowlisted event/action/time/public reschedule message | Renders history already in booking; no history endpoint | publicLookup, publicCancel | confirmed requirement; inspected source; BE-VISITS |
| [F10](#f10); [frontendx/apps/visitor/src/features/useCancellationWindow.ts](../../frontendx/apps/visitor/src/features/useCancellationWindow.ts); `useCancellationWindow` | /status | Time-based enabled/disabled state of the public cancellation button; Update cancel availability as time elapses; Server timing and local monotonic elapsed time | Local conservative boundary timer; mutation still authoritative | publicLookup, publicCancel | confirmed requirement; inspected source; BE-VISITS |
| [F11](#f11); [frontendx/apps/admin/src/main.tsx](../../frontendx/apps/admin/src/main.tsx); `Login` | /admin/login | Staff login email/password form and sign-in button; Sign in; Email/password -> TokenResult and refresh/CSRF cookies | Existing signIn still calls /staff/sessions and discards its response; new login/AuthResult consumer and Bearer transport are missing. | staffLogin | confirmed requirement; inspected source; FE-AUTH, BE-AUTH-TESTS, BE-PHASE1-AUTH |
| [F12](#f12); [frontendx/packages/data/src/session.tsx](../../frontendx/packages/data/src/session.tsx); `SessionProvider` | /admin/* | Workspace current-user load, fixed token expiry and local Sign out action; Load / recheck session / sign out; Current Session; no private cache reuse after authority loss | Existing provider uses GET/DELETE /staff/session with cookie assumptions; those calls are superseded by Phase 1 me plus local logout. | staffMe | confirmed requirement; inspected source; FE-AUTH, BE-PHASE1-AUTH |
| [F13](#f13); [frontendx/apps/admin/src/features/RoleControls.tsx](../../frontendx/apps/admin/src/features/RoleControls.tsx); `RoleSelection / RoleSwitcher` | /admin/* | Required-role selection screen and active-role switcher; Select required role / switch; Available assigned roles; confirmed role/revision | switchRole parses TokenResult as bare Session; reload clears cache |  | confirmed requirement; inspected source;  |
| [F14](#f14); [frontendx/apps/admin/src/features/PasswordChange.tsx](../../frontendx/apps/admin/src/features/PasswordChange.tsx); `PasswordChange` | /admin/* forced gate | Required password-change form: current password, new password and save button; Replace initial password; currentPassword, newPassword; policy min 15 code points / max 72 bytes | changePassword discards TokenResult; input maxLength 256 |  | confirmed requirement; inspected source;  |
| [F15](#f15); [frontendx/packages/data/src/session.tsx](../../frontendx/packages/data/src/session.tsx); `SessionProvider` | /admin/* | Workspace activity/expiry handling and access-token restoration after reload; consumer still missing; Continue actual activity / refresh access after reload; Authoritative idle/absolute timing, memory token and rotating refresh | No renewal, refresh, multi-tab rotation or expiry-warning consumer exists |  | confirmed earlier decisions D20-D25/D38-D54; missing implementation;  |
| [F16](#f16); [frontendx/apps/admin/src/pages/Bookings.tsx](../../frontendx/apps/admin/src/pages/Bookings.tsx); `Bookings` | /admin/booking; /admin/today | Janji temu / Kunjungan hari ini search box, status tabs, VIP filter, sort selector, cards/table and pagination; Search/filter/sort/page ordinary and restricted records; view=list, q/status/sort/dir/date/vip/page/pageSize=7 | Production readStaffBookings; preview full list then local selectBookings | staffBookings | confirmed requirement; inspected source; BE-READS, FE-CORRELATION |
| [F17](#f17); [frontendx/apps/admin/src/pages/Bookings.tsx](../../frontendx/apps/admin/src/pages/Bookings.tsx); `Bookings / BookingDetail` | /admin/booking?selected={id}; /admin/today?selected={id} | Selected booking-detail dialog opened from a row/card or selected URL parameter; Open detail independent of page; Authorized record and field projection for selected ID | useBookingDetail -> readStaffBooking; does not verify returned ID | staffBooking | confirmed requirement; inspected source; BE-READS, FE-CORRELATION |
| [F18](#f18); [frontendx/apps/admin/src/pages/Dashboard.tsx](../../frontendx/apps/admin/src/pages/Dashboard.tsx); `Dashboard` | /admin/dashboard | Dashboard metric cards, Agenda hari ini / Menunggu kedatangan table and Perlu perhatian / Sedang berkunjung list; Read metrics, agenda and attention; view=dashboard/date/page=1/pageSize=50; full-scope metrics plus bounded union | Shares booking read; Super Admin skips operational query | staffBookings | confirmed requirement; inspected source; BE-READS |
| [F19](#f19); [frontendx/apps/admin/src/components/BookingRefresh.tsx](../../frontendx/apps/admin/src/components/BookingRefresh.tsx); `ProductionRefresh` | /admin/dashboard; /admin/today; /admin/booking; /admin/check-in; /admin/check-out | Visible-page refresh timer, focus refresh and manual refresh for lists/details/reception; Visible-only timed/manual/focus refresh; Active scoped booking queries; 30 sec staff / 15 sec reception | Refetches active queries; no new endpoint; avoid auth renewal on polling | staffBookings, staffBooking | confirmed requirement; inspected source; BE-READS |
| [F20](#f20); [frontendx/apps/admin/src/components/DataRefresh.tsx](../../frontendx/apps/admin/src/components/DataRefresh.tsx); `DataRefresh` | /admin/* | DataRefresh freshness/offline/error notice and retry controls; Retry / show freshness and offline state; Existing query fetch status/error/update time | Pure query control; no persisted business data | staffBookings, staffEmployees, staffIntegrations | confirmed requirement; inspected source; BE-READS |
| [F21](#f21); [frontendx/apps/admin/src/features/CreateAppointment.tsx](../../frontendx/apps/admin/src/features/CreateAppointment.tsx); `CreateAppointment` | /admin/booking?new=1; /admin/today?new=1 | Staff standard-appointment dialog: official dropdown, schedule, representative/roster, note fields, document picker and submit; Open staff standard form / submit; Authorized active official catalog; ordinary fields, roster, notes, optional document | useCreateAppointment; UUID retained while payload unchanged; room/duration not collected | staffCatalogs, staffCreateBooking, staffUpload | confirmed requirement; inspected source; BE-VISITS, FE-UPLOAD |
| [F22](#f22); [frontendx/apps/admin/src/features/RestrictedAppointment.tsx](../../frontendx/apps/admin/src/features/RestrictedAppointment.tsx); `RestrictedAppointment` | /admin/booking?new=1&kind=restricted | Kunjungan Terbatas creation dialog: date/time, optional identity/target fields, notes and create button; Create restricted visit; date/time required; optional name/organization/serviceId/officerId and notes | Protocol create gate; no contact/identity number/roster/file required | staffCatalogs, staffCreateBooking | confirmed requirement; inspected source; BE-VISITS |
| [F23](#f23); [frontendx/apps/admin/src/features/RestrictedAppointment.tsx](../../frontendx/apps/admin/src/features/RestrictedAppointment.tsx); `RestrictedAppointment` | /admin/booking?selected={id} | Restricted visit edit dialog and Simpan perubahan button; Edit restricted identity/target/notes; Version; date/time; empty strings clear optional identity/target | useBookingAction(update); date/time inputs readOnly on existing record | staffRestrictedUpdate, staffCatalogs | confirmed requirement; inspected source; BE-VISITS |
| [F24](#f24); [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); `BookingDetail` | /admin/booking; /admin/today | Approve dialog: room dropdown, duration input, note fields and confirmation; Approve pending visit; Version, selected room, duration and public/private notes | Form sends room label and duration; target requires roomId | staffApprove, staffCatalogs | confirmed requirement; inspected source; BE-VISITS, FE-ROOM |
| [F25](#f25); [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); `BookingDetail` | /admin/booking; /admin/today | Reject dialog: public rejection-reason textarea, private note textarea and reject button; Reject pending visit; Version; public reason 5-1000; private note <=2000 | rejectionSchema -> useBookingAction; real backend absent | staffReject | confirmed requirement; inspected source; BE-VISITS |
| [F26](#f26); [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); `BookingDetail` | /admin/booking; /admin/today | Reschedule dialog: new date/time, room dropdown, duration, reason and notes; Reschedule; Version, future date/time, duration, room, reason, notes | room label sent; domain.can permits Approved only while D100 also selects Pending | staffReschedule, staffCatalogs | confirmed requirement; inspected source; BE-VISITS, FE-ROOM, FE-PENDING |
| [F27](#f27); [frontendx/apps/admin/src/features/BookingDetail.tsx](../../frontendx/apps/admin/src/features/BookingDetail.tsx); `BookingDetail` | /admin/booking; /admin/today | Staff cancellation dialog: public reason, internal cancellation note and confirm button; Staff cancel / read private notes; Version, reason, internalCancellationNote; decision/field scope | useBookingAction(cancel); separate from public pair authorization | staffCancel | confirmed requirement; inspected source; BE-VISITS |
| [F28](#f28); [frontendx/apps/admin/src/features/VisitNotesFields.tsx](../../frontendx/apps/admin/src/features/VisitNotesFields.tsx); `VisitNotesFields` | staff create / approve / reschedule / restricted edit dialogs | Shared Instruksi pengunjung and Catatan koordinasi internal textareas; Edit public instructions / private coordination; Separate optional 2000-code-point values; omit preserves; empty clears | Shared form fields; notes travel with existing command | staffCreateBooking, staffApprove, staffReschedule, staffRestrictedUpdate | confirmed requirement; inspected source; BE-VISITS, FE-BOUNDS |
| [F29](#f29); [frontendx/apps/admin/src/features/AttachmentAccess.tsx](../../frontendx/apps/admin/src/features/AttachmentAccess.tsx); `AttachmentAccess` | /admin/booking?selected={id} | Private supporting-document action in BookingDetail; existing action downloads only; Read attached private file; Authorized bytes, MIME, safe filename; view and download | downloadAttachment only; request lacks explicit disposition=attachment | staffAttachment | confirmed requirement; inspected source; BE-DOCUMENTS, FE-FILEVIEW |
| [F30](#f30); [frontendx/apps/admin/src/pages/Reception.tsx](../../frontendx/apps/admin/src/pages/Reception.tsx); `Reception` | /admin/check-in; /admin/check-out | Reception scheduled-arrivals / still-inside table and previous/next page controls; Read arrival / still-inside page; view=reception/mode; check-in today; checkout all dates; pageSize=25 | useBookings typed view; one endpoint reused | staffBookings | confirmed requirement; inspected source; BE-READS |
| [F31](#f31); [frontendx/apps/admin/src/pages/Reception.tsx](../../frontendx/apps/admin/src/pages/Reception.tsx); `Reception` | /admin/check-in; /admin/check-out | Manual visit-code form or QR resolve result before attendance confirmation; Resolve manual code or QR before confirmation; credential -> current authorized standard/restricted arrival projection | repository.resolve; no attendance side effect; restricted result kept local | staffResolve | confirmed requirement; inspected source; BE-ATTENDANCE |
| [F32](#f32); [frontendx/apps/admin/src/features/Scanner.tsx](../../frontendx/apps/admin/src/features/Scanner.tsx); `Scanner` | /admin/check-in; /admin/check-out | Camera scanner preview and decoded-QR handler; Scan camera QR; BarcodeDetector raw string passed to Reception.resolve | Camera local only; no image upload/storage/API | staffResolve | confirmed requirement; inspected source; BE-ATTENDANCE |
| [F33](#f33); [frontendx/apps/admin/src/pages/Reception.tsx](../../frontendx/apps/admin/src/pages/Reception.tsx); `Reception` | /admin/check-in | Reception identity-confirmation checkbox and check-in confirmation button; Confirm identity then check in; Matching credential and expected version, server timing eligibility | useBookingAction(checkin); checkbox local; no identityVerified wire field | staffCheckin | confirmed requirement; inspected source; BE-ATTENDANCE |
| [F34](#f34); [frontendx/apps/admin/src/pages/Reception.tsx](../../frontendx/apps/admin/src/pages/Reception.tsx); `Reception` | /admin/check-out | Reception identity-confirmation checkbox and check-out confirmation button; Confirm identity then check out; Credential/version; checked_in may be from earlier date | useBookingAction(checkout); consumed nonrevoked QR usable per D96 | staffCheckout | confirmed requirement; inspected source; BE-ATTENDANCE |
| [F35](#f35); [frontendx/apps/admin/src/features/EmployeeDirectory.tsx](../../frontendx/apps/admin/src/features/EmployeeDirectory.tsx); `EmployeeDirectory` | /admin/employees | Read-only employee directory search, cards/table, pagination and detail; editing controls deferred; Search/page/reload employee directory; q/page/pageSize=25, filtered/totalAll, password metadata | management.employees; local preview array; no backend admin routes | staffEmployees, staffEmployee | confirmed requirement; inspected source; BE-ADMIN, FE-DIRECTORY |
| [F36](#f36); [frontendx/apps/admin/src/features/EmployeeEditor.tsx](../../frontendx/apps/admin/src/features/EmployeeEditor.tsx); `EmployeeEditor` | /admin/employees | Employee create/edit dialog: profile fields, active checkbox, roles, assignments and initial password on create; Create employee / edit / deactivate; roles, profile strings, active, assignments, initial password only on create, version on edit | management.saveEmployee; preserves unavailable assignments |  | confirmed requirement; inspected source;  |
| [F37](#f37); [frontendx/apps/admin/src/features/RoleAssignments.tsx](../../frontendx/apps/admin/src/features/RoleAssignments.tsx); `RoleAssignments` | /admin/employees editor | Multi-role tag/search selector within the employee editor; Select multiple employee roles; Assigned role labels from fixed role enum; tag/search state | Local selector; saved atomically with employee; no join-row endpoint |  | confirmed requirement; inspected source;  |
| [F38](#f38); [frontendx/apps/admin/src/features/EmployeeEditor.tsx](../../frontendx/apps/admin/src/features/EmployeeEditor.tsx); `EmployeeEditor` | /admin/employees editor | PA non-VIP official-assignment checkboxes and retained-unavailable-assignment notice; Choose assigned non-VIP official / retain inactive assignment; Authorized catalog plus retained IDs on employee | useCatalogs; no truncation; current validator lacks max100/unique IDs |  | confirmed requirement; inspected source;  |
| [F39](#f39); [frontendx/apps/admin/src/features/RoleManagement.tsx](../../frontendx/apps/admin/src/features/RoleManagement.tsx); `RoleManagement / RoleEditor` | /admin/roles | Peran operasional cards, Edit akses dialog, permission checkboxes and Simpan akses button; List/reload/save role permissions; Permission dictionary in each RoleDefinition; assignability/editable/version | management.roles/saveRole; preview roles use names as IDs |  | confirmed requirement; inspected source;  |
| [F40](#f40); [frontendx/apps/admin/src/features/IntegrationManagement.tsx](../../frontendx/apps/admin/src/features/IntegrationManagement.tsx); `IntegrationManagement / IntegrationEditor` | /admin/integration | Integration service cards, Konfigurasi dialog, enabled checkbox, dynamic settings fields and secret keep/clear controls; Read/configure provider; keep/clear secret; versioned save; Safe field descriptors, configured secret flags, health; SMTP six-key schema | Generic live form uses server metadata; preview endpoint/bucket/credential examples are not selected providers | staffIntegrations, patchIntegration | confirmed requirement; inspected source; BE-INTEGRATIONS |
| [F41](#f41); [frontendx/apps/admin/src/pages/Reports.tsx](../../frontendx/apps/admin/src/pages/Reports.tsx); `Reports` | /admin/analytics; /admin/logs | Analytics/logs date range and status filters, total cards and per-status summary table/meters; Read filtered summary / refresh; Inclusive from/to/status; seven counts, scope, revision, generatedAt, canExport | useVisitSummary validates filter echo/sum; both tabs share operations | reportSummary | confirmed requirement; inspected source; BE-REPORTS |
| [F42](#f42); [frontendx/apps/admin/src/pages/Reports.tsx](../../frontendx/apps/admin/src/pages/Reports.tsx); `Reports` | /admin/analytics; /admin/logs | Unduh ringkasan CSV button and changed-summary conflict recovery; Export displayed revision / recover conflict; Same filter/revision -> filename,csv,revision JSON | repository.exportSummary -> Blob download; conflict refetch; summary rows only | reportExport, reportSummary | confirmed requirement; inspected source; BE-REPORTS |
| [F43](#f43); [frontendx/apps/admin/src/pages/Settings.tsx](../../frontendx/apps/admin/src/pages/Settings.tsx); `Settings` | /admin/analytics; /admin/logs; /admin/employees; /admin/roles; /admin/integration | Settings shell mounting read-only employees, integrations and reports; roles/editor routes deferred; Route shared management modules; No additional data beyond mounted module | Lazy router; guard permissions in main.tsx | staffEmployees, staffIntegrations, reportSummary | confirmed requirement; inspected source; BE-ADMIN |
| [F44](#f44); [frontendx/apps/admin/src/main.tsx](../../frontendx/apps/admin/src/main.tsx); `Shell / Guard` | /admin/* | Staff name/role display, fixed-permission navigation and route guards; Render permitted nav, user name, role gates; Session permissions, assignments and required-password/role state | UI guard is presentation; backend must independently enforce every route | staffMe | confirmed requirement; inspected source; FE-AUTH, BE-PHASE1-AUTH |
| [F45](#f45); [frontendx/packages/data/src/attachments.ts](../../frontendx/packages/data/src/attachments.ts); `uploadAttachment` | public/staff submission forms | Shared file-picker upload/retry transport for public and staff forms; Upload/replay selected file; Multipart file+UUID reference; expiry and serverTime; token only returned | WeakMap reuses reference for same File/audience; optional upload still attempted without capability check | publicUpload, staffUpload | confirmed requirement; inspected source; FE-UPLOAD, BE-DOCUMENTS |
| [F46](#f46); [frontendx/packages/data/src/http.ts](../../frontendx/packages/data/src/http.ts); `createHttpClient` | all API consumers | Shared Bearer-only HTTP transport and field-error/rate-limit handling; Execute request / recover error; JSON/multipart/blob, credentials, error fields, Retry-After | Existing shared client sends no Bearer and still includes cookies; Phase 1 must not rely on them. Its field-error parser expects string[]; current backend string-valued errors are filtered out (BE-ERRORS). | staffLogin, staffMe, publicCreateBooking | confirmed requirement; inspected source; FE-AUTH, BE-ERRORS, BE-PHASE1-AUTH |
| [F47](#f47); [frontendx/packages/data/src/repositories.ts](../../frontendx/packages/data/src/repositories.ts); `createRepository` | all booking consumers | Shared production repository used by booking/status components; preview adapter is explicit; Share production and explicit development adapters; Public/staff paths; strict response schemas; role only in __preview | Production list() legacy array branch exists but useBookings uses readStaffBookings instead | staffBookings, publicLookup | confirmed requirement; inspected source; BE-READS |
| [F48](#f48); [frontendx/apps/visitor/src/main.tsx](../../frontendx/apps/visitor/src/main.tsx); `App` | /; /booking; /status | Visitor landing shell with embedded registration/status views; hero/info content is static; Open landing/registration/status; No separate landing statistics/content API | Static hero/info; embedded Booking/Status supply dynamic data | publicCatalogs, publicLookup | confirmed requirement; inspected source; FE-CATALOG |

## F01

`frontendx/apps/visitor/src/pages/Booking.tsx` — BookingForm. Open destination step. Active officials, VIP classification, privacy notice. useCatalogs -> repository.catalogs; preview uses domain fixtures. Phase: phase1. Phase 1 operations: publicCatalogs. Gaps: FE-CATALOG.

**UI element:** Registration destination/official dropdown in the first booking step

| Operation | Request / returned-data use |
| --- | --- |
| [publicCatalogs](API-CONTRACT.md#publiccatalogs) | officers[].id/name populate officerId options; officers[].vip selects VIP presentation/requirements; privacyNotice supplies consent content. |

## F02

`frontendx/apps/visitor/src/pages/Booking.tsx` — BookingForm. Submit three-step ordinary request. Schedule, representative/contact, purpose, identity type, 0-19 companions converted to 1-20 names, ack, optional file token. BookingSubmission; actual create adapter exists; no backend booking route. Phase: phase1. Phase 1 operations: publicCreateBooking, publicUpload. Gaps: BE-VISITS, FE-BOUNDS, FE-UPLOAD.

**UI element:** Three-step booking form, companion-name rows, optional supporting-document picker and submit/receipt panel

| Operation | Request / returned-data use |
| --- | --- |
| [publicCreateBooking](API-CONTRACT.md#publiccreatebooking) | Send normalized representative/contact/schedule/officerId/purpose/visitors/identityType/ack, UUID idempotencyKey and optional attachmentToken/ndName/privacyVersion; PublicBooking.code/status/schedule/history populate the receipt. |
| [publicUpload](API-CONTRACT.md#publicupload) | Send the selected file and UUID reference; retain UploadResult.token as attachmentToken and check expiresAt/serverTime. New uploads require attachmentsEnabled=true. |

## F03

`frontendx/apps/visitor/src/features/PrivacyNotice.tsx` — PrivacyNotice. Read bilingual notice / retry. version and Indonesian/English notice text. Shares cached useCatalogs; real form blocks without published notice. Phase: phase1. Phase 1 operations: publicCatalogs. Gaps: FE-CATALOG.

**UI element:** Bilingual privacy notice dialog, consent text and retry button

| Operation | Request / returned-data use |
| --- | --- |
| [publicCatalogs](API-CONTRACT.md#publiccatalogs) | privacyNotice.version/id/en supply the notice and its language; the form uses the published version/ack for consent. Missing notice blocks genuine collection. |

## F04

`frontendx/apps/visitor/src/features/SubmissionRecovery.tsx` — SubmissionRecovery. Resolve uncertain create by email. Retained UUID reference; committed booking or closed not_committed. BookingForm -> BookingSubmission.recover; only reference in sessionStorage. Phase: phase1. Phase 1 operations: resolveSubmission. Gaps: BE-VISITS.

**UI element:** Uncertain-submission recovery panel and email recovery action

| Operation | Request / returned-data use |
| --- | --- |
| [resolveSubmission](API-CONTRACT.md#resolvesubmission) | Send retained idempotencyKey plus email; outcome=committed restores booking/receipt, not_committed allows a new submission after durable closure. |

## F05

`frontendx/apps/visitor/src/pages/Status.tsx` — Status. Lookup / refresh. Eight-digit code plus email -> complete public booking/history/timing. repository.lookup; invalid pair errors preserve staff auth. Phase: phase1. Phase 1 operations: publicLookup. Gaps: BE-VISITS.

**UI element:** Status lookup form: eight-digit code input, email input, View visit and Update buttons

| Operation | Request / returned-data use |
| --- | --- |
| [publicLookup](API-CONTRACT.md#publiclookup) | Send code/email in POST body; returned status/timing, schedule, host, roster, instructions and public history populate the visit status card. |

## F06

`frontendx/apps/visitor/src/pages/Status.tsx` — Status. Confirm cancellation. code, email, expected version; before-start eligibility. repository.cancel; conflict attempts a fresh lookup. Phase: phase1. Phase 1 operations: publicCancel, publicLookup. Gaps: BE-VISITS.

**UI element:** Cancel visit button and confirmation dialog on the status card

| Operation | Request / returned-data use |
| --- | --- |
| [publicCancel](API-CONTRACT.md#publiccancel) | Send selected id plus code/email/version; returned cancelled PublicBooking replaces the card/history. |
| [publicLookup](API-CONTRACT.md#publiclookup) | After a version/state conflict, repeat code/email lookup and replace the displayed booking with current authority. |

## F07

`frontendx/apps/visitor/src/features/AdmissionQR.tsx` — AdmissionQR. Issue / replace / renew QR. Opaque payload, expiry, serverTime; visit credentials/version. repository.admission; preview constructs VMS-DEMO credential locally. Phase: phase1. Phase 1 operations: publicAdmission. Gaps: BE-ATTENDANCE.

**UI element:** Admission QR image, expiry/countdown and Renew QR code/Try again buttons

| Operation | Request / returned-data use |
| --- | --- |
| [publicAdmission](API-CONTRACT.md#publicadmission) | Send id plus code/email/version; encode payload locally as QR, use expiresAt/serverTime for countdown and replacement. |

## F08

`frontendx/apps/visitor/src/features/VisitTicket.tsx` — VisitTicket. View / print ticket. Returned public schedule, code, roster and QR. Local rendering/print; shares existing booking; no ticket endpoint. Phase: phase1. Phase 1 operations: publicLookup, publicCreateBooking, publicAdmission. Gaps: BE-VISITS.

**UI element:** Booking receipt / printable VisitTicket card

| Operation | Request / returned-data use |
| --- | --- |
| [publicLookup](API-CONTRACT.md#publiclookup) | Use existing PublicBooking.code/date/time/host/room/visitors; no separate ticket fetch. |
| [publicCreateBooking](API-CONTRACT.md#publiccreatebooking) | Use the successful create response directly for the receipt and ticket. |
| [publicAdmission](API-CONTRACT.md#publicadmission) | Render the returned admission payload locally as QR; printing is a local browser action. |

## F09

`frontendx/apps/visitor/src/features/PublicVisitHistory.tsx` — PublicVisitHistory. Read public event history. Allowlisted event/action/time/public reschedule message. Renders history already in booking; no history endpoint. Phase: phase1. Phase 1 operations: publicLookup, publicCancel. Gaps: BE-VISITS.

**UI element:** Public visit-history timeline within the status view

| Operation | Request / returned-data use |
| --- | --- |
| [publicLookup](API-CONTRACT.md#publiclookup) | history[] and public reasons/instructions populate the timeline; no private actors/notes. |
| [publicCancel](API-CONTRACT.md#publiccancel) | Use the returned booking.history and cancellation state to refresh the same timeline. |

## F10

`frontendx/apps/visitor/src/features/useCancellationWindow.ts` — useCancellationWindow. Update cancel availability as time elapses. Server timing and local monotonic elapsed time. Local conservative boundary timer; mutation still authoritative. Phase: phase1. Phase 1 operations: publicLookup, publicCancel. Gaps: BE-VISITS.

**UI element:** Time-based enabled/disabled state of the public cancellation button

| Operation | Request / returned-data use |
| --- | --- |
| [publicLookup](API-CONTRACT.md#publiclookup) | timing.serverTime/canCancel and schedule establish the local conservative cancellation countdown. |
| [publicCancel](API-CONTRACT.md#publiccancel) | Send the current version and pair; the server rechecks the time boundary even if the button was enabled. |

## F11

`frontendx/apps/admin/src/main.tsx` — Login. Sign in. Email/password -> TokenResult and refresh/CSRF cookies. Existing signIn still calls /staff/sessions and discards its response; new login/AuthResult consumer and Bearer transport are missing.. Phase: phase1. Phase 1 operations: staffLogin. Gaps: FE-AUTH, BE-AUTH-TESTS, BE-PHASE1-AUTH.

**UI element:** Staff login email/password form and sign-in button

| Operation | Request / returned-data use |
| --- | --- |
| [staffLogin](API-CONTRACT.md#stafflogin) | POST email/password to /staff/auth/login; keep AuthResult.accessToken/accessExpiresAt/user in memory. No auth cookies, CSRF initialization or session/role/password gate. |

## F12

`frontendx/packages/data/src/session.tsx` — SessionProvider. Load / recheck session / sign out. Current Session; no private cache reuse after authority loss. Existing provider uses GET/DELETE /staff/session with cookie assumptions; those calls are superseded by Phase 1 me plus local logout.. Phase: phase1. Phase 1 operations: staffMe. Gaps: FE-AUTH, BE-PHASE1-AUTH.

Local action: Sign out: discard in-memory token/user, clear private caches and navigate to login; no API.

Local action: Token expiry or reload: sign in again; reject stale in-flight responses after logout.

**UI element:** Workspace current-user load, fixed token expiry and local Sign out action

| Operation | Request / returned-data use |
| --- | --- |
| [staffMe](API-CONTRACT.md#staffme) | GET /staff/auth/me with Bearer; CurrentUser drives authenticated state. Local logout clears token/user/private caches without HTTP. Reload/expiry requires login; no refresh or idle renewal. |

## F13

`frontendx/apps/admin/src/features/RoleControls.tsx` — RoleSelection / RoleSwitcher. Select required role / switch. Available assigned roles; confirmed role/revision. switchRole parses TokenResult as bare Session; reload clears cache. Phase: later. Phase 1 operations: none; deferred UI. Gaps: .

Existing UI is deferred and must not be mounted/call legacy endpoints in Phase 1. Accounts, single roles and PA assignments are provisioned through controlled database setup; password/role/session lifecycle UI is deferred.

**UI element:** Required-role selection screen and active-role switcher

| Operation | Request / returned-data use |
| --- | --- |

## F14

`frontendx/apps/admin/src/features/PasswordChange.tsx` — PasswordChange. Replace initial password. currentPassword, newPassword; policy min 15 code points / max 72 bytes. changePassword discards TokenResult; input maxLength 256. Phase: later. Phase 1 operations: none; deferred UI. Gaps: .

Existing UI is deferred and must not be mounted/call legacy endpoints in Phase 1. Accounts, single roles and PA assignments are provisioned through controlled database setup; password/role/session lifecycle UI is deferred.

**UI element:** Required password-change form: current password, new password and save button

| Operation | Request / returned-data use |
| --- | --- |

## F15

`frontendx/packages/data/src/session.tsx` — SessionProvider. Continue actual activity / refresh access after reload. Authoritative idle/absolute timing, memory token and rotating refresh. No renewal, refresh, multi-tab rotation or expiry-warning consumer exists. Phase: later. Phase 1 operations: none; deferred UI. Gaps: .

Existing UI is deferred and must not be mounted/call legacy endpoints in Phase 1. Accounts, single roles and PA assignments are provisioned through controlled database setup; password/role/session lifecycle UI is deferred.

**UI element:** Workspace activity/expiry handling and access-token restoration after reload; consumer still missing

| Operation | Request / returned-data use |
| --- | --- |

## F16

`frontendx/apps/admin/src/pages/Bookings.tsx` — Bookings. Search/filter/sort/page ordinary and restricted records. view=list, q/status/sort/dir/date/vip/page/pageSize=7. Production readStaffBookings; preview full list then local selectBookings. Phase: phase1. Phase 1 operations: staffBookings. Gaps: BE-READS, FE-CORRELATION.

**UI element:** Janji temu / Kunjungan hari ini search box, status tabs, VIP filter, sort selector, cards/table and pagination

| Operation | Request / returned-data use |
| --- | --- |
| [staffBookings](API-CONTRACT.md#staffbookings) | Send view=list plus q/status/sort/dir/date/vip/page/pageSize=7; items populate cards/table, total/totalScoped/page/pageSize control counts and pagination. |

## F17

`frontendx/apps/admin/src/pages/Bookings.tsx` — Bookings / BookingDetail. Open detail independent of page. Authorized record and field projection for selected ID. useBookingDetail -> readStaffBooking; does not verify returned ID. Phase: phase1. Phase 1 operations: staffBooking. Gaps: BE-READS, FE-CORRELATION.

**UI element:** Selected booking-detail dialog opened from a row/card or selected URL parameter

| Operation | Request / returned-data use |
| --- | --- |
| [staffBooking](API-CONTRACT.md#staffbooking) | Send path id; returned authorized StaffBooking variant populates the dialog even when the booking is outside the current list page. |

## F18

`frontendx/apps/admin/src/pages/Dashboard.tsx` — Dashboard. Read metrics, agenda and attention. view=dashboard/date/page=1/pageSize=50; full-scope metrics plus bounded union. Shares booking read; Super Admin skips operational query. Phase: phase1. Phase 1 operations: staffBookings. Gaps: BE-READS.

**UI element:** Dashboard metric cards, Agenda hari ini / Menunggu kedatangan table and Perlu perhatian / Sedang berkunjung list

| Operation | Request / returned-data use |
| --- | --- |
| [staffBookings](API-CONTRACT.md#staffbookings) | Send view=dashboard/date/page=1/pageSize=50; metrics.today/pending/arrivals/inside/completed supply cards, items supply the bounded agenda/attention union. Super Admin currently skips operational reads. |

## F19

`frontendx/apps/admin/src/components/BookingRefresh.tsx` — ProductionRefresh. Visible-only timed/manual/focus refresh. Active scoped booking queries; 30 sec staff / 15 sec reception. Refetches active queries; no new endpoint; avoid auth renewal on polling. Phase: phase1. Phase 1 operations: staffBookings, staffBooking. Gaps: BE-READS.

**UI element:** Visible-page refresh timer, focus refresh and manual refresh for lists/details/reception

| Operation | Request / returned-data use |
| --- | --- |
| [staffBookings](API-CONTRACT.md#staffbookings) | Reissue the active typed list/dashboard/reception query; replace rows/counts/metrics together. |
| [staffBooking](API-CONTRACT.md#staffbooking) | Reissue the selected-id read; replace only with a currently authorized matching detail. |

## F20

`frontendx/apps/admin/src/components/DataRefresh.tsx` — DataRefresh. Retry / show freshness and offline state. Existing query fetch status/error/update time. Pure query control; no persisted business data. Phase: phase1. Phase 1 operations: staffBookings, staffEmployees, staffIntegrations. Gaps: BE-READS.

**UI element:** DataRefresh freshness/offline/error notice and retry controls

| Operation | Request / returned-data use |
| --- | --- |
| [staffBookings](API-CONTRACT.md#staffbookings) | Refetch existing booking view; use query fetch/error/update state for the freshness control. |
| [staffEmployees](API-CONTRACT.md#staffemployees) | Refetch employee page; no separate freshness endpoint. |
| [staffIntegrations](API-CONTRACT.md#staffintegrations) | Refetch safe integration metadata; no separate freshness endpoint. |

## F21

`frontendx/apps/admin/src/features/CreateAppointment.tsx` — CreateAppointment. Open staff standard form / submit. Authorized active official catalog; ordinary fields, roster, notes, optional document. useCreateAppointment; UUID retained while payload unchanged; room/duration not collected. Phase: phase1. Phase 1 operations: staffCatalogs, staffCreateBooking, staffUpload. Gaps: BE-VISITS, FE-UPLOAD.

**UI element:** Staff standard-appointment dialog: official dropdown, schedule, representative/roster, note fields, document picker and submit

| Operation | Request / returned-data use |
| --- | --- |
| [staffCatalogs](API-CONTRACT.md#staffcatalogs) | officers[].id/name/vip populate and scope the official selector; attachmentsEnabled gates the optional file picker. |
| [staffCreateBooking](API-CONTRACT.md#staffcreatebooking) | Send visitKind=standard and ordinary fields, UUID idempotencyKey and optional notes/upload token; returned approved booking is the save result. |
| [staffUpload](API-CONTRACT.md#staffupload) | Upload file/reference and use token as attachmentToken on the create command; keep private bytes outside JSON. |

## F22

`frontendx/apps/admin/src/features/RestrictedAppointment.tsx` — RestrictedAppointment. Create restricted visit. date/time required; optional name/organization/serviceId/officerId and notes. Protocol create gate; no contact/identity number/roster/file required. Phase: phase1. Phase 1 operations: staffCatalogs, staffCreateBooking. Gaps: BE-VISITS.

**UI element:** Kunjungan Terbatas creation dialog: date/time, optional identity/target fields, notes and create button

| Operation | Request / returned-data use |
| --- | --- |
| [staffCatalogs](API-CONTRACT.md#staffcatalogs) | officers[].id/name populate optional officerId selector; no public contact/roster/file requirement is added. |
| [staffCreateBooking](API-CONTRACT.md#staffcreatebooking) | Send visitKind=restricted, idempotencyKey/date/time and optional name/organization/serviceId/officerId/notes; render returned authorized restricted booking. |

## F23

`frontendx/apps/admin/src/features/RestrictedAppointment.tsx` — RestrictedAppointment. Edit restricted identity/target/notes. Version; date/time; empty strings clear optional identity/target. useBookingAction(update); date/time inputs readOnly on existing record. Phase: phase1. Phase 1 operations: staffRestrictedUpdate, staffCatalogs. Gaps: BE-VISITS.

**UI element:** Restricted visit edit dialog and Simpan perubahan button

| Operation | Request / returned-data use |
| --- | --- |
| [staffRestrictedUpdate](API-CONTRACT.md#staffrestrictedupdate) | Send version/visitKind/date/time with optional identity/target/notes; omission preserves and empty values clear. Current schedule controls are read-only; schedule changes use reschedule. |
| [staffCatalogs](API-CONTRACT.md#staffcatalogs) | Use authorized official IDs/names and retained choices for the optional target selector. |

## F24

`frontendx/apps/admin/src/features/BookingDetail.tsx` — BookingDetail. Approve pending visit. Version, selected room, duration and public/private notes. Form sends room label and duration; target requires roomId. Phase: phase1. Phase 1 operations: staffApprove, staffCatalogs. Gaps: BE-VISITS, FE-ROOM.

**UI element:** Approve dialog: room dropdown, duration input, note fields and confirmation

| Operation | Request / returned-data use |
| --- | --- |
| [staffApprove](API-CONTRACT.md#staffapprove) | Send version/roomId/duration and optional notes; replace detail with returned approved StaffBooking. Current form sends room label and requires coordinated conversion. |
| [staffCatalogs](API-CONTRACT.md#staffcatalogs) | rooms[].id/name/active supply the approval room selector; selected value must become stable roomId. |

## F25

`frontendx/apps/admin/src/features/BookingDetail.tsx` — BookingDetail. Reject pending visit. Version; public reason 5-1000; private note <=2000. rejectionSchema -> useBookingAction; real backend absent. Phase: phase1. Phase 1 operations: staffReject. Gaps: BE-VISITS.

**UI element:** Reject dialog: public rejection-reason textarea, private note textarea and reject button

| Operation | Request / returned-data use |
| --- | --- |
| [staffReject](API-CONTRACT.md#staffreject) | Send version/rejectionReason/optional internalRejectionNote; returned rejected booking refreshes visible public/private sections according to authority. |

## F26

`frontendx/apps/admin/src/features/BookingDetail.tsx` — BookingDetail. Reschedule. Version, future date/time, duration, room, reason, notes. room label sent; domain.can permits Approved only while D100 also selects Pending. Phase: phase1. Phase 1 operations: staffReschedule, staffCatalogs. Gaps: BE-VISITS, FE-ROOM, FE-PENDING.

**UI element:** Reschedule dialog: new date/time, room dropdown, duration, reason and notes

| Operation | Request / returned-data use |
| --- | --- |
| [staffReschedule](API-CONTRACT.md#staffreschedule) | Send version/date/time/duration/optional roomId/reason/notes; returned schedule/version/timing refresh detail. Missing roomId clears the room under full schedule replacement. |
| [staffCatalogs](API-CONTRACT.md#staffcatalogs) | rooms[].id/name/active populate room options; current form still sends a room label. |

## F27

`frontendx/apps/admin/src/features/BookingDetail.tsx` — BookingDetail. Staff cancel / read private notes. Version, reason, internalCancellationNote; decision/field scope. useBookingAction(cancel); separate from public pair authorization. Phase: phase1. Phase 1 operations: staffCancel. Gaps: BE-VISITS.

**UI element:** Staff cancellation dialog: public reason, internal cancellation note and confirm button

| Operation | Request / returned-data use |
| --- | --- |
| [staffCancel](API-CONTRACT.md#staffcancel) | Send version/reason/optional internalCancellationNote; use returned cancelled booking and only authorized private-note fields. |

## F28

`frontendx/apps/admin/src/features/VisitNotesFields.tsx` — VisitNotesFields. Edit public instructions / private coordination. Separate optional 2000-code-point values; omit preserves; empty clears. Shared form fields; notes travel with existing command. Phase: phase1. Phase 1 operations: staffCreateBooking, staffApprove, staffReschedule, staffRestrictedUpdate. Gaps: BE-VISITS, FE-BOUNDS.

**UI element:** Shared Instruksi pengunjung and Catatan koordinasi internal textareas

| Operation | Request / returned-data use |
| --- | --- |
| [staffCreateBooking](API-CONTRACT.md#staffcreatebooking) | Include visitorInstructions/coordinationNote in create; returned fields render only for the permitted projection. |
| [staffApprove](API-CONTRACT.md#staffapprove) | Include optional note fields in approve; omission preserves, empty clears. |
| [staffReschedule](API-CONTRACT.md#staffreschedule) | Include optional note fields in reschedule; public/private projection remains distinct. |
| [staffRestrictedUpdate](API-CONTRACT.md#staffrestrictedupdate) | Include optional note fields in restricted update; full read alone does not grant coordination-note authority. |

## F29

`frontendx/apps/admin/src/features/AttachmentAccess.tsx` — AttachmentAccess. Read attached private file. Authorized bytes, MIME, safe filename; view and download. downloadAttachment only; request lacks explicit disposition=attachment. Phase: phase1. Phase 1 operations: staffAttachment. Gaps: BE-DOCUMENTS, FE-FILEVIEW.

**UI element:** Private supporting-document action in BookingDetail; existing action downloads only

| Operation | Request / returned-data use |
| --- | --- |
| [staffAttachment](API-CONTRACT.md#staffattachment) | Send path id and disposition=inline for view or attachment for download; MIME/Content-Disposition name the returned bytes. Current download helper omits explicit disposition=attachment. |

## F30

`frontendx/apps/admin/src/pages/Reception.tsx` — Reception. Read arrival / still-inside page. view=reception/mode; check-in today; checkout all dates; pageSize=25. useBookings typed view; one endpoint reused. Phase: phase1. Phase 1 operations: staffBookings. Gaps: BE-READS.

**UI element:** Reception scheduled-arrivals / still-inside table and previous/next page controls

| Operation | Request / returned-data use |
| --- | --- |
| [staffBookings](API-CONTRACT.md#staffbookings) | Send view=reception/mode/page/pageSize=25; check-in sends today date, checkout spans all dates. items and total/page/pageSize populate the table and pagination. |

## F31

`frontendx/apps/admin/src/pages/Reception.tsx` — Reception. Resolve manual code or QR before confirmation. credential -> current authorized standard/restricted arrival projection. repository.resolve; no attendance side effect; restricted result kept local. Phase: phase1. Phase 1 operations: staffResolve. Gaps: BE-ATTENDANCE.

**UI element:** Manual visit-code form or QR resolve result before attendance confirmation

| Operation | Request / returned-data use |
| --- | --- |
| [staffResolve](API-CONTRACT.md#staffresolve) | Send credential; authorized ArrivalBooking fields populate the identity/visit confirmation dialog. Resolve never changes attendance. |

## F32

`frontendx/apps/admin/src/features/Scanner.tsx` — Scanner. Scan camera QR. BarcodeDetector raw string passed to Reception.resolve. Camera local only; no image upload/storage/API. Phase: phase1. Phase 1 operations: staffResolve. Gaps: BE-ATTENDANCE.

**UI element:** Camera scanner preview and decoded-QR handler

| Operation | Request / returned-data use |
| --- | --- |
| [staffResolve](API-CONTRACT.md#staffresolve) | Send only BarcodeDetector raw string as credential to resolve; returned booking opens confirmation. Camera frames stay local. |

## F33

`frontendx/apps/admin/src/pages/Reception.tsx` — Reception. Confirm identity then check in. Matching credential and expected version, server timing eligibility. useBookingAction(checkin); checkbox local; no identityVerified wire field. Phase: phase1. Phase 1 operations: staffCheckin. Gaps: BE-ATTENDANCE.

**UI element:** Reception identity-confirmation checkbox and check-in confirmation button

| Operation | Request / returned-data use |
| --- | --- |
| [staffCheckin](API-CONTRACT.md#staffcheckin) | Send resolved id, matching credential and version; returned checked_in ArrivalBooking refreshes attendance. Checkbox stays local; no identityVerified wire field. |

## F34

`frontendx/apps/admin/src/pages/Reception.tsx` — Reception. Confirm identity then check out. Credential/version; checked_in may be from earlier date. useBookingAction(checkout); consumed nonrevoked QR usable per D96. Phase: phase1. Phase 1 operations: staffCheckout. Gaps: BE-ATTENDANCE.

**UI element:** Reception identity-confirmation checkbox and check-out confirmation button

| Operation | Request / returned-data use |
| --- | --- |
| [staffCheckout](API-CONTRACT.md#staffcheckout) | Send resolved id/credential/version; returned checked_out ArrivalBooking removes the visit from still-inside results; late/prior-date checkout is supported. |

## F35

`frontendx/apps/admin/src/features/EmployeeDirectory.tsx` — EmployeeDirectory. Search/page/reload employee directory. q/page/pageSize=25, filtered/totalAll, password metadata. management.employees; local preview array; no backend admin routes. Phase: phase1. Phase 1 operations: staffEmployees, staffEmployee. Gaps: BE-ADMIN, FE-DIRECTORY.

**UI element:** Read-only employee directory search, cards/table, pagination and detail; editing controls deferred

| Operation | Request / returned-data use |
| --- | --- |
| [staffEmployees](API-CONTRACT.md#staffemployees) | Send q/page/pageSize=25; EmployeePage.items/total/totalAll/page/pageSize populate the read-only directory. Password policy metadata and create/edit actions are deferred. |
| [staffEmployee](API-CONTRACT.md#staffemployee) | Read the employee id/profile/single-role array/assignments/version for an authorized read-only detail. Existing edit-conflict reload is a deferred use; no account writes in Phase 1. |

## F36

`frontendx/apps/admin/src/features/EmployeeEditor.tsx` — EmployeeEditor. Create employee / edit / deactivate. roles, profile strings, active, assignments, initial password only on create, version on edit. management.saveEmployee; preserves unavailable assignments. Phase: later. Phase 1 operations: none; deferred UI. Gaps: .

Existing UI is deferred and must not be mounted/call legacy endpoints in Phase 1. Accounts, single roles and PA assignments are provisioned through controlled database setup; password/role/session lifecycle UI is deferred.

**UI element:** Employee create/edit dialog: profile fields, active checkbox, roles, assignments and initial password on create

| Operation | Request / returned-data use |
| --- | --- |

## F37

`frontendx/apps/admin/src/features/RoleAssignments.tsx` — RoleAssignments. Select multiple employee roles. Assigned role labels from fixed role enum; tag/search state. Local selector; saved atomically with employee; no join-row endpoint. Phase: later. Phase 1 operations: none; deferred UI. Gaps: .

Existing UI is deferred and must not be mounted/call legacy endpoints in Phase 1. Accounts, single roles and PA assignments are provisioned through controlled database setup; password/role/session lifecycle UI is deferred.

**UI element:** Multi-role tag/search selector within the employee editor

| Operation | Request / returned-data use |
| --- | --- |

## F38

`frontendx/apps/admin/src/features/EmployeeEditor.tsx` — EmployeeEditor. Choose assigned non-VIP official / retain inactive assignment. Authorized catalog plus retained IDs on employee. useCatalogs; no truncation; current validator lacks max100/unique IDs. Phase: later. Phase 1 operations: none; deferred UI. Gaps: .

Existing UI is deferred and must not be mounted/call legacy endpoints in Phase 1. Accounts, single roles and PA assignments are provisioned through controlled database setup; password/role/session lifecycle UI is deferred.

**UI element:** PA non-VIP official-assignment checkboxes and retained-unavailable-assignment notice

| Operation | Request / returned-data use |
| --- | --- |

## F39

`frontendx/apps/admin/src/features/RoleManagement.tsx` — RoleManagement / RoleEditor. List/reload/save role permissions. Permission dictionary in each RoleDefinition; assignability/editable/version. management.roles/saveRole; preview roles use names as IDs. Phase: later. Phase 1 operations: none; deferred UI. Gaps: .

Existing UI is deferred and must not be mounted/call legacy endpoints in Phase 1. Accounts, single roles and PA assignments are provisioned through controlled database setup; password/role/session lifecycle UI is deferred.

**UI element:** Peran operasional cards, Edit akses dialog, permission checkboxes and Simpan akses button

| Operation | Request / returned-data use |
| --- | --- |

## F40

`frontendx/apps/admin/src/features/IntegrationManagement.tsx` — IntegrationManagement / IntegrationEditor. Read/configure provider; keep/clear secret; versioned save. Safe field descriptors, configured secret flags, health; SMTP six-key schema. Generic live form uses server metadata; preview endpoint/bucket/credential examples are not selected providers. Phase: phase1. Phase 1 operations: staffIntegrations, patchIntegration. Gaps: BE-INTEGRATIONS.

**UI element:** Integration service cards, Konfigurasi dialog, enabled checkbox, dynamic settings fields and secret keep/clear controls

| Operation | Request / returned-data use |
| --- | --- |
| [staffIntegrations](API-CONTRACT.md#staffintegrations) | Integration.name/description/health/enabled/fields populate cards and editor; secret fields expose configured flags, never values. |
| [patchIntegration](API-CONTRACT.md#patchintegration) | Send enabled/values/clearSecrets/version for path id; returned safe Integration refreshes card/health/descriptors without echoing secrets. |

## F41

`frontendx/apps/admin/src/pages/Reports.tsx` — Reports. Read filtered summary / refresh. Inclusive from/to/status; seven counts, scope, revision, generatedAt, canExport. useVisitSummary validates filter echo/sum; both tabs share operations. Phase: phase1. Phase 1 operations: reportSummary. Gaps: BE-REPORTS.

**UI element:** Analytics/logs date range and status filters, total cards and per-status summary table/meters

| Operation | Request / returned-data use |
| --- | --- |
| [reportSummary](API-CONTRACT.md#reportsummary) | Send inclusive from/to/status; filter echo/scope/generatedAt/counts/total/revision/canExport populate summary and export eligibility. Restricted records are excluded. |

## F42

`frontendx/apps/admin/src/pages/Reports.tsx` — Reports. Export displayed revision / recover conflict. Same filter/revision -> filename,csv,revision JSON. repository.exportSummary -> Blob download; conflict refetch; summary rows only. Phase: phase1. Phase 1 operations: reportExport, reportSummary. Gaps: BE-REPORTS.

**UI element:** Unduh ringkasan CSV button and changed-summary conflict recovery

| Operation | Request / returned-data use |
| --- | --- |
| [reportExport](API-CONTRACT.md#reportexport) | Send the displayed from/to/status/revision; filename/csv/revision build the local CSV download, containing summary rows only. |
| [reportSummary](API-CONTRACT.md#reportsummary) | On export 409, reload the same summary filter and display its current counts/revision before another export. |

## F43

`frontendx/apps/admin/src/pages/Settings.tsx` — Settings. Route shared management modules. No additional data beyond mounted module. Lazy router; guard permissions in main.tsx. Phase: phase1. Phase 1 operations: staffEmployees, staffIntegrations, reportSummary. Gaps: BE-ADMIN.

**UI element:** Settings shell mounting read-only employees, integrations and reports; roles/editor routes deferred

| Operation | Request / returned-data use |
| --- | --- |
| [staffEmployees](API-CONTRACT.md#staffemployees) | Delegate to employee directory; no additional shell response. |
| [staffIntegrations](API-CONTRACT.md#staffintegrations) | Delegate to integration cards/editor; no additional shell response. |
| [reportSummary](API-CONTRACT.md#reportsummary) | Delegate to Reports summary UI; no separate logs/analytics data API. |

## F44

`frontendx/apps/admin/src/main.tsx` — Shell / Guard. Render permitted nav, user name, role gates. Session permissions, assignments and required-password/role state. UI guard is presentation; backend must independently enforce every route. Phase: phase1. Phase 1 operations: staffMe. Gaps: FE-AUTH, BE-PHASE1-AUTH.

**UI element:** Staff name/role display, fixed-permission navigation and route guards

| Operation | Request / returned-data use |
| --- | --- |
| [staffMe](API-CONTRACT.md#staffme) | Use CurrentUser.name/role/permissions/assignedOfficerIds; there are no availableRoles, roleRevision, role-selection or required-password gates. Backend independently checks current authority. |

## F45

`frontendx/packages/data/src/attachments.ts` — uploadAttachment. Upload/replay selected file. Multipart file+UUID reference; expiry and serverTime; token only returned. WeakMap reuses reference for same File/audience; optional upload still attempted without capability check. Phase: phase1. Phase 1 operations: publicUpload, staffUpload. Gaps: FE-UPLOAD, BE-DOCUMENTS.

**UI element:** Shared file-picker upload/retry transport for public and staff forms

| Operation | Request / returned-data use |
| --- | --- |
| [publicUpload](API-CONTRACT.md#publicupload) | Send one file and stable per-File/audience UUID reference; token/expiresAt/serverTime feed the public create attachmentToken field. |
| [staffUpload](API-CONTRACT.md#staffupload) | Use the same multipart/result shape with Bearer/current create authority; token feeds staff standard creation. |

## F46

`frontendx/packages/data/src/http.ts` — createHttpClient. Execute request / recover error. JSON/multipart/blob, credentials, error fields, Retry-After. Existing shared client sends no Bearer and still includes cookies; Phase 1 must not rely on them. Its field-error parser expects string[]; current backend string-valued errors are filtered out (BE-ERRORS).. Phase: phase1. Phase 1 operations: staffLogin, staffMe, publicCreateBooking. Gaps: FE-AUTH, BE-ERRORS, BE-PHASE1-AUTH.

**UI element:** Shared Bearer-only HTTP transport and field-error/rate-limit handling

| Operation | Request / returned-data use |
| --- | --- |
| [staffLogin](API-CONTRACT.md#stafflogin) | POST JSON login with no auth-cookie/CSRF dependency; parse AuthResult and retain token/expiry/user in memory. Current signIn still calls the superseded session route and discards its result. |
| [staffMe](API-CONTRACT.md#staffme) | Attach the in-memory Bearer token; parse CurrentUser, clear auth/private state on expiry or unauthorized response. Current client still uses the old session endpoint/shape. |
| [publicCreateBooking](API-CONTRACT.md#publiccreatebooking) | Parse PublicBooking or ApiError.fieldErrors arrays; retain uncertainty/recovery behavior and Retry-After handling. |

## F47

`frontendx/packages/data/src/repositories.ts` — createRepository. Share production and explicit development adapters. Public/staff paths; strict response schemas; role only in __preview. Production list() legacy array branch exists but useBookings uses readStaffBookings instead. Phase: phase1. Phase 1 operations: staffBookings, publicLookup. Gaps: BE-READS.

**UI element:** Shared production repository used by booking/status components; preview adapter is explicit

| Operation | Request / returned-data use |
| --- | --- |
| [staffBookings](API-CONTRACT.md#staffbookings) | Use typed paged ReadModel for staff hooks; legacy array list() is not the target wire shape. |
| [publicLookup](API-CONTRACT.md#publiclookup) | Use pair-authorized PublicBooking for status/history; preview role hints never enter production authority. |

## F48

`frontendx/apps/visitor/src/main.tsx` — App. Open landing/registration/status. No separate landing statistics/content API. Static hero/info; embedded Booking/Status supply dynamic data. Phase: phase1. Phase 1 operations: publicCatalogs, publicLookup. Gaps: FE-CATALOG.

**UI element:** Visitor landing shell with embedded registration/status views; hero/info content is static

| Operation | Request / returned-data use |
| --- | --- |
| [publicCatalogs](API-CONTRACT.md#publiccatalogs) | Registration child consumes the shared catalogs snapshot for dropdown/notice; no landing statistics API. |
| [publicLookup](API-CONTRACT.md#publiclookup) | Status child consumes pair-authorized booking; the shell needs no extra landing endpoint. |

## Source review boundary

The checker inventories request() calls with the existing TypeScript parser and compares their normalized AST text to the reviewed snapshot. New/changed requests require a deliberate manifest update and endpoint mapping. Every application source file and shared data/transport module is classified below; data-bearing consumers are mapped above. Decorative UI, camera capture and local preferences create no additional API requirement.

| File | Classification | Reason |
| --- | --- | --- |
| frontendx/apps/admin/src/components/BookingRefresh.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/components/DataRefresh.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/components/lightswind/sidebar.tsx | presentation/local support | Local layout, input, language, navigation, focus or visual asset behavior; no direct request/persisted business operation. |
| frontendx/apps/admin/src/features/AttachmentAccess.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/BookingDetail.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/CreateAppointment.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/EmployeeDetails.tsx | presentation/local support | Read-only F35 employee detail projection using existing employee and catalog queries. |
| frontendx/apps/admin/src/features/EmployeeDirectory.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/EmployeeEditor.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/IntegrationManagement.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/ManagementDialog.tsx | presentation/local support | Local layout, input, language, navigation, focus or visual asset behavior; no direct request/persisted business operation. |
| frontendx/apps/admin/src/features/PasswordChange.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/RestrictedAppointment.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/RoleAssignments.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/RoleControls.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/RoleManagement.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/Scanner.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/features/visitLogExport.ts | presentation/local support | Client CSV/PDF rendering for the separately documented Super Admin visit-log extension. |
| frontendx/apps/admin/src/features/VisitNotesFields.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/hooks/managementFreshness.ts | presentation/local support | Shared query focus/reconnect/retry settings; uses existing management endpoints. |
| frontendx/apps/admin/src/hooks/useDebouncedValue.ts | presentation/local support | Local layout, input, language, navigation, focus or visual asset behavior; no direct request/persisted business operation. |
| frontendx/apps/admin/src/main.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/pages/Bookings.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/pages/Dashboard.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/pages/Reception.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/pages/Reports.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/pages/Settings.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/admin/src/pages/VisitLog.tsx | data consumer | Separately documented Super Admin visit-log extension; see SUPER-ADMIN-VISIT-LOG.md. Does not add Phase 1 operation promises. |
| frontendx/apps/visitor/src/features/AdmissionQR.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/visitor/src/features/hero-image.ts | presentation/local support | Local layout, input, language, navigation, focus or visual asset behavior; no direct request/persisted business operation. |
| frontendx/apps/visitor/src/features/landingMotion.ts | presentation/local support | Local layout, input, language, navigation, focus or visual asset behavior; no direct request/persisted business operation. |
| frontendx/apps/visitor/src/features/PrivacyNotice.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/visitor/src/features/PublicVisitHistory.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/visitor/src/features/SubmissionRecovery.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/visitor/src/features/useCancellationWindow.ts | data consumer | Mapped in feature matrix. |
| frontendx/apps/visitor/src/features/VisitTicket.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/visitor/src/hero-image-settings.ts | presentation/local support | Local layout, input, language, navigation, focus or visual asset behavior; no direct request/persisted business operation. |
| frontendx/apps/visitor/src/i18n.tsx | presentation/local support | Local layout, input, language, navigation, focus or visual asset behavior; no direct request/persisted business operation. |
| frontendx/apps/visitor/src/main.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/visitor/src/pages/Booking.tsx | data consumer | Mapped in feature matrix. |
| frontendx/apps/visitor/src/pages/Status.tsx | data consumer | Mapped in feature matrix. |
| frontendx/packages/data/src/attachments.ts | shared data/transport contract | Upload reference/Blob download adapter; F02/F21/F29/F45. |
| frontendx/packages/data/src/attendance.ts | shared data/transport contract | Credential resolve and check-in/out request adapter; F07/F31-F34. |
| frontendx/packages/data/src/config.ts | local/shared support | Configured API base and explicit preview flag; no new operation. |
| frontendx/packages/data/src/contact.ts | local/shared support | Local phone/WhatsApp formatting; no backend persistence. |
| frontendx/packages/data/src/contracts.ts | shared data/transport contract | Shared selected-role/session/booking type boundary; supports mapped consumers. |
| frontendx/packages/data/src/domain.ts | shared data/transport contract | Role/resource projection, local preview/filter order and timing; evidence, not backend authority. |
| frontendx/packages/data/src/errors.ts | shared data/transport contract | ApiError field arrays and retry classification; shared transport contract. |
| frontendx/packages/data/src/history.ts | shared data/transport contract | Public event allowlist; history is included in booking responses. |
| frontendx/packages/data/src/hooks.ts | shared data/transport contract | Existing query and mutation adapters; no independent route. |
| frontendx/packages/data/src/http.ts | shared data/transport contract | Shared credentials, field-error arrays, abort and authenticated-401 behavior. |
| frontendx/packages/data/src/index.tsx | local/shared support | Shared module exports; no independent data operation. |
| frontendx/packages/data/src/management-contract.ts | shared data/transport contract | Employee/role/integration response and editor schemas; F35-F40. |
| frontendx/packages/data/src/management.ts | shared data/transport contract | Directory/detail/save/roles/integrations adapters and explicit preview fixtures. |
| frontendx/packages/data/src/organization.ts | shared data/transport contract | Verified deputy/directorate reference and exact-title compatibility mapping; organizational metadata is not assignment authority. |
| frontendx/packages/data/src/query.tsx | local/shared support | Query provider/cache configuration; local support. |
| frontendx/packages/data/src/read-model.ts | shared data/transport contract | Typed booking-view/detail reads; F16-F19/F30. |
| frontendx/packages/data/src/reports.ts | shared data/transport contract | Summary/export schemas, filter/revision checks and shared report adapters; F41-F42. |
| frontendx/packages/data/src/repositories.ts | shared data/transport contract | Public/staff request adapters and explicit development repository boundary. |
| frontendx/packages/data/src/scheduling.ts | shared data/transport contract | Local WITA interval/timing calculations; server remains authoritative. |
| frontendx/packages/data/src/session-contract.ts | shared data/transport contract | Session parser/gate policy metadata; F11-F15/F44. |
| frontendx/packages/data/src/session.tsx | shared data/transport contract | Sign-in/session/role/password client and authority cache lifecycle; F11-F15. |
| frontendx/packages/data/src/submission.ts | shared data/transport contract | Create/reference recovery coordinator; F02/F04. |
| frontendx/packages/data/src/unsaved.ts | local/shared support | Local unsaved-draft navigation state; no persistence endpoint. |
| frontendx/packages/data/src/validation.ts | shared data/transport contract | Write schemas and form validation; bounds/variant mismatches remain documented. |
| frontendx/packages/data/src/visit-log.ts | shared data/transport contract | Typed filter/read/export models for the separately documented Super Admin visit-log extension. |
| frontendx/packages/data/src/wire.ts | shared data/transport contract | Actual response schemas and restricted field checks; shared wire compatibility evidence. |
