# Wire schemas

Generated from the canonical [OpenAPI](../BACKEND-OPENAPI-2026-11-12.json). Required lists, additionalProperties, oneOf/allOf, defaults and validation extensions are normative. No nullable types are declared: optional fields are omitted, not null. `x-*` state/byte constraints require application checks as described in the main contract. Examples are synthetic.

## BusinessId

```json
{
  "type": "string",
  "minLength": 1,
  "maxLength": 10,
  "pattern": "^(?:[1-9][0-9]{0,8}|1[0-9]{9}|20[0-9]{8}|21[0-3][0-9]{7}|214[0-6][0-9]{6}|2147[0-3][0-9]{5}|21474[0-7][0-9]{4}|214748[0-2][0-9]{3}|2147483[0-5][0-9]{2}|21474836[0-3][0-9]|214748364[0-7])$",
  "description": "D95: canonical decimal PostgreSQL INTEGER ID, 1 through 2147483647. Not a booking code or service ID."
}
```

## Version

```json
{
  "type": "integer",
  "minimum": 1,
  "maximum": 2147483647,
  "description": "D97: positive PostgreSQL INTEGER conflict version. Exhaustion rejects mutation; never wrap/reset."
}
```

## Role

```json
{
  "type": "string",
  "enum": [
    "PA",
    "Protocol Officer",
    "Receptionist",
    "Super Admin"
  ]
}
```

## Permission

```json
{
  "type": "string",
  "enum": [
    "bookings",
    "create",
    "approve",
    "reject",
    "reschedule",
    "cancel",
    "checkin",
    "checkout",
    "reports",
    "reportExport",
    "people",
    "integrations"
  ]
}
```

## Status

```json
{
  "type": "string",
  "enum": [
    "pending",
    "approved",
    "rejected",
    "cancelled",
    "checked_in",
    "checked_out"
  ]
}
```

## DisplayStatus

```json
{
  "type": "string",
  "enum": [
    "pending",
    "approved",
    "rejected",
    "cancelled",
    "checked_in",
    "checked_out",
    "missed"
  ]
}
```

## Date

```json
{
  "type": "string",
  "minLength": 10,
  "maxLength": 10,
  "format": "date",
  "pattern": "^[0-9]{4}-[0-9]{2}-[0-9]{2}$"
}
```

## StartTime

```json
{
  "type": "string",
  "minLength": 5,
  "maxLength": 5,
  "pattern": "^(?:(?:08|09|1[0-5]):[0-5][0-9]|16:00)$",
  "description": "Start only, Asia/Makassar. Must be in future on creation/schedule edits; end may exceed 16:00."
}
```

## Email

```json
{
  "type": "string",
  "minLength": 3,
  "maxLength": 254,
  "format": "email",
  "pattern": "^[\\x21-\\x7e]+$",
  "description": "One ASCII mailbox, no display-name/list/header input; trimmed, normalized per D82. Validate local/domain syntax and lengths.",
  "x-max-utf8-bytes": 254
}
```

## Password

```json
{
  "type": "string",
  "minLength": 15,
  "maxLength": 72,
  "x-max-utf8-bytes": 72,
  "x-length-unit": "unicode-code-points",
  "x-normalization": "none",
  "writeOnly": true,
  "description": "New password: >=15 code points AND <=72 UTF-8 bytes; bcrypt, no truncation/prehash."
}
```

## ExistingPassword

```json
{
  "type": "string",
  "minLength": 1,
  "maxLength": 72,
  "x-max-utf8-bytes": 72,
  "x-normalization": "none",
  "writeOnly": true,
  "description": "Verify existing credential; new-password minimum is not imposed on current-password/login verification."
}
```

## Reference

```json
{
  "type": "string",
  "minLength": 36,
  "maxLength": 36,
  "format": "uuid",
  "description": "UUID request/submission/upload reference, distinct from a business ID."
}
```

## BookingCode

```json
{
  "type": "string",
  "minLength": 8,
  "maxLength": 8,
  "pattern": "^[0-9]{8}$"
}
```

## Credential

```json
{
  "type": "string",
  "minLength": 1,
  "maxLength": 4096,
  "writeOnly": true,
  "description": "Manual eight-digit code or opaque QR. Admission and checkout have different eligibility; D96 permits the QR consumed on successful check-in for checkout after admission expiry, unless revoked.",
  "x-accepted-formats": {
    "manual": "8 ASCII decimal digits",
    "qr": "Proposed vmsq1. prefix plus43 unpadded base64url characters from32 random bytes; no embedded person/booking data."
  }
}
```

## ApiError

```json
{
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "minLength": 1,
      "description": "Stable machine code. Codes required for explicit D57/D58 and selected lifecycle branches are normative; the earlier common-code examples are not an exhaustive taxonomy for future modules. Define additional codes with their owning operation before implementation."
    },
    "message": {
      "type": "string",
      "minLength": 1
    },
    "fieldErrors": {
      "type": "object",
      "additionalProperties": {
        "type": "array",
        "items": {
          "type": "string",
          "minLength": 1
        },
        "minItems": 1
      }
    },
    "requestId": {
      "type": "string",
      "minLength": 1
    }
  },
  "required": [
    "code",
    "message",
    "requestId"
  ],
  "additionalProperties": false
}
```

