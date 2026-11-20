# 04: Uploads send their purpose

**What to build:** Public and staff uploads add the multipart field `purpose`: `document` for the Nota Dinas PDF, `visitorPhoto` for the representative photo. It goes alongside `file` and `reference`. No new visible control is added; the existing upload kind supplies the value.

- **Errors:** "too large" (413), "unsupported type" (415) and "content does not match" (422) are handled as clear field errors on the right upload.
- **Retry:** reuses the same `reference` with unchanged content.
- **Wrong-purpose tokens:** a booking submit rejected for a wrong-purpose token shows the error on the matching upload field.

Context:
- decision 22
- the upload-purpose part of `docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md`
- backend ticket `.scratch/backend-api-2026-11-18/issues/05-uploads.md`

**Blocked by:** None (can start immediately).

**Backend needed:** backend ticket 05.

**Status:** ready-for-agent

- [ ] Every upload request includes `purpose` matching its kind (tested on public and staff forms)
- [ ] 413, 415 and 422 responses show a field error on the right upload, and the user can pick another file
- [ ] A retried upload reuses its `reference` and accepts a 200 replay with the same token
- [ ] The preview mock backend validates `purpose` like the backend, and the public-submission and manual-appointment tests pass
