# VMS REST API contract: operation reference

Status: structured draft. Database alignment is partial, not certified.

This document uses the method/path, parameters, response and datatype organization present in SIHP Swagger and the existing SIHP-inspired VMS documentation. It does not copy SIHP business fields or response envelopes. Original uploaded SIHP Markdown was not available for exact format comparison.

See [field constraints and business rules](FRONTEND-DRIVEN-DETAILS-2026-11-16.md), [database mapping and gaps](DATABASE-MAPPING-2026-11-16.md), and [source/decision review](FRONTEND-DRIVEN-REVIEW-2026-11-16.md). All paths below include /v1. Each operation below includes synthetic JSON request/response examples and their field datatypes. Requiredness and validation are defined in the linked detailed contract. Database conflicts remain explicit in the readable-diagram mapping; examples do not certify schema compatibility.

Latest confirmed schedule rule (17 November 2026): officerId is required for all visits, including Terbatas; room is optional, including approval and rescheduling. The database owner is detail_kunjungan. Current frontend deviations require later fixes.

Confirmed room transport: writes use roomId; room dropdown options are objects with id/name/active. Response room remains a display label; roomId identifies the selection. Confirmed clearing rule: omit roomId to preserve it on existing visits; null clears it. On creation either means no room. Report filters now use room IDs under the existing room query key; options return id/name objects.

Report room filters: `room=3&room=8` selects those room IDs. Returned `filter.room` echoes IDs; `rooms` options are `{id:string,name:string}[]`. Empty string retains the existing no-filter meaning. List/export use identical filters; row `room` remains display text with `roomId` as identity.

## Common data types

Booking response models return `pejabat: {nama: string}` wherever that projection exposes the official's name. `pejabat.nama` comes from the official referenced by `detail_kunjungan.id_pejabat`. This also applies to booking list items, the `booking` object in a submission recovery response and reception booking responses. The UI displays the name under “Pejabat”. See the separate query parameter tables for the existing sort keys.

| Type | JSON representation | Notes |
| --- | --- | --- |
| ID | string | Local catalog/employee INTEGER IDs serialize to decimal strings; visit ID storage unverified |
| version/count/page | integer | Positive version/page; nonnegative counts |
| date | string | YYYY-MM-DD calendar date |
| time | string | HH:mm, WITA |
| timestamp | string | ISO timestamp with offset |
| serverTime | number | Epoch milliseconds |
| flags | boolean | Never string true/false |
| nullable field | declared type or null | Only when the response schema explicitly permits null |
| arrays | array | Typed elements; empty [] rather than invented records |

## Operations

### 01. GET /v1/public/catalogs

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/public/catalogs` |
| Authentication | Public proof where specified; no staff Bearer |
| Request / frontend use | Public registration |
| Response | officers `{id,name,vip,active,jobTitle?,directorate?,deputy?}`, rooms `{id,name,active}`, privacy notice `{version,id,en}` |
| Response datatype | Catalogs (object) |
| Success status | 200 with the response model below. |
| Key error cases | 429 rate limit or 503 catalog unavailable. |
| Database mapping | Pejabat and Ruangan, with organization joins and the published notice. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "officers": [
    {
      "id": "2",
      "name": "Pejabat Contoh",
      "vip": false,
      "active": true,
      "jobTitle": "Jabatan Contoh",
      "directorate": "Direktorat Contoh",
      "deputy": "Kedeputian Contoh"
    }
  ],
  "rooms": [
    {
      "id": "3",
      "name": "Ruang Contoh",
      "active": true
    }
  ],
  "privacyNotice": {
    "version": "notice-example",
    "id": "Contoh pemberitahuan privasi",
    "en": "Example privacy notice"
  }
}
```

| Request field | Datatype |
| --- | --- |
| `officers` | array of objects |
| `officers[].id` | string |
| `officers[].name` | string |
| `officers[].vip` | boolean |
| `officers[].active` | boolean |
| `officers[].jobTitle` | string |
| `officers[].directorate` | string |
| `officers[].deputy` | string |
| `rooms` | array of objects |
| `rooms[].id` | string |
| `rooms[].name` | string |
| `rooms[].active` | boolean |
| `privacyNotice` | object |
| `privacyNotice.version` | string |
| `privacyNotice.id` | string |
| `privacyNotice.en` | string |

### 02. POST /v1/public/attachments

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/public/attachments` |
| Authentication | Public proof where specified; no staff Bearer |
| Request / frontend use | Multipart fields file, reference and purpose for a public document or representative photo |
| Response | token, expiresAt, serverTime; distinct document/photo validation must be specified |
| Response datatype | UploadResult (object) |
| Success status | 201 for first creation. 200 for a committed replay with the same request. |
| Key error cases | 413 file too large, 415 unsupported media, 422 invalid purpose/file, or 409 reference reused with different input. |
| Database mapping | Uploads metadata and private file storage. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Multipart fields shown as JSON notation only; send actual multipart/form-data.
```json
{
  "file": "<binary file part>",
  "reference": "12345678-1234-4234-8234-123456789abc",
  "purpose": "document"
}
```

| Request field | Datatype |
| --- | --- |
| `file` | binary multipart part |
| `reference` | string |
| `purpose` | string enum: document or visitorPhoto |

Purpose is required. Example uses document; representative photo upload sends visitorPhoto. Validate content and token purpose; response shape is unchanged.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "token": "<opaque-upload-token>",
  "expiresAt": "2026-11-16T11:00:00+08:00",
  "serverTime": 1794794400000
}
```

| Response field | Datatype |
| --- | --- |
| `token` | string |
| `expiresAt` | string |
| `serverTime` | integer |

### 03. POST /v1/public/bookings

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/public/bookings` |
| Authentication | Public proof where specified; no staff Bearer |
| Request / frontend use | Public registration; representative/contact, officerId,date,time,purpose,identityType,visitors,optional ND token, required photo token, privacy acknowledgment/version, idempotencyKey |
| Response | Public booking receipt with code and status; roster includes representative first |
| Response datatype | PublicBooking (object) |
| Success status | 201 for first creation. 200 for a committed replay with the same request. |
| Key error cases | 422 validation or unusable upload token, 409 changed idempotency payload or stale notice, or 503 notice unavailable. |
| Database mapping | Kunjungan and both unique detail references, with Uploads and durable submission support. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "officerId": "2",
  "date": "2026-11-17",
  "time": "10:00",
  "purpose": "Pertemuan koordinasi",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "identityType": "KTP",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "visitorPhotoName": "representative.png",
  "visitorPhotoToken": "<opaque-upload-token>",
  "ack": true,
  "privacyVersion": "notice-example",
  "idempotencyKey": "12345678-1234-4234-8234-123456789abc"
}
```

| Request field | Datatype |
| --- | --- |
| `officerId` | string |
| `date` | string |
| `time` | string |
| `purpose` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `identityType` | string |
| `visitors` | string[] |
| `visitorPhotoName` | string |
| `visitorPhotoToken` | string |
| `ack` | boolean |
| `privacyVersion` | string |
| `idempotencyKey` | string |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1794794400000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": true
  },
  "history": [],
  "roomId": null
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array (element type in model) |
| `roomId` | string ID or null |

### 04. POST /v1/public/submissions/resolve

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/public/submissions/resolve` |
| Authentication | Public proof where specified; no staff Bearer |
| Request / frontend use | Uncertain submission; idempotencyKey,email |
| Response | Submission resolution; prevents duplicate registration after a lost response |
| Response datatype | SubmissionResolution (discriminated object) |
| Success status | 200 with the response model below. |
| Key error cases | Unavailable/mismatched proof must disclose no other visitor’s booking. Preserve the uncertain client state on failure. |
| Database mapping | Durable public submission reference and authorized booking projection. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "idempotencyKey": "12345678-1234-4234-8234-123456789abc",
  "email": "visitor@example.test"
}
```