## Room

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "active": {
      "type": "boolean"
    }
  },
  "required": [
    "id",
    "name",
    "active"
  ],
  "additionalProperties": false
}
```

## Officer

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "vip": {
      "type": "boolean"
    },
    "active": {
      "type": "boolean"
    },
    "jobTitle": {
      "type": "string",
      "maxLength": 255,
      "description": "Structural job title of the official; display metadata, not authorization."
    },
    "directorate": {
      "type": "string",
      "maxLength": 255,
      "description": "Directorate linked to the official position. Omitted when not applicable or unavailable."
    },
    "deputy": {
      "type": "string",
      "maxLength": 255,
      "description": "Deputy unit derived from the linked directorate, or the deputy-level position. Omitted when unavailable."
    }
  },
  "required": [
    "id",
    "name",
    "vip",
    "active"
  ],
  "additionalProperties": false
}
```

## Catalogs

```json
{
  "type": "object",
  "properties": {
    "officers": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/Officer"
      }
    },
    "rooms": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/Room"
      }
    },
    "attachmentsEnabled": {
      "type": "boolean"
    },
    "privacyNotice": {
      "type": "object",
      "properties": {
        "version": {
          "type": "string",
          "minLength": 1,
          "maxLength": 120
        },
        "id": {
          "type": "string",
          "minLength": 1
        },
        "en": {
          "type": "string",
          "minLength": 1
        }
      },
      "required": [
        "version",
        "id",
        "en"
      ],
      "additionalProperties": false
    }
  },
  "required": [
    "officers",
    "rooms"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Public active safe destinations only; no staff assignments.",
    "Optional capability omitted/false disables new attachments; genuine collection needs approved notice."
  ]
}
```

## VisitTiming

```json
{
  "type": "object",
  "properties": {
    "serverTime": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    },
    "opensAt": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    },
    "endsAt": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    },
    "displayStatus": {
      "$ref": "#/components/schemas/DisplayStatus"
    },
    "canCheckIn": {
      "type": "boolean"
    },
    "canCheckOut": {
      "type": "boolean"
    },
    "canCancel": {
      "type": "boolean"
    }
  },
  "required": [
    "serverTime",
    "opensAt",
    "endsAt",
    "displayStatus",
    "canCheckIn",
    "canCheckOut",
    "canCancel"
  ],
  "additionalProperties": false
}
```

## PublicEvent

```json
{
  "type": "object",
  "properties": {
    "at": {
      "type": "string",
      "format": "date-time"
    },
    "action": {
      "type": "string",
      "enum": [
        "received",
        "approved",
        "rejected",
        "rescheduled",
        "cancelled",
        "checked_in",
        "checked_out",
        "updated"
      ]
    },
    "publicMessage": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "at",
    "action"
  ],
  "additionalProperties": false,
  "allOf": [
    {
      "if": {
        "properties": {
          "action": {
            "not": {
              "const": "rescheduled"
            }
          }
        }
      },
      "then": {
        "not": {
          "required": [
            "publicMessage"
          ]
        }
      }
    }
  ]
}
```

## StaffEvent

```json
{
  "type": "object",
  "properties": {
    "at": {
      "type": "string",
      "format": "date-time"
    },
    "message": {
      "type": "string",
      "minLength": 1
    }
  },
  "required": [
    "at",
    "message"
  ],
  "additionalProperties": false
}
```

## PublicBooking

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "visitKind": {
      "const": "standard"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "type": "string",
      "minLength": 5,
      "maxLength": 5,
      "pattern": "^(?:[01][0-9]|2[0-3]):[0-5][0-9]$"
    },
    "duration": {
      "type": "integer",
      "minimum": 15,
      "maximum": 240
    },
    "roomId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "room": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255
    },
    "status": {
      "$ref": "#/components/schemas/Status"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "rescheduled": {
      "type": "boolean"
    },
    "timing": {
      "$ref": "#/components/schemas/VisitTiming"
    },
    "code": {
      "$ref": "#/components/schemas/BookingCode"
    },
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 120,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "organization": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "email": {
      "$ref": "#/components/schemas/Email"
    },
    "phone": {
      "type": "string",
      "minLength": 8,
      "maxLength": 20,
      "pattern": "^[0-9]{8,20}$"
    },
    "officerId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "host": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "vip": {
      "type": "boolean"
    },
    "purpose": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "visitors": {
      "type": "array",
      "items": {
        "type": "string",
        "minLength": 2,
        "maxLength": 120,
        "x-length-unit": "unicode-code-points",
        "x-normalization": "trim"
      },
      "minItems": 1,
      "maxItems": 20
    },
    "identityType": {
      "type": "string",
      "enum": [
        "KTP",
        "Paspor",
        "SIM"
      ]
    },
    "ndName": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "visitorInstructions": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "rejectionReason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "cancellationReason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "history": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/PublicEvent"
      }
    }
  },
  "required": [
    "id",
    "visitKind",
    "date",
    "time",
    "duration",
    "room",
    "status",
    "version",
    "timing",
    "code",
    "name",
    "organization",
    "email",
    "phone",
    "officerId",
    "host",
    "vip",
    "purpose",
    "visitors",
    "identityType",
    "history"
  ],
  "additionalProperties": false
}
```

## StandardStaffBooking

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "visitKind": {
      "const": "standard"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "type": "string",
      "minLength": 5,
      "maxLength": 5,
      "pattern": "^(?:[01][0-9]|2[0-3]):[0-5][0-9]$"
    },
    "duration": {
      "type": "integer",
      "minimum": 15,
      "maximum": 240
    },
    "roomId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "room": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255
    },
    "status": {
      "$ref": "#/components/schemas/Status"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "rescheduled": {
      "type": "boolean"
    },
    "timing": {
      "$ref": "#/components/schemas/VisitTiming"
    },
    "code": {
      "$ref": "#/components/schemas/BookingCode"
    },
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 120,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "organization": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "email": {
      "$ref": "#/components/schemas/Email"
    },
    "phone": {
      "type": "string",
      "minLength": 8,
      "maxLength": 20,
      "pattern": "^[0-9]{8,20}$"
    },
    "officerId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "host": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "vip": {
      "type": "boolean"
    },
    "purpose": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "visitors": {
      "type": "array",
      "items": {
        "type": "string",
        "minLength": 2,
        "maxLength": 120,
        "x-length-unit": "unicode-code-points",
        "x-normalization": "trim"
      },
      "minItems": 1,
      "maxItems": 20
    },
    "identityType": {
      "type": "string",
      "enum": [
        "KTP",
        "Paspor",
        "SIM"
      ]
    },
    "ndName": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "visitorInstructions": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "rejectionReason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "cancellationReason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "coordinationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "internalRejectionNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "internalCancellationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "createdBy": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255,
          "x-length-unit": "unicode-code-points",
          "x-normalization": "trim"
        },
        "role": {
          "$ref": "#/components/schemas/Role"
        }
      },
      "required": [
        "name",
        "role"
      ],
      "additionalProperties": false
    },
    "attachmentAvailable": {
      "type": "boolean"
    },
    "history": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/StaffEvent"
      }
    }
  },
  "required": [
    "id",
    "visitKind",
    "date",
    "time",
    "duration",
    "room",
    "status",
    "version",
    "timing",
    "code",
    "name",
    "organization",
    "email",
    "phone",
    "officerId",
    "host",
    "vip",
    "purpose",
    "visitors",
    "identityType",
    "history"
  ],
  "additionalProperties": false,
  "x-field-policy": "Private notes only to permitted PA/Protocol. Reception never receives private notes."
}
```

