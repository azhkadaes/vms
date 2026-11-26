# Spec (future): email notifications through mailgrid over SMTP

Label: spec · Status: **deferred**, not `ready-for-agent`. Start after the [backend spec](../backend-api-2026-11-18/spec.md) tickets 06–12 land, then run `/to-tickets` on this file. This resolves open decision D-06 in the [validation](../../contracts/UI-CONTRACT-VALIDATION-2026-11-18.md): email is a later effort, sent through mailgrid.

Agreed inputs:
- BR-KOM-01 ([business requirements](../../frontend/BUSINESS-REQUIREMENTS.md)): send a submission confirmation, plus decision, cancellation and reschedule notices, when an address exists.
- Earlier [plan](../../BACKEND-PLAN-2026-11-11.md) decisions:
  - D08: separate worker process
  - D23: retry for up to 24 hours
  - D36: waits of 1 m, 5 m, 15 m, 1 h, 3 h, then 6 h, with jitter
  - D98: authenticated SMTP over verified TLS, behind a delivery interface
  - §7: PostgreSQL outbox; no SMTP while holding locks; no recipients in logs

## Problem Statement

Visitors get no email today. The staff UI already promises visitor instructions "di detail kunjungan dan email pemberitahuan" (`VisitNotesFields`). Without notifications, a visitor learns about an approval, rejection, reschedule or cancellation only by checking the status page.

## Solution

Every visit command that should notify writes a notification intent into a PostgreSQL outbox, in the same transaction as the visit change. A separate worker process claims due intents, renders the email from embedded templates, and sends it through a `Mailer` interface. The first `Mailer` implementation wraps mailgrid's importable `email` package over authenticated, verified-TLS SMTP.

The outbox owns retries, deadlines and deduplication. Mailgrid only performs one SMTP attempt per call. Swapping mailgrid for another SMTP library later changes one adapter package.

## User Stories

1. As a visitor, I want a confirmation email with my booking code after I register, so that I don't lose the code.
2. As a visitor, I want an email when my visit is approved, with date, time, official, room and visitor instructions, so that I know when and where to go.
3. As a visitor, I want the approval email to point me to my admission QR, so that I can show it at reception.
4. As a visitor, I want an email with the public reason when my visit is rejected, so that I understand the decision.
5. As a visitor, I want an email when staff reschedule my visit, with the new time and the reason, and a note that my old QR no longer works, so that I go at the right time.
6. As a visitor, I want an email when staff cancel my visit, with the public reason, so that I don't travel for nothing.
7. As a visitor, I want a confirmation when I cancel my own visit, so that I know it took effect.
8. As a visitor whose visit a staff member created on my behalf, I want the same approval email, so that I get the details.
9. As a visitor, I want never to receive an email for a change that was rolled back, so that emails always match reality.
10. As a visitor, I want at most one email per change even if staff double-submit or the server restarts, so that I'm not spammed.
11. As a visitor, I want an outdated email suppressed if my visit changed again before it was sent, so that I'm not misled.
12. As a restricted (Terbatas) visitor, I want no email, because no contact details are collected.
13. As a staff member, I want my action to succeed even when the mail server is down, so that email outages never block operations.
14. As an operator, I want temporary SMTP failures retried with growing waits for up to 24 hours and then flagged, so that nothing is silently lost.
15. As an operator, I want permanent failures (for example a rejected mailbox) flagged immediately, so that I can follow up.
16. As an operator, I want backlog age, attempts and failures observable without recipient addresses in logs, so that I can run it safely.
17. As an operator, I want SMTP credentials loaded from files or environment, never from the database or a mailgrid JSON file, so that secrets follow the existing configuration rules.
18. As a developer, I want a fake mailer and a local SMTP sink for tests, so that tests never send real email.

## Implementation Decisions

### Events (BR-KOM-01)

| Event | Trigger (backend ticket) | Content (public fields only) |
| --- | --- | --- |
| `submitted` | Public create (06) | Code, schedule, official, pending status, status-page link |
| `approved` | Approve (11); staff create of an ordinary visit with an email (10) | Schedule, official, room, visitor instructions, status-page link to the static QR |
| `rejected` | Reject (11) | Public rejection reason |
| `rescheduled` | Reschedule with a changed date, time or duration (12) | New schedule, public reason, "the old QR no longer works" |
| `cancelled_by_staff` | Staff cancel (11) | Public cancellation reason |
| `cancelled_by_visitor` | Public cancel (07) | Confirmation |

**Exclusions:**
- No event for restricted visits, notes-only edits, check-in or check-out.
- Internal notes, coordination notes, service IDs and staff identities never appear in any email.

### Modules (follow the [Go conventions](../../BACKEND-GO-CONVENTIONS.md))

- **Mail adapter.** An infrastructure `Mailer` interface: `Send(ctx, Message) error`. Errors are classified as temporary (connection, timeout, SMTP 4xx) or permanent (SMTP 5xx, invalid address).
  - **`mailgrid` implementation** opens a client with mailgrid's context-aware SMTP connect, then sends one prepared message with its single-client send.
  - **Not used from mailgrid:** its CLI, scheduler daemon and BoltDB job store, bulk dispatcher, CSV success/failure logs, offset files, monitor server, package-level retry settings, and JSON `--env` file.
  - **Other implementations:** a no-op mailer for development, and a fake for tests.