| Request field | Datatype |
| --- | --- |
| `idempotencyKey` | string |
| `email` | string |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "outcome": "committed",
  "booking": {
    "id": "101",
    "code": "00123456",
    "name": "Pengunjung Contoh",
    "organization": "Instansi Contoh",
    "email": "visitor@example.test",
    "phone": "081234567890",
    "officerId": "2",
    "pejabat": {
      "nama": "Pejabat Contoh"
    },
    "vip": false,
    "date": "2026-11-17",
    "time": "10:00",
    "duration": 60,
    "room": "",
    "purpose": "Pertemuan koordinasi",
    "visitors": [
      "Pengunjung Contoh"
    ],
    "identityType": "KTP",
    "status": "pending",
    "version": 1,
    "timing": {
      "serverTime": 1794794400000,
      "opensAt": 1794879000000,
      "endsAt": 1794884400000,
      "displayStatus": "pending",
      "canCheckIn": false,
      "canCheckOut": false,
      "canCancel": true
    },
    "history": [],
    "roomId": null
  }
}
```

| Response field | Datatype |
| --- | --- |
| `outcome` | string |
| `booking` | object |
| `booking.id` | string |
| `booking.code` | string |
| `booking.name` | string |
| `booking.organization` | string |
| `booking.email` | string |
| `booking.phone` | string |
| `booking.officerId` | string |
| `booking.pejabat` | object |
| `booking.pejabat.nama` | string |
| `booking.vip` | boolean |
| `booking.date` | string |
| `booking.time` | string |
| `booking.duration` | integer |
| `booking.room` | string |
| `booking.purpose` | string |
| `booking.visitors` | string[] |
| `booking.identityType` | string |
| `booking.status` | string |
| `booking.version` | integer |
| `booking.timing` | object |
| `booking.timing.serverTime` | integer |
| `booking.timing.opensAt` | integer |
| `booking.timing.endsAt` | integer |
| `booking.timing.displayStatus` | string |
| `booking.timing.canCheckIn` | boolean |
| `booking.timing.canCheckOut` | boolean |
| `booking.timing.canCancel` | boolean |
| `booking.history` | array (element type in model) |
| `booking.roomId` | string ID or null |

### 05. POST /v1/public/lookup

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/public/lookup` |
| Authentication | Public proof where specified; no staff Bearer |
| Request / frontend use | code,email |
| Response | Public detail, schedule, timing, permitted history and reasons |
| Response datatype | PublicBooking (object) |
| Success status | 200 with the response model below. |
| Key error cases | 404 unavailable booking or mismatched code/email. |
| Database mapping | Kunjungan and its two detail records. Commands persist the new version and applicable audit events. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "code": "00123456",
  "email": "visitor@example.test"
}
```

| Request field | Datatype |
| --- | --- |
| `code` | string |
| `email` | string |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1794794400000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": true
  },
  "history": [],
  "roomId": null
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array (element type in model) |
| `roomId` | string ID or null |

### 06. POST /v1/public/bookings/{id}/cancel

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/public/bookings/{id}/cancel` |
| Authentication | Public proof where specified; no staff Bearer |
| Request / frontend use | code,email,version |
| Response | Updated public booking; visible cancellation eligibility |
| Response datatype | PublicBooking (object) |
| Success status | 200 with the response model below. |
| Key error cases | 404 unavailable proof, 409 stale version or cancellation no longer allowed. |
| Database mapping | Kunjungan and its two detail records. Commands persist the new version and applicable audit events. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "code": "00123456",
  "email": "visitor@example.test",
  "version": 1
}
```

| Request field | Datatype |
| --- | --- |
| `code` | string |
| `email` | string |
| `version` | integer |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "cancelled",
  "version": 2,
  "timing": {
    "serverTime": 1794794400000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "cancelled",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "history": [],
  "roomId": null
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array (element type in model) |
| `roomId` | string ID or null |

### 07. POST /v1/public/bookings/{id}/admission

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/public/bookings/{id}/admission` |
| Authentication | Public proof where specified; no staff Bearer |
| Request / frontend use | code,email,version |
| Response | QR payload, expiresAt,serverTime |
| Response datatype | Admission (object) |
| Success status | 200 with the response model below. |
| Key error cases | 404 unavailable proof, 409 stale version or QR issuance not allowed. |
| Database mapping | QR Kunjungan plus current Kunjungan status and detail_kunjungan schedule. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "code": "00123456",
  "email": "visitor@example.test",
  "version": 1
}
```

| Request field | Datatype |
| --- | --- |
| `code` | string |
| `email` | string |
| `version` | integer |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "payload": "<opaque-admission-credential>",
  "expiresAt": "2026-11-17T10:05:00+08:00",
  "serverTime": 1794880800000
}
```

| Response field | Datatype |
| --- | --- |
| `payload` | string |
| `expiresAt` | string |
| `serverTime` | integer |

### 08. GET /v1/staff/catalogs

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/catalogs` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Staff destination/room controls |
| Response | officers `{id,name,vip,active,jobTitle?,directorate?,deputy?}`, rooms `{id,name,active}`, privacy notice `{version,id,en}`. Scope per A-15 from the live role's permissions; the client-selected role is not authority |
| Response datatype | Catalogs (object) |
| Success status | 200 with the response model below. |
| Key error cases | 401 invalid session; 403 when the session holds neither bookings nor people, or a role selection or password change is pending. |
| Database mapping | Scoped Pejabat and Ruangan. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "officers": [
    {
      "id": "2",
      "name": "Pejabat Contoh",
      "vip": false,
      "active": true,
      "jobTitle": "Jabatan Contoh",
      "directorate": "Direktorat Contoh",
      "deputy": "Kedeputian Contoh"
    }
  ],
  "rooms": [
    {
      "id": "3",
      "name": "Ruang Contoh",
      "active": true
    }
  ],
  "privacyNotice": {
    "version": "notice-example",
    "id": "Contoh pemberitahuan privasi",
    "en": "Example privacy notice"
  }
}
```

| Request field | Datatype |
| --- | --- |
| `officers` | array of objects |
| `officers[].id` | string |
| `officers[].name` | string |
| `officers[].vip` | boolean |
| `officers[].active` | boolean |
| `officers[].jobTitle` | string |
| `officers[].directorate` | string |
| `officers[].deputy` | string |
| `rooms` | array of objects |
| `rooms[].id` | string |
| `rooms[].name` | string |
| `rooms[].active` | boolean |
| `privacyNotice` | object |
| `privacyNotice.version` | string |
| `privacyNotice.id` | string |
| `privacyNotice.en` | string |

