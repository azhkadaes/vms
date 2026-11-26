# Reconcile rescheduling eligibility

Type: grilling
Label: wayfinder:grilling
Status: resolved
Assignee: current review agent
Parent: ../map.md
Blocked by: none

## Question

The current frontend permission helper allows Approved and allows derived Missed through seven days after scheduled end. Earlier approved D100 permits Pending or Approved and excludes derived Missed. Which rule should this contract use?

## Evidence

`frontendx/packages/data/src/domain.ts` can() and `docs/BACKEND-PLAN-2026-11-11.md` D100. Both use a future replacement schedule, existing access scope and version checks. The choices differ materially for pending visits and missed visits.

## Answer

18 November 2026. The user selected current frontend eligibility, superseding D100 for this contract. Require stored approved status. An Approved visit whose display status is Missed remains eligible through the end of the seventh elapsed day after the original scheduled end, inclusive. Pending and other lifecycle states are ineligible. Apply current scope, pre-mutation version and time checks before replacing the schedule. Preserve approved status on success.
