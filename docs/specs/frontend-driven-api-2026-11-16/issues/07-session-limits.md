# Retain backend session limits and activity semantics

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Retain five-minute access-token maximum, eight-hour inactivity expiry and 24-hour absolute session expiry, with real staff activity extending inactivity but background refresh not extending it?

## Answer

User confirmed all these limits and activity semantics. Access expiry is also bounded by remaining idle/absolute lifetime. Activity renewal cannot extend absolute expiry. Document the missing frontend activity integration; no application implementation is authorized.