### 09. GET /v1/staff/bookings

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/bookings` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | view,q,status,sort,dir,date,vip,page,pageSize,mode |
| Response | `{items,total,totalScoped,page,pageSize,metrics?}`; dashboard requires metrics |
| Response datatype | StaffBookingPage (object) |
| Success status | 200 with the response model below. |
| Key error cases | 401 invalid session, 403 view denied, 422 invalid filter. |
| Database mapping | Scoped Kunjungan and its detail records. Dashboard metrics are computed. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `view` | query string enum | list, dashboard or reception |
| `date` | query date string | WITA date for today, dashboard or check-in. Checkout can span dates. |
| `mode` | query string enum | checkin or checkout when view=reception |
| `q` | query string | Operational search |
| `status` | query string enum | Scoped status selection or all |
| `vip` | query string enum | 1 selects VIP ordinary visits. 0 applies no VIP filter. |
| `sort` | query string enum | priority, name, organization, host, time or status |
| `dir` | query string enum | asc or desc |
| `page` | query integer | Positive page number |
| `pageSize` | query integer | 1–100 |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "items": [],
  "total": 0,
  "totalScoped": 0,
  "page": 1,
  "pageSize": 50,
  "metrics": {
    "today": 0,
    "pending": 0,
    "arrivals": 0,
    "inside": 0,
    "completed": 0
  }
}
```

| Request field | Datatype |
| --- | --- |
| `items` | array (element type in model) |
| `total` | integer |
| `totalScoped` | integer |
| `page` | integer |
| `pageSize` | integer |
| `metrics` | object |
| `metrics.today` | integer |
| `metrics.pending` | integer |
| `metrics.arrivals` | integer |
| `metrics.inside` | integer |
| `metrics.completed` | integer |

### 10. GET /v1/staff/bookings/{id}

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/bookings/{id}` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Authorized detail |
| Response | Role-projected staff booking, version, timing, permitted notes/history |
| Response datatype | StaffBooking (object; authorization projection) |
| Success status | 200 with the response model below. |
| Key error cases | 401 invalid session, 404 unavailable or outside permitted detail scope. |
| Database mapping | Kunjungan and its two detail records. Commands persist the new version and applicable audit events. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "pending",
  "version": 1,
  "timing": {
    "serverTime": 1794787200000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "pending",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": true
  },
  "history": [],
  "roomId": "3"
}
```

| Request field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array (element type in model) |
| `roomId` | string ID or null |

### 11. POST /v1/staff/bookings

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Standard or Terbatas appointment input plus idempotencyKey. Every visit requires officerId. Optional roomId identifies the selected room. |
| Response | Created staff booking; different input shapes, not unrelated visitor creation |
| Response datatype | StaffBooking (object; authorization projection) |
| Success status | 201 for first creation. 200 for a committed replay with the same request. |
| Key error cases | 403 creation outside scope, 422 invalid fields/upload, 409 schedule or idempotency conflict. |
| Database mapping | Kunjungan and its two detail records. Commands persist the new version and applicable audit events. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "officerId": "2",
  "date": "2026-11-17",
  "time": "10:00",
  "purpose": "Pertemuan koordinasi",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "identityType": "KTP",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "visitorPhotoName": "representative.png",
  "visitorPhotoToken": "<opaque-upload-token>",
  "idempotencyKey": "12345678-1234-4234-8234-123456789abc",
  "duration": 60,
  "visitKind": "standard",
  "roomId": "3"
}
```

| Request field | Datatype |
| --- | --- |
| `officerId` | string |
| `date` | string |
| `time` | string |
| `purpose` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `identityType` | string |
| `visitors` | string[] |
| `visitorPhotoName` | string |
| `visitorPhotoToken` | string |
| `idempotencyKey` | string |
| `duration` | integer |
| `visitKind` | string |
| `roomId` | string ID or null |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "approved",
  "version": 1,
  "timing": {
    "serverTime": 1794794400000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "approved",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": true
  },
  "history": [],
  "roomId": "3"
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array (element type in model) |
| `roomId` | string ID or null |

### 12. POST /v1/staff/bookings/{id}/approve

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings/{id}/approve` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | version, duration, optional roomId and appointment notes |
| Response | Updated booking; enforce schedule/room conflicts atomically |
| Response datatype | StaffBooking (object; authorization projection) |
| Success status | 200 with the response model below. |
| Key error cases | 404 unavailable record, 409 stale version/state or schedule conflict, 422 invalid duration/room. |
| Database mapping | Kunjungan and its two detail records. Commands persist the new version and applicable audit events. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "duration": 60,
  "visitorInstructions": "",
  "coordinationNote": "",
  "version": 1,
  "roomId": "3"
}
```

| Request field | Datatype |
| --- | --- |
| `duration` | integer |
| `visitorInstructions` | string |
| `coordinationNote` | string |
| `version` | integer |
| `roomId` | string ID or null |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "approved",
  "version": 2,
  "timing": {
    "serverTime": 1794794400000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "approved",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": true
  },
  "history": [],
  "roomId": "3"
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array (element type in model) |
| `roomId` | string ID or null |

### 13. POST /v1/staff/bookings/{id}/reject

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings/{id}/reject` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | rejectionReason,internalRejectionNote,version |
| Response | Updated booking; separate public reason from internal note |
| Response datatype | StaffBooking (object; authorization projection) |
| Success status | 200 with the response model below. |
| Key error cases | 404 unavailable record, 409 stale version/state, 422 invalid reason. |
| Database mapping | Kunjungan and its two detail records. Commands persist the new version and applicable audit events. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "rejectionReason": "Jadwal belum tersedia",
  "internalRejectionNote": "",
  "version": 1
}
```

| Request field | Datatype |
| --- | --- |
| `rejectionReason` | string |
| `internalRejectionNote` | string |
| `version` | integer |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "rejected",
  "version": 2,
  "timing": {
    "serverTime": 1794794400000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "rejected",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "history": [],
  "roomId": "3"
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array (element type in model) |
| `roomId` | string ID or null |

### 14. POST /v1/staff/bookings/{id}/reschedule

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings/{id}/reschedule` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | version, date, time, duration, reason, optional roomId and appointment notes |
| Eligibility | Approved only. When display status is Missed, the current server time must be at or before the original scheduled end plus seven elapsed days. |
| Response | Updated Approved visit with rescheduled set to true, timing based on its new schedule and incremented version. Prior admission credentials are revoked. |
| Response datatype | StaffBooking (object; authorization projection) |
| Success status | 200 with the response model below. |
| Key error cases | 404 unavailable record. 409 stale version, ineligible status, expired reschedule window or schedule conflict. 422 invalid replacement fields. |
| Database mapping | Read eligibility from Kunjungan and the original detail_kunjungan schedule. Replace the schedule, increment version, revoke prior admission credentials and append an audit event in the same transaction. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

In this example, the original Approved visit ended at 11:00 WITA on 17 November. An authorized staff member submits the change at 11:00 WITA on 24 November, exactly at the cutoff. The replacement starts at 10:00 WITA on 25 November.
```json
{
  "date": "2026-11-25",
  "time": "10:00",
  "duration": 60,
  "reason": "Perubahan agenda pejabat",
  "visitorInstructions": "",
  "coordinationNote": "",
  "version": 1,
  "roomId": "3"
}
```

| Request field | Datatype |
| --- | --- |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `reason` | string |
| `visitorInstructions` | string |
| `coordinationNote` | string |
| `version` | integer |
| `roomId` | string ID or null |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-25",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "approved",
  "version": 2,
  "timing": {
    "serverTime": 1795489200000,
    "opensAt": 1795570200000,
    "endsAt": 1795575600000,
    "displayStatus": "approved",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": true
  },
  "history": [
    {
      "at": "2026-11-24T11:00:00+08:00",
      "message": "Jadwal ditetapkan ulang: Perubahan agenda pejabat"
    }
  ],
  "rescheduled": true,
  "roomId": "3"
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array of objects |
| `history[].at` | timestamp string |
| `history[].message` | string |
| `rescheduled` | boolean |
| `roomId` | string ID or null |

