# 05: Uploads (ops 02, 19)

**What to build:** Visitors and staff can upload a Nota Dinas (PDF) or a representative photo (JPEG/PNG). They get back a token that is bound to that purpose and valid for one hour.
- **Storage:** bytes are kept privately through a storage adapter (a local filesystem implementation for now).
- **Metadata:** document metadata lives in `form_documents`, exactly the diagram's columns plus the agreed `purpose`, without `file_data`. The upload lifecycle (reference, owner, hash, token, expiry, link state) lives in `support.form_document_uploads`.
- **Verification:** content is checked by signature and decoding, never by the client-declared MIME type.
- **Replay:** the same reference with the same content returns the same token within its validity. A reused reference with different content or purpose is a conflict.
- **Rate limits:** the D31 allowance applies, and public calls stay under the public ceiling.

Context:
- [physical schema](../../../contracts/PHYSICAL-SCHEMA-2026-11-18.md): tables 18 and 29
- [spec](../spec.md): file storage
- [field rules](../../../contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md): Reception and files; upload purpose
- decisions 21 and 22
- earlier D15, D17, D18, D31, D99

**Blocked by:** 01, 02, 03 (03 depends on 00).

**Status:** done (merge 0c9fa60 on vms/backend-api)

**Verification:** Upload route captures and PostgreSQL tests cover purpose binding, replay, shared admission and rollback.

- [ ] **Migrations:** two files, one per table, each with Up and Down:
  - [ ] `form_documents` with exactly the physical-schema columns; `storage_path` holds the private key, and purpose and MIME type must be consistent
  - [ ] `support.form_document_uploads` with the lifecycle columns
- [ ] **Request:** `POST /v1/public/attachments` and `POST /v1/staff/attachments` accept multipart `file`, `reference` (canonical UUID) and `purpose` (`document` | `visitorPhoto`). The staff route requires a ready staff session.
- [ ] **Success and conflicts:**
  - [x] 201 on first creation
  - [x] 200 with the identical token for a same-input replay
  - [x] 409 `idempotency_conflict` when the reference is reused with different input
- [ ] **Rejections:**
  - [ ] 413 over 5 MiB
  - [ ] 415 for an unsupported type
  - [ ] 422 for an empty file, invalid purpose, or content that doesn't match the purpose (e.g. a PNG sent as a document, or renamed text)
- [ ] **Response:** `{token, expiresAt, serverTime}`. Only the token hash is stored, and no storage key, path or URL appears in any response.
- [ ] **Failure handling:** a storage or database failure part-way through leaves no usable token
- [x] **Rate limit:** the 61st new upload from one IP within an hour gets 429 with `Retry-After`; replays don't count
- [x] **Integration test:** parallel uploads with the same reference produce one stored object and one token