- **Notification module.** It follows the house layering:
  - **Repository:** outbox writes inside the caller's transaction, plus lease-based claiming with `SKIP LOCKED`.
  - **Usecase:** enqueue, and deliver-batch.
  - **Serializer:** event to template data, using only fields a visitor may see.
- **Hook into booking writes.** The booking transaction interface gains one enqueue method. Each notifying command adds one call beside its existing audit write. This is the only change to the already-built tickets.
- **Templates.** Embedded `html/template` and `text/template` files, one pair per event, in Indonesian (English to be confirmed). The subject is rendered the same way. We render; mailgrid only sends the finished body.
- **Worker.** A new `cmd/vms-worker` entrypoint from the same module and release (D08). It claims a bounded batch, sends outside any transaction, records the outcome only while it still holds the lease, and drains on shutdown.
- **Configuration.** `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD` (or `SMTP_PASSWORD_FILE`), `SMTP_FROM`, `SMTP_TLS_MODE` (STARTTLS or implicit) and an optional `SMTP_CA_FILE`. These load through the existing config loader and map in memory to mailgrid's SMTP config. TLS verification is always on (D98), and an all-absent block means the no-op mailer. A partial block fails validation, as Swagger credentials already do.
- **Recipient.** Resolved at send time from the visit's `detail_pengunjung.email`. The outbox stores no address. If the email is missing, the intent is suppressed.

### Schema (one migration, `support` schema per the earlier plan)

`support.notification_outbox`:

| Column | Meaning |
| --- | --- |
| `id` | Primary key |
| `event_key` | **Unique** logical key, e.g. `kunjungan:<id>:<event>:v<version>` (deduplication, user story 10) |
| `event` | One of the six events |
| `id_kunjungan` | FK to the visit |
| `visit_version` | Version the event describes. If the visit moved on, the worker rechecks and suppresses outdated `approved`/`rescheduled` mail (user story 11) |
| `state` | pending / sending / sent / suppressed / failed |
| `attempts` | Attempt count |
| `next_attempt_at` | When the next attempt is due |
| `deadline_at` | Set at the first failure to +24 h (D23) |
| `lease_owner`, `lease_expires_at` | Worker claim |
| `last_error_class` | Error classification; never raw SMTP text containing an address |
| `created_at`, `sent_at` | Timestamps |

No diagram table changes.

### QR in email (follows the static-QR decision)

The approval email links to the status page, where the static QR is shown. Embedding the QR image is a later option: render the static payload with piglig/go-qr v2 to PNG. Mailgrid attachments are file paths, so the adapter would need a private temporary file, deleted after the send. That is decided by the spike below, not now.

## Testing Decisions

- **Seams.** These are the existing ones: usecase with a fake mailer and fake outbox transaction; HTTP not involved; real PostgreSQL for claim, lease and dedup; an adapter test against a local SMTP sink container behind a build tag. No live email in any test (earlier plan §7).
- **Required cases:**
  - rollback writes no intent;
  - a double submit gives one intent;
  - a crash after send but before recording leaves the result uncertain, with no duplicate claim while the lease is valid;
  - the D36 schedule and jitter bounds;
  - the D23 deadline stops attempts;
  - a permanent error fails immediately;
  - an outdated version is suppressed;
  - a restricted visit gets no intent;
  - templates never contain internal fields (assert on rendered output).

## Out of Scope

- Staff-facing emails, password-reset email, bulk campaigns, open or click tracking.
- An email preview UI, and resending from the UI.
- Mailgrid's scheduler and monitor.

## Further Notes

**Mailgrid facts checked 18 November 2026** (README, pkg.go.dev, docs; not the source):
- License: BSD-3-Clause.
- The `email` package exports a context-aware SMTP connect, a single-client send and a bulk dispatcher.
- SMTP settings: host, port, username, password and from, with TLS 1.2+ and STARTTLS.
- No documented job-submission API.
- The CLI writes `success.csv`/`failed.csv` and offset files in the working directory.
- v1.1.0 was seen; a newer tag exists.

**Spike first** (throwaway `/prototype`, kept on a `prototype/mailgrid-smtp` branch). It answers:
1. The exact SMTP config fields.
2. Whether a single send performs exactly one attempt and writes **no** files. That matters for distroless containers and for keeping visitor addresses out of disk logs.
3. Whether SMTP reply codes reach the caller, so temporary and permanent failures can be told apart.
4. Whether package-level globals affect a single send.
5. The pinned version and its Go toolchain compatibility with Go 1.27.4.
6. How attachments and inline images behave, for the optional QR image.

If item 2 or 3 fails, keep this design and implement the `Mailer` with another SMTP library instead. Nothing outside the adapter changes.

**Proposed tickets** (publish with `/to-tickets` when this work starts):

| # | Ticket | Blocked by |
| --- | --- | --- |
| E1 | Spike: mailgrid single-send adapter answers the six questions | — |
| E2 | Outbox table, notification module and worker with the fake mailer (claim, lease, D23/D36, metrics) | backend tickets 06, 03 |
| E3 | Mailgrid SMTP adapter and configuration, tested against a local sink | E1, E2 |
| E4 | Enqueue the six events from booking commands, plus templates | E2; backend tickets 07, 10, 11, 12 |
| E5 | Optional: static QR image in the approval email via piglig/go-qr | E3, E4; backend ticket 08 |

Sources: [mailgrid repository](https://github.com/bravo1goingdark/mailgrid), [mailgrid email package](https://pkg.go.dev/github.com/bravo1goingdark/mailgrid/email), [piglig/go-qr](https://github.com/piglig/go-qr).