### 15. POST /v1/staff/bookings/{id}/cancel

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings/{id}/cancel` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | reason,internalCancellationNote,version |
| Response | Updated booking; scope and lifecycle checked server-side |
| Response datatype | StaffBooking (object; authorization projection) |
| Success status | 200 with the response model below. |
| Key error cases | 404 unavailable record, 409 stale version/state, 422 invalid reason. |
| Database mapping | Kunjungan and its two detail records. Commands persist the new version and applicable audit events. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "reason": "Permintaan dibatalkan petugas",
  "internalCancellationNote": "",
  "version": 1
}
```

| Request field | Datatype |
| --- | --- |
| `reason` | string |
| `internalCancellationNote` | string |
| `version` | integer |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "code": "00123456",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "email": "visitor@example.test",
  "phone": "081234567890",
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "purpose": "Pertemuan koordinasi",
  "visitors": [
    "Pengunjung Contoh"
  ],
  "identityType": "KTP",
  "status": "cancelled",
  "version": 2,
  "timing": {
    "serverTime": 1794794400000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "cancelled",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "history": [],
  "roomId": "3"
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `code` | string |
| `name` | string |
| `organization` | string |
| `email` | string |
| `phone` | string |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `purpose` | string |
| `visitors` | string[] |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `history` | array (element type in model) |
| `roomId` | string ID or null |

### 16. POST /v1/staff/bookings/{id}/update

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings/{id}/update` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | version, required officerId and editable Terbatas details. Optional roomId uses the preservation rule. |
| Response | Updated permitted restricted projection |
| Response datatype | StaffBooking (object; authorization projection) |
| Success status | 200 with the response model below. |
| Key error cases | 404 unavailable record, 409 stale version/state or schedule conflict, 422 invalid detail fields. |
| Database mapping | Kunjungan and its two detail records. Commands persist the new version and applicable audit events. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "visitKind": "restricted",
  "date": "2026-11-17",
  "time": "10:00",
  "name": "",
  "organization": "",
  "officerId": "2",
  "serviceId": "",
  "visitorInstructions": "",
  "coordinationNote": "",
  "version": 1,
  "roomId": null
}
```

| Request field | Datatype |
| --- | --- |
| `visitKind` | string |
| `date` | string |
| `time` | string |
| `name` | string |
| `organization` | string |
| `officerId` | string |
| `serviceId` | string |
| `visitorInstructions` | string |
| `coordinationNote` | string |
| `version` | integer |
| `roomId` | string ID or null |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "",
  "status": "approved",
  "version": 2,
  "timing": {
    "serverTime": 1794794400000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "approved",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": true
  },
  "visitKind": "restricted",
  "reference": "TRB-101",
  "restrictedView": "full",
  "roomId": null,
  "officerId": "2",
  "pejabat": {
    "nama": "Pejabat Contoh"
  }
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `visitKind` | string |
| `reference` | string |
| `restrictedView` | string |
| `roomId` | string ID or null |
| `officerId` | string |
| `pejabat` | object |
| `pejabat.nama` | string |

### 17. POST /v1/staff/resolve

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/resolve` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | credential from code or QR |
| Response | Eligible role-projected record for reception confirmation |
| Response datatype | StaffBooking (object; authorization projection) |
| Success status | 200 with the response model below. |
| Key error cases | 403 reception access denied or 404 unavailable booking/proof. |
| Database mapping | Kunjungan and QR Kunjungan with the authorized arrival projection. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "credential": "00123456"
}
```

| Request field | Datatype |
| --- | --- |
| `credential` | string |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "101",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "identityType": "KTP",
  "status": "approved",
  "version": 1,
  "timing": {
    "serverTime": 1794881100000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "approved",
    "canCheckIn": true,
    "canCheckOut": false,
    "canCancel": false
  },
  "roomId": "3",
  "visitorCount": 1,
  "visitorPhotoAvailable": true
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `name` | string |
| `organization` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `roomId` | string ID or null |
| `visitorCount` | integer |
| `visitorPhotoAvailable` | boolean |

### 18. POST /v1/staff/bookings/{id}/attendance

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings/{id}/attendance` |
| Authentication | Bearer. Enforce the selected action's reception permission, visit scope and matching credential. |
| Request / frontend use | The operator confirms the visitor's identity, then the Check in or Check out button submits its explicit action, credential and current booking version. |
| Response | Updated authorized arrival projection with the resulting status and incremented version. |
| Response datatype | StaffBooking (object, arrival projection) |
| Success status | 200 |
| Key error cases | 401 invalid session. 403 action forbidden. 404 unavailable visit or proof. 409 stale version or ineligible transition, including a repeated check-in or checkout. 422 missing or invalid action, credential or version. |
| Database mapping | Update Kunjungan status, matching attendance timestamp and version, and append the audit event atomically. Apply QR consumption rules for check-in. |

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required visit ID. |
| `action` | JSON string enum | Required. `check_in` or `check_out`. |
| `credential` | JSON string | Required. Manual eight-digit code or matching opaque admission QR. |
| `version` | JSON integer | Required positive expected booking version. |

| Action | Required permission | Eligibility | Effect |
| --- | --- | --- | --- |
| `check_in` | `checkin` | Stored `approved` and within the inclusive check-in window. QR must be valid for admission. | Move to `checked_in`. Record server arrival time and consume the QR for admission if used. |
| `check_out` | `checkout` | Stored `checked_in`, including visits from earlier days. A matching, nonrevoked consumed QR can be used after admission expiry according to the approved checkout rule. | Move to `checked_out` and record server departure time. |

`action` selects exactly one transition. Status is a server-managed response field. A stale retry cannot choose or perform the opposite action. Credential resolution through `POST /v1/staff/resolve` remains a separate read step before identity confirmation. See [attendance rules](FRONTEND-DRIVEN-DETAILS-2026-11-16.md#reception-and-files).

**Check-in example request**

```json
{
  "action": "check_in",
  "credential": "00123456",
  "version": 1
}
```

**Check-in example response**

```json
{
  "id": "101",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "identityType": "KTP",
  "status": "checked_in",
  "version": 2,
  "timing": {
    "serverTime": 1794881100000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "checked_in",
    "canCheckIn": false,
    "canCheckOut": true,
    "canCancel": false
  },
  "roomId": "3",
  "visitorCount": 1
}
```

**Checkout example request**

The operator uses the current version from the check-in result. This example checks out the next day in WITA.

```json
{
  "action": "check_out",
  "credential": "00123456",
  "version": 2
}
```

**Checkout example response**

```json
{
  "id": "101",
  "name": "Pengunjung Contoh",
  "organization": "Instansi Contoh",
  "pejabat": {
    "nama": "Pejabat Contoh"
  },
  "vip": false,
  "date": "2026-11-17",
  "time": "10:00",
  "duration": 60,
  "room": "Ruang Contoh",
  "identityType": "KTP",
  "status": "checked_out",
  "version": 3,
  "timing": {
    "serverTime": 1794967200000,
    "opensAt": 1794879000000,
    "endsAt": 1794884400000,
    "displayStatus": "checked_out",
    "canCheckIn": false,
    "canCheckOut": false,
    "canCancel": false
  },
  "roomId": "3",
  "visitorCount": 1
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `name` | string |
| `organization` | string |
| `pejabat` | object |
| `pejabat.nama` | string |
| `vip` | boolean |
| `date` | string |
| `time` | string |
| `duration` | integer |
| `room` | string |
| `identityType` | string |
| `status` | string |
| `version` | integer |
| `timing` | object |
| `timing.serverTime` | integer |
| `timing.opensAt` | integer |
| `timing.endsAt` | integer |
| `timing.displayStatus` | string |
| `timing.canCheckIn` | boolean |
| `timing.canCheckOut` | boolean |
| `timing.canCancel` | boolean |
| `roomId` | string ID or null |
| `visitorCount` | integer |

### 19. POST /v1/staff/attachments

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/attachments` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Multipart fields file, reference and purpose for a staff document or representative photo |
| Response | Upload token/expiry/server time |
| Response datatype | UploadResult (object) |
| Success status | 201 for first creation. 200 for a committed replay with the same request. |
| Key error cases | 403 upload denied, 413 file too large, 415 unsupported media, 422 invalid purpose/file or 409 reference reuse conflict. |
| Database mapping | Uploads metadata and private file storage. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Multipart fields shown as JSON notation only; send actual multipart/form-data.
```json
{
  "file": "<binary file part>",
  "reference": "12345678-1234-4234-8234-123456789abc",
  "purpose": "document"
}
```

| Request field | Datatype |
| --- | --- |
| `file` | binary multipart part |
| `reference` | string |
| `purpose` | string enum: document or visitorPhoto |

Purpose is required. Example uses document; representative photo upload sends visitorPhoto. Validate content and token purpose; response shape is unchanged.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "token": "<opaque-upload-token>",
  "expiresAt": "2026-11-16T11:00:00+08:00",
  "serverTime": 1794794400000
}
```

| Response field | Datatype |
| --- | --- |
| `token` | string |
| `expiresAt` | string |
| `serverTime` | integer |

### 20. GET /v1/staff/bookings/{id}/attachment

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/bookings/{id}/attachment` |
| Authentication | Bearer and ordinary staff file-read scope. Reception uses the credential-bearing POST operation. |
| Request / frontend use | Permitted document display/download |
| Response | Validated binary; reception proof binding unresolved |
| Response datatype | binary file (not JSON) |
| Success status | 200 with the authorized file bytes. |
| Key error cases | 404 unavailable file or visit outside the permitted procedure. |
| Database mapping | Uploads and the owning visit reference. Stream bytes from private storage. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Binary PDF for attachment, or JPEG/PNG for visitor photo. Do not wrap file bytes in JSON.

### 21. GET /v1/staff/bookings/{id}/visitor-photo

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/bookings/{id}/visitor-photo` |
| Authentication | Bearer and ordinary staff file-read scope. Reception uses the credential-bearing POST operation. |
| Request / frontend use | Permitted photo display |
| Response | Image binary; reception proof binding unresolved |
| Response datatype | binary file (not JSON) |
| Success status | 200 with the authorized file bytes. |
| Key error cases | 404 unavailable photo or visit outside the permitted procedure. |
| Database mapping | Uploads and the owning visit reference. Stream bytes from private storage. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Binary PDF for attachment, or JPEG/PNG for visitor photo. Do not wrap file bytes in JSON.

### 22. GET /v1/staff/reports/summary

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/reports/summary` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | from,to,status |
| Response | filter,scope,generatedAt,canExport,revision,counts,total |
| Response datatype | VisitSummary (object) |
| Success status | 200 with the response model below. |
| Key error cases | 403 report denied or 422 invalid filters. |
| Database mapping | Authorized visit/detail joins and report aggregation or snapshot projection. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `from` | query date string | Inclusive first date |
| `to` | query date string | Inclusive last date. Up to 366 days with from. |
| `status` | query string or repeated strings | all or selected display statuses |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "filter": {
    "from": "2026-11-01",
    "to": "2026-11-30",
    "status": "all"
  },
  "scope": "assigned",
  "generatedAt": "2026-11-16T02:00:00Z",
  "canExport": true,
  "revision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "counts": {
    "pending": 0,
    "approved": 0,
    "rejected": 0,
    "cancelled": 0,
    "checked_in": 0,
    "checked_out": 0,
    "missed": 0
  },
  "total": 0
}
```

| Request field | Datatype |
| --- | --- |
| `filter` | object |
| `filter.from` | string |
| `filter.to` | string |
| `filter.status` | string |
| `scope` | string |
| `generatedAt` | string |
| `canExport` | boolean |
| `revision` | string |
| `counts` | object |
| `counts.pending` | integer |
| `counts.approved` | integer |
| `counts.rejected` | integer |
| `counts.cancelled` | integer |
| `counts.checked_in` | integer |
| `counts.checked_out` | integer |
| `counts.missed` | integer |
| `total` | integer |

### 23. GET /v1/staff/reports/export

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/reports/export` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Same filters plus revision |
| Response | filename,csv,revision; separate export permission |
| Response datatype | SummaryExport (object) |
| Success status | 200 with the response model below. |
| Key error cases | 403 export denied, 409 revision changed or 422 invalid filters. |
| Database mapping | Authorized visit/detail joins and report aggregation or snapshot projection. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `from` | query date string | Inclusive first date |
| `to` | query date string | Inclusive last date. Up to 366 days with from. |
| `status` | query string or repeated strings | all or selected display statuses |
| `revision` | query string | 64 lowercase hex characters from summary read |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "filename": "kunjungan-ringkasan-2026-11-01-2026-11-30.csv",
  "csv": "status,jumlah\npending,0\n",
  "revision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
```

| Request field | Datatype |
| --- | --- |
| `filename` | string |
| `csv` | string |
| `revision` | string |

### 24. GET /v1/staff/reports/visits

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/reports/visits` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Visit log filters, including room and officer IDs, plus sorting and pagination |
| Response | Filtered rows, summary,total,paging,revision,generatedAt,canExport,officers,rooms |
| Response datatype | VisitLogResponse (object) |
| Success status | 200 with the response model below. |
| Key error cases | 403 visit log denied or 422 invalid filters. |
| Database mapping | Authorized visit/detail joins and report aggregation or snapshot projection. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `from / to` | query date strings | Optional inclusive bounds |
| `q` | query string | Search authorized report projection |
| `status` | query string or repeated strings | all or selected display statuses |
| `kind` | query string or repeated strings | all or selected standard, vip, restricted |
| `officer` | query string or repeated strings | Officer IDs |
| `room` | query string or repeated strings | Room IDs |
| `sort` | query string enum | date, name, organization, host, status, kind or visitors |
| `dir` | query string enum | asc or desc |
| `page` | query integer | Positive page number |
| `pageSize` | query integer | Positive page size |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "filter": {
    "from": "",
    "to": "",
    "q": "",
    "status": "all",
    "kind": "all",
    "officer": "",
    "room": "",
    "sort": "date",
    "dir": "desc"
  },
  "items": [],
  "summary": {
    "total": 0,
    "visitors": 0,
    "inside": 0,
    "completed": 0,
    "pending": 0
  },
  "generatedAt": "2026-11-16T02:00:00Z",
  "revision": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "page": 1,
  "pageSize": 25,
  "total": 0,
  "canExport": true,
  "officers": [],
  "rooms": []
}
```

| Request field | Datatype |
| --- | --- |
| `filter` | object |
| `filter.from` | string |
| `filter.to` | string |
| `filter.q` | string |
| `filter.status` | string |
| `filter.kind` | string |
| `filter.officer` | string |
| `filter.room` | string |
| `filter.sort` | string |
| `filter.dir` | string |
| `items` | array (element type in model) |
| `summary` | object |
| `summary.total` | integer |
| `summary.visitors` | integer |
| `summary.inside` | integer |
| `summary.completed` | integer |
| `summary.pending` | integer |
| `generatedAt` | string |
| `revision` | string |
| `page` | integer |
| `pageSize` | integer |
| `total` | integer |
| `canExport` | boolean |
| `officers` | array (element type in model) |
| `rooms` | array (element type in model) |

### 25. GET /v1/staff/reports/visits/export

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/reports/visits/export` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Same Visit Log filters and sort as the read, plus revision. Omit pagination. |
| Response | Complete matching snapshot; UI creates downloadable report |
| Response datatype | VisitLogSnapshot (object) |
| Success status | 200 with the response model below. |
| Key error cases | 403 export denied, 409 revision changed or 422 invalid filters. |
| Database mapping | Authorized visit/detail joins and report aggregation or snapshot projection. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `from / to` | query date strings | Optional inclusive bounds |
| `q` | query string | Search authorized report projection |
| `status` | query string or repeated strings | all or selected display statuses |
| `kind` | query string or repeated strings | all or selected standard, vip, restricted |
| `officer` | query string or repeated strings | Officer IDs |
| `room` | query string or repeated strings | Room IDs |
| `sort` | query string enum | date, name, organization, host, status, kind or visitors |
| `dir` | query string enum | asc or desc |
| `revision` | query string | 64 lowercase hex characters from visit-log read |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "filter": {
    "from": "",
    "to": "",
    "q": "",
    "status": "all",
    "kind": "all",
    "officer": "",
    "room": "",
    "sort": "date",
    "dir": "desc"
  },
  "items": [],
  "summary": {
    "total": 0,
    "visitors": 0,
    "inside": 0,
    "completed": 0,
    "pending": 0
  },
  "generatedAt": "2026-11-16T02:00:00Z",
  "revision": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
}
```

| Request field | Datatype |
| --- | --- |
| `filter` | object |
| `filter.from` | string |
| `filter.to` | string |
| `filter.q` | string |
| `filter.status` | string |
| `filter.kind` | string |
| `filter.officer` | string |
| `filter.room` | string |
| `filter.sort` | string |
| `filter.dir` | string |
| `items` | array (element type in model) |
| `summary` | object |
| `summary.total` | integer |
| `summary.visitors` | integer |
| `summary.inside` | integer |
| `summary.completed` | integer |
| `summary.pending` | integer |
| `generatedAt` | string |
| `revision` | string |

### 26. GET /v1/staff/employees

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/employees` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | q,page,pageSize |
| Response | items,total,totalAll,page,pageSize,passwordMinLength |
| Response datatype | EmployeePage (object) |
| Success status | 200 with the response model below. |
| Key error cases | 403 employee list denied or 422 invalid paging. |
| Database mapping | Pegawai, Role Pegawai and Pegawai Pejabat. Credential data is server-only. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `q` | query string | Employee search |
| `page` | query integer | Positive page number |
| `pageSize` | query integer | 1–250 |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "items": [
    {
      "id": "7",
      "name": "Petugas Contoh",
      "email": "staff@example.test",
      "roles": [
        "PA"
      ],
      "active": true,
      "assignedOfficerIds": [
        "2"
      ],
      "version": 1
    }
  ],
  "total": 1,
  "totalAll": 1,
  "page": 1,
  "pageSize": 25,
  "passwordMinLength": 8
}
```

| Request field | Datatype |
| --- | --- |
| `items` | array of objects |
| `items[].id` | string |
| `items[].name` | string |
| `items[].email` | string |
| `items[].roles` | string[] |
| `items[].active` | boolean |
| `items[].assignedOfficerIds` | string[] |
| `items[].version` | integer |
| `total` | integer |
| `totalAll` | integer |
| `page` | integer |
| `pageSize` | integer |
| `passwordMinLength` | integer |

### 27. GET /v1/staff/employees/{id}

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/employees/{id}` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Employee detail/editor |
| Response | Employee profile,roles,assignment,active,version |
| Response datatype | Employee (object) |
| Success status | 200 with the response model below. |
| Key error cases | 403 employee read denied or 404 unavailable employee. |
| Database mapping | Pegawai, Role Pegawai and Pegawai Pejabat. Credential data is server-only. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "7",
  "name": "Petugas Contoh",
  "email": "staff@example.test",
  "roles": [
    "PA"
  ],
  "active": true,
  "assignedOfficerIds": [
    "2"
  ],
  "version": 1
}
```

| Request field | Datatype |
| --- | --- |
| `id` | string |
| `name` | string |
| `email` | string |
| `roles` | string[] |
| `active` | boolean |
| `assignedOfficerIds` | string[] |
| `version` | integer |

### 28. POST /v1/staff/employees

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/employees` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Employee form and initial password policy |
| Response | Created employee; no password returned |
| Response datatype | Employee (object) |
| Success status | 201 with the created employee. Proposed status for this draft. |
| Key error cases | 403 employee creation denied or 422 invalid form fields. Duplicate-email outcome remains under review. |
| Database mapping | Pegawai, Role Pegawai and Pegawai Pejabat. Credential data is server-only. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "name": "Petugas Contoh",
  "email": "staff@example.test",
  "roles": [
    "PA"
  ],
  "active": true,
  "assignedOfficerIds": [
    "2"
  ],
  "initialPassword": "<administrator-supplied-password>",
  "requirePasswordChange": true
}
```

| Request field | Datatype |
| --- | --- |
| `name` | string |
| `email` | string |
| `roles` | string[] |
| `active` | boolean |
| `assignedOfficerIds` | string[] |
| `initialPassword` | string |
| `requirePasswordChange` | boolean |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "7",
  "name": "Petugas Contoh",
  "email": "staff@example.test",
  "roles": [
    "PA"
  ],
  "active": true,
  "assignedOfficerIds": [
    "2"
  ],
  "version": 1
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `name` | string |
| `email` | string |
| `roles` | string[] |
| `active` | boolean |
| `assignedOfficerIds` | string[] |
| `version` | integer |

### 29. PATCH /v1/staff/employees/{id}

| Property | Contract |
| --- | --- |
| Method | PATCH |
| Endpoint | `/v1/staff/employees/{id}` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Profile,roles,assignment,active,version |
| Response | Updated employee |
| Response datatype | Employee (object) |
| Success status | 200 with the response model below. |
| Key error cases | 403 employee edit denied, 404 unavailable employee, 409 stale version or 422 invalid form fields. |
| Database mapping | Pegawai, Role Pegawai and Pegawai Pejabat. Credential data is server-only. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "name": "Petugas Contoh",
  "email": "staff@example.test",
  "roles": [
    "PA"
  ],
  "active": true,
  "assignedOfficerIds": [
    "2"
  ],
  "version": 1
}
```

| Request field | Datatype |
| --- | --- |
| `name` | string |
| `email` | string |
| `roles` | string[] |
| `active` | boolean |
| `assignedOfficerIds` | string[] |
| `version` | integer |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "7",
  "name": "Petugas Contoh",
  "email": "staff@example.test",
  "roles": [
    "PA"
  ],
  "active": true,
  "assignedOfficerIds": [
    "2"
  ],
  "version": 1
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `name` | string |
| `email` | string |
| `roles` | string[] |
| `active` | boolean |
| `assignedOfficerIds` | string[] |
| `version` | integer |

### 30. GET /v1/staff/roles

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/roles` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Peran & akses |
| Response | Existing role definitions with permissions,assignablePermissions,editable,version |
| Response datatype | RoleDefinition[] (array of objects) |
| Success status | 200 with the response model below. |
| Key error cases | 403 role administration denied. |
| Database mapping | Role, Role Policy, Policy, Policy Permission and Permission. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
[
  {
    "id": "1",
    "name": "PA",
    "description": "Asisten pejabat",
    "permissions": [
      "bookings",
      "create"
    ],
    "assignablePermissions": [
      "bookings",
      "create"
    ],
    "editable": true,
    "version": 1
  }
]
```

| Request field | Datatype |
| --- | --- |
| `[]` | array of objects |
| `[].id` | string |
| `[].name` | string |
| `[].description` | string |
| `[].permissions` | string[] |
| `[].assignablePermissions` | string[] |
| `[].editable` | boolean |
| `[].version` | integer |

### 31. PATCH /v1/staff/roles/{id}

| Property | Contract |
| --- | --- |
| Method | PATCH |
| Endpoint | `/v1/staff/roles/{id}` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | permissions,version |
| Response | Updated definition; no arbitrary role creation |
| Response datatype | RoleDefinition (object) |
| Success status | 200 with the response model below. |
| Key error cases | 403 policy edit denied, 404 unavailable role, 409 stale version or 422 invalid permissions. |
| Database mapping | Role, Role Policy, Policy, Policy Permission and Permission. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "permissions": [
    "bookings",
    "create"
  ],
  "version": 1
}
```

| Request field | Datatype |
| --- | --- |
| `permissions` | string[] |
| `version` | integer |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "id": "1",
  "name": "PA",
  "description": "Asisten pejabat",
  "permissions": [
    "bookings",
    "create"
  ],
  "assignablePermissions": [
    "bookings",
    "create"
  ],
  "editable": true,
  "version": 1
}
```

| Response field | Datatype |
| --- | --- |
| `id` | string |
| `name` | string |
| `description` | string |
| `permissions` | string[] |
| `assignablePermissions` | string[] |
| `editable` | boolean |
| `version` | integer |

### 32. POST /v1/staff/sessions

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/sessions` |
| Authentication | Anonymous login bootstrap cookies, X-CSRF-Token and same-origin checks. |
| Request / frontend use | email and password. Read TokenResult and process its session. |
| Response | accessToken:string, tokenType:string (Bearer), accessExpiresAt:timestamp, session:Session; adopt replacement token before further calls |
| Response datatype | TokenResult (object; confirmed Bearer integration) |
| Success status | 200 with the response model below. |
| Key error cases | 401 invalid credentials, 403 invalid origin or CSRF proof, 429 rate limit. |
| Database mapping | Staff session and credential support plus active employee/role authorization. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "email": "staff@example.test",
  "password": "<password>"
}
```

| Request field | Datatype |
| --- | --- |
| `email` | string |
| `password` | string |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "accessToken": "<opaque-access-token>",
  "tokenType": "Bearer",
  "accessExpiresAt": "2026-11-16T10:05:00+08:00",
  "session": {
    "active": true,
    "id": "7",
    "name": "Petugas Contoh",
    "role": "PA",
    "availableRoles": [
      "PA"
    ],
    "roleSelectionRequired": false,
    "roleRevision": 1,
    "permissions": [
      "bookings",
      "create"
    ],
    "assignedOfficerIds": [
      "2"
    ],
    "mustChangePassword": false,
    "passwordMinLength": 8
  }
}
```

