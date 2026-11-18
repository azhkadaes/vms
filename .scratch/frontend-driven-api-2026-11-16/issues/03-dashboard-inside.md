# Define the dashboard occupants count

Type: grilling
Label: wayfinder:grilling
Status: resolved
Assignee: current review agent
Parent: ../map.md
Blocked by: none

## Question

Should 'Sedang berkunjung' count all currently checked-in visits, including earlier dates, matching the linked list and checkout queue; or only visits scheduled today, matching the dashboard's fallback calculation?

## Evidence

`frontendx/apps/admin/src/pages/Dashboard.tsx`: current/inside calculation and metric links; `Reception.tsx`: no date filter for checkout. No independent endpoint is required for either choice; define `metrics.inside` precisely.

## Answer

16 November 2026: user selected all visits still checked in, including overnight visits. `metrics.inside` counts authorized booking records with stored status checked_in across dates, independent of the dashboard's today filter and pagination. It counts visits, not summed roster members. The today-only frontend fallback is a later correction. Other metrics retain their own date semantics.