## RestrictedReception

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "visitKind": {
      "const": "restricted"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "type": "string",
      "minLength": 5,
      "maxLength": 5,
      "pattern": "^(?:[01][0-9]|2[0-3]):[0-5][0-9]$"
    },
    "duration": {
      "type": "integer",
      "minimum": 15,
      "maximum": 240
    },
    "roomId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "room": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255
    },
    "status": {
      "$ref": "#/components/schemas/Status"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "rescheduled": {
      "type": "boolean"
    },
    "timing": {
      "$ref": "#/components/schemas/VisitTiming"
    },
    "reference": {
      "type": "string",
      "minLength": 1
    },
    "restrictedView": {
      "const": "reception"
    }
  },
  "required": [
    "id",
    "visitKind",
    "date",
    "time",
    "duration",
    "room",
    "status",
    "version",
    "timing",
    "reference",
    "restrictedView"
  ],
  "additionalProperties": false,
  "x-field-policy": "Omit absent identity fields. Scope before search/counts; full is not automatic coordination-note permission."
}
```

## RestrictedArrival

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "visitKind": {
      "const": "restricted"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "type": "string",
      "minLength": 5,
      "maxLength": 5,
      "pattern": "^(?:[01][0-9]|2[0-3]):[0-5][0-9]$"
    },
    "duration": {
      "type": "integer",
      "minimum": 15,
      "maximum": 240
    },
    "roomId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "room": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255
    },
    "status": {
      "$ref": "#/components/schemas/Status"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "rescheduled": {
      "type": "boolean"
    },
    "timing": {
      "$ref": "#/components/schemas/VisitTiming"
    },
    "reference": {
      "type": "string",
      "minLength": 1
    },
    "restrictedView": {
      "const": "arrival"
    },
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 120,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "officerId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "host": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "serviceIdLast4": {
      "type": "string",
      "minLength": 1,
      "maxLength": 4
    }
  },
  "required": [
    "id",
    "visitKind",
    "date",
    "time",
    "duration",
    "room",
    "status",
    "version",
    "timing",
    "reference",
    "restrictedView"
  ],
  "additionalProperties": false,
  "x-field-policy": "Omit absent identity fields. Scope before search/counts; full is not automatic coordination-note permission."
}
```

## RestrictedAssignedPA

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "visitKind": {
      "const": "restricted"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "type": "string",
      "minLength": 5,
      "maxLength": 5,
      "pattern": "^(?:[01][0-9]|2[0-3]):[0-5][0-9]$"
    },
    "duration": {
      "type": "integer",
      "minimum": 15,
      "maximum": 240
    },
    "roomId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "room": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255
    },
    "status": {
      "$ref": "#/components/schemas/Status"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "rescheduled": {
      "type": "boolean"
    },
    "timing": {
      "$ref": "#/components/schemas/VisitTiming"
    },
    "reference": {
      "type": "string",
      "minLength": 1
    },
    "restrictedView": {
      "const": "assigned-pa"
    },
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 120,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "officerId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "host": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "serviceIdLast4": {
      "type": "string",
      "minLength": 1,
      "maxLength": 4
    },
    "organization": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "id",
    "visitKind",
    "date",
    "time",
    "duration",
    "room",
    "status",
    "version",
    "timing",
    "reference",
    "restrictedView"
  ],
  "additionalProperties": false,
  "x-field-policy": "Omit absent identity fields. Scope before search/counts; full is not automatic coordination-note permission."
}
```

## RestrictedFull

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "visitKind": {
      "const": "restricted"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "type": "string",
      "minLength": 5,
      "maxLength": 5,
      "pattern": "^(?:[01][0-9]|2[0-3]):[0-5][0-9]$"
    },
    "duration": {
      "type": "integer",
      "minimum": 15,
      "maximum": 240
    },
    "roomId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "room": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255
    },
    "status": {
      "$ref": "#/components/schemas/Status"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "rescheduled": {
      "type": "boolean"
    },
    "timing": {
      "$ref": "#/components/schemas/VisitTiming"
    },
    "reference": {
      "type": "string",
      "minLength": 1
    },
    "restrictedView": {
      "const": "full"
    },
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 120,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "officerId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "host": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "serviceIdLast4": {
      "type": "string",
      "minLength": 1,
      "maxLength": 4
    },
    "organization": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "code": {
      "$ref": "#/components/schemas/BookingCode"
    },
    "serviceId": {
      "type": "string",
      "minLength": 1,
      "maxLength": 100,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "ndName": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "visitorInstructions": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "rejectionReason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "cancellationReason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "coordinationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "internalRejectionNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "internalCancellationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "createdBy": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255,
          "x-length-unit": "unicode-code-points",
          "x-normalization": "trim"
        },
        "role": {
          "$ref": "#/components/schemas/Role"
        }
      },
      "required": [
        "name",
        "role"
      ],
      "additionalProperties": false
    },
    "history": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/StaffEvent"
      }
    }
  },
  "required": [
    "id",
    "visitKind",
    "date",
    "time",
    "duration",
    "room",
    "status",
    "version",
    "timing",
    "reference",
    "restrictedView"
  ],
  "additionalProperties": false,
  "x-field-policy": "Omit absent identity fields. Scope before search/counts; full is not automatic coordination-note permission."
}
```