| Response field | Datatype |
| --- | --- |
| `accessToken` | string |
| `tokenType` | string |
| `accessExpiresAt` | string |
| `session` | object |
| `session.active` | boolean |
| `session.id` | string |
| `session.name` | string |
| `session.role` | string |
| `session.availableRoles` | string[] |
| `session.roleSelectionRequired` | boolean |
| `session.roleRevision` | integer |
| `session.permissions` | string[] |
| `session.assignedOfficerIds` | string[] |
| `session.mustChangePassword` | boolean |
| `session.passwordMinLength` | integer |

### 33. GET /v1/staff/session

| Property | Contract |
| --- | --- |
| Method | GET |
| Endpoint | `/v1/staff/session` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Session bootstrap/refresh |
| Response | Active or role-selection-required principal, permissions,assignments,password policy |
| Response datatype | Session (object) |
| Success status | 200 with the response model below. |
| Key error cases | 401 expired, revoked or invalid session. |
| Database mapping | Staff session and credential support plus active employee/role authorization. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "active": true,
  "id": "7",
  "name": "Petugas Contoh",
  "role": "PA",
  "availableRoles": [
    "PA"
  ],
  "roleSelectionRequired": false,
  "roleRevision": 1,
  "permissions": [
    "bookings",
    "create"
  ],
  "assignedOfficerIds": [
    "2"
  ],
  "mustChangePassword": false,
  "passwordMinLength": 8
}
```

| Request field | Datatype |
| --- | --- |
| `active` | boolean |
| `id` | string |
| `name` | string |
| `role` | string |
| `availableRoles` | string[] |
| `roleSelectionRequired` | boolean |
| `roleRevision` | integer |
| `permissions` | string[] |
| `assignedOfficerIds` | string[] |
| `mustChangePassword` | boolean |
| `passwordMinLength` | integer |

### 34. POST /v1/staff/session/role

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/session/role` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | Requested assigned role. Replace the access token and apply result.session. |
| Response | accessToken:string, tokenType:string (Bearer), accessExpiresAt:timestamp, session:Session; adopt replacement token before further calls |
| Response datatype | TokenResult (object; confirmed Bearer integration) |
| Success status | 200 with the response model below. |
| Key error cases | 401 invalid session, 403 role not assigned or 409 stale session role revision. |
| Database mapping | Staff session and credential support plus active employee/role authorization. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "role": "PA"
}
```

| Request field | Datatype |
| --- | --- |
| `role` | string |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "accessToken": "<opaque-access-token>",
  "tokenType": "Bearer",
  "accessExpiresAt": "2026-11-16T10:05:00+08:00",
  "session": {
    "active": true,
    "id": "7",
    "name": "Petugas Contoh",
    "role": "PA",
    "availableRoles": [
      "PA"
    ],
    "roleSelectionRequired": false,
    "roleRevision": 1,
    "permissions": [
      "bookings",
      "create"
    ],
    "assignedOfficerIds": [
      "2"
    ],
    "mustChangePassword": false,
    "passwordMinLength": 8
  }
}
```

