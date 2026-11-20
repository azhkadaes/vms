# Protect booking changes with an expected version

Type: grilling
Label: wayfinder:grilling
Status: resolved
Parent: ../map.md
Blocked by: none

## Question

Add a booking change counter to support the frontend's existing expected-version mutation requests?

## Answer

17 November 2026: after an explanation of concurrent staff changes, user agreed to Kunjungan.version INTEGER NOT NULL DEFAULT 1. Each committed booking change increments the counter; stale expected versions are rejected. This is contract/schema planning approval, not execution of a migration.