## StaffBooking

```json
{
  "oneOf": [
    {
      "$ref": "#/components/schemas/StandardStaffBooking"
    },
    {
      "$ref": "#/components/schemas/RestrictedReception"
    },
    {
      "$ref": "#/components/schemas/RestrictedArrival"
    },
    {
      "$ref": "#/components/schemas/RestrictedAssignedPA"
    },
    {
      "$ref": "#/components/schemas/RestrictedFull"
    }
  ]
}
```

## ArrivalBooking

```json
{
  "oneOf": [
    {
      "$ref": "#/components/schemas/StandardStaffBooking"
    },
    {
      "$ref": "#/components/schemas/RestrictedArrival"
    }
  ]
}
```

## PublicCreate

```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 120,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "organization": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "email": {
      "$ref": "#/components/schemas/Email"
    },
    "phone": {
      "type": "string",
      "minLength": 8,
      "maxLength": 20,
      "pattern": "^[0-9]{8,20}$"
    },
    "officerId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "purpose": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "visitors": {
      "type": "array",
      "items": {
        "type": "string",
        "minLength": 2,
        "maxLength": 120,
        "x-length-unit": "unicode-code-points",
        "x-normalization": "trim"
      },
      "minItems": 1,
      "maxItems": 20
    },
    "identityType": {
      "type": "string",
      "enum": [
        "KTP",
        "Paspor",
        "SIM"
      ]
    },
    "visitKind": {
      "const": "standard"
    },
    "idempotencyKey": {
      "$ref": "#/components/schemas/Reference"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "$ref": "#/components/schemas/StartTime"
    },
    "ndName": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "attachmentToken": {
      "type": "string",
      "minLength": 16,
      "maxLength": 4096,
      "writeOnly": true
    },
    "ack": {
      "const": true
    },
    "privacyVersion": {
      "type": "string",
      "minLength": 1,
      "maxLength": 120
    }
  },
  "required": [
    "name",
    "organization",
    "email",
    "phone",
    "officerId",
    "purpose",
    "visitors",
    "identityType",
    "visitKind",
    "idempotencyKey",
    "date",
    "time",
    "ack"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "roster[0] equals representative name after normalization",
    "Pending; duration=60; no room input/reservation",
    "D90 omitted privacyVersion resolves current server version; D91 supplied stale/unknown fails before new write; committed retry preserves original association.",
    "Attachment token and display name refer to same verified owned upload."
  ]
}
```

## StandardStaffCreate

```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 120,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "organization": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "email": {
      "$ref": "#/components/schemas/Email"
    },
    "phone": {
      "type": "string",
      "minLength": 8,
      "maxLength": 20,
      "pattern": "^[0-9]{8,20}$"
    },
    "officerId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "purpose": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "visitors": {
      "type": "array",
      "items": {
        "type": "string",
        "minLength": 2,
        "maxLength": 120,
        "x-length-unit": "unicode-code-points",
        "x-normalization": "trim"
      },
      "minItems": 1,
      "maxItems": 20
    },
    "identityType": {
      "type": "string",
      "enum": [
        "KTP",
        "Paspor",
        "SIM"
      ]
    },
    "visitKind": {
      "const": "standard"
    },
    "idempotencyKey": {
      "$ref": "#/components/schemas/Reference"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "$ref": "#/components/schemas/StartTime"
    },
    "ndName": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "attachmentToken": {
      "type": "string",
      "minLength": 16,
      "maxLength": 4096,
      "writeOnly": true
    },
    "visitorInstructions": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "coordinationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "name",
    "organization",
    "email",
    "phone",
    "officerId",
    "purpose",
    "visitors",
    "identityType",
    "visitKind",
    "idempotencyKey",
    "date",
    "time"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Approved; duration=60; no room input; reserve full interval under locks.",
    "No consent requirement; actor from authenticated authority."
  ]
}
```

## RestrictedCreate

```json
{
  "type": "object",
  "properties": {
    "visitKind": {
      "const": "restricted"
    },
    "idempotencyKey": {
      "$ref": "#/components/schemas/Reference"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "$ref": "#/components/schemas/StartTime"
    },
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 120,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "organization": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "serviceId": {
      "type": "string",
      "minLength": 1,
      "maxLength": 100,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "officerId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "visitorInstructions": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "coordinationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "visitKind",
    "idempotencyKey",
    "date",
    "time"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Protocol create authority; Approved; default duration60; identity/official optional; no public contact/roster/consent/upload."
  ]
}
```

## StaffCreate