| Response field | Datatype |
| --- | --- |
| `accessToken` | string |
| `tokenType` | string |
| `accessExpiresAt` | string |
| `session` | object |
| `session.active` | boolean |
| `session.id` | string |
| `session.name` | string |
| `session.role` | string |
| `session.availableRoles` | string[] |
| `session.roleSelectionRequired` | boolean |
| `session.roleRevision` | integer |
| `session.permissions` | string[] |
| `session.assignedOfficerIds` | string[] |
| `session.mustChangePassword` | boolean |
| `session.passwordMinLength` | integer |

### 35. DELETE /v1/staff/session

| Property | Contract |
| --- | --- |
| Method | DELETE |
| Endpoint | `/v1/staff/session` |
| Authentication | Refresh cookie, X-CSRF-Token and same-origin checks. |
| Request / frontend use | Logout |
| Response | HTTP 204 with no body. |
| Response datatype | no body (204 in inspected backend) |
| Success status | 204, empty body. |
| Key error cases | 401 invalid session or 403 invalid origin or CSRF proof. |
| Database mapping | Staff session and credential support plus active employee/role authorization. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

No JSON body. Query/path parameters are listed above and in the detailed contract.

**Example response**

HTTP 204; no response body.

### 36. POST /v1/staff/password

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/password` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | currentPassword and newPassword. Read replacement TokenResult. |
| Response | accessToken:string, tokenType:string (Bearer), accessExpiresAt:timestamp, session:Session; adopt replacement token before further calls |
| Response datatype | TokenResult (object; confirmed Bearer integration) |
| Success status | 200 with the response model below. |
| Key error cases | 401 invalid session or current password, 422 invalid new password. |
| Database mapping | Staff session and credential support plus active employee/role authorization. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "currentPassword": "<current-password>",
  "newPassword": "<new-password>"
}
```

