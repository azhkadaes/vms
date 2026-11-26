# Preserve login across reload and access-token expiry

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Restore and refresh access while the server session remains valid, or require login after reload/access-token expiry?

## Answer

User confirmed automatic restoration and refresh while the server session is valid, then sign-in when that session expires. Preserve current screens. This authorizes documenting the supporting refresh operation as an explicit integration requirement; it does not authorize application changes or invent session durations.
