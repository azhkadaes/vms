# Visitor photo confirmation

Admin visit details show **Lihat foto** when the standard staff booking response contains `visitorPhotoAvailable: true`. The viewer opens in a dialog and loads the image on demand, with loading, retry, and image decoding error states. `visitorPhotoName` is optional metadata. Standard visit forms accept JPEG or PNG for the representative’s photo and PDF for new Nota Dinas uploads. Each file may be up to 5 MB.

The production client expects `GET /v1/staff/bookings/{id}/visitor-photo` to return JPEG or PNG bytes, at most 5 MB, under the same authenticated booking scope as the detail response. Return 404 when no photo is available and do not publish storage URLs or upload tokens. This backend route, photo storage, and the response metadata are still pending implementation alongside the existing booking/upload backend work; adding the UI does not implement those endpoints.

Local development supports the complete flow using an in-memory photo store in the preview server. Photos disappear on server restart. The visitor uploads raw image bytes to the local-only `/__preview/visitor-photos` endpoint and submits the resulting token with the booking. The preview photo read route checks existing booking scope. Public booking projections omit photo tokens and metadata, and restricted booking projections do not gain photo access.

Verification: `node --experimental-strip-types --test tests/visitor-photo.integration.mjs` covers image bytes, response parsing, public projection exclusion, out-of-scope reads, unsupported image types, and no-store responses.