| Request field | Datatype |
| --- | --- |
| `currentPassword` | string |
| `newPassword` | string |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "accessToken": "<opaque-access-token>",
  "tokenType": "Bearer",
  "accessExpiresAt": "2026-11-16T10:05:00+08:00",
  "session": {
    "active": true,
    "id": "7",
    "name": "Petugas Contoh",
    "role": "PA",
    "availableRoles": [
      "PA"
    ],
    "roleSelectionRequired": false,
    "roleRevision": 1,
    "permissions": [
      "bookings",
      "create"
    ],
    "assignedOfficerIds": [
      "2"
    ],
    "mustChangePassword": false,
    "passwordMinLength": 8
  }
}
```

| Response field | Datatype |
| --- | --- |
| `accessToken` | string |
| `tokenType` | string |
| `accessExpiresAt` | string |
| `session` | object |
| `session.active` | boolean |
| `session.id` | string |
| `session.name` | string |
| `session.role` | string |
| `session.availableRoles` | string[] |
| `session.roleSelectionRequired` | boolean |
| `session.roleRevision` | integer |
| `session.permissions` | string[] |
| `session.assignedOfficerIds` | string[] |
| `session.mustChangePassword` | boolean |
| `session.passwordMinLength` | integer |

### 37. POST /v1/staff/bookings/{id}/attachment

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings/{id}/attachment` |
| Authentication | Bearer plus current reception scope and matching visit credential. |
| Request / frontend use | JSON {credential}; authorized reception; Bearer |
| Response | Document binary after code/QR revalidation |
| Response datatype | binary file (not JSON) |
| Success status | 200 with the authorized file bytes. |
| Key error cases | 404 unavailable file, visit or matching proof. A denied reception role cannot use this route. |
| Database mapping | Uploads and the owning visit reference. Stream bytes from private storage. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "credential": "00123456"
}
```

| Request field | Datatype |
| --- | --- |
| `credential` | string |

**Example response**

Binary PDF for attachment, or JPEG/PNG for visitor photo. Do not wrap file bytes in JSON.

### 38. POST /v1/staff/bookings/{id}/visitor-photo

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/bookings/{id}/visitor-photo` |
| Authentication | Bearer plus current reception scope and matching visit credential. |
| Request / frontend use | JSON {credential}; authorized reception; Bearer |
| Response | Image binary after code/QR revalidation |
| Response datatype | binary file (not JSON) |
| Success status | 200 with the authorized file bytes. |
| Key error cases | 404 unavailable photo, visit or matching proof. A denied reception role cannot use this route. |
| Database mapping | Uploads and the owning visit reference. Stream bytes from private storage. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Path and query parameters**