```json
{
  "oneOf": [
    {
      "$ref": "#/components/schemas/StandardStaffCreate"
    },
    {
      "$ref": "#/components/schemas/RestrictedCreate"
    }
  ],
  "discriminator": {
    "propertyName": "visitKind",
    "mapping": {
      "standard": "#/components/schemas/StandardStaffCreate",
      "restricted": "#/components/schemas/RestrictedCreate"
    }
  }
}
```

## Approve

```json
{
  "type": "object",
  "properties": {
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "roomId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "duration": {
      "type": "integer",
      "minimum": 15,
      "maximum": 240
    },
    "visitorInstructions": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "coordinationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "version"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Pending only. Ordinary public approval requires valid room/duration; restricted may omit.",
    "Omitted room preserves existing; omitted notes preserve, empty clears."
  ]
}
```

## Reject

```json
{
  "type": "object",
  "properties": {
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "rejectionReason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "internalRejectionNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "version",
    "rejectionReason"
  ],
  "additionalProperties": false
}
```

## Reschedule

```json
{
  "type": "object",
  "properties": {
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "$ref": "#/components/schemas/StartTime"
    },
    "duration": {
      "type": "integer",
      "minimum": 15,
      "maximum": 240
    },
    "roomId": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "reason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "visitorInstructions": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "coordinationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "version",
    "date",
    "time",
    "duration"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "D100 Pending or Approved excluding derived Missed; new start future. Coordinate frontend can()/action visibility, which currently only allows Approved/non-Missed.",
    "Ordinary requires room/reason, restricted may omit. Omitted room clears; omitted notes preserve and empty clears.",
    "Preserve lifecycle; Pending does not reserve; Approved uses full interval conflict locks. Revoke incompatible admission credentials, audit and enqueue notification atomically."
  ]
}
```

## RestrictedUpdate

```json
{
  "type": "object",
  "properties": {
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "visitKind": {
      "const": "restricted"
    },
    "date": {
      "$ref": "#/components/schemas/Date"
    },
    "time": {
      "$ref": "#/components/schemas/StartTime"
    },
    "name": {
      "oneOf": [
        {
          "type": "string",
          "minLength": 2,
          "maxLength": 120,
          "x-length-unit": "unicode-code-points",
          "x-normalization": "trim"
        },
        {
          "const": ""
        }
      ]
    },
    "organization": {
      "oneOf": [
        {
          "type": "string",
          "minLength": 2,
          "maxLength": 255,
          "x-length-unit": "unicode-code-points",
          "x-normalization": "trim"
        },
        {
          "const": ""
        }
      ]
    },
    "serviceId": {
      "oneOf": [
        {
          "type": "string",
          "minLength": 1,
          "maxLength": 100,
          "x-length-unit": "unicode-code-points",
          "x-normalization": "trim"
        },
        {
          "const": ""
        }
      ]
    },
    "officerId": {
      "oneOf": [
        {
          "$ref": "#/components/schemas/BusinessId"
        },
        {
          "const": ""
        }
      ]
    },
    "visitorInstructions": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "coordinationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "version",
    "visitKind",
    "date",
    "time"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Protocol create authority; restricted Pending/Approved; recheck future submitted date/time, scope and conflicts. This remains separate from D100 Reschedule availability.",
    "Omitted identity/target/notes preserve; empty clears. No ordinary personal data."
  ]
}
```

## StaffCancel

```json
{
  "type": "object",
  "properties": {
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "reason": {
      "type": "string",
      "minLength": 5,
      "maxLength": 1000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "internalCancellationNote": {
      "type": "string",
      "minLength": 0,
      "maxLength": 2000,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    }
  },
  "required": [
    "version",
    "reason"
  ],
  "additionalProperties": false
}
```

## Attendance

```json
{
  "type": "object",
  "properties": {
    "credential": {
      "$ref": "#/components/schemas/Credential"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    }
  },
  "required": [
    "credential",
    "version"
  ],
  "additionalProperties": false
}
```

## Resolve

```json
{
  "type": "object",
  "properties": {
    "credential": {
      "$ref": "#/components/schemas/Credential"
    }
  },
  "required": [
    "credential"
  ],
  "additionalProperties": false
}
```

## Pair

```json
{
  "type": "object",
  "properties": {
    "code": {
      "$ref": "#/components/schemas/BookingCode"
    },
    "email": {
      "$ref": "#/components/schemas/Email"
    }
  },
  "required": [
    "code",
    "email"
  ],
  "additionalProperties": false
}
```

## PairVersion

```json
{
  "type": "object",
  "properties": {
    "code": {
      "$ref": "#/components/schemas/BookingCode"
    },
    "email": {
      "$ref": "#/components/schemas/Email"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    }
  },
  "required": [
    "code",
    "email",
    "version"
  ],
  "additionalProperties": false
}
```

## SubmissionResolve

```json
{
  "type": "object",
  "properties": {
    "idempotencyKey": {
      "$ref": "#/components/schemas/Reference"
    },
    "email": {
      "$ref": "#/components/schemas/Email"
    }
  },
  "required": [
    "idempotencyKey",
    "email"
  ],
  "additionalProperties": false
}
```

## Resolution

```json
{
  "oneOf": [
    {
      "type": "object",
      "properties": {
        "outcome": {
          "const": "committed"
        },
        "booking": {
          "$ref": "#/components/schemas/PublicBooking"
        }
      },
      "required": [
        "outcome",
        "booking"
      ],
      "additionalProperties": false
    },
    {
      "type": "object",
      "properties": {
        "outcome": {
          "const": "not_committed"
        }
      },
      "required": [
        "outcome"
      ],
      "additionalProperties": false
    }
  ]
}
```

