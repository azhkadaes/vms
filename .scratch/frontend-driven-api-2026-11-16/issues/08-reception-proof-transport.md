# Resend reception proof for each file request

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Send the already-entered code/QR proof with every file request, or store an expiring verification grant in the session?

## Answer

User selected resending proof on every file request, matching preview, with no extra operator input. Define POST /v1/staff/bookings/{id}/attachment and POST /v1/staff/bookings/{id}/visitor-photo, JSON {credential}, Bearer authorization, binary response. Recheck current permission, record scope, credential and attendance state every time. Do not put credentials in URLs or create a new server-side verification grant. Other authorized staff retain scoped GET operations.