| Parameter | Datatype | Meaning |
| --- | --- | --- |
| `id` | path string ID | Required. Identifies the visit, employee or role named by the path. |

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "credential": "00123456"
}
```

| Request field | Datatype |
| --- | --- |
| `credential` | string |

**Example response**

Binary PDF for attachment, or JPEG/PNG for visitor photo. Do not wrap file bytes in JSON.

### 39. POST /v1/staff/session/refresh

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/session/refresh` |
| Authentication | Refresh cookie, X-CSRF-Token and same-origin checks. |
| Request / frontend use | JSON {refreshRequestId}; refresh cookie and X-CSRF-Token |
| Response | accessToken:string, tokenType:string (Bearer), accessExpiresAt:timestamp, session:Session; adopt replacement token before further calls |
| Response datatype | TokenResult (object; confirmed Bearer integration) |
| Success status | 200 with the response model below. |
| Key error cases | 401 expired or invalid server session, 403 invalid origin or CSRF proof, 409 refresh request conflict, 422 invalid request ID. |
| Database mapping | Staff session and credential support plus active employee/role authorization. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{
  "refreshRequestId": "12345678-1234-4234-8234-123456789abc"
}
```

| Request field | Datatype |
| --- | --- |
| `refreshRequestId` | string |

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "accessToken": "<opaque-access-token>",
  "tokenType": "Bearer",
  "accessExpiresAt": "2026-11-16T10:05:00+08:00",
  "session": {
    "active": true,
    "id": "7",
    "name": "Petugas Contoh",
    "role": "PA",
    "availableRoles": [
      "PA"
    ],
    "roleSelectionRequired": false,
    "roleRevision": 1,
    "permissions": [
      "bookings",
      "create"
    ],
    "assignedOfficerIds": [
      "2"
    ],
    "mustChangePassword": false,
    "passwordMinLength": 8
  }
}
```

| Response field | Datatype |
| --- | --- |
| `accessToken` | string |
| `tokenType` | string |
| `accessExpiresAt` | string |
| `session` | object |
| `session.active` | boolean |
| `session.id` | string |
| `session.name` | string |
| `session.role` | string |
| `session.availableRoles` | string[] |
| `session.roleSelectionRequired` | boolean |
| `session.roleRevision` | integer |
| `session.permissions` | string[] |
| `session.assignedOfficerIds` | string[] |
| `session.mustChangePassword` | boolean |
| `session.passwordMinLength` | integer |

### 40. POST /v1/staff/session/renew

| Property | Contract |
| --- | --- |
| Method | POST |
| Endpoint | `/v1/staff/session/renew` |
| Authentication | Bearer and the operation-specific role/permission and visit scope. |
| Request / frontend use | JSON {}; Bearer; actual user activity |
| Response | Session |
| Response datatype | Session (object) |
| Success status | 200 with the response model below. |
| Key error cases | 401 expired, revoked or invalid session. |
| Database mapping | Staff session and credential support plus active employee/role authorization. |

**Request and response field definitions:** use the corresponding section in [contract details](FRONTEND-DRIVEN-DETAILS-2026-11-16.md); validation and nullable fields must not be inferred from the database table name.

**Example request**

Synthetic request; placeholders are not live credentials.
```json
{}
```

**Example response**

Synthetic shape example. Optional fields may be omitted; state-dependent values require server calculation.
```json
{
  "active": true,
  "id": "7",
  "name": "Petugas Contoh",
  "role": "PA",
  "availableRoles": [
    "PA"
  ],
  "roleSelectionRequired": false,
  "roleRevision": 1,
  "permissions": [
    "bookings",
    "create"
  ],
  "assignedOfficerIds": [
    "2"
  ],
  "mustChangePassword": false,
  "passwordMinLength": 8
}
```

| Response field | Datatype |
| --- | --- |
| `active` | boolean |
| `id` | string |
| `name` | string |
| `role` | string |
| `availableRoles` | string[] |
| `roleSelectionRequired` | boolean |
| `roleRevision` | integer |
| `permissions` | string[] |
| `assignedOfficerIds` | string[] |
| `mustChangePassword` | boolean |
| `passwordMinLength` | integer |


## Shared errors

| Field | Datatype | Meaning |
| --- | --- | --- |
| message | string | User-facing error |
| code | string, optional | Stable machine-readable category |
| fieldErrors | object of string arrays, optional | Errors keyed by actual form field |

Operation tables list success and key error cases for the target draft. The shared body carries 401, 403, 404, 409, 422 or 429 where relevant. The full error inventory and wire conformance remain under review. Use Retry-After on rate-limit responses where available.

Inventory: 40 method/path operations. No application or database changes made.