## Upload

```json
{
  "type": "object",
  "properties": {
    "file": {
      "type": "string",
      "format": "binary",
      "description": "One nonempty verified PDF/JPEG/PNG <=5242880 bytes.",
      "x-max-bytes": 5242880
    },
    "reference": {
      "$ref": "#/components/schemas/Reference"
    }
  },
  "required": [
    "file",
    "reference"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Same reference plus same content/audience/owner replays only an eligible accepted upload. Changed content/reference binding conflicts.",
    "One nonempty verified PDF/JPEG/PNG; unsupported extra multipart fields/parts rejected."
  ]
}
```

## UploadResult

```json
{
  "type": "object",
  "properties": {
    "token": {
      "type": "string",
      "minLength": 16,
      "maxLength": 4096,
      "x-derived-format": "Proposed vmsu1.<keyVersion>.<HMAC-SHA256-base64url>; keyVersion1..32 ASCII URL-safe characters. Dedicated key and immutable versioned derivation metadata (D99)."
    },
    "expiresAt": {
      "type": "string",
      "format": "date-time"
    },
    "serverTime": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    }
  },
  "required": [
    "token",
    "expiresAt",
    "serverTime"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "D99: same-reference accepted upload replay regenerates identical token using original dedicated key/version and immutable metadata, within original expiry.",
    "No recoverable/plaintext upload token persisted; private lookup digest allowed. No new object quota or sliding expiry."
  ]
}
```

## AdmissionResult

```json
{
  "type": "object",
  "properties": {
    "payload": {
      "type": "string",
      "minLength": 1,
      "maxLength": 4096
    },
    "expiresAt": {
      "type": "string",
      "format": "date-time"
    },
    "serverTime": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    }
  },
  "required": [
    "payload",
    "expiresAt",
    "serverTime"
  ],
  "additionalProperties": false
}
```

