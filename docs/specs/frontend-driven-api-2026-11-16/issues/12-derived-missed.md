# Derive missed visits rather than storing another status

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Store terlewat as an independent lifecycle status or derive it as the frontend currently does?

## Answer

17 November 2026: user confirmed derived terlewat/missed, with no scheduled database status update. Stored status remains approved; displayStatus becomes missed when server time is strictly later than scheduled end and no check-in has occurred. Checked-in visits never become missed. Exclude terlewat from the proposed stored enum; preserve missed in API display/filter/report enums. Do not rewrite existing data without a separate migration decision.
