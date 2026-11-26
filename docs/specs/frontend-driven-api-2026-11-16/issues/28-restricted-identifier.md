# Map restricted No ID Dinas to detail_kunjungan.no_nd

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## User direction

User selected detail_kunjungan.no_nd instead of proposed detail_pengunjung.service_id for the existing API serviceId. Preserve this owner/name choice. Confirm semantics and storage type: diagram no_nd is int, while frontend serviceId is optional text up to 100 characters and means No ID Dinas. Do not assume it is the uploaded Nota Dinas number or change the UI label silently.

## Answer

User explicitly confirmed no_nd means the Terbatas visitor's No ID Dinas, mapped to serviceId, with optional VARCHAR(100) rather than INTEGER. Preserve letters and leading zeros. nota_dinas remains the separate upload reference. No API label change, migration or ordinary identity-number collection is authorized.