## Employee

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "email": {
      "$ref": "#/components/schemas/Email"
    },
    "roles": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/Role"
      },
      "minItems": 1,
      "maxItems": 1,
      "uniqueItems": true
    },
    "active": {
      "type": "boolean"
    },
    "assignedOfficerIds": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/BusinessId"
      },
      "minItems": 0,
      "maxItems": 1,
      "uniqueItems": true,
      "description": "At most one current official assignment per employee; historical inactive rows do not add scope."
    },
    "jobTitle": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "deputy": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "directorate": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "phone": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "nip": {
      "type": "string",
      "minLength": 0,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    }
  },
  "required": [
    "id",
    "name",
    "email",
    "roles",
    "active",
    "assignedOfficerIds",
    "version"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Exactly one database-assigned role in roles; read-only directory projection. No password/hash/credential or editable policy graph."
  ]
}
```

## EmployeePage

```json
{
  "type": "object",
  "properties": {
    "items": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/Employee"
      },
      "minItems": 0,
      "maxItems": 250
    },
    "total": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    },
    "totalAll": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    },
    "page": {
      "type": "integer",
      "minimum": 1,
      "maximum": 2147483647
    },
    "pageSize": {
      "type": "integer",
      "minimum": 1,
      "maximum": 250
    }
  },
  "required": [
    "items",
    "total",
    "totalAll",
    "page",
    "pageSize"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Read-only Phase 1 directory; total is filtered, totalAll is before search; stable paging. Account/role/assignment mutation and password metadata/forms are deferred."
  ]
}
```

## IntegrationField

```json
{
  "oneOf": [
    {
      "type": "object",
      "properties": {
        "key": {
          "const": "host"
        },
        "label": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255
        },
        "kind": {
          "const": "text"
        },
        "required": {
          "const": true
        },
        "value": {
          "type": "string",
          "maxLength": 253
        }
      },
      "required": [
        "key",
        "label",
        "kind",
        "required"
      ],
      "additionalProperties": false
    },
    {
      "type": "object",
      "properties": {
        "key": {
          "const": "port"
        },
        "label": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255
        },
        "kind": {
          "const": "number"
        },
        "required": {
          "const": true
        },
        "value": {
          "type": "string",
          "maxLength": 5
        }
      },
      "required": [
        "key",
        "label",
        "kind",
        "required"
      ],
      "additionalProperties": false
    },
    {
      "type": "object",
      "properties": {
        "key": {
          "const": "tlsMode"
        },
        "label": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255
        },
        "kind": {
          "const": "text"
        },
        "required": {
          "const": true
        },
        "value": {
          "type": "string",
          "maxLength": 12
        }
      },
      "required": [
        "key",
        "label",
        "kind",
        "required"
      ],
      "additionalProperties": false
    },
    {
      "type": "object",
      "properties": {
        "key": {
          "const": "username"
        },
        "label": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255
        },
        "kind": {
          "const": "text"
        },
        "required": {
          "const": true
        },
        "value": {
          "type": "string",
          "maxLength": 255
        }
      },
      "required": [
        "key",
        "label",
        "kind",
        "required"
      ],
      "additionalProperties": false
    },
    {
      "type": "object",
      "properties": {
        "key": {
          "const": "sender"
        },
        "label": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255
        },
        "kind": {
          "const": "email"
        },
        "required": {
          "const": true
        },
        "value": {
          "type": "string",
          "maxLength": 254
        }
      },
      "required": [
        "key",
        "label",
        "kind",
        "required"
      ],
      "additionalProperties": false
    },
    {
      "type": "object",
      "properties": {
        "key": {
          "const": "password"
        },
        "label": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255
        },
        "kind": {
          "const": "secret"
        },
        "required": {
          "const": true
        },
        "configured": {
          "type": "boolean"
        }
      },
      "required": [
        "key",
        "label",
        "kind",
        "required",
        "configured"
      ],
      "additionalProperties": false
    }
  ]
}
```

## Integration

```json
{
  "type": "object",
  "properties": {
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "description": {
      "type": "string"
    },
    "enabled": {
      "type": "boolean"
    },
    "health": {
      "type": "string",
      "enum": [
        "not_configured",
        "ready",
        "degraded"
      ]
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    },
    "fields": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/IntegrationField"
      },
      "minItems": 0,
      "maxItems": 6
    }
  },
  "required": [
    "id",
    "name",
    "description",
    "enabled",
    "health",
    "version",
    "fields"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "SMTP field dictionary only for initial adapter; secret never has value.",
    "Deferred storage metadata, if exposed, stays disabled with no writable provider fields.",
    "SMTP fields unique by key; expose six known field descriptors, never duplicate keys or private paths."
  ]
}
```

## IntegrationPatch

```json
{
  "type": "object",
  "properties": {
    "enabled": {
      "type": "boolean"
    },
    "values": {
      "$ref": "#/components/schemas/SmtpValues"
    },
    "clearSecrets": {
      "type": "array",
      "items": {
        "const": "password"
      },
      "maxItems": 1,
      "uniqueItems": true
    },
    "version": {
      "$ref": "#/components/schemas/Version"
    }
  },
  "required": [
    "enabled",
    "values",
    "clearSecrets",
    "version"
  ],
  "additionalProperties": false,
  "x-provider-validation": {
    "provider": "smtp",
    "requiredEffectiveFields": [
      "host",
      "port",
      "tlsMode",
      "username",
      "password",
      "sender"
    ],
    "tlsModes": [
      "implicit_tls",
      "starttls"
    ],
    "portRange": [
      1,
      65535
    ],
    "maxHostBytes": 253,
    "maxUsernameCodePoints": 255,
    "maxPasswordBytes": 4096,
    "sender": "Email",
    "tls": "Verified CA/hostname, no insecure fallback; AUTH only after TLS",
    "storage": "Deferred settings not writable; no arbitrary preview endpoint/bucket keys."
  },
  "x-server-invariants": [
    "Keys must belong to supported provider schema; no unsupported fields.",
    "Omitted/blank configured secret keeps; explicit clear removes, reject required missing secret while enabled.",
    "No secret echoes/logs; safe field-key errors."
  ]
}
```

## ReportFilter

```json
{
  "type": "object",
  "properties": {
    "from": {
      "$ref": "#/components/schemas/Date"
    },
    "to": {
      "$ref": "#/components/schemas/Date"
    },
    "status": {
      "type": "string",
      "enum": [
        "all",
        "pending",
        "approved",
        "rejected",
        "cancelled",
        "checked_in",
        "checked_out",
        "missed"
      ]
    }
  },
  "required": [
    "from",
    "to",
    "status"
  ],
  "additionalProperties": false
}
```

## Revision

```json
{
  "type": "string",
  "minLength": 64,
  "maxLength": 64,
  "pattern": "^[0-9a-f]{64}$"
}
```

## Summary

```json
{
  "type": "object",
  "properties": {
    "filter": {
      "$ref": "#/components/schemas/ReportFilter"
    },
    "scope": {
      "type": "string",
      "enum": [
        "organization",
        "assigned",
        "authorized"
      ]
    },
    "generatedAt": {
      "type": "string",
      "format": "date-time"
    },
    "canExport": {
      "type": "boolean"
    },
    "revision": {
      "$ref": "#/components/schemas/Revision"
    },
    "counts": {
      "type": "object",
      "properties": {
        "pending": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "approved": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "rejected": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "cancelled": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "checked_in": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "checked_out": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "missed": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        }
      },
      "required": [
        "pending",
        "approved",
        "rejected",
        "cancelled",
        "checked_in",
        "checked_out",
        "missed"
      ],
      "additionalProperties": false
    },
    "total": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    }
  },
  "required": [
    "filter",
    "scope",
    "generatedAt",
    "canExport",
    "revision",
    "counts",
    "total"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Counts sum to total; other status counts zero for specific-status filter; exclude restricted before aggregation.",
    "Inclusive dates, from<=to and elapsed range<=365 days; consistent authorized snapshot."
  ]
}
```

## Export

```json
{
  "type": "object",
  "properties": {
    "filename": {
      "type": "string",
      "minLength": 1
    },
    "csv": {
      "type": "string"
    },
    "revision": {
      "$ref": "#/components/schemas/Revision"
    }
  },
  "required": [
    "filename",
    "csv",
    "revision"
  ],
  "additionalProperties": false
}
```

## ReadModel

```json
{
  "type": "object",
  "properties": {
    "items": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/StaffBooking"
      },
      "minItems": 0,
      "maxItems": 100
    },
    "total": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    },
    "totalScoped": {
      "type": "integer",
      "minimum": 0,
      "maximum": 9007199254740991
    },
    "page": {
      "type": "integer",
      "minimum": 1,
      "maximum": 2147483647
    },
    "pageSize": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100
    },
    "metrics": {
      "type": "object",
      "properties": {
        "today": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "pending": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "arrivals": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "inside": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        },
        "completed": {
          "type": "integer",
          "minimum": 0,
          "maximum": 9007199254740991
        }
      },
      "required": [
        "today",
        "pending",
        "arrivals",
        "inside",
        "completed"
      ],
      "additionalProperties": false
    }
  },
  "required": [
    "items",
    "total",
    "totalScoped",
    "page",
    "pageSize"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "Scope/projection before filtering/counts; one consistent authorized snapshot.",
    "Dashboard metrics required; first5 agenda plus first4 attention covered in deduplicated bounded union."
  ]
}
```

## Login

```json
{
  "type": "object",
  "properties": {
    "email": {
      "$ref": "#/components/schemas/Email"
    },
    "password": {
      "$ref": "#/components/schemas/ExistingPassword"
    }
  },
  "required": [
    "email",
    "password"
  ],
  "additionalProperties": false
}
```

## SmtpValues

```json
{
  "type": "object",
  "properties": {
    "host": {
      "type": "string",
      "minLength": 0,
      "maxLength": 253,
      "pattern": "^[A-Za-z0-9.-]*$",
      "x-max-utf8-bytes": 253,
      "description": "Optional while disabled. If effective enabled: valid nonempty ASCII DNS/punycode labels, no URL/port/userinfo; server-owned outbound allowlist and address validation."
    },
    "port": {
      "oneOf": [
        {
          "const": ""
        },
        {
          "type": "string",
          "pattern": "^(?:[1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$"
        }
      ],
      "description": "Canonical decimal port 1-65535; blank only while disabled."
    },
    "tlsMode": {
      "type": "string",
      "enum": [
        "",
        "implicit_tls",
        "starttls"
      ]
    },
    "username": {
      "type": "string",
      "maxLength": 255,
      "x-normalization": "trim",
      "x-length-unit": "unicode-code-points"
    },
    "password": {
      "type": "string",
      "maxLength": 4096,
      "x-max-utf8-bytes": 4096,
      "x-normalization": "none",
      "writeOnly": true
    },
    "sender": {
      "oneOf": [
        {
          "const": ""
        },
        {
          "$ref": "#/components/schemas/Email"
        }
      ]
    }
  },
  "additionalProperties": false,
  "description": "SMTP-only patch values; omitted/blank configured password keeps existing. Validate merged effective configuration when enabled, not this partial object alone."
}
```

## CurrentUser

```json
{
  "type": "object",
  "properties": {
    "active": {
      "const": true
    },
    "id": {
      "$ref": "#/components/schemas/BusinessId"
    },
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 255,
      "x-length-unit": "unicode-code-points",
      "x-normalization": "trim"
    },
    "role": {
      "$ref": "#/components/schemas/Role"
    },
    "permissions": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/Permission"
      },
      "minItems": 0,
      "maxItems": 12,
      "uniqueItems": true
    },
    "assignedOfficerIds": {
      "type": "array",
      "items": {
        "$ref": "#/components/schemas/BusinessId"
      },
      "minItems": 0,
      "maxItems": 1,
      "uniqueItems": true,
      "description": "At most one current official assignment per employee; historical inactive rows do not add scope."
    }
  },
  "required": [
    "active",
    "id",
    "name",
    "role",
    "permissions",
    "assignedOfficerIds"
  ],
  "additionalProperties": false,
  "allOf": [
    {
      "if": {
        "properties": {
          "role": {
            "const": "PA"
          }
        },
        "required": [
          "role"
        ]
      },
      "then": {
        "properties": {
          "permissions": {
            "const": [
              "bookings",
              "create",
              "approve",
              "reject",
              "reschedule",
              "cancel",
              "reports",
              "reportExport"
            ]
          }
        }
      }
    },
    {
      "if": {
        "properties": {
          "role": {
            "const": "Protocol Officer"
          }
        },
        "required": [
          "role"
        ]
      },
      "then": {
        "properties": {
          "permissions": {
            "const": [
              "bookings",
              "create",
              "approve",
              "reject",
              "reschedule",
              "cancel",
              "reports",
              "reportExport"
            ]
          }
        }
      }
    },
    {
      "if": {
        "properties": {
          "role": {
            "const": "Receptionist"
          }
        },
        "required": [
          "role"
        ]
      },
      "then": {
        "properties": {
          "permissions": {
            "const": [
              "bookings",
              "checkin",
              "checkout"
            ]
          }
        }
      }
    },
    {
      "if": {
        "properties": {
          "role": {
            "const": "Super Admin"
          }
        },
        "required": [
          "role"
        ]
      },
      "then": {
        "properties": {
          "permissions": {
            "const": [
              "bookings",
              "people",
              "integrations",
              "reports",
              "reportExport"
            ]
          }
        }
      }
    },
    {
      "if": {
        "properties": {
          "role": {
            "const": "PA"
          }
        },
        "required": [
          "role"
        ]
      },
      "else": {
        "properties": {
          "assignedOfficerIds": {
            "maxItems": 0
          }
        }
      }
    }
  ],
  "x-server-invariants": [
    "Exactly one assigned database role; no role picker, role revision, permission union, mustChangePassword or session timing.",
    "permissions is the deterministic fixed backend mapping; database policy grants and token role claims cannot add permissions.",
    "Only active authorized PA assignments are returned; non-PA assignment list is empty. Missing/inactive accounts cannot produce CurrentUser."
  ]
}
```

## AuthResult

```json
{
  "type": "object",
  "properties": {
    "accessToken": {
      "type": "string",
      "minLength": 1
    },
    "tokenType": {
      "const": "Bearer"
    },
    "accessExpiresAt": {
      "type": "string",
      "format": "date-time"
    },
    "user": {
      "$ref": "#/components/schemas/CurrentUser"
    }
  },
  "required": [
    "accessToken",
    "tokenType",
    "accessExpiresAt",
    "user"
  ],
  "additionalProperties": false,
  "x-server-invariants": [
    "accessExpiresAt equals the signed token exp instant from required finite server lifetime configuration. No refresh/CSRF/anonymous cookies or server-side session is created.",
    "Only accessToken is returned as a credential; token and user are retained in browser memory only."
  ]
}
```
