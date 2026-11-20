# Use one attendance endpoint

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Answer

The user chose `POST /v1/staff/bookings/{id}/attendance` for both attendance transitions. Its required body contains `action` (`check_in` or `check_out`), `credential`, and the expected booking `version`. Keep separate permissions, credential eligibility and state rules for each action. Return the updated arrival projection after the atomic transition. Keep `/staff/resolve` as the credential lookup step before identity confirmation. The revised target has 40 method/path operations.
