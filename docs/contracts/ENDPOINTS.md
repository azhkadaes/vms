# Endpoint specifications

The complete SIHP-structured endpoint reference is now in [API-CONTRACT](API-CONTRACT.md#api-endpoints). This file preserves existing operation links as a navigation index. Each linked specification includes frontend controls/field usage, method/path, authentication, query/path/header/cookie parameters, request fields/examples, business logic, response fields/examples and operation-specific errors.

## staffLogin

[POST /v1/staff/auth/login — Staff email/password login form](API-CONTRACT.md#stafflogin)

## staffMe

[GET /v1/staff/auth/me — Current user, navigation and fixed-role access checks](API-CONTRACT.md#staffme)

## publicCatalogs

[GET /v1/public/catalogs — Registration destination dropdown and bilingual privacy dialog](API-CONTRACT.md#publiccatalogs)

## staffCatalogs

[GET /v1/staff/catalogs — Staff official/room selectors and PA assignment checkboxes](API-CONTRACT.md#staffcatalogs)

## publicCreateBooking

[POST /v1/public/bookings — Public three-step submission and receipt card](API-CONTRACT.md#publiccreatebooking)

## resolveSubmission

[POST /v1/public/submissions/resolve — Uncertain-submission recovery panel](API-CONTRACT.md#resolvesubmission)

## publicLookup

[POST /v1/public/lookup — Code/email status form, visit card, ticket and history](API-CONTRACT.md#publiclookup)

## publicCancel

[POST /v1/public/bookings/{id}/cancel — Status-card cancellation confirmation](API-CONTRACT.md#publiccancel)

## publicAdmission

[POST /v1/public/bookings/{id}/admission — Admission QR image, countdown and renewal](API-CONTRACT.md#publicadmission)

## publicUpload

[POST /v1/public/attachments — Public optional supporting-document picker](API-CONTRACT.md#publicupload)

## staffUpload

[POST /v1/staff/attachments — Staff standard-appointment document picker](API-CONTRACT.md#staffupload)

## staffAttachment

[GET /v1/staff/bookings/{id}/attachment — Private document view/download in booking detail](API-CONTRACT.md#staffattachment)

## staffBookings

[GET /v1/staff/bookings — Booking search/table/cards, dashboard cards/agenda and reception table](API-CONTRACT.md#staffbookings)

## staffBooking

[GET /v1/staff/bookings/{id} — Selected booking-detail dialog](API-CONTRACT.md#staffbooking)

## staffCreateBooking

[POST /v1/staff/bookings — Standard and restricted staff create dialogs](API-CONTRACT.md#staffcreatebooking)

## staffApprove

[POST /v1/staff/bookings/{id}/approve — Approval room/duration/notes form](API-CONTRACT.md#staffapprove)

## staffReject

[POST /v1/staff/bookings/{id}/reject — Public rejection reason and internal note form](API-CONTRACT.md#staffreject)

## staffReschedule

[POST /v1/staff/bookings/{id}/reschedule — New schedule/room/duration/reason form](API-CONTRACT.md#staffreschedule)

## staffRestrictedUpdate

[POST /v1/staff/bookings/{id}/update — Restricted identity/target/notes edit dialog](API-CONTRACT.md#staffrestrictedupdate)

## staffCancel

[POST /v1/staff/bookings/{id}/cancel — Staff cancellation reason/internal-note dialog](API-CONTRACT.md#staffcancel)

## staffCheckin

[POST /v1/staff/bookings/{id}/checkin — Identity-confirmed check-in button](API-CONTRACT.md#staffcheckin)

## staffCheckout

[POST /v1/staff/bookings/{id}/checkout — Identity-confirmed check-out button](API-CONTRACT.md#staffcheckout)

## staffResolve

[POST /v1/staff/resolve — Manual code and camera-QR confirmation dialog](API-CONTRACT.md#staffresolve)

## staffEmployees

[GET /v1/staff/employees — Employee search, cards/table, counts and pagination](API-CONTRACT.md#staffemployees)

## staffEmployee

[GET /v1/staff/employees/{id} — Employee edit-conflict reload](API-CONTRACT.md#staffemployee)

## staffIntegrations

[GET /v1/staff/integrations — Integration cards and safe dynamic-field metadata](API-CONTRACT.md#staffintegrations)

## patchIntegration

[PATCH /v1/staff/integrations/{id} — Provider settings and secret keep/clear save form](API-CONTRACT.md#patchintegration)

## reportSummary

[GET /v1/staff/reports/summary — Analytics/logs filters, summary cards and status table/meters](API-CONTRACT.md#reportsummary)

## reportExport

[GET /v1/staff/reports/export — Displayed-summary CSV download button](API-CONTRACT.md#reportexport)

## Error examples

See [common error codes and examples](API-CONTRACT.md#error-codes).
